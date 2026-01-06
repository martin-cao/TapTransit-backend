# TapTransit-backend

本项目是公交刷卡收费系统的后端服务，使用Go语言、Gin框架、GORM和PostgreSQL开发。

> [!NOTE]
>
> This is an course-design project for **2025 Fall** _RFID Principle and Application_ by student ID: 0055713db0a7edac6c4d1743a6ca0fcd9aebe8b6d2656fb03cb4dfadc731ed5f, 7c665580e94e95d1ca7b9652e15bbcde769bbe519d6ce3c2e7f6c6eebc92da24,
> 200f1716837e93e0f5aa07b5ec8a43d31646144c85fdbb512f5b3f718f5e2751,
> f402c025592e7097d3f0de7dede33d39b3f4d5862b8ed66007122f08c5f9581e  (SHA256)

## 项目结构

```
awesomeProject/
├── config/              # 配置文件
│   ├── config.yaml      # 应用配置
│   └── config.go        # 配置加载逻辑
├── models/              # 数据库模型
│   ├── card.go          # IC卡模型
│   ├── route.go         # 线路模型
│   ├── station.go       # 站点模型
│   ├── route_station.go # 线路-站点关联模型
│   ├── fare.go          # 票价规则模型
│   ├── transfer.go      # 换乘优惠模型
│   ├── discount_policy.go # 折扣策略模型
│   ├── transaction.go   # 交易记录模型
│   ├── device.go        # 设备模型
│   └── user.go          # 用户模型
├── controllers/         # 控制器层
│   ├── bus_controller.go      # 公交数据控制器
│   ├── card_controller.go     # 卡片控制器
│   ├── config_controller.go   # 配置控制器
│   ├── transaction_controller.go # 交易记录控制器
│   └── route_controller.go    # 线路控制器
├── services/            # 业务服务层
│   ├── fare_service.go  # 计费服务
│   ├── upload_service.go # 上传服务
│   └── card_service.go  # 卡片服务
├── routes/              # 路由配置
│   └── routes.go        # 路由定义
├── middleware/          # 中间件
│   └── logger.go        # 日志中间件
├── utils/               # 工具函数
│   ├── database.go      # 数据库初始化
│   └── redis.go         # Redis工具函数
├── main.go              # 应用入口
├── Dockerfile           # Docker构建文件
└── go.mod               # Go模块定义
```

## 功能特性

- **批量上传接口**：网关可以批量上传乘车记录
- **计费系统**：支持单程计费、换乘优惠、月度累计折扣等多种计费策略
- **卡片管理**：IC卡信息查询和管理
- **线路配置**：线路和站点信息配置查询
- **交易记录查询**：支持多条件查询交易记录

## 技术栈

- **框架**：Gin (Web框架)
- **ORM**：GORM
- **数据库**：PostgreSQL
- **缓存**：Redis
- **配置**：YAML

## 安装和运行

### 前置要求

- Go 1.21 或更高版本
- PostgreSQL 数据库
- Redis 服务器（可选，但推荐）
- Docker 和 Docker Compose（推荐，用于快速启动数据库）

### 快速启动（使用Docker）

如果安装了Docker，可以使用docker-compose快速启动PostgreSQL和Redis：

```bash
docker-compose up -d
```

这将在后台启动PostgreSQL和Redis服务。

### 手动安装数据库

如果没有Docker，需要：

1. **安装PostgreSQL**
   - Windows: 下载并安装 [PostgreSQL for Windows](https://www.postgresql.org/download/windows/)
   - 或使用包管理器安装

2. **创建数据库**
   ```sql
   CREATE DATABASE bus_fare_system;
   ```

3. **安装Redis**（可选但推荐）
   - Windows: 下载 [Redis for Windows](https://github.com/microsoftarchive/redis/releases)
   - 或使用WSL安装

### 安装依赖

```bash
go mod download
```

或

```bash
go mod tidy
```

### 配置

编辑 `config/config.yaml` 文件，配置数据库和Redis连接信息：

```yaml
database:
  host: "localhost"
  port: 5432
  user: "postgres"
  password: "postgres"  # 根据实际情况修改
  dbname: "bus_fare_system"
  sslmode: "disable"
  timezone: "Asia/Shanghai"

redis:
  host: "localhost"
  port: 6379
  password: ""
  db: 0
```

**注意**：如果使用docker-compose启动，默认配置已经匹配，无需修改。

### 创建数据库

**如果使用docker-compose，数据库会自动创建，可跳过此步骤。**

如果手动安装PostgreSQL，需要先创建数据库。有两种方式：

**方式1：使用SQL命令（推荐）**
```sql
-- 连接到PostgreSQL（使用默认的postgres数据库）
psql -U postgres

-- 创建数据库
CREATE DATABASE bus_fare_system;
```

**方式2：使用Go脚本**
```bash
go run scripts/create_database.go
```

### 数据库初始化

系统启动时会自动执行数据库迁移，创建所需的表结构。**首次运行前请确保：**
1. 数据库服务已启动
2. 数据库 `bus_fare_system` 已创建
3. 配置文件中的密码正确

#### 初始化示例数据（可选）

如果需要初始化示例数据（线路、站点、票价规则等），可以运行：

```bash
go run scripts/seed_data.go
```

或者使用SQL脚本：

```bash
psql -U postgres -d bus_fare_system -f scripts/init_db.sql
```

这将创建：
- 示例线路（1路、2路、快速1路）
- 示例站点（市中心站、火车站等）
- 票价规则（统一票价和分段计价示例）
- 换乘优惠规则
- 月度累计折扣策略

### 运行

```bash
go run main.go
```

服务器将默认在 `http://localhost:8080` 启动。

### Docker 运行

```bash
# 构建镜像
docker build -t bus-backend .

# 运行容器
docker run -p 8080:8080 bus-backend
```

## API 接口

### 公交数据接口

#### 批量上传乘车记录
```
POST /api/v1/bus/batchRecords
Content-Type: application/json

[
  {
    "card_id": "12345678",
    "board_time": "2026-01-03T08:30:25Z",
    "board_station": "站点A",
    "alight_time": "2026-01-03T08:50:10Z",
    "alight_station": "站点B",
    "route_id": 1,
    "gateway_id": "gateway001"
  }
]
```

#### 获取线路配置
```
GET /api/v1/bus/config?route_id=1
```

### 卡片接口

#### 查询卡片信息
```
GET /api/v1/card/{card_id}
```

### 交易记录接口

#### 查询交易记录
```
GET /api/v1/transactions?date=2026-01-03&route_id=1&page=1&page_size=20
```

### 线路接口

#### 获取线路列表
```
GET /api/v1/routes
```

## 计费策略

系统支持以下计费策略：

1. **单程票价**：根据上车站和下车站计算基础票价
2. **换乘优惠**：在指定换乘站和时间窗口内换乘享受优惠
3. **月度累计折扣**：当月累计消费达到阈值后享受折扣
4. **卡类型折扣**：学生卡、老人卡等特殊卡类型享受折扣

## 开发计划

本项目按照开发计划文档，在2026年1月6日前完成主要功能开发。

## 许可证

本项目为内部项目，仅供开发团队使用。
