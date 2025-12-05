package service

import (
	"context"
	"fmt"
	"sort"
	"stock_data/internal/config"
	"stock_data/internal/database"
	"stock_data/internal/models"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

// DataFetcher 数据抓取服务
type DataFetcher struct {
	tushareClient *TushareClient
	db            *gorm.DB
	config        *config.FetcherConfig
	logger        *zap.Logger
	rateLimiter   *time.Ticker
}

// NewDataFetcher 创建数据抓取服务
func NewDataFetcher(tushareClient *TushareClient, cfg *config.FetcherConfig, logger *zap.Logger) *DataFetcher {
	return &DataFetcher{
		tushareClient: tushareClient,
		db:            database.GetDB(),
		config:        cfg,
		logger:        logger,
		rateLimiter:   time.NewTicker(time.Minute / time.Duration(cfg.RateLimit)),
	}
}

// FetchStockBasic 抓取股票基本信息
func (f *DataFetcher) FetchStockBasic() error {
	f.logger.Info("开始抓取股票基本信息")

	stocks, err := f.tushareClient.GetStockBasic()
	if err != nil {
		return fmt.Errorf("获取股票基本信息失败: %w", err)
	}

	f.logger.Info("获取股票列表成功", zap.Int("count", len(stocks)))

	// 批量插入
	if err := f.batchInsertStockBasic(stocks); err != nil {
		return fmt.Errorf("保存股票基本信息失败: %w", err)
	}

	f.logger.Info("股票基本信息抓取完成", zap.Int("total", len(stocks)))
	return nil
}

// FetchDailyData 抓取日线数据
func (f *DataFetcher) FetchDailyData(ctx context.Context, startDate, endDate string) (*models.FetchTask, error) {
	// 创建任务记录
	task := &models.FetchTask{
		TaskID:    fmt.Sprintf("task_%d", time.Now().Unix()),
		StartDate: startDate,
		EndDate:   endDate,
		Status:    "running",
		StartTime: time.Now(),
	}

	if err := f.db.Create(task).Error; err != nil {
		return nil, fmt.Errorf("创建任务记录失败: %w", err)
	}

	f.logger.Info("开始抓取日线数据",
		zap.String("task_id", task.TaskID),
		zap.String("start_date", startDate),
		zap.String("end_date", endDate))

	// 获取股票列表
	var stocks []models.StockBasic
	if err := f.db.Find(&stocks).Error; err != nil {
		return nil, fmt.Errorf("获取股票列表失败: %w", err)
	}

	// 生成日期列表
	dates := f.generateDateRange(startDate, endDate)

	totalTasks := len(stocks) * len(dates)
	task.TotalCount = totalTasks
	f.db.Save(task)

	f.logger.Info("任务规模",
		zap.Int("stocks", len(stocks)),
		zap.Int("dates", len(dates)),
		zap.Int("total_tasks", totalTasks))

	// 并发抓取
	var successCount, failedCount int64
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, f.config.Concurrency)

	for _, stock := range stocks {
		for _, date := range dates {
			wg.Add(1)
			go func(tsCode, tradeDate string) {
				defer wg.Done()

				// 限流
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				<-f.rateLimiter.C

				// 抓取数据
				if err := f.fetchAndSaveDailyData(tsCode, tradeDate); err != nil {
					atomic.AddInt64(&failedCount, 1)
					f.logger.Error("抓取失败",
						zap.String("ts_code", tsCode),
						zap.String("trade_date", tradeDate),
						zap.Error(err))
				} else {
					atomic.AddInt64(&successCount, 1)
				}

				// 更新进度
				total := atomic.LoadInt64(&successCount) + atomic.LoadInt64(&failedCount)
				progress := int(total * 100 / int64(totalTasks))

				if total%100 == 0 {
					f.updateTaskProgress(task.ID, progress, int(successCount), int(failedCount))
					f.logger.Info("抓取进度",
						zap.Int("progress", progress),
						zap.Int64("success", successCount),
						zap.Int64("failed", failedCount))
				}
			}(stock.TSCode, date)
		}
	}

	wg.Wait()

	// 更新任务状态
	now := time.Now()
	task.EndTime = &now
	task.Status = "completed"
	task.Progress = 100
	task.SuccessCount = int(successCount)
	task.FailedCount = int(failedCount)
	f.db.Save(task)

	f.logger.Info("日线数据抓取完成",
		zap.String("task_id", task.TaskID),
		zap.Int64("success", successCount),
		zap.Int64("failed", failedCount))

	return task, nil
}

