// +build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"crypto-bridge/internal/models"
)

// RedisTestSuite Redis 集成测试套件
// 测试用例 ID: IT-101 ~ IT-105
type RedisTestSuite struct {
	suite.Suite
	client *redis.Client
	logger *zap.Logger
	ctx    context.Context
}

// SetupSuite 初始化测试环境
func (s *RedisTestSuite) SetupSuite() {
	s.logger = zap.NewNop()
	s.ctx = context.Background()

	host := getEnv("TEST_REDIS_HOST", "localhost")
	port := getEnv("TEST_REDIS_PORT", "6379")
	password := getEnv("TEST_REDIS_PASSWORD", "")
	db := getEnv("TEST_REDIS_DB", "0")

	s.client = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", host, port),
		Password: password,
		DB:       parseInt(db),
	})

	// 测试连接
	err := s.client.Ping(s.ctx).Err()
	s.Require().NoError(err, "无法连接到 Redis，请确保 Redis 服务正在运行")

	// 清空测试数据库
	s.Require().NoError(s.client.FlushDB(s.ctx).Err())
}

// TearDownSuite 清理测试环境
func (s *RedisTestSuite) TearDownSuite() {
	if s.client != nil {
		s.client.FlushDB(s.ctx)
		s.client.Close()
	}
}

// SetupTest 每个测试前的清理
func (s *RedisTestSuite) SetupTest() {
	s.Require().NoError(s.client.FlushDB(s.ctx).Err())
}

// IT-101: 余额缓存读写测试
func (s *RedisTestSuite) TestBalanceCacheReadWrite() {
	userID := "user_cache_101"
	chain := models.ChainBTC
	cacheKey := fmt.Sprintf("balance:%s:%s", userID, chain)
	balance := "1000000"

	// 写入缓存
	err := s.client.Set(s.ctx, cacheKey, balance, 30*time.Minute).Err()
	s.Require().NoError(err)

	// 读取缓存
	val, err := s.client.Get(s.ctx, cacheKey).Result()
	s.Require().NoError(err)
	s.Equal(balance, val)

	// 验证过期时间设置
	ttl, err := s.client.TTL(s.ctx, cacheKey).Result()
	s.Require().NoError(err)
	s.Greater(ttl, time.Duration(0), "缓存应该有过期时间")
	s.LessOrEqual(ttl, 31*time.Minute, "TTL 应该接近设置的时间")
}

// IT-102: 缓存过期测试
func (s *RedisTestSuite) TestCacheExpiration() {
	userID := "user_cache_102"
	chain := models.ChainBTC
	cacheKey := fmt.Sprintf("balance:%s:%s", userID, chain)
	balance := "500000"

	// 写入一个短过期时间的缓存 (1秒)
	err := s.client.Set(s.ctx, cacheKey, balance, 1*time.Second).Err()
	s.Require().NoError(err)

	// 立即读取应该成功
	val, err := s.client.Get(s.ctx, cacheKey).Result()
	s.Require().NoError(err)
	s.Equal(balance, val)

	// 等待过期
	time.Sleep(2 * time.Second)

	// 再次读取应该失败（缓存已过期）
	_, err = s.client.Get(s.ctx, cacheKey).Result()
	s.Equal(redis.Nil, err, "过期后的缓存应该返回 redis.Nil")
}

