# Crypto Bridge 集成测试

## 概述

本目录包含 Crypto Bridge 多链钱包系统的集成测试套件。集成测试验证服务间交互与外部依赖的正确性。

## 前置要求

### 本地服务

确保以下服务已在本地运行：

| 服务 | 默认配置 | 说明 |
|------|----------|------|
| PostgreSQL | localhost:5432 | 用户: anthony, 密码: (空) |
| Redis | localhost:6379 | 无密码 |
| RocketMQ | localhost:9876 | NameServer 地址 |

### 安装测试依赖

```bash
go get github.com/stretchr/testify/assert
go get github.com/stretchr/testify/suite
go get github.com/redis/go-redis/v9
go get github.com/apache/rocketmq-client-go/v2
```

## 运行测试

### 运行所有集成测试

```bash
# 使用默认配置（连接本地服务）
go test -v -tags=integration ./test/integration/...
```

### 运行单个测试套件

```bash
# 数据库测试
go test -v -tags=integration ./test/integration/database_test.go

# Redis 测试
go test -v -tags=integration ./test/integration/redis_test.go

# RocketMQ 测试
go test -v -tags=integration ./test/integration/mq_test.go

# 链节点测试
go test -v -tags=integration ./test/integration/chain_test.go
```

### 运行特定测试用例

```bash
go test -v -tags=integration ./test/integration/database_test.go -run TestDatabaseTestSuite/TestDepositTransactionIntegrity
```

## 环境变量

可以通过环境变量覆盖默认配置：

```bash
# PostgreSQL
export TEST_DB_HOST=localhost
export TEST_DB_PORT=5432
export TEST_DB_USER=anthony
export TEST_DB_PASSWORD=""

# Redis
export TEST_REDIS_HOST=localhost
export TEST_REDIS_PORT=6379
export TEST_REDIS_PASSWORD=""

# RocketMQ
export TEST_ROCKETMQ_ADDR=127.0.0.1:9876
export TEST_DEPOSIT_TOPIC=wallet.deposit.events
export TEST_WITHDRAW_TOPIC=wallet.withdrawal.events

# 区块链网络
export TEST_NET_MODE=testnet
```

## 测试范围

### 1. 数据库集成测试 (IT-001 ~ IT-005)
- **文件**: `database_test.go`
- IT-001: 充值事务完整性
- IT-002: 提现事务完整性
- IT-003: 事务回滚
- IT-004: 并发写入
- IT-005: 唯一约束

### 2. Redis 集成测试 (IT-101 ~ IT-105)
- **文件**: `redis_test.go`
- IT-101: 余额缓存读写
- IT-102: 缓存过期
- IT-103: 充值去重
- IT-104: 分布式锁
- IT-105: 限流计数

### 3. RocketMQ 集成测试 (IT-201 ~ IT-205)
- **文件**: `mq_test.go`
- IT-201: 充值事件发送
- IT-202: 提现事件发送
- IT-203: 消息格式验证
- IT-204: 消息 Tag 测试
- IT-205: 消费幂等性

### 4. 链节点集成测试 (IT-301 ~ IT-305)
- **文件**: `chain_test.go`
- IT-301: 节点连接
- IT-302: 获取区块高度
- IT-303: 查询地址余额
- IT-304: 查询交易
- IT-305: 广播交易

## 注意事项

1. **数据库自动创建**: 测试会自动创建独立的测试数据库并在完成后删除
2. **测试网使用**: 链节点测试使用 Bitcoin Testnet，无需本地节点
3. **跳过策略**: 如果外部服务不可用，测试会自动跳过并记录日志
4. **隔离性**: 每个测试前会清空相关数据，确保测试独立

## 故障排查

### PostgreSQL 连接失败
```
Error: failed to connect to database
```
- 检查 PostgreSQL 是否运行: `pg_isready`
- 验证端口和用户配置

### Redis 连接失败
```
Error: 无法连接到 Redis
```
- 检查 Redis 是否运行: `redis-cli ping`
- 验证端口配置

### RocketMQ 测试跳过
```
--- SKIP: TestMQTestSuite (0.00s)
```
- 检查 RocketMQ 是否运行
- 验证 NameServer 地址配置

### 测试网连接失败
```
--- SKIP: TestChainTestSuite (0.00s)
```
- 检查网络连接
- 测试网节点可能暂时不可用