// FetchDailyDataOptimized 优化版：按日期并发抓取
func (f *DataFetcher) FetchDailyDataOptimized(ctx context.Context, startDate, endDate string) (*models.FetchTask, error) {
	// 创建任务记录
	task := &models.FetchTask{
		TaskID:    fmt.Sprintf("task_%d", time.Now().Unix()),
		StartDate: startDate,
		EndDate:   endDate,
		Status:    "running",
		StartTime: time.Now(),
	}

	if err := f.db.Create(task).Error; err != nil {
		return nil, fmt.Errorf("创建任务记录失败: %w", err)
	}

	// 生成日期列表
	dates := f.generateDateRange(startDate, endDate)
	task.TotalCount = len(dates)
	f.db.Save(task)

	f.logger.Info("开始抓取日线数据（按日期）",
		zap.String("task_id", task.TaskID),
		zap.Int("total_dates", len(dates)))

	// 使用 errgroup 并发抓取
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(f.config.Concurrency)

	var successCount, failedCount int64

	for i, date := range dates {
		date := date
		index := i

		g.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			<-f.rateLimiter.C

			// 抓取该日期的所有数据
			dailyData, err := f.tushareClient.GetDailyData(date, "")
			if err != nil {
				atomic.AddInt64(&failedCount, 1)
				f.logger.Error("抓取日期数据失败",
					zap.String("date", date),
					zap.Error(err))
				return nil // 不中断其他任务
			}

			// 批量保存
			if len(dailyData) > 0 {
				if err := f.batchInsertDailyData(dailyData); err != nil {
					atomic.AddInt64(&failedCount, 1)
					f.logger.Error("保存日期数据失败",
						zap.String("date", date),
						zap.Error(err))
				} else {
					atomic.AddInt64(&successCount, 1)
					f.logger.Info("日期数据保存成功",
						zap.String("date", date),
						zap.Int("count", len(dailyData)))
				}
			}

			// 更新进度
			progress := (index + 1) * 100 / len(dates)
			f.updateTaskProgress(task.ID, progress, int(successCount), int(failedCount))

			return nil
		})
	}

	// 等待所有任务完成
	if err := g.Wait(); err != nil {
		f.logger.Error("抓取过程出错", zap.Error(err))
	}

	// 更新任务状态
	now := time.Now()
	task.EndTime = &now
	task.Status = "completed"
	task.Progress = 100
	task.SuccessCount = int(successCount)
	task.FailedCount = int(failedCount)
	f.db.Save(task)

	f.logger.Info("日线数据抓取完成",
		zap.String("task_id", task.TaskID),
		zap.Int64("success", successCount),
		zap.Int64("failed", failedCount))

	return task, nil
}

// fetchAndSaveDailyData 抓取并保存单条日线数据
func (f *DataFetcher) fetchAndSaveDailyData(tsCode, tradeDate string) error {
	dailyData, err := f.tushareClient.GetDailyData(tradeDate, tsCode)
	if err != nil {
		return err
	}

	if len(dailyData) == 0 {
		return nil
	}

	return f.batchInsertDailyData(dailyData)
}

