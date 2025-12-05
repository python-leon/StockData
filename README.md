# Tushare 数据采集系统

## 项目概述

基于 Golang + Gin 框架的金融数据采集系统，从 Tushare 接口抓取大规模数据并存储到本地数据库。

### 数据规模
- 总数据量: 20 × 5000 × 250 = 25,000,000 条记录
- 数据源: Tushare Pro API
- 存储方案: PostgreSQL/MySQL

## 技术栈

- **语言**: Go 1.24+
- **Web框架**: Gin
- **数据库**: PostgreSQL (推荐)
- **ORM**: GORM
- **配置管理**: Viper
- **日志**: Zap
- **并发控制**: errgroup + context
- **API客户端**: 基于 net/http 封装

## 项目结构

```
.
├── cmd/
│   └── main.go                 # 程序入口
├── internal/
│   ├── api/
│   │   └── handler.go          # HTTP 处理器
│   ├── config/
│   │   └── config.go           # 配置管理
│   ├── database/
│   │   └── db.go               # 数据库连接
│   ├── models/
│   │   └── stock.go            # 数据模型
│   ├── service/
│   │   ├── tushare.go          # Tushare API 客户端
│   │   └── data_fetcher.go     # 数据抓取服务
│   └── worker/
│       └── worker_pool.go      # 工作池
├── config/
│   └── config.yaml             # 配置文件
├── logs/                       # 日志目录
├── docs/
│   ├── API.md                  # API 文档
│   ├── DATABASE.md             # 数据库设计文档
│   └── DEPLOYMENT.md           # 部署文档
├── go.mod
├── go.sum
└── README.md
```

## 快速开始

### 1. 环境准备

```bash
# 安装 Go 1.21+
# 安装 PostgreSQL 14+

# 克隆项目
cd e:\AIdata\Qoder
```

### 2. 配置数据库

```bash
# 创建数据库
psql -U postgres
CREATE DATABASE tushare_data;
```

### 3. 配置文件

编辑 `config/config.yaml`，填入你的配置：

```yaml
tushare:
  token: "your_tushare_token_here"
  
database:
  type: "postgres"
  host: "localhost"
  port: 5432
  user: "postgres"
  password: "your_password"
  dbname: "tushare_data"
```

### 4. 安装依赖

```bash
go mod download
```

### 5. 运行程序

```bash
# 开发模式
go run cmd/main.go

# 编译运行
go build -o tushare-fetcher cmd/main.go
./tushare-fetcher
```

## 核心功能

### 1. 数据抓取

- 支持并发抓取（可配置并发数）
- 断点续传功能
- 失败重试机制
- 进度监控

### 2. 数据存储

- 批量插入优化
- 数据去重
- 索引优化
- 分区表支持（可选）

### 3. API 接口

- 启动数据抓取任务
- 查询抓取进度
- 数据查询接口
- 健康检查

## API 使用

### 启动抓取任务

```bash
POST http://localhost:8080/api/v1/fetch/start
Content-Type: application/json

{
  "start_date": "20200101",
  "end_date": "20231231",
  "concurrency": 10
}
```

### 查询进度

```bash
GET http://localhost:8080/api/v1/fetch/progress
```

### 查询数据

```bash
GET http://localhost:8080/api/v1/stocks?date=20231201&limit=100
```

## 性能优化

1. **并发控制**: 使用 worker pool 控制并发数，避免 API 限流
2. **批量插入**: 每次批量插入 1000 条数据
3. **连接池**: 配置合理的数据库连接池大小
4. **索引优化**: 在关键字段上创建索引
5. **分区表**: 对于大数据量，按日期分区

## 监控与日志

- 日志文件位置: `./logs/app.log`
- 日志级别: Debug/Info/Warn/Error
- 监控指标: 抓取速度、成功率、错误统计

## 数据库选型建议

### PostgreSQL (推荐)
- ✅ 支持大数据量
- ✅ 分区表功能强大
- ✅ JSON 字段支持好
- ✅ 并发性能优秀

### MySQL
- ✅ 使用广泛
- ✅ 性能稳定
- ⚠️ 分区表支持较弱

## 注意事项

1. Tushare API 有调用频率限制，注意控制并发数
2. 建议使用积分较高的 Tushare 账号
3. 大数据量插入建议在非业务高峰期进行
4. 定期备份数据库

## 开发计划

- [ ] 基础框架搭建
- [ ] Tushare API 客户端
- [ ] 数据库设计与初始化
- [ ] 数据抓取服务
- [ ] API 接口开发
- [ ] 性能优化
- [ ] 监控和日志
- [ ] 测试与文档

## 许可证

MIT License
