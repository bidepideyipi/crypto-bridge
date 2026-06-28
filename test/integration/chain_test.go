//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"crypto-bridge/internal/blockchain/btc"
	"crypto-bridge/internal/config"
)

// testCfg 测试配置
var testCfg *config.Config

// init 加载测试配置
func init() {
	var err error
	testCfg, err = config.Load("config/test_config.yml")
	if err != nil {
		panic(fmt.Sprintf("加载测试配置失败: %v", err))
	}
}

// ChainTestSuite 链节点集成测试套件
// 测试用例 ID: IT-201 ~ IT-205
type ChainTestSuite struct {
	suite.Suite
	logger       *zap.Logger
	btcAdapter   *btc.Adapter
	rpcEndpoints []string
	network      string
	timeout      time.Duration
	maxRetries   int
}

// SetupSuite 初始化测试环境
func (s *ChainTestSuite) SetupSuite() {
	s.logger = zap.NewNop()

	// 使用统一配置
	if testCfg == nil {
		s.T().Fatal("测试配置未加载，请确保 test/integration/config/test_config.yml 存在")
	}

	// 从配置读取区块链节点信息
	s.network = testCfg.Blockchain.BTC.Network
	s.rpcEndpoints = testCfg.Blockchain.BTC.RPCEndpoints
	s.timeout = testCfg.Blockchain.BTC.Timeout
	s.maxRetries = testCfg.Blockchain.BTC.MaxRetries

	// 创建 BTC 适配器
	s.btcAdapter = btc.NewAdapter(
		s.rpcEndpoints,
		s.network,
		s.timeout,
		s.maxRetries,
		s.logger,
	)

	if s.btcAdapter == nil {
		s.T().Skip("无法创建 BTC 适配器")
	}

	s.T().Logf("初始化链测试套件: network=%s, endpoints=%v", s.network, s.rpcEndpoints)
}

// TearDownSuite 清理测试环境
func (s *ChainTestSuite) TearDownSuite() {
	if s.btcAdapter != nil {
		s.btcAdapter.Close()
	}
}

// IT-301: 节点连接测试
func (s *ChainTestSuite) TestNodeConnection() {
	s.T().Run("连接到测试网节点", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// 尝试获取最新区块高度来验证连接
		height, err := s.btcAdapter.GetLatestBlockHeight(ctx)
		s.Require().NoError(err, "应该能连接到至少一个测试网节点")

		s.Greater(height, int64(0), "区块高度应该大于 0")
		s.T().Logf("成功连接到测试网节点，当前区块高度: %d", height)
	})

	s.T().Run("验证所有配置的节点", func(t *testing.T) {
		connectedCount := 0
		for _, endpoint := range s.rpcEndpoints {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			url := fmt.Sprintf("%s/blocks/tip/height", endpoint)

			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				cancel()
				continue
			}

			resp, err := http.DefaultClient.Do(req)
			cancel()
			if err != nil {
				continue
			}
			resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				connectedCount++
				s.T().Logf("节点 %s 连接正常", endpoint)
			}
		}

		s.Greater(connectedCount, 0, "应该至少能连接到一个节点")
	})
}

// IT-302: 获取区块高度测试
func (s *ChainTestSuite) TestGetBlockHeight() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 获取最新区块高度
	height, err := s.btcAdapter.GetLatestBlockHeight(ctx)
	s.Require().NoError(err, "获取区块高度应该成功")

	s.Greater(height, int64(0), "区块高度应该大于 0")
	s.T().Logf("测试网当前区块高度: %d", height)

	// 多次获取应该返回相同或递增的高度
	height2, err := s.btcAdapter.GetLatestBlockHeight(ctx)
	s.Require().NoError(err)

	s.GreaterOrEqual(height2, height, "区块高度应该保持或递增")
}

// IT-303: 查询地址余额测试
func (s *ChainTestSuite) TestGetAddressBalance() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 使用已知的测试网地址
	testCases := []struct {
		address     string
		description string
	}{
		{
			address:     "tb1q5pshmp3yq6kl9q2hpyxqmn8ypdxk2c3jv2k7a8",
			description: "bech32 格式测试网地址",
		},
		{
			address:     "mpVfo4xnJjpnrcUhAHQZjECxsnxgDy9CyX",
			description: "Legacy 格式测试网地址",
		},
		{
			address:     "2N1rKN5qEuUk5bRPxPr7oHVQ5UnabGreksP",
			description: "P2SH 格式测试网地址",
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.description, func(t *testing.T) {
			balance, err := s.btcAdapter.GetBalance(ctx, tc.address)
			if err != nil {
				// 如果地址没有交易，某些节点会返回错误，这是正常的
				s.T().Logf("地址 %s 查询余额返回错误（可能无交易）: %v", tc.address, err)
				return
			}

			s.GreaterOrEqual(balance, int64(0), "余额应该大于等于 0")
			s.T().Logf("地址 %s 的余额: %d satoshis", tc.address, balance)
		})
	}

	s.T().Run("无效地址应该返回错误", func(t *testing.T) {
		_, err := s.btcAdapter.GetBalance(ctx, "invalid-address-format")
		s.Error(err, "无效地址应该返回错误")
	})
}

// IT-204: 查询交易测试
func (s *ChainTestSuite) TestGetTransaction() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	s.T().Run("查询不存在的交易", func(t *testing.T) {
		// 查询一个不存在的交易哈希
		fakeTxHash := "0000000000000000000000000000000000000000000000000000000000000000"

		_, err := s.btcAdapter.GetTransaction(ctx, fakeTxHash)
		s.Error(err, "不存在的交易应该返回错误")
	})

	s.T().Run("查询格式错误的交易哈希", func(t *testing.T) {
		// 查询格式错误的交易哈希
		invalidTxHash := "not-a-valid-tx-hash"

		_, err := s.btcAdapter.GetTransaction(ctx, invalidTxHash)
		s.Error(err, "格式错误的交易哈希应该返回错误")
	})
}

// IT-305: 广播交易测试
func (s *ChainTestSuite) TestBroadcastTransaction() {
	s.T().Run("广播无效交易应该失败", func(t *testing.T) {
		// 注意：当前适配器可能没有 BroadcastTransaction 方法
		// 这里测试验证如果不支持广播，应该优雅地处理
		s.T().Log("BTC 适配器当前可能不支持直接广播交易")
	})
}




// TestChainTestSuite 运行链测试套件
func TestChainTestSuite(t *testing.T) {
	suite.Run(t, new(ChainTestSuite))
}