// batchInsertStockBasic 批量插入股票基本信息
func (f *DataFetcher) batchInsertStockBasic(stocks []StockBasicData) error {
	batchSize := f.config.BatchSize

	for i := 0; i < len(stocks); i += batchSize {
		end := i + batchSize
		if end > len(stocks) {
			end = len(stocks)
		}

		batch := stocks[i:end]
		records := make([]models.StockBasic, 0, len(batch))

		for _, stock := range batch {
			records = append(records, models.StockBasic{
				TSCode:     stock.TSCode,
				Symbol:     stock.Symbol,
				Name:       stock.Name,
				Area:       stock.Area,
				Industry:   stock.Industry,
				Market:     stock.Market,
				ListDate:   stock.ListDate,
				ListStatus: stock.ListStatus,
			})
		}

		// 使用 ON CONFLICT 处理重复数据（仅 PostgreSQL）
		if err := f.db.CreateInBatches(records, batchSize).Error; err != nil {
			return err
		}
	}

	return nil
}

// batchInsertDailyData 批量插入日线数据
func (f *DataFetcher) batchInsertDailyData(dailyData []StockDailyData) error {
	batchSize := f.config.BatchSize

	for i := 0; i < len(dailyData); i += batchSize {
		end := i + batchSize
		if end > len(dailyData) {
			end = len(dailyData)
		}

		batch := dailyData[i:end]
		records := make([]models.StockDaily, 0, len(batch))

		for _, data := range batch {
			// 解析日期字符串为 time.Time
			tradeDate, err := time.Parse("20060102", data.TradeDate)
			if err != nil {
				f.logger.Warn("日期格式错误", zap.String("trade_date", data.TradeDate))
			}
			records = append(records, models.StockDaily{
				TSCode:    data.TSCode,
				TradeDate: tradeDate,
				Open:      data.Open,
				High:      data.High,
				Low:       data.Low,
				Close:     data.Close,
				PreClose:  data.PreClose,
				Change:    data.Change,
				PctChg:    data.PctChg,
				Vol:       data.Vol,
				Amount:    data.Amount,
			})
		}

		if err := f.db.CreateInBatches(records, batchSize).Error; err != nil {
			return err
		}
	}

	return nil
}

// updateTaskProgress 更新任务进度
func (f *DataFetcher) updateTaskProgress(taskID uint, progress, successCount, failedCount int) {
	f.db.Model(&models.FetchTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
		"progress":      progress,
		"success_count": successCount,
		"failed_count":  failedCount,
	})
}

// generateDateRange 生成日期范围（使用真实交易日历）
func (f *DataFetcher) generateDateRange(startDate, endDate string) []string {
	// 调用 getTradeDates 获取真实交易日历
	tradeDates, err := f.getTradeDates(startDate, endDate)
	if err != nil {
		f.logger.Error("获取交易日历失败，降级为周末过滤",
			zap.String("start_date", startDate),
			zap.String("end_date", endDate),
			zap.Error(err))

		// 降级方案：简单过滤周末
		return f.generateDateRangeFallback(startDate, endDate)
	}

	return tradeDates
}

// generateDateRangeFallback 生成日期范围的降级方案（仅过滤周末）
func (f *DataFetcher) generateDateRangeFallback(startDate, endDate string) []string {
	start, _ := time.Parse("20060102", startDate)
	end, _ := time.Parse("20060102", endDate)

	var dates []string
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		// 只包含工作日
		if d.Weekday() != time.Saturday && d.Weekday() != time.Sunday {
			dates = append(dates, d.Format("20060102"))
		}
	}

	f.logger.Warn("使用降级方案生成日期列表",
		zap.Int("date_count", len(dates)))

	return dates
}

