//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"crypto-bridge/internal/config"
	"crypto-bridge/internal/models"
	"crypto-bridge/internal/service"
)

// 测试配置
var testCfg *config.Config

// init 加载测试配置
func init() {
	// 尝试从多个位置加载测试配置
	configPaths := []string{
		"./test/integration/config/test_config.yml",
		"../../test/integration/config/test_config.yml",
		"./config/test_config.yml",
		"../../config/test_config.yml",
	}

	for _, path := range configPaths {
		if _, err := os.ReadFile(path); err == nil {
			if cfg, err := config.Load(path); err == nil {
				testCfg = cfg
				return
			}
		}
	}

	// 找不到配置文件，设置 nil 标记
	testCfg = nil
}

// DatabaseTestSuite 数据库集成测试套件
// 测试用例 ID: IT-001 ~ IT-005
type DatabaseTestSuite struct {
	suite.Suite
	db         *gorm.DB
	sqlxDB     *sqlx.DB
	logger     *zap.Logger
	depositSvc *service.DepositService
	testDBName string
}

// SetupSuite 初始化测试环境
func (s *DatabaseTestSuite) SetupSuite() {
	// 初始化 logger
	s.logger = zap.NewNop()

	// 检查配置是否加载
	if testCfg == nil {
		s.T().Fatal("测试配置未加载，请确保 test/integration/config/test_config.yml 存在")
	}

	// 从配置获取数据库配置
	host := testCfg.Database.Postgres.Host
	port := fmt.Sprintf("%d", testCfg.Database.Postgres.Port)
	user := testCfg.Database.Postgres.User
	password := testCfg.Database.Postgres.Password

	// 创建唯一的测试数据库名称
	s.testDBName = fmt.Sprintf("crypto_bridge_test_%d", time.Now().UnixNano())

	// 先连接到 postgres 数据库创建测试数据库
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s sslmode=%s dbname=postgres",
		host, port, user, password, testCfg.Database.Postgres.SSLMode)
	db, err := sqlx.Connect("postgres", dsn)
	s.Require().NoError(err)

	// 创建测试数据库
	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", s.testDBName))
	s.Require().NoError(err)
	db.Close()

	// 连接到测试数据库
	testDSN := fmt.Sprintf("host=%s port=%s user=%s password=%s sslmode=%s dbname=%s",
		host, port, user, password, testCfg.Database.Postgres.SSLMode, s.testDBName)

	s.db, err = gorm.Open(postgres.Open(testDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	s.Require().NoError(err)

	s.sqlxDB, err = sqlx.Connect("postgres", testDSN)
	s.Require().NoError(err)

	// 运行数据库迁移脚本
	s.RunMigration()

	// 初始化服务
	s.depositSvc = service.NewDepositService(s.db, s.logger)
}

// TearDownSuite 清理测试环境
func (s *DatabaseTestSuite) TearDownSuite() {
	if s.sqlxDB != nil {
		s.sqlxDB.Close()
	}
	if s.db != nil {
		sqlDB, _ := s.db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}

	if testCfg == nil {
		return
	}

	// 从配置获取数据库配置
	host := testCfg.Database.Postgres.Host
	port := fmt.Sprintf("%d", testCfg.Database.Postgres.Port)
	user := testCfg.Database.Postgres.User
	password := testCfg.Database.Postgres.Password

	// 删除测试数据库
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s sslmode=%s dbname=postgres",
		host, port, user, password, testCfg.Database.Postgres.SSLMode)
	db, err := sqlx.Connect("postgres", dsn)
	if err == nil {
		// 终止所有连接到测试数据库的连接
		db.Exec(fmt.Sprintf("SELECT pg_terminate_backend(pg_stat_activity.pid) FROM pg_stat_activity WHERE pg_stat_activity.datname = '%s' AND pid <> pg_backend_pid()", s.testDBName))
		// 删除数据库
		db.Exec(fmt.Sprintf("DROP DATABASE %s", s.testDBName))
		db.Close()
	}
}

// SetupTest 每个测试前的清理
func (s *DatabaseTestSuite) SetupTest() {
	// 清理表数据（按依赖关系顺序）
	// 先删除引用其他表的数据
	s.db.Exec("DELETE FROM balance_transactions")
	// 再删除主表数据
	s.db.Exec("DELETE FROM user_balances")
	s.db.Exec("DELETE FROM deposits")
	s.db.Exec("DELETE FROM user_addresses")
	s.db.Exec("DELETE FROM withdrawals")
}

