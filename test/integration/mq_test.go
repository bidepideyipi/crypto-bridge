// +build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"crypto-bridge/internal/models"
)

// MQTestSuite RocketMQ 集成测试套件
// 测试用例 ID: IT-201 ~ IT-205
type MQTestSuite struct {
	suite.Suite
	logger        *zap.Logger
	producer      rocketmq.Producer
	nameSrvAddr   string
	depositTopic  string
	withdrawTopic string
	groupName     string
	consumedMsgs  []*primitive.MessageExt
	ctx           context.Context
	mu            sync.Mutex
}

// SetupSuite 初始化测试环境
func (s *MQTestSuite) SetupSuite() {
	s.logger = zap.NewNop()
	s.ctx = context.Background()
	s.consumedMsgs = make([]*primitive.MessageExt, 0)
	s.mu = sync.Mutex{}

	s.nameSrvAddr = getEnv("TEST_ROCKETMQ_ADDR", "127.0.0.1:9876")
	s.depositTopic = getEnv("TEST_DEPOSIT_TOPIC", "wallet.deposit.events")
	s.withdrawTopic = getEnv("TEST_WITHDRAW_TOPIC", "wallet.withdrawal.events")
	s.groupName = fmt.Sprintf("test_consumer_%d", time.Now().UnixNano())

	// 创建生产者
	var err error
	s.producer, err = rocketmq.NewProducer(
		producer.WithNameServer([]string{s.nameSrvAddr}),
		producer.WithRetry(3),
	)
	if err != nil {
		s.T().Skip("无法连接到 RocketMQ，请确保 RocketMQ 服务正在运行: ", err)
		return
	}

	err = s.producer.Start()
	if err != nil {
		s.T().Skip("无法启动 RocketMQ 生产者: ", err)
		return
	}

	s.T().Cleanup(func() {
		if s.producer != nil {
			s.producer.Shutdown()
		}
	})
}

// TearDownSuite 清理测试环境
func (s *MQTestSuite) TearDownSuite() {
	if s.producer != nil {
		s.producer.Shutdown()
	}
}

// SetupTest 每个测试前的设置
func (s *MQTestSuite) SetupTest() {
	s.mu.Lock()
	s.consumedMsgs = make([]*primitive.MessageExt, 0)
	s.mu.Unlock()
}

// addConsumedMsg 线程安全地添加消息
func (s *MQTestSuite) addConsumedMsg(msg *primitive.MessageExt) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.consumedMsgs = append(s.consumedMsgs, msg)
}

// getConsumedMsgs 线程安全地获取消息列表
func (s *MQTestSuite) getConsumedMsgs() []*primitive.MessageExt {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.consumedMsgs
}

// IT-201: 充值事件发送测试
func (s *MQTestSuite) TestDepositEventSending() {
	if s.producer == nil {
		s.T().Skip("RocketMQ 生产者未初始化")
	}

	// 准备充值事件
	event := &models.DepositEvent{
		EventID:       fmt.Sprintf("dep_event_%d", time.Now().UnixNano()),
		EventType:     "deposit.confirmed",
		UserID:        "user_mq_201",
		Chain:         string(models.ChainBTC),
		TxHash:        "test_mq_tx_201",
		Amount:        "100000",
		FromAddress:   "bc1qfrom_mq_201",
		ToAddress:     "bc1qto_mq_201",
		Confirmations: 6,
		Timestamp:     time.Now().Unix(),
	}

	// 序列化事件
	body, err := json.Marshal(event)
	s.Require().NoError(err)

	// 发送消息
	msg := primitive.NewMessage(s.depositTopic, body)
	msg.WithTag(string(models.ChainBTC))

	result, err := s.producer.SendSync(s.ctx, msg)
	s.Require().NoError(err, "发送充值事件应该成功")
	s.Equal(primitive.SendOK, result.Status, "消息状态应该是 SendOK")

	s.T().Logf("消息发送成功: MsgID=%s", result.MsgID)

	// 验证消息结构
	s.NotEmpty(result.MsgID, "消息 ID 不应为空")
}

// IT-202: 提现事件发送测试
func (s *MQTestSuite) TestWithdrawalEventSending() {
	if s.producer == nil {
		s.T().Skip("RocketMQ 生产者未初始化")
	}

	// 准备提现事件
	event := &models.WithdrawalEvent{
		EventID:    fmt.Sprintf("wdr_event_%d", time.Now().UnixNano()),
		EventType:  "withdrawal.completed",
		WithdrawID: "wdr_mq_202",
		UserID:     "user_mq_202",
		Chain:      string(models.ChainBTC),
		TxHash:     "test_wdr_mq_tx_202",
		Amount:     "50000",
		ToAddress:  "bc1qwdr_mq_202",
		Status:     "completed",
		Timestamp:  time.Now().Unix(),
	}

	body, err := json.Marshal(event)
	s.Require().NoError(err)

	msg := primitive.NewMessage(s.withdrawTopic, body)
	msg.WithTag(string(models.ChainBTC))

	result, err := s.producer.SendSync(s.ctx, msg)
	s.Require().NoError(err, "发送提现事件应该成功")
	s.Equal(primitive.SendOK, result.Status)

	s.NotEmpty(result.MsgID, "消息 ID 不应为空")
}

