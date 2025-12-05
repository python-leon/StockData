package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"stock_data/internal/config"
	"time"
)

// TushareClient Tushare API 客户端
type TushareClient struct {
	token   string
	baseURL string
	timeout time.Duration
	retry   int
	client  *http.Client
}

// TushareRequest Tushare API 请求结构
type TushareRequest struct {
	APIName string                 `json:"api_name"`
	Token   string                 `json:"token"`
	Params  map[string]interface{} `json:"params"`
	Fields  string                 `json:"fields,omitempty"`
}

// TushareResponse Tushare API 响应结构
type TushareResponse struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

// TushareData 数据结构
type TushareData struct {
	Fields []string        `json:"fields"`
	Items  [][]interface{} `json:"items"`
}

// StockDailyData 日线数据
type StockDailyData struct {
	TSCode    string  `json:"ts_code"`
	TradeDate string  `json:"trade_date"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Close     float64 `json:"close"`
	PreClose  float64 `json:"pre_close"`
	Change    float64 `json:"change"`
	PctChg    float64 `json:"pct_chg"`
	Vol       float64 `json:"vol"`
	Amount    float64 `json:"amount"`
}

// StockBasicData 股票基本信息
type StockBasicData struct {
	TSCode     string `json:"ts_code"`
	Symbol     string `json:"symbol"`
	Name       string `json:"name"`
	Area       string `json:"area"`
	Industry   string `json:"industry"`
	Market     string `json:"market"`
	ListDate   string `json:"list_date"`
	ListStatus string `json:"list_status"`
}

// TradeCal 交易日历
type TradeCal struct {
	Exchange     string `json:"exchange"`      // 交易所 SSE上交所 SZSE深交所
	CalDate      string `json:"cal_date"`      // 日历日期
	IsOpen       int    `json:"is_open"`       // 是否交易 0休市 1交易
	PreTradeDate string `json:"pretrade_date"` // 上一个交易日
}

// StockWeeklyData 周线数据
type StockWeeklyData struct {
	TSCode    string `json:"ts_code"`    // 股票代码
	TradeDate string `json:"trade_date"` // 交易日期（周五或月末）
	EndDate   string `json:"end_date"`   // 计算截至日期

	// 未复权价格
	Open     float64 `json:"open"`
	High     float64 `json:"high"`
	Low      float64 `json:"low"`
	Close    float64 `json:"close"`
	PreClose float64 `json:"pre_close"`

	// 前复权价格
	OpenQfq  float64 `json:"open_qfq"`
	HighQfq  float64 `json:"high_qfq"`
	LowQfq   float64 `json:"low_qfq"`
	CloseQfq float64 `json:"close_qfq"`

	// 后复权价格
	OpenHfq  float64 `json:"open_hfq"`
	HighHfq  float64 `json:"high_hfq"`
	LowHfq   float64 `json:"low_hfq"`
	CloseHfq float64 `json:"close_hfq"`

	// 成交数据
	Vol    float64 `json:"vol"`
	Amount float64 `json:"amount"`

	// 涨跌数据
	Change float64 `json:"change"`
	PctChg float64 `json:"pct_chg"`
}

// StockMonthlyData 月线数据
type StockMonthlyData struct {
	TSCode    string    `json:"ts_code"`
	TradeDate time.Time `json:"trade_date"`
	EndDate   time.Time `json:"end_date"`

	// 未复权价格
	Open     float64 `json:"open"`
	High     float64 `json:"high"`
	Low      float64 `json:"low"`
	Close    float64 `json:"close"`
	PreClose float64 `json:"pre_close"`

	// 前复权价格
	OpenQfq  float64 `json:"open_qfq"`
	HighQfq  float64 `json:"high_qfq"`
	LowQfq   float64 `json:"low_qfq"`
	CloseQfq float64 `json:"close_qfq"`

	// 后复权价格
	OpenHfq  float64 `json:"open_hfq"`
	HighHfq  float64 `json:"high_hfq"`
	LowHfq   float64 `json:"low_hfq"`
	CloseHfq float64 `json:"close_hfq"`

	// 成交数据
	Vol    float64 `json:"vol"`
	Amount float64 `json:"amount"`

	// 涨跌数据
	Change float64 `json:"change"`
	PctChg float64 `json:"pct_chg"`
}

// NewTushareClient 创建 Tushare 客户端
func NewTushareClient(cfg *config.TushareConfig) *TushareClient {
	return &TushareClient{
		token:   cfg.Token,
		baseURL: cfg.BaseURL,
		timeout: time.Duration(cfg.Timeout) * time.Second,
		retry:   cfg.Retry,
		client: &http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
		},
	}
}

// request 发送请求
func (c *TushareClient) request(apiName string, params map[string]interface{}, fields string) (*TushareData, error) {
	reqData := TushareRequest{
		APIName: apiName,
		Token:   c.token,
		Params:  params,
		Fields:  fields,
	}

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	var resp *TushareResponse
	var lastErr error

	// 重试机制
	for i := 0; i <= c.retry; i++ {
		resp, lastErr = c.doRequest(jsonData)
		if lastErr == nil && resp.Code == 0 {
			break
		}
		if i < c.retry {
			time.Sleep(time.Second * time.Duration(i+1))
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}

	if resp.Code != 0 {
		return nil, fmt.Errorf("API 返回错误: %s", resp.Msg)
	}

	var data TushareData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, fmt.Errorf("解析响应数据失败: %w", err)
	}

	return &data, nil
}

// doRequest 执行 HTTP 请求
func (c *TushareClient) doRequest(jsonData []byte) (*TushareResponse, error) {
	req, err := http.NewRequest("POST", c.baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	httpResp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var resp TushareResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &resp, nil
}

// GetStockBasic 获取股票基本信息
func (c *TushareClient) GetStockBasic() ([]StockBasicData, error) {
	params := map[string]interface{}{
		"list_status": "L", // 只获取上市状态的股票
	}

	data, err := c.request("stock_basic", params, "")
	if err != nil {
		return nil, err
	}

	return c.parseStockBasic(data)
}

// GetDailyData 获取日线数据
func (c *TushareClient) GetDailyData(tradeDate string, tsCode string) ([]StockDailyData, error) {
	params := map[string]interface{}{}

	if tradeDate != "" {
		params["trade_date"] = tradeDate
	}
	if tsCode != "" {
		params["ts_code"] = tsCode
	}

	data, err := c.request("daily", params, "")
	if err != nil {
		return nil, err
	}

	return c.parseDailyData(data)
}

// parseStockBasic 解析股票基本信息
func (c *TushareClient) parseStockBasic(data *TushareData) ([]StockBasicData, error) {
	result := make([]StockBasicData, 0, len(data.Items))

	fieldMap := make(map[string]int)
	for i, field := range data.Fields {
		fieldMap[field] = i
	}

	for _, item := range data.Items {
		stock := StockBasicData{
			TSCode:     getString(item, fieldMap["ts_code"]),
			Symbol:     getString(item, fieldMap["symbol"]),
			Name:       getString(item, fieldMap["name"]),
			Area:       getString(item, fieldMap["area"]),
			Industry:   getString(item, fieldMap["industry"]),
			Market:     getString(item, fieldMap["market"]),
			ListDate:   getString(item, fieldMap["list_date"]),
			ListStatus: getString(item, fieldMap["list_status"]),
		}
		result = append(result, stock)
	}

	return result, nil
}

// parseDailyData 解析日线数据
func (c *TushareClient) parseDailyData(data *TushareData) ([]StockDailyData, error) {
	result := make([]StockDailyData, 0, len(data.Items))

	fieldMap := make(map[string]int)
	for i, field := range data.Fields {
		fieldMap[field] = i
	}

	for _, item := range data.Items {
		daily := StockDailyData{
			TSCode:    getString(item, fieldMap["ts_code"]),
			TradeDate: getString(item, fieldMap["trade_date"]),
			Open:      getFloat(item, fieldMap["open"]),
			High:      getFloat(item, fieldMap["high"]),
			Low:       getFloat(item, fieldMap["low"]),
			Close:     getFloat(item, fieldMap["close"]),
			PreClose:  getFloat(item, fieldMap["pre_close"]),
			Change:    getFloat(item, fieldMap["change"]),
			PctChg:    getFloat(item, fieldMap["pct_chg"]),
			Vol:       getFloat(item, fieldMap["vol"]),
			Amount:    getFloat(item, fieldMap["amount"]),
		}
		result = append(result, daily)
	}

	return result, nil
}

// GetTradeCal 获取交易日历
// startDate: 开始日期 YYYYMMDD
// endDate: 结束日期 YYYYMMDD
// isOpen: 是否只获取交易日 1-交易日 0-休市日 空-全部
func (c *TushareClient) GetTradeCal(startDate, endDate string, isOpen int) ([]TradeCal, error) {
	params := map[string]interface{}{
		"exchange": "SSE", // 上交所
	}

	if startDate != "" {
		params["start_date"] = startDate
	}
	if endDate != "" {
		params["end_date"] = endDate
	}
	if isOpen > 0 {
		params["is_open"] = isOpen
	}

	data, err := c.request("trade_cal", params, "")
	if err != nil {
		return nil, err
	}

	return c.parseTradeCal(data)
}

// parseTradeCal 解析交易日历数据
func (c *TushareClient) parseTradeCal(data *TushareData) ([]TradeCal, error) {
	result := make([]TradeCal, 0, len(data.Items))

	fieldMap := make(map[string]int)
	for i, field := range data.Fields {
		fieldMap[field] = i
	}

	for _, item := range data.Items {
		cal := TradeCal{
			Exchange:     getString(item, fieldMap["exchange"]),
			CalDate:      getString(item, fieldMap["cal_date"]),
			IsOpen:       int(getFloat(item, fieldMap["is_open"])),
			PreTradeDate: getString(item, fieldMap["pretrade_date"]),
		}
		result = append(result, cal)
	}

	return result, nil
}

// GetWeeklyData 获取周线数据
// tradeDate: 交易日期 YYYYMMDD
func (c *TushareClient) GetWeeklyData(tradeDate string) ([]StockWeeklyData, error) {
	params := map[string]interface{}{
		"freq": "week", // 频率：周
	}
	if tradeDate != "" {
		params["trade_date"] = tradeDate
	}

	data, err := c.request("stk_week_month_adj", params, "")
	if err != nil {
		return nil, err
	}

	return c.parseWeeklyData(data)
}

// parseWeeklyData 解析周线数据
func (c *TushareClient) parseWeeklyData(data *TushareData) ([]StockWeeklyData, error) {
	result := make([]StockWeeklyData, 0, len(data.Items))

	fieldMap := make(map[string]int)
	for i, field := range data.Fields {
		fieldMap[field] = i
	}

	for _, item := range data.Items {
		weekly := StockWeeklyData{
			TSCode:    getString(item, fieldMap["ts_code"]),
			TradeDate: getString(item, fieldMap["trade_date"]),
			EndDate:   getString(item, fieldMap["end_date"]),

			// 未复权价格
			Open:     getFloat(item, fieldMap["open"]),
			High:     getFloat(item, fieldMap["high"]),
			Low:      getFloat(item, fieldMap["low"]),
			Close:    getFloat(item, fieldMap["close"]),
			PreClose: getFloat(item, fieldMap["pre_close"]),

			// 前复权价格
			OpenQfq:  getFloat(item, fieldMap["open_qfq"]),
			HighQfq:  getFloat(item, fieldMap["high_qfq"]),
			LowQfq:   getFloat(item, fieldMap["low_qfq"]),
			CloseQfq: getFloat(item, fieldMap["close_qfq"]),

			// 后复权价格
			OpenHfq:  getFloat(item, fieldMap["open_hfq"]),
			HighHfq:  getFloat(item, fieldMap["high_hfq"]),
			LowHfq:   getFloat(item, fieldMap["low_hfq"]),
			CloseHfq: getFloat(item, fieldMap["close_hfq"]),

			// 成交数据
			Vol:    getFloat(item, fieldMap["vol"]),
			Amount: getFloat(item, fieldMap["amount"]),

			// 涨跌数据
			Change: getFloat(item, fieldMap["change"]),
			PctChg: getFloat(item, fieldMap["pct_chg"]),
		}
		result = append(result, weekly)
	}

	return result, nil
}

// GetMonthlyData 获取月线数据（月线复权行情）
// tradeDate: 交易日期（月末最后一个交易日），格式 YYYYMMDD
// tsCode: 股票代码，为空则获取该日期所有股票
func (c *TushareClient) GetMonthlyData(tradeDate string, tsCode string) ([]StockMonthlyData, error) {
	params := map[string]interface{}{
		"freq": "month", // 频率：月
	}

	if tradeDate != "" {
		params["trade_date"] = tradeDate
	}
	if tsCode != "" {
		params["ts_code"] = tsCode
	}

	// 调用 Tushare 月线复权行情接口
	data, err := c.request("stk_week_month_adj", params, "")
	if err != nil {
		return nil, err
	}

	return c.parseMonthlyData(data)
}

// parseMonthlyData 解析月线数据
func (c *TushareClient) parseMonthlyData(data *TushareData) ([]StockMonthlyData, error) {
	result := make([]StockMonthlyData, 0, len(data.Items))

	fieldMap := make(map[string]int)
	for i, field := range data.Fields {
		fieldMap[field] = i
	}

	for _, item := range data.Items {
		monthly := StockMonthlyData{
			TSCode:    getString(item, fieldMap["ts_code"]),
			TradeDate: getTime(item, fieldMap["trade_date"]),
			EndDate:   getTime(item, fieldMap["end_date"]),

			// 未复权价格
			Open:     getFloat(item, fieldMap["open"]),
			High:     getFloat(item, fieldMap["high"]),
			Low:      getFloat(item, fieldMap["low"]),
			Close:    getFloat(item, fieldMap["close"]),
			PreClose: getFloat(item, fieldMap["pre_close"]),

			// 前复权价格
			OpenQfq:  getFloat(item, fieldMap["open_qfq"]),
			HighQfq:  getFloat(item, fieldMap["high_qfq"]),
			LowQfq:   getFloat(item, fieldMap["low_qfq"]),
			CloseQfq: getFloat(item, fieldMap["close_qfq"]),

			// 后复权价格
			OpenHfq:  getFloat(item, fieldMap["open_hfq"]),
			HighHfq:  getFloat(item, fieldMap["high_hfq"]),
			LowHfq:   getFloat(item, fieldMap["low_hfq"]),
			CloseHfq: getFloat(item, fieldMap["close_hfq"]),

			// 成交数据
			Vol:    getFloat(item, fieldMap["vol"]),
			Amount: getFloat(item, fieldMap["amount"]),

			// 涨跌数据
			Change: getFloat(item, fieldMap["change"]),
			PctChg: getFloat(item, fieldMap["pct_chg"]),
		}
		result = append(result, monthly)
	}

	return result, nil
}

// 辅助函数
func getString(item []interface{}, index int) string {
	if index < 0 || index >= len(item) || item[index] == nil {
		return ""
	}
	if str, ok := item[index].(string); ok {
		return str
	}
	return ""
}

func getFloat(item []interface{}, index int) float64 {
	if index < 0 || index >= len(item) || item[index] == nil {
		return 0
	}
	switch v := item[index].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case string:
		return 0
	default:
		return 0
	}
}

func getTime(item []interface{}, index int) time.Time {
	if index < 0 || index >= len(item) || item[index] == nil {
		return time.Time{}
	}
	if str, ok := item[index].(string); ok {
		// Tushare 日期格式: YYYYMMDD
		if t, err := time.Parse("20060102", str); err == nil {
			return t
		}
	}
	return time.Time{}
}
