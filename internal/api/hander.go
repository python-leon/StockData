package api

import (
	"context"
	"net/http"
	"stock_data/internal/database"
	"stock_data/internal/models"
	"stock_data/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler API 处理器
type Handler struct {
	dataFetcher *service.DataFetcher
	logger      *zap.Logger
}

// NewHandler 创建处理器
func NewHandler(dataFetcher *service.DataFetcher, logger *zap.Logger) *Handler {
	return &Handler{
		dataFetcher: dataFetcher,
		logger:      logger,
	}
}

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// FetchRequest 抓取请求
type FetchRequest struct {
	StartDate   string `json:"start_date" binding:"required"`
	EndDate     string `json:"end_date" binding:"required"`
	Concurrency int    `json:"concurrency"`
}

// RegisterRoutes 注册路由
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1")
	{
		// 健康检查
		api.GET("/health", h.HealthCheck)

		// 抓取相关
		fetch := api.Group("/fetch")
		{
			fetch.POST("/stock-basic", h.FetchStockBasic)
			fetch.POST("/daily", h.FetchDaily)
			fetch.GET("/progress/:task_id", h.GetProgress)
			fetch.GET("/tasks", h.ListTasks)
			fetch.POST("/weekly", h.FetchWeekly) // 新增：周线数据抓取
			fetch.POST("/monthly", h.FetchMonthly)
		}

		// 数据查询
		data := api.Group("/data")
		{
			data.GET("/stocks", h.GetStocks)
			data.GET("/daily", h.GetDailyData)
			data.GET("/stock/:ts_code", h.GetStockInfo)
		}
	}
}

// HealthCheck 健康检查
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "OK",
		Data: gin.H{
			"status": "healthy",
		},
	})
}

// FetchStockBasic 抓取股票基本信息
func (h *Handler) FetchStockBasic(c *gin.Context) {
	h.logger.Info("收到股票基本信息抓取请求")

	if err := h.dataFetcher.FetchStockBasic(); err != nil {
		h.logger.Error("抓取股票基本信息失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "抓取成功",
	})
}

// FetchDaily 抓取日线数据
func (h *Handler) FetchDaily(c *gin.Context) {
	var req FetchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "参数错误: " + err.Error(),
		})
		return
	}

	h.logger.Info("收到日线数据抓取请求",
		zap.String("start_date", req.StartDate),
		zap.String("end_date", req.EndDate))

	// 异步执行抓取任务
	go func() {
		ctx := context.Background()
		_, err := h.dataFetcher.FetchDailyDataOptimized(ctx, req.StartDate, req.EndDate)
		if err != nil {
			h.logger.Error("抓取日线数据失败", zap.Error(err))
		}
	}()

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "任务已启动，请查询进度",
	})
}

// GetProgress 获取抓取进度
func (h *Handler) GetProgress(c *gin.Context) {
	taskID := c.Param("task_id")

	task, err := h.dataFetcher.GetTaskProgress(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: "任务不存在",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    task,
	})
}

// ListTasks 获取任务列表
func (h *Handler) ListTasks(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	var tasks []models.FetchTask
	var total int64

	db := database.GetDB()
	db.Model(&models.FetchTask{}).Count(&total)
	db.Order("created_at desc").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&tasks)

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data: gin.H{
			"list":  tasks,
			"total": total,
			"page":  page,
		},
	})
}

// GetStocks 获取股票列表
func (h *Handler) GetStocks(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var stocks []models.StockBasic
	var total int64

	db := database.GetDB()
	db.Model(&models.StockBasic{}).Count(&total)
	db.Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&stocks)

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data: gin.H{
			"list":  stocks,
			"total": total,
			"page":  page,
		},
	})
}

// GetDailyData 获取日线数据
func (h *Handler) GetDailyData(c *gin.Context) {
	tsCode := c.Query("ts_code")
	tradeDate := c.Query("trade_date")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "100"))

	db := database.GetDB().Model(&models.StockDaily{})

	if tsCode != "" {
		db = db.Where("ts_code = ?", tsCode)
	}
	if tradeDate != "" {
		db = db.Where("trade_date = ?", tradeDate)
	}
	if startDate != "" {
		db = db.Where("trade_date >= ?", startDate)
	}
	if endDate != "" {
		db = db.Where("trade_date <= ?", endDate)
	}

	var dailyData []models.StockDaily
	var total int64

	db.Count(&total)
	db.Order("trade_date desc").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&dailyData)

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data: gin.H{
			"list":  dailyData,
			"total": total,
			"page":  page,
		},
	})
}

// GetStockInfo 获取股票详细信息
func (h *Handler) GetStockInfo(c *gin.Context) {
	tsCode := c.Param("ts_code")

	var stock models.StockBasic
	if err := database.GetDB().Where("ts_code = ?", tsCode).First(&stock).Error; err != nil {
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: "股票不存在",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    stock,
	})
}

// FetchWeekly 抓取周线数据
func (h *Handler) FetchWeekly(c *gin.Context) {
	var req FetchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "参数错误: " + err.Error(),
		})
		return
	}

	h.logger.Info("收到周线数据抓取请求",
		zap.String("start_date", req.StartDate),
		zap.String("end_date", req.EndDate))

	// 异步执行抓取任务
	go func() {
		ctx := context.Background()
		_, err := h.dataFetcher.FetchWeeklyData(ctx, req.StartDate, req.EndDate)
		if err != nil {
			h.logger.Error("抓取周线数据失败", zap.Error(err))
		}
	}()

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "周线数据抓取任务已启动，请查询进度",
	})
}

// FetchMonthly 抓取月线数据
func (h *Handler) FetchMonthly(c *gin.Context) {
	var req FetchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "参数错误: " + err.Error(),
		})
		return
	}

	h.logger.Info("收到月线数据抓取请求",
		zap.String("start_date", req.StartDate),
		zap.String("end_date", req.EndDate))

	// 异步执行抓取任务
	go func() {
		ctx := context.Background()
		_, err := h.dataFetcher.FetchMonthlyData(ctx, req.StartDate, req.EndDate)
		if err != nil {
			h.logger.Error("抓取月线数据失败", zap.Error(err))
		}
	}()

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "月线数据抓取任务已启动，请查询进度",
	})
}

// GetMonthlyData 获取月线数据
func (h *Handler) GetMonthlyData(c *gin.Context) {
	tsCode := c.Query("ts_code")
	tradeDate := c.Query("trade_date")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "100"))

	db := database.GetDB().Model(&models.StockMonthly{})

	if tsCode != "" {
		db = db.Where("ts_code = ?", tsCode)
	}
	if tradeDate != "" {
		db = db.Where("trade_date = ?", tradeDate)
	}
	if startDate != "" {
		db = db.Where("trade_date >= ?", startDate)
	}
	if endDate != "" {
		db = db.Where("trade_date <= ?", endDate)
	}

	var monthlyData []models.StockMonthly
	var total int64

	db.Count(&total)
	db.Order("trade_date desc").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&monthlyData)

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data: gin.H{
			"list":  monthlyData,
			"total": total,
			"page":  page,
		},
	})
}
