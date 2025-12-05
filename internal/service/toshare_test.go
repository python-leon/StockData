package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"stock_data/internal/config"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetDailyData_Success 测试成功获取日线数据
func TestGetDailyData_Success(t *testing.T) {
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求方法
		assert.Equal(t, "POST", r.Method)

		// 验证请求头
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// 解析请求体
		var req TushareRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		// 验证请求参数
		assert.Equal(t, "daily", req.APIName)
		assert.Equal(t, "test_token", req.Token)

		// 构造模拟响应
		mockData := TushareData{
			Fields: []string{"ts_code", "trade_date", "open", "high", "low", "close", "pre_close", "change", "pct_chg", "vol", "amount"},
			Items: [][]interface{}{
				{"000001.SZ", "20231201", 10.5, 11.0, 10.2, 10.8, 10.6, 0.2, 1.89, 123456.78, 1234567.89},
				{"000002.SZ", "20231201", 20.5, 21.0, 20.2, 20.8, 20.6, 0.2, 0.97, 234567.89, 4876543.21},
			},
		}

		dataBytes, _ := json.Marshal(mockData)

		resp := TushareResponse{
			Code: 0,
			Msg:  "success",
			Data: dataBytes,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 创建测试客户端
	cfg := &config.TushareConfig{
		Token:   "test_token",
		BaseURL: server.URL,
		Timeout: 30,
		Retry:   0,
	}
	client := NewTushareClient(cfg)

	// 执行测试
	data, err := client.GetDailyData("20231201", "")

	// 验证结果
	require.NoError(t, err)
	require.NotNil(t, data)
	assert.Len(t, data, 2)

	// 验证第一条数据
	assert.Equal(t, "000001.SZ", data[0].TSCode)
	assert.Equal(t, "20231201", data[0].TradeDate)
	assert.Equal(t, 10.5, data[0].Open)
	assert.Equal(t, 11.0, data[0].High)
	assert.Equal(t, 10.2, data[0].Low)
	assert.Equal(t, 10.8, data[0].Close)
	assert.Equal(t, 10.6, data[0].PreClose)
	assert.Equal(t, 0.2, data[0].Change)
	assert.Equal(t, 1.89, data[0].PctChg)
	assert.Equal(t, 123456.78, data[0].Vol)
	assert.Equal(t, 1234567.89, data[0].Amount)

	// 验证第二条数据
	assert.Equal(t, "000002.SZ", data[1].TSCode)
	assert.Equal(t, "20231201", data[1].TradeDate)
}

// TestGetDailyData_WithTSCode 测试按股票代码获取数据
func TestGetDailyData_WithTSCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req TushareRequest
		json.NewDecoder(r.Body).Decode(&req)

		// 验证请求参数中包含 ts_code
		assert.Equal(t, "000001.SZ", req.Params["ts_code"])

		mockData := TushareData{
			Fields: []string{"ts_code", "trade_date", "open", "high", "low", "close", "pre_close", "change", "pct_chg", "vol", "amount"},
			Items: [][]interface{}{
				{"000001.SZ", "20231201", 10.5, 11.0, 10.2, 10.8, 10.6, 0.2, 1.89, 123456.78, 1234567.89},
			},
		}

		dataBytes, _ := json.Marshal(mockData)
		resp := TushareResponse{Code: 0, Msg: "success", Data: dataBytes}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.TushareConfig{
		Token:   "test_token",
		BaseURL: server.URL,
		Timeout: 30,
		Retry:   0,
	}
	client := NewTushareClient(cfg)

	// 测试按股票代码查询
	data, err := client.GetDailyData("", "000001.SZ")

	require.NoError(t, err)
	require.Len(t, data, 1)
	assert.Equal(t, "000001.SZ", data[0].TSCode)
}

// TestGetDailyData_WithBothParams 测试同时传递 trade_date 和 ts_code
func TestGetDailyData_WithBothParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req TushareRequest
		json.NewDecoder(r.Body).Decode(&req)

		// 验证两个参数都传递了
		assert.Equal(t, "20231201", req.Params["trade_date"])
		assert.Equal(t, "000001.SZ", req.Params["ts_code"])

		mockData := TushareData{
			Fields: []string{"ts_code", "trade_date", "open", "high", "low", "close", "pre_close", "change", "pct_chg", "vol", "amount"},
			Items: [][]interface{}{
				{"000001.SZ", "20231201", 10.5, 11.0, 10.2, 10.8, 10.6, 0.2, 1.89, 123456.78, 1234567.89},
			},
		}

		dataBytes, _ := json.Marshal(mockData)
		resp := TushareResponse{Code: 0, Msg: "success", Data: dataBytes}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.TushareConfig{
		Token:   "test_token",
		BaseURL: server.URL,
		Timeout: 30,
		Retry:   0,
	}
	client := NewTushareClient(cfg)

	data, err := client.GetDailyData("20231201", "000001.SZ")

	require.NoError(t, err)
	require.Len(t, data, 1)
}

