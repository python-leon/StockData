# API 接口文档

## 基础信息

- **Base URL**: `http://localhost:8080/api/v1`
- **Content-Type**: `application/json`
- **字符编码**: UTF-8

## 响应格式

所有接口统一返回以下格式：

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

- `code`: 状态码，0 表示成功，非 0 表示错误
- `message`: 响应消息
- `data`: 响应数据

## 接口列表

### 1. 健康检查

**接口**: `GET /health`

**描述**: 检查服务是否正常运行

**请求示例**:
```bash
curl http://localhost:8080/api/v1/health
```

**响应示例**:
```json
{
  "code": 0,
  "message": "OK",
  "data": {
    "status": "healthy"
  }
}
```

---

### 2. 抓取股票基本信息

**接口**: `POST /fetch/stock-basic`

**描述**: 从 Tushare 抓取所有上市股票的基本信息

**请求示例**:
```bash
curl -X POST http://localhost:8080/api/v1/fetch/stock-basic
```

**响应示例**:
```json
{
  "code": 0,
  "message": "抓取成功"
}
```

---

### 3. 抓取日线数据

**接口**: `POST /fetch/daily`

**描述**: 从 Tushare 抓取股票日线数据（异步任务）

**请求参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| start_date | string | 是 | 开始日期，格式 YYYYMMDD |
| end_date | string | 是 | 结束日期，格式 YYYYMMDD |
| concurrency | int | 否 | 并发数，默认使用配置值 |

**请求示例**:
```bash
curl -X POST http://localhost:8080/api/v1/fetch/daily \
  -H "Content-Type: application/json" \
  -d '{
    "start_date": "20230101",
    "end_date": "20231231",
    "concurrency": 10
  }'
```

**响应示例**:
```json
{
  "code": 0,
  "message": "任务已启动，请查询进度"
}
```

---

### 4. 查询抓取进度

**接口**: `GET /fetch/progress/:task_id`

**描述**: 查询指定任务的抓取进度

**路径参数**:
- `task_id`: 任务ID

**请求示例**:
```bash
curl http://localhost:8080/api/v1/fetch/progress/task_1701600000
```

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "task_id": "task_1701600000",
    "start_date": "20230101",
    "end_date": "20231231",
    "status": "running",
    "progress": 45,
    "total_count": 250,
    "success_count": 112,
    "failed_count": 3,
    "error_msg": "",
    "start_time": "2023-12-03T10:00:00Z",
    "end_time": null,
    "created_at": "2023-12-03T10:00:00Z",
    "updated_at": "2023-12-03T10:30:00Z"
  }
}
```

**状态说明**:
- `pending`: 等待中
- `running`: 运行中
- `completed`: 已完成
- `failed`: 失败

---

### 5. 获取任务列表

**接口**: `GET /fetch/tasks`

**描述**: 获取所有抓取任务列表

**查询参数**:

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| page | int | 否 | 1 | 页码 |
| page_size | int | 否 | 10 | 每页数量 |

**请求示例**:
```bash
curl "http://localhost:8080/api/v1/fetch/tasks?page=1&page_size=10"
```

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": 1,
        "task_id": "task_1701600000",
        "start_date": "20230101",
        "end_date": "20231231",
        "status": "completed",
        "progress": 100,
        "total_count": 250,
        "success_count": 247,
        "failed_count": 3,
        "start_time": "2023-12-03T10:00:00Z",
        "end_time": "2023-12-03T12:00:00Z",
        "created_at": "2023-12-03T10:00:00Z",
        "updated_at": "2023-12-03T12:00:00Z"
      }
    ],
    "total": 5,
    "page": 1
  }
}
```

---

### 6. 获取股票列表

**接口**: `GET /data/stocks`

**描述**: 获取股票基本信息列表

**查询参数**:

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| page | int | 否 | 1 | 页码 |
| page_size | int | 否 | 20 | 每页数量 |

