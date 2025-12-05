package models

import (
	"time"
)

// StockDaily 股票日线数据
type StockDaily struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TSCode    string    `gorm:"type:varchar(20);index:idx_ts_code_date,priority:1;not null" json:"ts_code"`                  // 股票代码
	TradeDate time.Time `gorm:"type:date;index:idx_ts_code_date,priority:2;index:idx_trade_date;not null" json:"trade_date"` // 交易日期
	Open      float64   `gorm:"type:decimal(10,2)" json:"open"`                                                              // 开盘价
	High      float64   `gorm:"type:decimal(10,2)" json:"high"`                                                              // 最高价
	Low       float64   `gorm:"type:decimal(10,2)" json:"low"`                                                               // 最低价
	Close     float64   `gorm:"type:decimal(10,2)" json:"close"`                                                             // 收盘价
	PreClose  float64   `gorm:"type:decimal(10,2)" json:"pre_close"`                                                         // 昨收价
	Change    float64   `gorm:"type:decimal(10,2)" json:"change"`                                                            // 涨跌额
	PctChg    float64   `gorm:"type:decimal(10,4)" json:"pct_chg"`                                                           // 涨跌幅
	Vol       float64   `gorm:"type:decimal(20,2)" json:"vol"`                                                               // 成交量（手）
	Amount    float64   `gorm:"type:decimal(20,2)" json:"amount"`                                                            // 成交额（千元）
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (StockDaily) TableName() string {
	return "stock_daily"
}

// StockBasic 股票基本信息
type StockBasic struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	TSCode     string    `gorm:"type:varchar(20);uniqueIndex;not null" json:"ts_code"` // 股票代码
	Symbol     string    `gorm:"type:varchar(10)" json:"symbol"`                       // 股票简称
	Name       string    `gorm:"type:varchar(50)" json:"name"`                         // 股票名称
	Area       string    `gorm:"type:varchar(20)" json:"area"`                         // 地域
	Industry   string    `gorm:"type:varchar(50)" json:"industry"`                     // 行业
	Market     string    `gorm:"type:varchar(10)" json:"market"`                       // 市场类型
	ListDate   string    `gorm:"type:varchar(8)" json:"list_date"`                     // 上市日期
	ListStatus string    `gorm:"type:varchar(1)" json:"list_status"`                   // 上市状态
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// TableName 指定表名
func (StockBasic) TableName() string {
	return "stock_basic"
}

// FetchTask 抓取任务记录
type FetchTask struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	TaskID       string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"task_id"` // 任务ID
	StartDate    string     `gorm:"type:varchar(8)" json:"start_date"`                    // 开始日期
	EndDate      string     `gorm:"type:varchar(8)" json:"end_date"`                      // 结束日期
	Status       string     `gorm:"type:varchar(20)" json:"status"`                       // 状态：pending/running/completed/failed
	Progress     int        `gorm:"type:int" json:"progress"`                             // 进度（0-100）
	TotalCount   int        `gorm:"type:int" json:"total_count"`                          // 总数
	SuccessCount int        `gorm:"type:int" json:"success_count"`                        // 成功数
	FailedCount  int        `gorm:"type:int" json:"failed_count"`                         // 失败数
	ErrorMsg     string     `gorm:"type:text" json:"error_msg"`                           // 错误信息
	StartTime    time.Time  `json:"start_time"`
	EndTime      *time.Time `json:"end_time"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// TableName 指定表名
func (FetchTask) TableName() string {
	return "fetch_tasks"
}

