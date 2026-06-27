# Crypto Bridge 集成测试

## 概述

本目录包含 Crypto Bridge 多链钱包系统的集成测试套件。集成测试验证服务间交互与外部依赖的正确性。

## 前置要求

### 本地服务

确保以下服务已在本地运行：

| 服务 | 默认配置 | 说明 |
|------|----------|------|
| PostgreSQL | localhost:5432 | 用户: anthony, 密码: (空) |

### 安装测试依赖

```bash
go get github.com/stretchr/testify/assert
go get github.com/stretchr/testify/suite
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

### 2. 链节点集成测试 (IT-201 ~ IT-205)
- **文件**: `chain_test.go`
- IT-201: 节点连接
- IT-202: 获取区块高度
- IT-203: 查询地址余额
- IT-204: 查询交易
- IT-205: 广播交易

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

### 测试网连接失败
```
--- SKIP: TestChainTestSuite (0.00s)
```
- 检查网络连接
- 测试网节点可能暂时不可用