// RunMigration 执行数据库迁移
func (s *DatabaseTestSuite) RunMigration() {
	// 尝试从多个位置查找 schema.sql
	schemaPaths := []string{
		"../../db/schema.sql",
		"./db/schema.sql",
		"../../../db/schema.sql",
	}

	var schemaContent []byte
	var schemaPath string
	for _, path := range schemaPaths {
		// 获取测试文件的绝对路径
		testFile := getTestFilePath()
		schemaPath = filepath.Join(filepath.Dir(testFile), path)

		if data, err := os.ReadFile(schemaPath); err == nil {
			schemaContent = data
			break
		}
	}

	if len(schemaContent) == 0 {
		s.T().Fatalf("无法找到 db/schema.sql 文件，尝试的路径: %v", schemaPaths)
	}

	// 执行 schema.sql
	err := s.db.Exec(string(schemaContent)).Error
	s.Require().NoError(err, "执行数据库迁移失败: %s", schemaPath)

	s.T().Logf("数据库迁移完成: %s", schemaPath)
}

// getTestFilePath 获取当前测试文件的路径
func getTestFilePath() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		// 如果获取失败，使用当前工作目录
		return ""
	}
	return file
}

// IT-001: 充值事务完整性测试
func (s *DatabaseTestSuite) TestDepositTransactionIntegrity() {
	// 准备测试数据
	userID := "test_user_001"
	depositAddr := "bc1qtestaddr001"
	fromAddr := "bc1qfromaddr001"
	txHash := "test_tx_hash_001"
	amount := int64(100000) // 0.001 BTC in satoshis

	// 创建用户地址
	userAddr := models.UserAddress{
		UserID:      userID,
		Chain:       models.ChainBTC,
		Address:     depositAddr,
		AddressType: models.AddressTypeDeposit,
		Status:      models.AddressStatusActive,
		CreatedAt:   time.Now().Unix(),
		UpdatedAt:   time.Now().Unix(),
	}
	s.Require().NoError(s.db.Create(&userAddr).Error)

	// 处理充值（确认数达到阈值）
	depositInfo := &service.DepositInfo{
		TxHash:        txHash,
		FromAddress:   fromAddr,
		ToAddress:     depositAddr,
		Amount:        amount,
		BlockHeight:   12345,
		BlockHash:     "block_hash_001",
		Timestamp:     time.Now().Unix(),
		Confirmations: 6, // 达到确认数阈值
		Status:        "confirmed",
	}

	err := s.depositSvc.HandleDeposit(context.Background(), depositInfo)
	s.Require().NoError(err)

	// 验证三表数据一致性

	// 1. 验证 deposits 表
	var deposit models.Deposit
	err = s.db.Where("tx_hash = ?", txHash).First(&deposit).Error
	s.Require().NoError(err)
	s.Equal(models.DepositStatusCompleted, deposit.Status)
	s.Equal(userID, deposit.UserID)

	// 解析并验证金额
	var depositAmount int64
	fmt.Sscanf(deposit.Amount, "%d", &depositAmount)
	s.Equal(amount, depositAmount)
	s.NotNil(deposit.CompletedAt)

	// 2. 验证 balance_transactions 表
	var balanceTxn models.BalanceTransaction
	err = s.db.Where("related_id = ?", deposit.DepositID).First(&balanceTxn).Error
	s.Require().NoError(err)
	s.Equal(userID, balanceTxn.UserID)
	s.Equal(models.TransactionTypeDeposit, balanceTxn.Type)

	// 解析并验证金额
	var txnAmount, balanceBefore, balanceAfter int64
	fmt.Sscanf(balanceTxn.Amount, "%d", &txnAmount)
	fmt.Sscanf(balanceTxn.BalanceBefore, "%d", &balanceBefore)
	fmt.Sscanf(balanceTxn.BalanceAfter, "%d", &balanceAfter)
	s.Equal(amount, txnAmount)
	s.Equal(int64(0), balanceBefore)
	s.Equal(amount, balanceAfter)

	// 3. 验证 user_balances 表
	var userBalance models.UserBalance
	err = s.db.Where("user_id = ? AND chain = ?", userID, models.ChainBTC).First(&userBalance).Error
	s.Require().NoError(err)

	// 解析并验证余额
	var userBalanceAmount, lockedBalance int64
	fmt.Sscanf(userBalance.Balance, "%d", &userBalanceAmount)
	fmt.Sscanf(userBalance.LockedBalance, "%d", &lockedBalance)
	s.Equal(amount, userBalanceAmount)
	s.Equal(int64(0), lockedBalance)

	// 验证三者之间的关联关系
	s.Equal(deposit.DepositID, balanceTxn.RelatedID)
	s.Equal(deposit.UserID, userBalance.UserID)
	s.Equal(deposit.Amount, balanceTxn.Amount)
	s.Equal(userBalance.Balance, balanceTxn.BalanceAfter)
}

