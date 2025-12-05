package database

import (
	"fmt"
	"stock_data/internal/config"
	"stock_data/internal/models"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDB(cfg *config.DatabaseConfig) error {
	var dialector gorm.Dialector

	dsn := cfg.GetDSN()

	switch cfg.Type {
	case "mysql":
		dialector = mysql.Open(dsn)
	case "postgres":
		dialector = postgres.Open(dsn)
	default:
		return fmt.Errorf("不支持的数据库类型: %s", cfg.Type)
	}
	// 配置 GORM
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().Local()
		},
	}
	var err error
	DB, err = gorm.Open(dialector, gormConfig)
	if err != nil {
		return err
	}
	// 获取底层数据库连接
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("获取数据库连接失败: %w", err)
	}
	// 设置连接池
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)

	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("数据库连接测试失败: %w", err)
	}

	//// 自动迁移
	//if err := autoMigrate(); err != nil {
	//	return fmt.Errorf("数据库迁移失败: %w", err)
	//}

	return nil
}

// autoMigrate 自动迁移数据库表结构
func autoMigrate() error {
	return DB.AutoMigrate(
		&models.StockDaily{},
	)
}

// Close 关闭数据库连接
func Close() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

// GetDB 获取数据库实例
func GetDB() *gorm.DB {
	return DB
}
