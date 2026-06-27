-- Web3 多链钱包系统数据库表结构
-- 版本: v0.1.0
-- 数据库: PostgreSQL 14+

-- ============================================
-- 用户地址表 (user_addresses)
-- ============================================
CREATE TABLE IF NOT EXISTS user_addresses (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL,
    chain VARCHAR(16) NOT NULL,
    address VARCHAR(128) NOT NULL,
    address_type VARCHAR(16) NOT NULL DEFAULT 'deposit',
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    CONSTRAINT uq_user_address UNIQUE (user_id, chain, address_type),
    CONSTRAINT uq_address UNIQUE (address)
);

CREATE INDEX IF NOT EXISTS idx_user_addresses_user_id ON user_addresses(user_id);
CREATE INDEX IF NOT EXISTS idx_user_addresses_address ON user_addresses(address);
CREATE INDEX IF NOT EXISTS idx_user_addresses_chain ON user_addresses(chain);

COMMENT ON TABLE user_addresses IS '用户地址映射表';
COMMENT ON COLUMN user_addresses.address_type IS '地址类型: deposit(充值), hot_wallet(热钱包), cold_wallet(冷钱包)';
COMMENT ON COLUMN user_addresses.status IS '状态: active(活跃), inactive(停用)';

-- ============================================
-- 充值记录表 (deposits)
-- ============================================
CREATE TABLE IF NOT EXISTS deposits (
    id BIGSERIAL PRIMARY KEY,
    deposit_id VARCHAR(64) NOT NULL UNIQUE,
    user_id VARCHAR(64) NOT NULL,
    chain VARCHAR(16) NOT NULL,
    tx_hash VARCHAR(128) NOT NULL,
    from_address VARCHAR(128),
    to_address VARCHAR(128) NOT NULL,
    amount NUMERIC(36, 18) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    confirmations INTEGER NOT NULL DEFAULT 0,
    required_confirmations INTEGER NOT NULL,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    completed_at BIGINT,
    CONSTRAINT uq_deposit_tx UNIQUE (chain, tx_hash)
);

CREATE INDEX IF NOT EXISTS idx_deposits_user_id ON deposits(user_id);

CREATE INDEX IF NOT EXISTS idx_deposits_user_id ON deposits(user_id);
CREATE INDEX IF NOT EXISTS idx_deposits_tx_hash ON deposits(tx_hash);
CREATE INDEX IF NOT EXISTS idx_deposits_status ON deposits(status);
CREATE INDEX IF NOT EXISTS idx_deposits_chain ON deposits(chain);
CREATE INDEX IF NOT EXISTS idx_deposits_created_at ON deposits(created_at);

COMMENT ON TABLE deposits IS '充值记录表';
COMMENT ON COLUMN deposits.status IS '状态: pending(待确认), confirmed(已确认), completed(已完成)';

-- ============================================
-- 提现记录表 (withdrawals)
-- ============================================
CREATE TABLE IF NOT EXISTS withdrawals (
    id BIGSERIAL PRIMARY KEY,
    withdraw_id VARCHAR(64) NOT NULL UNIQUE,
    user_id VARCHAR(64) NOT NULL,
    chain VARCHAR(16) NOT NULL,
    to_address VARCHAR(128) NOT NULL,
    amount NUMERIC(36, 18) NOT NULL,
    fee NUMERIC(36, 18) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    tx_hash VARCHAR(128),
    confirmations INTEGER NOT NULL DEFAULT 0,
    required_confirmations INTEGER NOT NULL,
    memo TEXT,
    reject_reason TEXT,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    approved_at BIGINT,
    completed_at BIGINT,
    CONSTRAINT uq_withdraw_tx UNIQUE (chain, tx_hash)
);

CREATE INDEX IF NOT EXISTS idx_withdrawals_user_id ON withdrawals(user_id);

CREATE INDEX IF NOT EXISTS idx_withdrawals_user_id ON withdrawals(user_id);
CREATE INDEX IF NOT EXISTS idx_withdrawals_status ON withdrawals(status);
CREATE INDEX IF NOT EXISTS idx_withdrawals_chain ON withdrawals(chain);
CREATE INDEX IF NOT EXISTS idx_withdrawals_tx_hash ON withdrawals(tx_hash);
CREATE INDEX IF NOT EXISTS idx_withdrawals_created_at ON withdrawals(created_at);

COMMENT ON TABLE withdrawals IS '提现记录表';
COMMENT ON COLUMN withdrawals.status IS '状态: pending(待审批), approved(已批准), rejected(已拒绝), completed(已完成), failed(失败)';

-- ============================================
-- 用户余额表 (user_balances)
-- ============================================
CREATE TABLE IF NOT EXISTS user_balances (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL,
    chain VARCHAR(16) NOT NULL,
    balance NUMERIC(36, 18) NOT NULL DEFAULT 0,
    locked_balance NUMERIC(36, 18) NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    CONSTRAINT uq_user_balance UNIQUE (user_id, chain)
);

CREATE INDEX IF NOT EXISTS idx_user_balances_user_id ON user_balances(user_id);
CREATE INDEX IF NOT EXISTS idx_user_balances_chain ON user_balances(chain);