// GetTaskProgress 获取任务进度
func (f *DataFetcher) GetTaskProgress(taskID string) (*models.FetchTask, error) {
	var task models.FetchTask
	if err := f.db.Where("task_id = ?", taskID).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// getTradeDates 获取交易日列表
func (f *DataFetcher) getTradeDates(startDate, endDate string) ([]string, error) {
	// 从 Tushare 获取交易日历
	calData, err := f.tushareClient.GetTradeCal(startDate, endDate, 1) // 1 = 只获取交易日
	if err != nil {
		return nil, fmt.Errorf("调用 Tushare API 失败: %w", err)
	}

	// 提取交易日期
	tradeDates := make([]string, 0, len(calData))
	for _, cal := range calData {
		if cal.IsOpen == 1 {
			tradeDates = append(tradeDates, cal.CalDate)
		}
	}

	// 按日期排序（从旧到新）
	sort.Strings(tradeDates)

	return tradeDates, nil
}

// FetchWeeklyData 抓取周线数据
func (f *DataFetcher) FetchWeeklyData(ctx context.Context, startDate, endDate string) (*models.FetchTask, error) {
	// 创建任务记录
	task := &models.FetchTask{
		TaskID:    fmt.Sprintf("weekly_task_%d", time.Now().Unix()),
		StartDate: startDate,
		EndDate:   endDate,
		Status:    "running",
		StartTime: time.Now(),
	}

	if err := f.db.Create(task).Error; err != nil {
		return nil, fmt.Errorf("创建任务记录失败: %w", err)
	}

	f.logger.Info("开始抓取周线数据",
		zap.String("task_id", task.TaskID),
		zap.String("start_date", startDate),
		zap.String("end_date", endDate))

	// 生成周线日期范围（每周最后一个交易日）
	dates := f.generateWeekDateRange(startDate, endDate)
	task.TotalCount = len(dates)
	f.db.Save(task)
	f.logger.Info("任务规模",
		zap.Int("weeks", len(dates)),
		zap.Int("total_tasks", task.TotalCount))
	// 并发抓取
	var successCount, failedCount int64
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(f.config.Concurrency)

	for _, date := range dates {
		week_date := date
		g.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			// 限流
			<-f.rateLimiter.C

			// 抓取周线数据
			weeklyData, err := f.tushareClient.GetWeeklyData(week_date)
			if err != nil {
				atomic.AddInt64(&failedCount, 1)
				f.logger.Error("抓取周线数据失败",
					zap.String("date", date),
					zap.Error(err))
				return nil // 不中断其他任务
			}

			// 批量保存
			if len(weeklyData) > 0 {
				if err := f.batchInsertWeeklyData(weeklyData); err != nil {
					atomic.AddInt64(&failedCount, 1)
					f.logger.Error("保存周线数据失败",
						zap.String("date", date),
						zap.Error(err))
				} else {
					atomic.AddInt64(&successCount, 1)
					f.logger.Info("周线数据保存成功",
						zap.String("date", date),
						zap.Int("count", len(weeklyData)))
				}
			} else {
				// 无数据也算成功
				atomic.AddInt64(&successCount, 1)
				f.logger.Debug("该日期无周线数据",
					zap.String("date", date))
			}

			// 更新进度
			total := atomic.LoadInt64(&successCount) + atomic.LoadInt64(&failedCount)
			progress := int(total * 100 / int64(task.TotalCount))
			f.updateTaskProgress(task.ID, progress, int(successCount), int(failedCount))

			return nil
		})
	}

	// 等待所有任务完成
	if err := g.Wait(); err != nil {
		f.logger.Error("抓取过程出错", zap.Error(err))
	}

	// 更新任务状态
	now := time.Now()
	task.EndTime = &now
	task.Status = "completed"
	task.Progress = 100
	task.SuccessCount = int(successCount)
	task.FailedCount = int(failedCount)
	f.db.Save(task)

	f.logger.Info("周线数据抓取完成",
		zap.String("task_id", task.TaskID),
		zap.Int64("success", successCount),
		zap.Int64("failed", failedCount),
		zap.Duration("elapsed", time.Since(task.StartTime)))

	return task, nil
}