// IT-002: 提现事务完整性测试
func (s *DatabaseTestSuite) TestWithdrawalTransactionIntegrity() {
	// 准备测试数据
	userID := "test_user_002"
	depositAddr := "bc1qtestaddr002"
	withdrawAddr := "bc1qwithdrawaddr002"

	// 创建用户充值地址
	userAddr := models.UserAddress{
		UserID:      userID,
		Chain:       models.ChainBTC,
		Address:     depositAddr,
		AddressType: models.AddressTypeDeposit,
		Status:      models.AddressStatusActive,
		CreatedAt:   time.Now().Unix(),
		UpdatedAt:   time.Now().Unix(),
	}
	s.Require().NoError(s.db.Create(&userAddr).Error)

	// 先充值以获得余额
	depositInfo := &service.DepositInfo{
		TxHash:        "test_deposit_tx_002",
		FromAddress:   "bc1qfromaddr002",
		ToAddress:     depositAddr,
		Amount:        1000000, // 0.01 BTC
		Timestamp:     time.Now().Unix(),
		Confirmations: 6,
		Status:        "confirmed",
	}
	s.Require().NoError(s.depositSvc.HandleDeposit(context.Background(), depositInfo))

	// 获取当前余额
	var balanceBefore models.UserBalance
	s.Require().NoError(s.db.Where("user_id = ? AND chain = ?", userID, models.ChainBTC).First(&balanceBefore).Error)

	// 在事务中执行提现冻结操作
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 创建提现记录
		withdrawID := fmt.Sprintf("wdr_%d", time.Now().UnixNano())
		withdrawal := models.Withdrawal{
			WithdrawID: withdrawID,
			UserID:     userID,
			Chain:      models.ChainBTC,
			ToAddress:  withdrawAddr,
			Amount:     "50000", // 0.0005 BTC
			Fee:        "1000",  // 手续费
			Status:     models.WithdrawStatusPending,
			CreatedAt:  time.Now().Unix(),
			UpdatedAt:  time.Now().Unix(),
		}
		if err := tx.Create(&withdrawal).Error; err != nil {
			return err
		}

		// 获取当前余额（在事务内重新查询）
		var currentBalance models.UserBalance
		if err := tx.Where("user_id = ? AND chain = ?", userID, models.ChainBTC).
			First(&currentBalance).Error; err != nil {
			return err
		}

		// 解析余额
		var currentBal, lockedBal, withdrawAmount int64
		fmt.Sscanf(currentBalance.Balance, "%d", &currentBal)
		fmt.Sscanf(currentBalance.LockedBalance, "%d", &lockedBal)
		fmt.Sscanf(withdrawal.Amount, "%d", &withdrawAmount)

		// 冻结余额
		newBalance := currentBal - withdrawAmount
		newLockedBalance := lockedBal + withdrawAmount

		// 创建余额流水
		balanceTxn := models.BalanceTransaction{
			TransactionID: fmt.Sprintf("txn_%d", time.Now().UnixNano()),
			UserID:        userID,
			Chain:         models.ChainBTC,
			Type:          models.TransactionTypeFreeze,
			Amount:        withdrawal.Amount,
			BalanceBefore: currentBalance.Balance,
			BalanceAfter:  fmt.Sprintf("%d", newBalance),
			RelatedID:     withdrawID,
			CreatedAt:     time.Now().Unix(),
		}
		if err := tx.Create(&balanceTxn).Error; err != nil {
			return err
		}

		// 更新用户余额
		return tx.Model(&currentBalance).
			Updates(map[string]interface{}{
				"balance":        fmt.Sprintf("%d", newBalance),
				"locked_balance": fmt.Sprintf("%d", newLockedBalance),
				"updated_at":     time.Now().Unix(),
			}).Error
	})
	s.Require().NoError(err)

	// 验证三表数据一致性
	// 1. withdrawals 表
	var withdrawal models.Withdrawal
	s.Require().NoError(s.db.Where("withdraw_id LIKE ?", "wdr_%").First(&withdrawal).Error)
	s.Equal(models.WithdrawStatusPending, withdrawal.Status)

	// 2. balance_transactions 表
	var balanceTxn models.BalanceTransaction
	s.Require().NoError(s.db.Where("type = ?", models.TransactionTypeFreeze).First(&balanceTxn).Error)
	s.Equal(models.TransactionTypeFreeze, balanceTxn.Type)

	// 3. user_balances 表 - 重新查询验证
	var updatedBalance models.UserBalance
	s.Require().NoError(s.db.Where("user_id = ? AND chain = ?", userID, models.ChainBTC).First(&updatedBalance).Error)

	// 验证余额变化：充值 1000000，提现冻结 50000，结果余额应为 950000，冻结 50000
	var balAfter, lockedAfter int64
	fmt.Sscanf(updatedBalance.Balance, "%d", &balAfter)
	fmt.Sscanf(updatedBalance.LockedBalance, "%d", &lockedAfter)

	s.Equal(int64(950000), balAfter, "提现冻结后可用余额应为 950000")
	s.Equal(int64(50000), lockedAfter, "提现冻结后冻结余额应为 50000")
}