// StockWeekly 股票周线数据（复权）
type StockWeekly struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TSCode    string    `gorm:"type:varchar(20);index:idx_weekly_ts_code_date,priority:1;not null" json:"ts_code"`                         // 股票代码
	TradeDate time.Time `gorm:"type:date;index:idx_weekly_ts_code_date,priority:2;index:idx_weekly_trade_date;not null" json:"trade_date"` // 交易日期（周五或月末）
	EndDate   time.Time `gorm:"type:date" json:"end_date"`                                                                                 // 计算截至日期

	// 未复权价格
	Open     float64 `gorm:"type:decimal(10,2)" json:"open"`      // 周开盘价
	High     float64 `gorm:"type:decimal(10,2)" json:"high"`      // 周最高价
	Low      float64 `gorm:"type:decimal(10,2)" json:"low"`       // 周最低价
	Close    float64 `gorm:"type:decimal(10,2)" json:"close"`     // 周收盘价
	PreClose float64 `gorm:"type:decimal(10,2)" json:"pre_close"` // 上周收盘价（除权价，前复权）

	// 前复权价格
	OpenQfq  float64 `gorm:"type:decimal(10,2)" json:"open_qfq"`  // 前复权周开盘价
	HighQfq  float64 `gorm:"type:decimal(10,2)" json:"high_qfq"`  // 前复权周最高价
	LowQfq   float64 `gorm:"type:decimal(10,2)" json:"low_qfq"`   // 前复权周最低价
	CloseQfq float64 `gorm:"type:decimal(10,2)" json:"close_qfq"` // 前复权周收盘价

	// 后复权价格
	OpenHfq  float64 `gorm:"type:decimal(10,2)" json:"open_hfq"`  // 后复权周开盘价
	HighHfq  float64 `gorm:"type:decimal(10,2)" json:"high_hfq"`  // 后复权周最高价
	LowHfq   float64 `gorm:"type:decimal(10,2)" json:"low_hfq"`   // 后复权周最低价
	CloseHfq float64 `gorm:"type:decimal(10,2)" json:"close_hfq"` // 后复权周收盘价

	// 成交数据
	Vol    float64 `gorm:"type:decimal(20,2)" json:"vol"`    // 周成交量（手）
	Amount float64 `gorm:"type:decimal(20,2)" json:"amount"` // 周成交额（千元）

	// 涨跌数据
	Change float64 `gorm:"type:decimal(10,2)" json:"change"`  // 周涨跌额
	PctChg float64 `gorm:"type:decimal(10,4)" json:"pct_chg"` // 周涨跌幅（基于除权后的昨收）

	CreatedAt time.Time `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName 指定表名
func (StockWeekly) TableName() string {
	return "stock_weekly"
}

// StockMonthly 股票月线数据
type StockMonthly struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TSCode    string    `gorm:"type:varchar(20);index:idx_monthly_ts_code_date,priority:1;not null" json:"ts_code"`                          // 股票代码
	TradeDate time.Time `gorm:"type:date;index:idx_monthly_ts_code_date,priority:2;index:idx_monthly_trade_date;not null" json:"trade_date"` // 交易日期（月末最后一个交易日）
	EndDate   time.Time `gorm:"type:date" json:"end_date"`                                                                                   // 计算截至日期

	// 未复权价格
	Open     float64 `gorm:"type:decimal(10,2)" json:"open"`      // 月开盘价
	High     float64 `gorm:"type:decimal(10,2)" json:"high"`      // 月最高价
	Low      float64 `gorm:"type:decimal(10,2)" json:"low"`       // 月最低价
	Close    float64 `gorm:"type:decimal(10,2)" json:"close"`     // 月收盘价
	PreClose float64 `gorm:"type:decimal(10,2)" json:"pre_close"` // 上月收盘价（除权价，前复权）

	// 前复权价格
	OpenQfq  float64 `gorm:"type:decimal(10,2)" json:"open_qfq"`  // 前复权月开盘价
	HighQfq  float64 `gorm:"type:decimal(10,2)" json:"high_qfq"`  // 前复权月最高价
	LowQfq   float64 `gorm:"type:decimal(10,2)" json:"low_qfq"`   // 前复权月最低价
	CloseQfq float64 `gorm:"type:decimal(10,2)" json:"close_qfq"` // 前复权月收盘价

	// 后复权价格
	OpenHfq  float64 `gorm:"type:decimal(10,2)" json:"open_hfq"`  // 后复权月开盘价
	HighHfq  float64 `gorm:"type:decimal(10,2)" json:"high_hfq"`  // 后复权月最高价
	LowHfq   float64 `gorm:"type:decimal(10,2)" json:"low_hfq"`   // 后复权月最低价
	CloseHfq float64 `gorm:"type:decimal(10,2)" json:"close_hfq"` // 后复权月收盘价

	// 成交数据
	Vol    float64 `gorm:"type:decimal(20,2)" json:"vol"`    // 月成交量（手）
	Amount float64 `gorm:"type:decimal(20,2)" json:"amount"` // 月成交额（千元）

	// 涨跌数据
	Change float64 `gorm:"type:decimal(10,2)" json:"change"`  // 月涨跌额
	PctChg float64 `gorm:"type:decimal(10,4)" json:"pct_chg"` // 月涨跌幅（基于除权后的昨收）

	CreatedAt time.Time `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName 指定表名
func (StockMonthly) TableName() string {
	return "stock_monthly"
}