// batchInsertWeeklyData 批量插入周线数据
func (f *DataFetcher) batchInsertWeeklyData(weeklyData []StockWeeklyData) error {
	batchSize := f.config.BatchSize

	for i := 0; i < len(weeklyData); i += batchSize {
		end := i + batchSize
		if end > len(weeklyData) {
			end = len(weeklyData)
		}

		batch := weeklyData[i:end]
		records := make([]models.StockWeekly, 0, len(batch))

		for _, data := range batch {
			// 解析交易日期
			tradeDate, err := time.Parse("20060102", data.TradeDate)
			if err != nil {
				f.logger.Warn("周线交易日期格式错误", zap.String("trade_date", data.TradeDate))
				continue
			}

			// 解析截至日期
			endDate, err := time.Parse("20060102", data.EndDate)
			if err != nil {
				f.logger.Warn("周线end_date日期格式错误", zap.String("end_date", data.EndDate))
				continue
			}

			records = append(records, models.StockWeekly{
				TSCode:    data.TSCode,
				TradeDate: tradeDate,
				EndDate:   endDate,

				// 未复权价格
				Open:     data.Open,
				High:     data.High,
				Low:      data.Low,
				Close:    data.Close,
				PreClose: data.PreClose,

				// 前复权价格
				OpenQfq:  data.OpenQfq,
				HighQfq:  data.HighQfq,
				LowQfq:   data.LowQfq,
				CloseQfq: data.CloseQfq,

				// 后复权价格
				OpenHfq:  data.OpenHfq,
				HighHfq:  data.HighHfq,
				LowHfq:   data.LowHfq,
				CloseHfq: data.CloseHfq,

				// 成交数据
				Vol:    data.Vol,
				Amount: data.Amount,

				// 涨跌数据
				Change: data.Change,
				PctChg: data.PctChg,
			})
		}

		if err := f.db.CreateInBatches(records, batchSize).Error; err != nil {
			return err
		}
	}

	return nil
}

// generateWeekDateRange 生成周线交易日期范围（每周最后一个交易日）
func (f *DataFetcher) generateWeekDateRange(startDate, endDate string) []string {
	// 获取所有交易日
	allTradeDates, err := f.getTradeDates(startDate, endDate)
	if err != nil {
		f.logger.Error("获取交易日历失败，降级为周末过滤",
			zap.String("start_date", startDate),
			zap.String("end_date", endDate),
			zap.Error(err))

		// 降级方案：使用周末过滤并获取每周最后一个交易日
		return f.generateWeekDateRangeFallback(startDate, endDate)
	}
	if len(allTradeDates) == 0 {
		return []string{}
	}
	// 将字符串日期转换为 time.Time
	var dates []time.Time
	for _, dateStr := range allTradeDates {
		if t, err := time.Parse("20060102", dateStr); err == nil {
			dates = append(dates, t)
		}
	}
	if len(dates) == 0 {
		return []string{}
	}

	// 按周分组，获取每周最后一个交易日
	var weekEndDates []string
	currentWeekYear, currentWeekNum := dates[0].ISOWeek()
	lastDateOfWeek := dates[0]

	for _, date := range dates {
		weekYear, weekNum := date.ISOWeek()

		// 如果进入新的周
		if weekYear != currentWeekYear || weekNum != currentWeekNum {
			// 添加上周的最后一个交易日
			weekEndDates = append(weekEndDates, lastDateOfWeek.Format("20060102"))

			// 更新当前周信息
			currentWeekYear = weekYear
			currentWeekNum = weekNum
		}

		// 更新本周最后一个交易日
		lastDateOfWeek = date
	}

	// 添加最后一周的最后一个交易日
	weekEndDates = append(weekEndDates, lastDateOfWeek.Format("20060102"))

	f.logger.Info("生成周线交易日期完成",
		zap.Int("total_weeks", len(weekEndDates)))

	return weekEndDates
}

// generateDateRangeFallback 生成日期范围的降级方案（仅过滤周末）
func (f *DataFetcher) generateWeekDateRangeFallback(startDate, endDate string) []string {
	start, _ := time.Parse("20060102", startDate)
	end, _ := time.Parse("20060102", endDate)

	var dates []string
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		// 只包含工作日
		if d.Weekday() != time.Saturday && d.Weekday() != time.Sunday {
			dates = append(dates, d.Format("20060102"))
		}
	}

	f.logger.Warn("使用降级方案生成日期列表",
		zap.Int("date_count", len(dates)))

	return dates
}