// IT-003: 事务回滚测试
func (s *DatabaseTestSuite) TestTransactionRollback() {
	userID := "test_user_003"

	// 创建初始余额
	userBalance := models.UserBalance{
		UserID:        userID,
		Chain:         models.ChainBTC,
		Balance:       "100000",
		LockedBalance: "0",
		CreatedAt:     time.Now().Unix(),
		UpdatedAt:     time.Now().Unix(),
	}
	s.Require().NoError(s.db.Create(&userBalance).Error)

	// 记录初始余额
	var initialBalance int64
	fmt.Sscanf(userBalance.Balance, "%d", &initialBalance)

	// 尝试执行一个会失败的事务（余额不足）
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 先扣减余额
		var balance models.UserBalance
		if err := tx.Where("user_id = ? AND chain = ?", userID, models.ChainBTC).First(&balance).Error; err != nil {
			return err
		}

		var currentBalance int64
		fmt.Sscanf(balance.Balance, "%d", &currentBalance)

		// 尝试扣除超出余额的金额
		newBalance := currentBalance - 999999999
		if err := tx.Model(&balance).
			Updates(map[string]interface{}{
				"balance":    fmt.Sprintf("%d", newBalance),
				"updated_at": time.Now().Unix(),
			}).Error; err != nil {
			return err
		}

		// 创建余额流水
		balanceTxn := models.BalanceTransaction{
			TransactionID: "txn_rollback_test",
			UserID:        userID,
			Chain:         models.ChainBTC,
			Type:          models.TransactionTypeWithdraw,
			Amount:        "999999999",
			BalanceBefore: balance.Balance,
			BalanceAfter:  fmt.Sprintf("%d", newBalance),
			CreatedAt:     time.Now().Unix(),
		}
		if err := tx.Create(&balanceTxn).Error; err != nil {
			return err
		}

		// 模拟业务逻辑失败
		return fmt.Errorf("insufficient balance")
	})

	// 事务应该失败
	s.Error(err)

	// 验证数据没有变化 - 余额应该保持不变
	var finalBalance models.UserBalance
	s.Require().NoError(s.db.Where("user_id = ? AND chain = ?", userID, models.ChainBTC).First(&finalBalance).Error)

	var finalBalanceValue int64
	fmt.Sscanf(finalBalance.Balance, "%d", &finalBalanceValue)

	s.Equal(initialBalance, finalBalanceValue, "余额应该在事务回滚后保持不变")

	// 验证流水表没有记录
	var count int64
	s.db.Model(&models.BalanceTransaction{}).Where("user_id = ?", userID).Count(&count)
	s.Equal(int64(0), count, "事务失败时不应该有流水记录")
}

