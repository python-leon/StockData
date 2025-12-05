package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"log"
	"net/http"
	"os"
	"os/signal"
	"stock_data/internal/api"
	"stock_data/internal/config"
	"stock_data/internal/database"
	"stock_data/internal/service"
	"syscall"
	"time"
)

func main() {
	cfg, err := config.LoadConfig("./config/config.yaml")
	if err != nil {
		log.Fatalf("load config error: %v", err)
	}
	// 初始化日志
	logger, err := initLogger(cfg.Log)
	if err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}
	defer func(logger *zap.Logger) {
		err := logger.Sync()
		if err != nil {

		}
	}(logger)
	logger.Info("配置加载成功")

	// 初始化数据库
	if err := database.InitDB(&cfg.Database); err != nil {
		logger.Fatal("初始化数据库失败", zap.Error(err))
	}
	defer database.Close()

	// 创建 Tushare 客户端
	tushareClient := service.NewTushareClient(&cfg.Tushare)
	logger.Info("Tushare 客户端初始化成功")

	// 创建数据抓取服务
	dataFetcher := service.NewDataFetcher(tushareClient, &cfg.Fetcher, logger)

	// 设置 Gin 模式
	gin.SetMode(cfg.Server.Mode)

	// 创建 Gin 引擎
	r := gin.Default()

	// 创建 API 处理器
	handler := api.NewHandler(dataFetcher, logger)
	handler.RegisterRoutes(r)

	// 启动服务器
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: r,
	}

	// 启动服务器
	go func() {
		logger.Info("服务器启动", zap.Int("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("服务器启动失败", zap.Error(err))
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("正在关闭服务器...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("服务器强制关闭", zap.Error(err))
	}

	logger.Info("服务器已关闭")
}

// initLogger 初始化日志
func initLogger(cfg config.LogConfig) (*zap.Logger, error) {
	// 创建日志目录
	if err := os.MkdirAll("./logs", 0755); err != nil {
		return nil, err
	}

	// 配置日志
	zapCfg := zap.NewProductionConfig()
	zapCfg.OutputPaths = []string{
		"stdout",
		cfg.File,
	}

	// 设置日志级别
	switch cfg.Level {
	case "debug":
		zapCfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		zapCfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		zapCfg.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		zapCfg.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		zapCfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	return zapCfg.Build()
}
