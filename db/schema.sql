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
-- 钱包配置表 (wallet_configs)
-- ============================================
CREATE TABLE IF NOT EXISTS wallet_configs (
    id BIGSERIAL PRIMARY KEY,
    chain VARCHAR(16) NOT NULL,
    wallet_type VARCHAR(16) NOT NULL,
    address VARCHAR(128) NOT NULL,
    config JSONB NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    CONSTRAINT uq_wallet_config UNIQUE (chain, wallet_type, address)
);

CREATE INDEX IF NOT EXISTS idx_wallet_configs_chain ON wallet_configs(chain);
CREATE INDEX IF NOT EXISTS idx_wallet_configs_type ON wallet_configs(wallet_type);
CREATE INDEX IF NOT EXISTS idx_wallet_configs_status ON wallet_configs(status);

COMMENT ON TABLE wallet_configs IS '钱包配置表';
COMMENT ON COLUMN wallet_configs.wallet_type IS '钱包类型: hot_wallet(热钱包), cold_wallet(冷钱包)';
COMMENT ON COLUMN wallet_configs.status IS '状态: active(活跃), inactive(停用), maintenance(维护中)';
COMMENT ON COLUMN wallet_configs.config IS '钱包配置: {sign_type, hsm_key_id, m, n, signers}';

-- ============================================
-- 归档记录表 (archive_records)
-- ============================================
CREATE TABLE IF NOT EXISTS archive_records (
    id BIGSERIAL PRIMARY KEY,
    archive_id VARCHAR(64) NOT NULL UNIQUE,
    chain VARCHAR(16) NOT NULL,
    from_address VARCHAR(128) NOT NULL,
    to_address VARCHAR(128) NOT NULL,
    amount NUMERIC(36, 18) NOT NULL,
    tx_hash VARCHAR(128),
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    confirmations INTEGER NOT NULL DEFAULT 0,
    required_confirmations INTEGER NOT NULL,
    trigger_reason VARCHAR(32) NOT NULL,
    trigger_balance NUMERIC(36, 18),
    threshold NUMERIC(36, 18),
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    confirmed_at BIGINT,
    CONSTRAINT uq_archive_tx UNIQUE (chain, tx_hash)
);

CREATE INDEX IF NOT EXISTS idx_archive_records_chain ON archive_records(chain);
CREATE INDEX IF NOT EXISTS idx_archive_records_status ON archive_records(status);
CREATE INDEX IF NOT EXISTS idx_archive_records_tx_hash ON archive_records(tx_hash);
CREATE INDEX IF NOT EXISTS idx_archive_records_created_at ON archive_records(created_at);

COMMENT ON TABLE archive_records IS '归档记录表';
COMMENT ON COLUMN archive_records.status IS '状态: pending(待确认), confirmed(已确认), failed(失败)';
COMMENT ON COLUMN archive_records.trigger_reason IS '触发原因: threshold(超阈值), scheduled(定时), manual(手动)';

-- ============================================
-- 多签审批表 (multisig_approvals)
-- ============================================
CREATE TABLE IF NOT EXISTS multisig_approvals (
    id BIGSERIAL PRIMARY KEY,
    approval_id VARCHAR(64) NOT NULL UNIQUE,
    related_type VARCHAR(16) NOT NULL,
    related_id VARCHAR(64) NOT NULL,
    chain VARCHAR(16) NOT NULL,
    required_signatures INTEGER NOT NULL,
    total_signers INTEGER NOT NULL,
    tx_data TEXT NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    expire_at BIGINT NOT NULL,
    created_at BIGINT NOT NULL,
    updated_at BIGINT,
    completed_at BIGINT
);

CREATE INDEX IF NOT EXISTS idx_multisig_approvals_related ON multisig_approvals(related_type, related_id);
CREATE INDEX IF NOT EXISTS idx_multisig_approvals_status ON multisig_approvals(status);
CREATE INDEX IF NOT EXISTS idx_multisig_approvals_chain ON multisig_approvals(chain);