// IT-004: 并发写入测试
func (s *DatabaseTestSuite) TestConcurrentWrites() {
	userID := "test_user_004"
	depositAddr := "bc1qtestaddr004"

	// 创建用户地址
	userAddr := models.UserAddress{
		UserID:      userID,
		Chain:       models.ChainBTC,
		Address:     depositAddr,
		AddressType: models.AddressTypeDeposit,
		Status:      models.AddressStatusActive,
		CreatedAt:   time.Now().Unix(),
		UpdatedAt:   time.Now().Unix(),
	}
	s.Require().NoError(s.db.Create(&userAddr).Error)

	// 并发处理多笔充值
	concurrency := 10
	var wg sync.WaitGroup
	errors := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			depositInfo := &service.DepositInfo{
				TxHash:        fmt.Sprintf("concurrent_tx_%d", index),
				FromAddress:   fmt.Sprintf("bc1qfrom_%d", index),
				ToAddress:     depositAddr,
				Amount:        10000, // 每笔 0.0001 BTC
				Timestamp:     time.Now().Unix(),
				Confirmations: 6,
				Status:        "confirmed",
			}

			if err := s.depositSvc.HandleDeposit(context.Background(), depositInfo); err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// 检查是否有错误
	for err := range errors {
		s.Require().NoError(err)
	}

	// 验证最终余额
	var userBalance models.UserBalance
	s.Require().NoError(s.db.Where("user_id = ? AND chain = ?", userID, models.ChainBTC).First(&userBalance).Error)

	var finalBalance int64
	fmt.Sscanf(userBalance.Balance, "%d", &finalBalance)

	expectedBalance := int64(concurrency * 10000) // 10 笔，每笔 10000 satoshis
	s.Equal(expectedBalance, finalBalance, "并发充值后余额应该等于所有充值金额之和")

	// 验证流水记录数量
	var txnCount int64
	s.db.Model(&models.BalanceTransaction{}).Where("user_id = ?", userID).Count(&txnCount)
	s.Equal(int64(concurrency), txnCount, "应该有等量的流水记录")
}

// IT-005: 唯一约束测试
func (s *DatabaseTestSuite) TestUniqueConstraint() {
	userID := "test_user_005"
	depositAddr := "bc1qtestaddr005"
	txHash := "duplicate_tx_hash_005"

	// 创建用户地址
	userAddr := models.UserAddress{
		UserID:      userID,
		Chain:       models.ChainBTC,
		Address:     depositAddr,
		AddressType: models.AddressTypeDeposit,
		Status:      models.AddressStatusActive,
		CreatedAt:   time.Now().Unix(),
		UpdatedAt:   time.Now().Unix(),
	}
	s.Require().NoError(s.db.Create(&userAddr).Error)

	// 第一次处理充值
	depositInfo := &service.DepositInfo{
		TxHash:        txHash,
		FromAddress:   "bc1qfrom_005",
		ToAddress:     depositAddr,
		Amount:        50000,
		Timestamp:     time.Now().Unix(),
		Confirmations: 3,
		Status:        "pending",
	}

	err := s.depositSvc.HandleDeposit(context.Background(), depositInfo)
	s.Require().NoError(err)

	// 验证充值记录已创建
	var deposit models.Deposit
	s.Require().NoError(s.db.Where("tx_hash = ?", txHash).First(&deposit).Error)
	s.Equal(models.DepositStatusPending, deposit.Status)

	// 第二次处理相同交易（模拟重新检测到）
	depositInfo.Confirmations = 4
	err = s.depositSvc.HandleDeposit(context.Background(), depositInfo)
	s.Require().NoError(err, "重复处理同一交易应该返回成功")

	// 验证只创建了一条充值记录
	var count int64
	s.db.Model(&models.Deposit{}).Where("tx_hash = ?", txHash).Count(&count)
	s.Equal(int64(1), count, "相同交易哈希只应该有一条记录")

	// 验证确认数已更新
	s.db.Where("tx_hash = ?", txHash).First(&deposit)
	s.Equal(4, deposit.Confirmations, "确认数应该被更新")

	// 第三次处理，确认数达到阈值，验证不会重复入账
	depositInfo.Confirmations = 6
	depositInfo.Status = "confirmed"
	err = s.depositSvc.HandleDeposit(context.Background(), depositInfo)
	s.Require().NoError(err)

	// 验证余额只增加一次（不是三次）
	var userBalance models.UserBalance
	s.db.Where("user_id = ? AND chain = ?", userID, models.ChainBTC).First(&userBalance)

	var balance int64
	fmt.Sscanf(userBalance.Balance, "%d", &balance)
	s.Equal(int64(50000), balance, "余额应该只增加一次原始充值金额")
}

// TestDatabaseTestSuite 运行数据库测试套件
func TestDatabaseTestSuite(t *testing.T) {
	suite.Run(t, new(DatabaseTestSuite))
}