// TestGetDailyData_EmptyResult 测试返回空数据
func TestGetDailyData_EmptyResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockData := TushareData{
			Fields: []string{"ts_code", "trade_date", "open", "high", "low", "close", "pre_close", "change", "pct_chg", "vol", "amount"},
			Items:  [][]interface{}{}, // 空数据
		}

		dataBytes, _ := json.Marshal(mockData)
		resp := TushareResponse{Code: 0, Msg: "success", Data: dataBytes}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.TushareConfig{
		Token:   "test_token",
		BaseURL: server.URL,
		Timeout: 30,
		Retry:   0,
	}
	client := NewTushareClient(cfg)

	data, err := client.GetDailyData("20231201", "")

	require.NoError(t, err)
	require.NotNil(t, data)
	assert.Len(t, data, 0)
}

// TestGetDailyData_APIError 测试 API 返回错误
func TestGetDailyData_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := TushareResponse{
			Code: 4001,
			Msg:  "权限不足",
			Data: nil,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.TushareConfig{
		Token:   "test_token",
		BaseURL: server.URL,
		Timeout: 30,
		Retry:   0,
	}
	client := NewTushareClient(cfg)

	data, err := client.GetDailyData("20231201", "")

	require.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "权限不足")
}

// TestGetDailyData_NetworkError 测试网络错误
func TestGetDailyData_NetworkError(t *testing.T) {
	cfg := &config.TushareConfig{
		Token:   "test_token",
		BaseURL: "http://invalid-url-that-does-not-exist.local",
		Timeout: 1, // 1秒超时
		Retry:   0,
	}
	client := NewTushareClient(cfg)

	data, err := client.GetDailyData("20231201", "")

	require.Error(t, err)
	assert.Nil(t, data)
}

// TestGetDailyData_RetryMechanism 测试重试机制
func TestGetDailyData_RetryMechanism(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		// 前两次调用返回错误，第三次返回成功
		if callCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		mockData := TushareData{
			Fields: []string{"ts_code", "trade_date", "open", "high", "low", "close", "pre_close", "change", "pct_chg", "vol", "amount"},
			Items: [][]interface{}{
				{"000001.SZ", "20231201", 10.5, 11.0, 10.2, 10.8, 10.6, 0.2, 1.89, 123456.78, 1234567.89},
			},
		}

		dataBytes, _ := json.Marshal(mockData)
		resp := TushareResponse{Code: 0, Msg: "success", Data: dataBytes}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.TushareConfig{
		Token:   "test_token",
		BaseURL: server.URL,
		Timeout: 30,
		Retry:   3, // 重试3次
	}
	client := NewTushareClient(cfg)

	data, err := client.GetDailyData("20231201", "")

	// 应该成功（第3次重试成功）
	require.NoError(t, err)
	require.Len(t, data, 1)
	assert.Equal(t, 3, callCount)
}

// TestGetDailyData_NullValues 测试处理 null 值
func TestGetDailyData_NullValues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockData := TushareData{
			Fields: []string{"ts_code", "trade_date", "open", "high", "low", "close", "pre_close", "change", "pct_chg", "vol", "amount"},
			Items: [][]interface{}{
				// 包含 null 值的数据
				{"000001.SZ", "20231201", nil, nil, 10.2, 10.8, nil, nil, nil, nil, nil},
			},
		}

		dataBytes, _ := json.Marshal(mockData)
		resp := TushareResponse{Code: 0, Msg: "success", Data: dataBytes}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.TushareConfig{
		Token:   "test_token",
		BaseURL: server.URL,
		Timeout: 30,
		Retry:   0,
	}
	client := NewTushareClient(cfg)

	data, err := client.GetDailyData("20231201", "")

	require.NoError(t, err)
	require.Len(t, data, 1)

	// null 值应该被转换为 0
	assert.Equal(t, 0.0, data[0].Open)
	assert.Equal(t, 0.0, data[0].High)
	assert.Equal(t, 10.2, data[0].Low)
	assert.Equal(t, 10.8, data[0].Close)
}

// TestGetDailyData_Timeout 测试超时
func TestGetDailyData_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 模拟慢响应
		time.Sleep(3 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.TushareConfig{
		Token:   "test_token",
		BaseURL: server.URL,
		Timeout: 1, // 1秒超时
		Retry:   0,
	}
	client := NewTushareClient(cfg)

	data, err := client.GetDailyData("20231201", "")

	require.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "deadline exceeded")
}

// Benchmark 性能测试
func BenchmarkGetDailyData(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockData := TushareData{
			Fields: []string{"ts_code", "trade_date", "open", "high", "low", "close", "pre_close", "change", "pct_chg", "vol", "amount"},
			Items:  make([][]interface{}, 5000), // 模拟5000条数据
		}

		for i := 0; i < 5000; i++ {
			mockData.Items[i] = []interface{}{
				"000001.SZ", "20231201", 10.5, 11.0, 10.2, 10.8, 10.6, 0.2, 1.89, 123456.78, 1234567.89,
			}
		}

		dataBytes, _ := json.Marshal(mockData)
		resp := TushareResponse{Code: 0, Msg: "success", Data: dataBytes}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.TushareConfig{
		Token:   "test_token",
		BaseURL: server.URL,
		Timeout: 30,
		Retry:   0,
	}
	client := NewTushareClient(cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.GetDailyData("20231201", "")
	}
}