**请求示例**:
```bash
curl "http://localhost:8080/api/v1/data/stocks?page=1&page_size=20"
```

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": 1,
        "ts_code": "000001.SZ",
        "symbol": "000001",
        "name": "平安银行",
        "area": "深圳",
        "industry": "银行",
        "market": "主板",
        "list_date": "19910403",
        "list_status": "L",
        "created_at": "2023-12-03T10:00:00Z",
        "updated_at": "2023-12-03T10:00:00Z"
      }
    ],
    "total": 5000,
    "page": 1
  }
}
```

---

### 7. 获取股票详情

**接口**: `GET /data/stock/:ts_code`

**描述**: 获取指定股票的详细信息

**路径参数**:
- `ts_code`: 股票代码，如 000001.SZ

**请求示例**:
```bash
curl http://localhost:8080/api/v1/data/stock/000001.SZ
```

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "ts_code": "000001.SZ",
    "symbol": "000001",
    "name": "平安银行",
    "area": "深圳",
    "industry": "银行",
    "market": "主板",
    "list_date": "19910403",
    "list_status": "L",
    "created_at": "2023-12-03T10:00:00Z",
    "updated_at": "2023-12-03T10:00:00Z"
  }
}
```

---

### 8. 获取日线数据

**接口**: `GET /data/daily`

**描述**: 查询股票日线数据

**查询参数**:

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| ts_code | string | 否 | - | 股票代码 |
| trade_date | string | 否 | - | 交易日期 YYYYMMDD |
| start_date | string | 否 | - | 开始日期 YYYYMMDD |
| end_date | string | 否 | - | 结束日期 YYYYMMDD |
| page | int | 否 | 1 | 页码 |
| page_size | int | 否 | 100 | 每页数量 |

**请求示例**:

```bash
# 查询某只股票的数据
curl "http://localhost:8080/api/v1/data/daily?ts_code=000001.SZ&start_date=20230101&end_date=20230131"

# 查询某个日期的所有股票数据
curl "http://localhost:8080/api/v1/data/daily?trade_date=20231201"

# 日期范围查询
curl "http://localhost:8080/api/v1/data/daily?start_date=20230101&end_date=20230131&page=1&page_size=100"
```

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": 1,
        "ts_code": "000001.SZ",
        "trade_date": "20231201",
        "open": 10.50,
        "high": 10.80,
        "low": 10.45,
        "close": 10.75,
        "pre_close": 10.60,
        "change": 0.15,
        "pct_chg": 1.42,
        "vol": 123456.00,
        "amount": 1320000.50,
        "created_at": "2023-12-03T10:00:00Z",
        "updated_at": "2023-12-03T10:00:00Z"
      }
    ],
    "total": 250,
    "page": 1
  }
}
```

---

## 错误码

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| 400 | 请求参数错误 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

## 使用示例

### 完整流程示例

```bash
# 1. 检查服务健康状态
curl http://localhost:8080/api/v1/health

# 2. 抓取股票基本信息（首次运行时必须）
curl -X POST http://localhost:8080/api/v1/fetch/stock-basic

# 3. 启动日线数据抓取任务
curl -X POST http://localhost:8080/api/v1/fetch/daily \
  -H "Content-Type: application/json" \
  -d '{
    "start_date": "20230101",
    "end_date": "20231231"
  }'

# 4. 查询任务进度
curl http://localhost:8080/api/v1/fetch/progress/task_1701600000

# 5. 查询数据
curl "http://localhost:8080/api/v1/data/daily?ts_code=000001.SZ&trade_date=20231201"
```
## 注意事项

1. **速率限制**: Tushare API 有调用频率限制，建议控制并发数
2. **数据量**: 查询大量数据时建议使用分页
3. **日期格式**: 所有日期格式为 YYYYMMDD
4. **异步任务**: 数据抓取为异步任务，需要通过进度接口查询状态