// FetchMonthlyData 抓取月线数据（仅获取每月最后一个交易日的数据）
func (f *DataFetcher) FetchMonthlyData(ctx context.Context, startDate, endDate string) (*models.FetchTask, error) {
	// 创建任务记录
	task := &models.FetchTask{
		TaskID:    fmt.Sprintf("monthly_task_%d", time.Now().Unix()),
		StartDate: startDate,
		EndDate:   endDate,
		Status:    "running",
		StartTime: time.Now(),
	}

	if err := f.db.Create(task).Error; err != nil {
		return nil, fmt.Errorf("创建任务记录失败: %w", err)
	}

	// 生成月末日期列表
	monthEndDates := f.generateMonthEndDates(startDate, endDate)
	task.TotalCount = len(monthEndDates)
	f.db.Save(task)

	f.logger.Info("开始抓取月线数据",
		zap.String("task_id", task.TaskID),
		zap.Int("total_months", len(monthEndDates)))

	// 使用 errgroup 并发抓取
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(f.config.Concurrency)

	var successCount, failedCount int64

	for i, date := range monthEndDates {
		date := date
		index := i

		g.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			<-f.rateLimiter.C

			// 抓取该月末日期的所有数据
			monthlyData, err := f.tushareClient.GetMonthlyData(date, "")
			if err != nil {
				atomic.AddInt64(&failedCount, 1)
				f.logger.Error("抓取月线数据失败",
					zap.String("date", date),
					zap.Error(err))
				return nil
			}

			// 批量保存
			if len(monthlyData) > 0 {
				if err := f.batchInsertMonthlyData(monthlyData); err != nil {
					atomic.AddInt64(&failedCount, 1)
					f.logger.Error("保存月线数据失败",
						zap.String("date", date),
						zap.Error(err))
				} else {
					atomic.AddInt64(&successCount, 1)
					f.logger.Info("月线数据保存成功",
						zap.String("date", date),
						zap.Int("count", len(monthlyData)))
				}
			}

			// 更新进度
			progress := (index + 1) * 100 / len(monthEndDates)
			f.updateTaskProgress(task.ID, progress, int(successCount), int(failedCount))

			return nil
		})
	}

	// 等待所有任务完成
	if err := g.Wait(); err != nil {
		f.logger.Error("抓取过程出错", zap.Error(err))
	}

	// 更新任务状态
	now := time.Now()
	task.EndTime = &now
	task.Status = "completed"
	task.Progress = 100
	task.SuccessCount = int(successCount)
	task.FailedCount = int(failedCount)
	f.db.Save(task)

	f.logger.Info("月线数据抓取完成",
		zap.String("task_id", task.TaskID),
		zap.Int64("success", successCount),
		zap.Int64("failed", failedCount))

	return task, nil
}