COMMENT ON TABLE user_balances IS '用户余额表';
COMMENT ON COLUMN user_balances.balance IS '可用余额';
COMMENT ON COLUMN user_balances.locked_balance IS '冻结余额(提现中)';

-- ============================================
-- 余额变更流水表 (balance_transactions)
-- ============================================
CREATE TABLE IF NOT EXISTS balance_transactions (
    id BIGSERIAL PRIMARY KEY,
    transaction_id VARCHAR(64) NOT NULL UNIQUE,
    user_id VARCHAR(64) NOT NULL,
    chain VARCHAR(16) NOT NULL,
    type VARCHAR(16) NOT NULL,
    amount NUMERIC(36, 18) NOT NULL,
    balance_before NUMERIC(36, 18) NOT NULL,
    balance_after NUMERIC(36, 18) NOT NULL,
    related_id VARCHAR(64),
    created_at BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_balance_transactions_user_id ON balance_transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_balance_transactions_chain ON balance_transactions(chain);
CREATE INDEX IF NOT EXISTS idx_balance_transactions_type ON balance_transactions(type);
CREATE INDEX IF NOT EXISTS idx_balance_transactions_related_id ON balance_transactions(related_id);
CREATE INDEX IF NOT EXISTS idx_balance_transactions_created_at ON balance_transactions(created_at);

COMMENT ON TABLE balance_transactions IS '余额变更流水表';
COMMENT ON COLUMN balance_transactions.type IS '变更类型: deposit(充值), withdraw(提现), freeze(冻结), unfreeze(解冻), transfer(转账)';

-- ============================================
-- 链上交易追踪表 (chain_transactions)
-- ============================================
CREATE TABLE IF NOT EXISTS chain_transactions (
    id BIGSERIAL PRIMARY KEY,
    chain VARCHAR(16) NOT NULL,
    tx_hash VARCHAR(128) NOT NULL,
    type VARCHAR(16) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    confirmations INTEGER NOT NULL DEFAULT 0,
    required_confirmations INTEGER NOT NULL,
    raw_tx JSONB,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    CONSTRAINT uq_chain_tx UNIQUE (chain, tx_hash)
);

CREATE INDEX IF NOT EXISTS idx_chain_transactions_chain ON chain_transactions(chain);
CREATE INDEX IF NOT EXISTS idx_chain_transactions_tx_hash ON chain_transactions(tx_hash);
CREATE INDEX IF NOT EXISTS idx_chain_transactions_status ON chain_transactions(status);

COMMENT ON TABLE chain_transactions IS '链上交易追踪表';
COMMENT ON COLUMN chain_transactions.type IS '交易类型: deposit(充值), withdraw(提现), archive(归档)';
COMMENT ON COLUMN chain_transactions.status IS '状态: pending(待确认), confirmed(已确认), failed(失败)';

-- ============================================
-- 系统配置表 (system_configs)
-- ============================================
CREATE TABLE IF NOT EXISTS system_configs (
    id BIGSERIAL PRIMARY KEY,
    config_key VARCHAR(64) NOT NULL UNIQUE,
    config_value TEXT NOT NULL,
    description TEXT,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_system_configs_key ON system_configs(config_key);

COMMENT ON TABLE system_configs IS '系统配置表';

-- ============================================
-- 初始化配置数据
-- ============================================
INSERT INTO system_configs (config_key, config_value, description, created_at, updated_at) VALUES
('btc.min_deposit', '1000', 'BTC 最小充值金额(satoshis)', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
('btc.max_withdraw', '10000000', 'BTC 单笔最大提现金额(satoshis)', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
('btc.confirmations', '6', 'BTC 充值确认数', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
('btc.fee', '1000', 'BTC 默认手续费(satoshis/byte)', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
('eth.min_deposit', '1000000000000000', 'ETH 最小充值金额(wei)', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
('eth.max_withdraw', '10000000000000000000', 'ETH 单笔最大提现金额(wei)', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
('eth.confirmations', '12', 'ETH 充值确认数', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
('eth.gas_limit', '21000', 'ETH 默认 Gas Limit', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
('withdraw.daily_limit', '10', '用户每日最大提现次数', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (config_key) DO NOTHING;

-- ============================================
-- 初始化热钱包地址 (测试用，生产环境需替换)
-- ============================================
INSERT INTO user_addresses (user_id, chain, address, address_type, status, created_at, updated_at) VALUES
('system', 'BTC', '1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa', 'hot_wallet', 'active', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
('system', 'ETH', '0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb', 'hot_wallet', 'active', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (user_id, chain, address_type) DO NOTHING;

-- ============================================
-- 创建更新时间触发器函数
-- ============================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 为需要的表添加更新时间触发器
CREATE TRIGGER update_user_addresses_updated_at BEFORE UPDATE ON user_addresses
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_deposits_updated_at BEFORE UPDATE ON deposits
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_withdrawals_updated_at BEFORE UPDATE ON withdrawals
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_user_balances_updated_at BEFORE UPDATE ON user_balances
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_chain_transactions_updated_at BEFORE UPDATE ON chain_transactions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_system_configs_updated_at BEFORE UPDATE ON system_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