// IT-103: 充值去重测试
func (s *RedisTestSuite) TestDepositDeduplication() {
	txHash := "test_dedup_tx_103"
	dedupeKey := fmt.Sprintf("deposit:dedup:%s", txHash)

	// 第一次检查是否已处理
	exists, err := s.client.Exists(s.ctx, dedupeKey).Result()
	s.Require().NoError(err)
	s.Equal(int64(0), exists, "第一次检查时 key 不应该存在")

	// 标记为已处理
	err = s.client.Set(s.ctx, dedupeKey, "1", 24*time.Hour).Err()
	s.Require().NoError(err)

	// 第二次检查
	exists, err = s.client.Exists(s.ctx, dedupeKey).Result()
	s.Require().NoError(err)
	s.Equal(int64(1), exists, "处理后 key 应该存在")

	// 尝试设置（使用 SetNX 保证原子性）
	result, err := s.client.SetNX(s.ctx, dedupeKey, "1", 24*time.Hour).Result()
	s.Require().NoError(err)
	s.False(result, "SetNX 应该返回 false，表示 key 已存在")

	// 验证数据完整性
	val, err := s.client.Get(s.ctx, dedupeKey).Result()
	s.Require().NoError(err)
	s.Equal("1", val)

	// 获取剩余 TTL
	ttl, err := s.client.TTL(s.ctx, dedupeKey).Result()
	s.Require().NoError(err)
	s.Greater(ttl, time.Duration(0), "去重 key 应该有合理的 TTL")
}