// 通过调用Tushare交易日历接口获取真实的交易日，并找出每月最后一个交易日
func (f *DataFetcher) generateMonthEndDates(startDate, endDate string) []string {
	start, _ := time.Parse("20060102", startDate)
	end, _ := time.Parse("20060102", endDate)

	var dates []string

	// 首先尝试调用Tushare交易日历接口获取真实交易日
	tradeCals, err := f.tushareClient.GetTradeCal(startDate, endDate, 1) // 1表示只获取交易日
	if err != nil {
		f.logger.Warn("获取交易日历失败，使用降级方案（排除周末）",
			zap.Error(err))
		return f.generateMonthEndDatesFallback(startDate, endDate)
	}

	if len(tradeCals) == 0 {
		f.logger.Warn("交易日历为空，使用降级方案")
		return f.generateMonthEndDatesFallback(startDate, endDate)
	}

	// 按月份分组，找出每月最后一个交易日
	monthlyLastTrade := make(map[string]string) // key: YYYYMM, value: 最后一个交易日

	for _, cal := range tradeCals {
		if cal.IsOpen == 1 {
			// 提取年月 YYYYMM
			if len(cal.CalDate) >= 6 {
				yearMonth := cal.CalDate[:6]
				// 保留每月最大的日期（最后一个交易日）
				if existing, ok := monthlyLastTrade[yearMonth]; !ok || cal.CalDate > existing {
					monthlyLastTrade[yearMonth] = cal.CalDate
				}
			}
		}
	}

	// 按时间顺序排序并生成日期列表
	current := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, start.Location())
	for !current.After(end) {
		yearMonth := current.Format("200601")
		if lastTradeDate, ok := monthlyLastTrade[yearMonth]; ok {
			dates = append(dates, lastTradeDate)
			f.logger.Debug("找到月末交易日",
				zap.String("year_month", yearMonth),
				zap.String("last_trade_date", lastTradeDate))
		}
		// 移动到下个月
		current = current.AddDate(0, 1, 0)
	}

	f.logger.Info("生成月末交易日列表完成",
		zap.Int("total_months", len(dates)),
		zap.Strings("dates", dates))

	return dates
}

// generateMonthEndDatesFallback 降级方案：简单排除周末
func (f *DataFetcher) generateMonthEndDatesFallback(startDate, endDate string) []string {
	start, _ := time.Parse("20060102", startDate)
	end, _ := time.Parse("20060102", endDate)

	var dates []string

	// 调整到当月第一天
	current := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, start.Location())

	for !current.After(end) {
		// 获取当月最后一天
		monthEnd := current.AddDate(0, 1, -1)

		// 确保不超过结束日期
		if monthEnd.After(end) {
			monthEnd = end
		}

		// 如果月末不是交易日（周末），回退到最后一个周五
		switch monthEnd.Weekday() {
		case time.Saturday:
			monthEnd = monthEnd.AddDate(0, 0, -1) // 周六退1天到周五
		case time.Sunday:
			monthEnd = monthEnd.AddDate(0, 0, -2) // 周日退2天到周五
		}

		// 确保不早于开始日期
		if !monthEnd.Before(start) {
			dates = append(dates, monthEnd.Format("20060102"))
		}

		// 移动到下个月
		current = current.AddDate(0, 1, 0)
	}

	return dates
}

// batchInsertMonthlyData 批量插入月线数据
func (f *DataFetcher) batchInsertMonthlyData(monthlyData []StockMonthlyData) error {
	batchSize := f.config.BatchSize

	for i := 0; i < len(monthlyData); i += batchSize {
		end := i + batchSize
		if end > len(monthlyData) {
			end = len(monthlyData)
		}

		batch := monthlyData[i:end]
		records := make([]models.StockMonthly, 0, len(batch))

		for _, data := range batch {
			records = append(records, models.StockMonthly{
				TSCode:    data.TSCode,
				TradeDate: data.TradeDate,
				EndDate:   data.EndDate,

				// 未复权价格
				Open:     data.Open,
				High:     data.High,
				Low:      data.Low,
				Close:    data.Close,
				PreClose: data.PreClose,

				// 前复权价格
				OpenQfq:  data.OpenQfq,
				HighQfq:  data.HighQfq,
				LowQfq:   data.LowQfq,
				CloseQfq: data.CloseQfq,

				// 后复权价格
				OpenHfq:  data.OpenHfq,
				HighHfq:  data.HighHfq,
				LowHfq:   data.LowHfq,
				CloseHfq: data.CloseHfq,

				// 成交数据
				Vol:    data.Vol,
				Amount: data.Amount,

				// 涨跌数据
				Change: data.Change,
				PctChg: data.PctChg,
			})
		}

		if err := f.db.CreateInBatches(records, batchSize).Error; err != nil {
			return err
		}
	}

	return nil
}