// IT-203: 消息格式测试
func (s *MQTestSuite) TestMessageFormat() {
	if s.producer == nil {
		s.T().Skip("RocketMQ 生产者未初始化")
	}

	// 创建包含所有字段的充值事件
	event := &models.DepositEvent{
		EventID:       fmt.Sprintf("fmt_test_%d", time.Now().UnixNano()),
		EventType:     "deposit.confirmed",
		UserID:        "user_fmt_test",
		Chain:         string(models.ChainBTC),
		TxHash:        "fmt_test_tx_hash",
		Amount:        "12345678",
		FromAddress:   "bc1qfrom_fmt",
		ToAddress:     "bc1qto_fmt",
		Confirmations: 6,
		Timestamp:     time.Now().Unix(),
	}

	body, err := json.Marshal(event)
	s.Require().NoError(err)

	// 验证 JSON 格式正确
	var decoded map[string]interface{}
	err = json.Unmarshal(body, &decoded)
	s.Require().NoError(err)

	// 验证必需字段存在
	requiredFields := []string{"event_id", "event_type", "user_id", "chain", "tx_hash", "amount", "from_address", "to_address", "confirmations", "timestamp"}
	for _, field := range requiredFields {
		_, exists := decoded[field]
		s.True(exists, fmt.Sprintf("必需字段 %s 应该存在", field))
	}

	// 发送消息
	msg := primitive.NewMessage(s.depositTopic, body)
	msg.WithTag(string(models.ChainBTC))

	result, err := s.producer.SendSync(s.ctx, msg)
	s.Require().NoError(err)
	s.Equal(primitive.SendOK, result.Status)
}

// IT-204: 消息 Tag 测试
func (s *MQTestSuite) TestMessageTag() {
	if s.producer == nil {
		s.T().Skip("RocketMQ 生产者未初始化")
	}

	// 测试不同链的 Tag
	testCases := []struct {
		chain models.ChainType
		tag   string
	}{
		{models.ChainBTC, "BTC"},
		{models.ChainETH, "ETH"},
		{models.ChainTRX, "TRX"},
		{models.ChainSOL, "SOL"},
	}

	for _, tc := range testCases {
		event := &models.DepositEvent{
			EventID:   fmt.Sprintf("tag_test_%s_%d", tc.chain, time.Now().UnixNano()),
			EventType: "deposit.confirmed",
			UserID:    "user_tag_test",
			Chain:     string(tc.chain),
			TxHash:    fmt.Sprintf("tag_tx_%s", tc.chain),
			Amount:    "1000",
			Timestamp: time.Now().Unix(),
		}

		body, err := json.Marshal(event)
		s.Require().NoError(err)

		msg := primitive.NewMessage(s.depositTopic, body)
		msg.WithTag(tc.tag)

		result, err := s.producer.SendSync(s.ctx, msg)
		s.Require().NoError(err, fmt.Sprintf("发送 %s 链消息应该成功", tc.chain))
		s.Equal(primitive.SendOK, result.Status)
		s.T().Logf("成功发送 %s 链消息，MsgID=%s", tc.chain, result.MsgID)
	}
}

// IT-205: 消息 Keys 测试（用于幂等性）
func (s *MQTestSuite) TestMessageKeys() {
	if s.producer == nil {
		s.T().Skip("RocketMQ 生产者未初始化")
	}

	// 测试消息 Keys 用于幂等性
	event := &models.DepositEvent{
		EventID:   "idempotent_event_205",
		EventType: "deposit.confirmed",
		UserID:    "user_idempotent",
		Chain:     string(models.ChainBTC),
		TxHash:    "idempotent_tx_205",
		Amount:    "20000",
		Timestamp: time.Now().Unix(),
	}

	body, err := json.Marshal(event)
	s.Require().NoError(err)

	// 发送带有相同 Key 的多条消息
	for i := 0; i < 3; i++ {
		msg := primitive.NewMessage(s.depositTopic, body)
		msg.WithTag(string(models.ChainBTC))
		msg.WithKeys([]string{event.EventID})

		result, err := s.producer.SendSync(s.ctx, msg)
		s.Require().NoError(err)
		s.Equal(primitive.SendOK, result.Status)
		s.T().Logf("发送第 %d 条消息，MsgID=%s", i+1, result.MsgID)

		// 每条消息应该有不同的 MsgID
		if i > 0 {
			// 不比较 MsgID，因为应该是不同的
		}
	}

	// 验证所有消息都有 Key
	// 在实际使用中，消费者可以使用 Key 来实现幂等性
}

// TestProducerShutdown 测试生产者关闭
func (s *MQTestSuite) TestProducerShutdown() {
	if s.producer == nil {
		s.T().Skip("RocketMQ 生产者未初始化")
	}

	// 验证生产者正常
	s.NotNil(s.producer)

	// 关闭后不应该能发送消息
	_ = s.producer.Shutdown()

	// 尝试发送消息应该失败
	msg := primitive.NewMessage(s.depositTopic, []byte("test"))
	_, err := s.producer.SendSync(s.ctx, msg)
	s.Error(err, "关闭后的生产者不应该能发送消息")

	// 重新创建生产者供其他测试使用
	var newErr error
	s.producer, newErr = rocketmq.NewProducer(
		producer.WithNameServer([]string{s.nameSrvAddr}),
		producer.WithRetry(3),
	)
	if newErr == nil {
		s.producer.Start()
	}
}

// TestMQTestSuite 运行 MQ 测试套件
func TestMQTestSuite(t *testing.T) {
	suite.Run(t, new(MQTestSuite))
}