// IT-104: 分布式锁测试
func (s *RedisTestSuite) TestDistributedLock() {
	lockKey := "lock:withdraw:user_lock_104"
	lockValue := fmt.Sprintf("%d", time.Now().UnixNano())
	lockTTL := 10 * time.Second

	// 尝试获取锁
	acquired, err := s.client.SetNX(s.ctx, lockKey, lockValue, lockTTL).Result()
	s.Require().NoError(err)
	s.True(acquired, "第一次获取锁应该成功")

	// 验证锁存在
	val, err := s.client.Get(s.ctx, lockKey).Result()
	s.Require().NoError(err)
	s.Equal(lockValue, val)

	// 尝试再次获取锁（应该失败）
	acquired, err = s.client.SetNX(s.ctx, lockKey, "another_value", lockTTL).Result()
	s.Require().NoError(err)
	s.False(acquired, "锁已被占用，再次获取应该失败")

	// 验证原锁值不变
	val, err = s.client.Get(s.ctx, lockKey).Result()
	s.Require().NoError(err)
	s.Equal(lockValue, val, "锁值应该保持不变")

	// 释放锁（使用 Lua 脚本保证原子性）
	releaseLockScript := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`
	result, err := s.client.Eval(s.ctx, releaseLockScript, []string{lockKey}, lockValue).Result()
	s.Require().NoError(err)
	s.Equal(int64(1), result, "释放锁应该成功")

	// 验证锁已释放
	exists, err := s.client.Exists(s.ctx, lockKey).Result()
	s.Require().NoError(err)
	s.Equal(int64(0), exists, "释放后锁不应该存在")

	// 现在可以重新获取锁
	acquired, err = s.client.SetNX(s.ctx, lockKey, "new_lock_value", lockTTL).Result()
	s.Require().NoError(err)
	s.True(acquired, "释放后重新获取锁应该成功")

	// 清理
	s.client.Del(s.ctx, lockKey)
}

// 测试锁的自动过期机制
func (s *RedisTestSuite) TestLockAutoExpiry() {
	lockKey := "lock:auto_expiry_104"
	lockValue := "lock_holder"
	shortTTL := 1 * time.Second

	// 获取锁
	acquired, err := s.client.SetNX(s.ctx, lockKey, lockValue, shortTTL).Result()
	s.Require().NoError(err)
	s.True(acquired)

	// 等待锁过期
	time.Sleep(2 * time.Second)

	// 验证锁已自动过期
	exists, err := s.client.Exists(s.ctx, lockKey).Result()
	s.Require().NoError(err)
	s.Equal(int64(0), exists, "锁应该自动过期")

	// 现在可以获取锁
	acquired, err = s.client.SetNX(s.ctx, lockKey, "new_holder", shortTTL).Result()
	s.Require().NoError(err)
	s.True(acquired, "过期后应该能获取锁")
}

// IT-105: 限流计数测试
func (s *RedisTestSuite) TestRateLimiting() {
	userID := "user_ratelimit_105"
	rateLimitKey := fmt.Sprintf("ratelimit:withdraw:%s", userID)
	limit := int64(5)       // 5次
	window := 10 * time.Second // 10秒窗口

	// 使用 INCR 和 EXPIRE 实现限流
	count, err := s.client.Incr(s.ctx, rateLimitKey).Result()
	s.Require().NoError(err)
	s.Equal(int64(1), count)

	// 第一次设置过期时间
	ttl := s.client.TTL(s.ctx, rateLimitKey)
	s.Require().NoError(ttl.Err())
	if ttl.Val() < 0 {
		// 只有当 key 没有过期时间时才设置
		s.client.Expire(s.ctx, rateLimitKey, window)
	}

	// 模拟多次请求
	for i := 2; i <= int(limit); i++ {
		count, err = s.client.Incr(s.ctx, rateLimitKey).Result()
		s.Require().NoError(err)
		s.Equal(int64(i), count)
	}

	// 验证当前计数
	val, err := s.client.Get(s.ctx, rateLimitKey).Result()
	s.Require().NoError(err)
	s.Equal(strconv.FormatInt(limit, 10), val)

	// 下一次请求应该超过限制
	count, err = s.client.Incr(s.ctx, rateLimitKey).Result()
	s.Require().NoError(err)
	s.Greater(count, limit, "计数应该超过限制")

	// 等待窗口过期
	time.Sleep(window + 1*time.Second)

	// 验证计数已过期（通过 TTL）
	ttlVal, err := s.client.TTL(s.ctx, rateLimitKey).Result()
	s.Require().NoError(err)
	s.Less(ttlVal, time.Duration(0), "过期后 TTL 应该为负数或已不存在")

	// 重新开始计数
	s.client.Del(s.ctx, rateLimitKey)
	count, err = s.client.Incr(s.ctx, rateLimitKey).Result()
	s.Require().NoError(err)
	s.Equal(int64(1), count, "过期后应该重新开始计数")
}

// 测试滑动窗口限流
func (s *RedisTestSuite) TestSlidingWindowRateLimit() {
	userID := "user_sliding_105"
	key := fmt.Sprintf("ratelimit:sliding:%s", userID)

	// 使用 ZSet 实现滑动窗口限流
	now := time.Now().Unix()
	window := int64(10) // 10秒窗口
	limit := int64(3)   // 最多3次

	// 添加第一次请求
	err := s.client.ZAdd(s.ctx, key, redis.Z{
		Score:  float64(now),
		Member: "req1",
	}).Err()
	s.Require().NoError(err)

	// 设置过期时间
	s.client.Expire(s.ctx, key, 20*time.Second)

	// 检查当前窗口内的请求数
	minScore := float64(now - window)
	maxScore := float64(now + 1)
	count, err := s.client.ZCount(s.ctx, key, strconv.FormatFloat(minScore, 'f', 0, 64), strconv.FormatFloat(maxScore, 'f', 0, 64)).Result()
	s.Require().NoError(err)
	s.Equal(int64(1), count)

	// 添加更多请求
	for i := 2; i <= int(limit); i++ {
		err := s.client.ZAdd(s.ctx, key, redis.Z{
			Score:  float64(now),
			Member: fmt.Sprintf("req%d", i),
		}).Err()
		s.Require().NoError(err)
	}

	// 检查是否达到限制
	count, err = s.client.ZCount(s.ctx, key, strconv.FormatFloat(minScore, 'f', 0, 64), strconv.FormatFloat(maxScore, 'f', 0, 64)).Result()
	s.Require().NoError(err)
	s.Equal(limit, count)

	// 尝试添加第 limit+1 次请求
	isAllowed := count < limit
	s.False(isAllowed, "达到限制后应该拒绝新请求")

	// 清理过期成员
	s.client.ZRemRangeByScore(s.ctx, key, "0", strconv.FormatFloat(minScore, 'f', 0, 64))

	// 清理
	s.client.Del(s.ctx, key)
}

// TestRedisTestSuite 运行 Redis 测试套件
func TestRedisTestSuite(t *testing.T) {
	suite.Run(t, new(RedisTestSuite))
}

func parseInt(s string) int {
	val, _ := strconv.Atoi(s)
	return val
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