COMMENT ON TABLE multisig_approvals IS '多签审批表';
COMMENT ON COLUMN multisig_approvals.related_type IS '关联类型: withdrawal(提现), archive(归档)';
COMMENT ON COLUMN multisig_approvals.status IS '状态: pending(待签名), approved(已通过), rejected(已拒绝), expired(已过期)';

-- ============================================
-- 多签签名记录表 (multisig_signatures)
-- ============================================
CREATE TABLE IF NOT EXISTS multisig_signatures (
    id BIGSERIAL PRIMARY KEY,
    approval_id VARCHAR(64) NOT NULL,
    signer_id VARCHAR(64) NOT NULL,
    signature VARCHAR(256) NOT NULL,
    signed_at BIGINT NOT NULL,
    created_at BIGINT NOT NULL,
    CONSTRAINT uq_approval_signer UNIQUE (approval_id, signer_id)
);

CREATE INDEX IF NOT EXISTS idx_multisig_signatures_approval ON multisig_signatures(approval_id);
CREATE INDEX IF NOT EXISTS idx_multisig_signatures_signer ON multisig_signatures(signer_id);

COMMENT ON TABLE multisig_signatures IS '多签签名记录表';

-- ============================================
-- 归档策略配置表 (archive_policies)
-- ============================================
CREATE TABLE IF NOT EXISTS archive_policies (
    id BIGSERIAL PRIMARY KEY,
    chain VARCHAR(16) NOT NULL UNIQUE,
    enabled BOOLEAN NOT NULL DEFAULT true,
    threshold NUMERIC(36, 18) NOT NULL,
    reserve_amount NUMERIC(36, 18) NOT NULL DEFAULT 0,
    min_archive_amount NUMERIC(36, 18) NOT NULL,
    schedule VARCHAR(64),
    required_confirmations INTEGER NOT NULL DEFAULT 6,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_archive_policies_chain ON archive_policies(chain);
CREATE INDEX IF NOT EXISTS idx_archive_policies_enabled ON archive_policies(enabled);

COMMENT ON TABLE archive_policies IS '归档策略配置表';

-- ============================================
-- 审批者信息表 (approvers)
-- ============================================
CREATE TABLE IF NOT EXISTS approvers (
    id BIGSERIAL PRIMARY KEY,
    approver_id VARCHAR(64) NOT NULL UNIQUE,
    name VARCHAR(64) NOT NULL,
    email VARCHAR(128),
    pubkey TEXT,
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    role VARCHAR(32),
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_approvers_status ON approvers(status);
CREATE INDEX IF NOT EXISTS idx_approvers_role ON approvers(role);

COMMENT ON TABLE approvers IS '审批者信息表';
COMMENT ON COLUMN approvers.status IS '状态: active(活跃), inactive(停用)';
COMMENT ON COLUMN approvers.role IS '角色: finance(财务), security(安全), executive(高管)';

-- ============================================
-- 审批日志表 (approval_logs)
-- ============================================
CREATE TABLE IF NOT EXISTS approval_logs (
    id BIGSERIAL PRIMARY KEY,
    approval_id VARCHAR(64) NOT NULL,
    approver_id VARCHAR(64),
    action VARCHAR(32) NOT NULL,
    details JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_approval_logs_approval ON approval_logs(approval_id);
CREATE INDEX IF NOT EXISTS idx_approval_logs_created_at ON approval_logs(created_at);

COMMENT ON TABLE approval_logs IS '审批日志表';
COMMENT ON COLUMN approval_logs.action IS '操作类型: created, signed, approved, rejected, expired';

-- ============================================
-- 冷钱包余额追踪表 (cold_wallet_balances)
-- ============================================
CREATE TABLE IF NOT EXISTS cold_wallet_balances (
    id BIGSERIAL PRIMARY KEY,
    chain VARCHAR(16) NOT NULL,
    address VARCHAR(128) NOT NULL,
    balance NUMERIC(36, 18) NOT NULL DEFAULT 0,
    updated_at BIGINT NOT NULL,
    CONSTRAINT uq_cold_wallet UNIQUE (chain, address)
);

CREATE INDEX IF NOT EXISTS idx_cold_wallet_balances_chain ON cold_wallet_balances(chain);

COMMENT ON TABLE cold_wallet_balances IS '冷钱包余额追踪表';

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
('withdraw.daily_limit', '10', '用户每日最大提现次数', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
('hot_wallet.btc.threshold', '50000000', 'BTC 热钱包归档阈值(satoshis)', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
('hot_wallet.btc.reserve', '10000000', 'BTC 热钱包保留金额(satoshis)', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
('hot_wallet.btc.min_archive', '5000000', 'BTC 最小归档金额(satoshis)', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
('hot_wallet.eth.threshold', '10000000000000000000', 'ETH 热钱包归档阈值(wei)', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
('hot_wallet.eth.reserve', '2000000000000000000', 'ETH 热钱包保留金额(wei)', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
('hot_wallet.eth.min_archive', '1000000000000000000', 'ETH 最小归档金额(wei)', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
('withdraw.hot_limit', '10000000', '热钱包直接提现限额(satoshis)', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (config_key) DO NOTHING;

-- ============================================
-- 初始化钱包地址 (测试用，生产环境需替换)
-- ============================================
-- 热钱包地址
INSERT INTO user_addresses (user_id, chain, address, address_type, status, created_at, updated_at) VALUES
('system', 'BTC', '1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa', 'hot_wallet', 'active', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
('system', 'ETH', '0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb', 'hot_wallet', 'active', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (user_id, chain, address_type) DO NOTHING;

-- 冷钱包地址
INSERT INTO user_addresses (user_id, chain, address, address_type, status, created_at, updated_at) VALUES
('system', 'BTC', 'bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh', 'cold_wallet', 'active', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
('system', 'ETH', '0x000000000000000000000000000000000000Dead', 'cold_wallet', 'active', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (user_id, chain, address_type) DO NOTHING;

-- ============================================
-- 初始化钱包配置
-- ============================================
INSERT INTO wallet_configs (chain, wallet_type, address, config, status, created_at, updated_at) VALUES
('BTC', 'hot_wallet', '1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa', '{"sign_type": "hsm", "hsm_key_id": "hot_wallet_btc", "auto_sign": true}'::jsonb, 'active', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
('BTC', 'cold_wallet', 'bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh', '{"sign_type": "multisig", "m": 3, "n": 5}'::jsonb, 'active', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
('ETH', 'hot_wallet', '0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb', '{"sign_type": "hsm", "hsm_key_id": "hot_wallet_eth", "auto_sign": true}'::jsonb, 'active', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
('ETH', 'cold_wallet', '0x000000000000000000000000000000000000Dead', '{"sign_type": "shamir", "threshold": 3, "shares": 5}'::jsonb, 'active', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (chain, wallet_type, address) DO NOTHING;

-- ============================================
-- 初始化归档策略
-- ============================================
INSERT INTO archive_policies (chain, enabled, threshold, reserve_amount, min_archive_amount, schedule, required_confirmations, created_at, updated_at) VALUES
('BTC', true, 50000000, 10000000, 5000000, '0 */4 * * *', 6, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
('ETH', true, 10000000000000000000, 2000000000000000000, 1000000000000000000, '0 */4 * * *', 12, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (chain) DO NOTHING;

-- ============================================
-- 初始化冷钱包余额
-- ============================================
INSERT INTO cold_wallet_balances (chain, address, balance, updated_at) VALUES
('BTC', 'bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh', 0, EXTRACT(EPOCH FROM NOW())::BIGINT),
('ETH', '0x000000000000000000000000000000000000Dead', 0, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (chain, address) DO NOTHING;

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

CREATE TRIGGER update_wallet_configs_updated_at BEFORE UPDATE ON wallet_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_archive_records_updated_at BEFORE UPDATE ON archive_records
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_multisig_approvals_updated_at BEFORE UPDATE ON multisig_approvals
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_approvers_updated_at BEFORE UPDATE ON approvers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_archive_policies_updated_at BEFORE UPDATE ON archive_policies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_cold_wallet_balances_updated_at BEFORE UPDATE ON cold_wallet_balances
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
