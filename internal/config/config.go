package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Server     ServerConfig              `yaml:"server"`
	Database   DatabaseConfig            `yaml:"database"`
	Redis      RedisConfig               `yaml:"redis"`
	RocketMQ   RocketMQConfig            `yaml:"rocketmq"`
	Blockchain BlockchainConfig          `yaml:"blockchain"`
	HotWallet  HotWalletConfig           `yaml:"hot_wallet"`
	ColdWallet ColdWalletConfig          `yaml:"cold_wallet"`
	Deposit    DepositConfig             `yaml:"deposit"`
	Withdraw   WithdrawConfig            `yaml:"withdraw"`
	Multisig   MultisigConfig             `yaml:"multisig"`
	Reconcile  ReconciliationConfig       `yaml:"reconciliation"`
	Logging    LoggingConfig              `yaml:"logging"`
	Monitoring MonitoringConfig          `yaml:"monitoring"`
	Security   SecurityConfig             `yaml:"security"`
	Test       TestConfig                 `yaml:"test"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	Mode            string        `yaml:"mode"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Postgres PostgresConfig `yaml:"postgres"`
}

// PostgresConfig holds PostgreSQL configuration
type PostgresConfig struct {
	Host              string        `yaml:"host"`
	Port              int           `yaml:"port"`
	Database          string        `yaml:"database"`
	User              string        `yaml:"user"`
	Password          string        `yaml:"password"`
	SSLMode           string        `yaml:"ssl_mode"`
	MaxOpenConns      int           `yaml:"max_open_conns"`
	MaxIdleConns      int           `yaml:"max_idle_conns"`
	ConnMaxLifetime   time.Duration `yaml:"conn_max_lifetime"`
	ConnMaxIdleTime   time.Duration `yaml:"conn_max_idle_time"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	Password     string `yaml:"password"`
	Database     int    `yaml:"database"`
	PoolSize     int    `yaml:"pool_size"`
	MinIdleConns int    `yaml:"min_idle_conns"`
}

// RocketMQConfig holds RocketMQ configuration
type RocketMQConfig struct {
	NameSrvAddr      string            `yaml:"name_srv_addr"`
	GroupName        string            `yaml:"group_name"`
	ConsumeTimeout   time.Duration     `yaml:"consume_timeout"`
	RetryTimes       int               `yaml:"retry_times"`
	Topics           map[string]string `yaml:"topics"`
}

// BlockchainConfig holds blockchain configuration
type BlockchainConfig struct {
	BTC  ChainConfig `yaml:"btc"`
	ETH  ChainConfig `yaml:"eth"`
	TRX  ChainConfig `yaml:"tron"`
	SOL  ChainConfig `yaml:"sol"`
}

// ChainConfig holds individual chain configuration
type ChainConfig struct {
	Enabled           bool              `yaml:"enabled"`
	Network           string            `yaml:"network"`
	RPCEndpoints      []string          `yaml:"rpc_endpoints"`
	WSEndpoint        string            `yaml:"websocket_endpoint"`
	Timeout           time.Duration     `yaml:"timeout"`
	MaxRetries        int               `yaml:"max_retries"`
	RetryDelay        time.Duration     `yaml:"retry_delay"`
}

// HotWalletConfig holds hot wallet configuration
type HotWalletConfig struct {
	PrivateKeys      map[string]string `yaml:"private_keys"`
	ArchiveThreshold map[string]string `yaml:"archive_threshold"`
	MinBalance       map[string]string `yaml:"min_balance"`
}

// ColdWalletConfig holds cold wallet configuration
type ColdWalletConfig struct {
	HardwareWallet HardwareWalletConfig `yaml:"hardware_wallet"`
	HSM            HSMConfig            `yaml:"hsm"`
	Shamir         ShamirConfig         `yaml:"shamir"`
	Addresses      map[string]string    `yaml:"addresses"`
}

// HardwareWalletConfig holds hardware wallet configuration
type HardwareWalletConfig struct {
	Enabled bool   `yaml:"enabled"`
	Type    string `yaml:"type"`
}

// HSMConfig holds HSM configuration
type HSMConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Provider string `yaml:"provider"`
	Endpoint string `yaml:"endpoint"`
	Pin      string `yaml:"pin"`
}

// ShamirConfig holds Shamir secret sharing configuration
type ShamirConfig struct {
	Enabled   bool `yaml:"enabled"`
	Threshold int  `yaml:"threshold"`
	Shares    int  `yaml:"shares"`
}

// DepositConfig holds deposit configuration
type DepositConfig struct {
	Confirmations map[string]int `yaml:"confirmations"`
	BatchSize     int            `yaml:"batch_size"`
	CheckInterval time.Duration  `yaml:"check_interval"`
}

// WithdrawConfig holds withdrawal configuration
type WithdrawConfig struct {
	Limits            map[string]LimitConfig    `yaml:"limits"`
	DailyLimit         int                       `yaml:"daily_limit"`
	Fees               map[string]interface{}    `yaml:"fees"`
	ApprovalThreshold  map[string]float64        `yaml:"approval_threshold"`
	ApprovalTimeout    time.Duration             `yaml:"approval_timeout"`
}

// LimitConfig holds withdrawal limit configuration
type LimitConfig struct {
	Min string `yaml:"min"`
	Max string `yaml:"max"`
}

// GetBTCFee returns the BTC fee in satoshis
func (w *WithdrawConfig) GetBTCFee() int {
	if val, ok := w.Fees["btc"]; ok {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		case string:
			// Try to parse string as int
			var fee int
			fmt.Sscanf(v, "%d", &fee)
			return fee
		}
	}
	return 1000 // Default fallback
}

// GetETHGasConfig returns the ETH gas configuration
func (w *WithdrawConfig) GetETHGasConfig() (gasLimit int, gasPrice string) {
	if val, ok := w.Fees["eth"]; ok {
		if m, ok := val.(map[string]interface{}); ok {
			if gl, ok := m["gas_limit"]; ok {
				switch v := gl.(type) {
				case int:
					gasLimit = v
				case float64:
					gasLimit = int(v)
				}
			}
			if gp, ok := m["gas_price"]; ok {
				if s, ok := gp.(string); ok {
					gasPrice = s
				}
			}
		}
	}
	// Defaults
	if gasLimit == 0 {
		gasLimit = 21000
	}
	if gasPrice == "" {
		gasPrice = "auto"
	}
	return
}

// MultisigConfig holds multisig configuration
type MultisigConfig struct {
	Enabled bool     `yaml:"enabled"`
	K       int      `yaml:"k"`
	N       int      `yaml:"n"`
	Signers []string `yaml:"signers"`
}

// ReconciliationConfig holds reconciliation configuration
type ReconciliationConfig struct {
	Schedule              string        `yaml:"schedule"`
	FullCheckEnabled      bool          `yaml:"full_check_enabled"`
	IncrementalInterval   time.Duration `yaml:"incremental_interval"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
	File   struct {
		Path       string `yaml:"path"`
		MaxSize    int    `yaml:"max_size"`
		MaxBackups int    `yaml:"max_backups"`
		MaxAge     int    `yaml:"max_age"`
		Compress   bool   `yaml:"compress"`
	} `yaml:"file"`
}

// MonitoringConfig holds monitoring configuration
type MonitoringConfig struct {
	HealthCheck     HealthCheckConfig    `yaml:"health_check"`
	ChainMonitor    ChainMonitorConfig   `yaml:"chain_monitor"`
	FailoverThreshold int                 `yaml:"failover_threshold"`
}

// HealthCheckConfig holds health check configuration
type HealthCheckConfig struct {
	Enabled  bool          `yaml:"enabled"`
	Interval time.Duration `yaml:"interval"`
}

// ChainMonitorConfig holds chain monitor configuration
type ChainMonitorConfig struct {
	Enabled        bool          `yaml:"enabled"`
	Interval       time.Duration `yaml:"interval"`
	LagThreshold   int           `yaml:"lag_threshold"`
}

// SecurityConfig holds security configuration
type SecurityConfig struct {
	JWT       JWTConfig       `yaml:"jwt"`
	APIKey    APIKeyConfig    `yaml:"api_key"`
	IPWhitelist IPWhitelistConfig `yaml:"ip_whitelist"`
	RateLimit RateLimitConfig `yaml:"rate_limit"`
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret  string        `yaml:"secret"`
	Expiry  time.Duration `yaml:"expiry"`
}

// APIKeyConfig holds API key configuration
type APIKeyConfig struct {
	Enabled bool     `yaml:"enabled"`
	Keys    []string `yaml:"keys"`
}

// IPWhitelistConfig holds IP whitelist configuration
type IPWhitelistConfig struct {
	Enabled    bool     `yaml:"enabled"`
	AllowedIPs []string `yaml:"allowed_ips"`
}

// RateLimitConfig holds rate limit configuration
type RateLimitConfig struct {
	Enabled           bool `yaml:"enabled"`
	RequestsPerMinute int  `yaml:"requests_per_minute"`
	Burst             int  `yaml:"burst"`
}

// TestConfig holds test configuration
type TestConfig struct {
	Testnet  bool          `yaml:"testnet"`
	TestUsers []TestUser   `yaml:"test_users"`
}

// TestUser represents a test user
type TestUser struct {
	UserID  string `yaml:"user_id"`
	Chain   string `yaml:"chain"`
	Address string `yaml:"address"`
}

// Load loads the configuration from a file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// LoadOrDefault loads the configuration from default paths
func LoadOrDefault() (*Config, error) {
	// Get the executable's directory to find relative paths
	exePath, err := os.Executable()
	if err != nil {
		// If we can't get the exe path, try relative paths
		exePath = "."
	}
	exeDir := "."
	if exePath != "." {
		exeDir = filepath.Dir(exePath)
	}

	paths := []string{
		"./config/config.yml",
		"./config/config.yaml",
		filepath.Join(exeDir, "config/config.yml"),
		filepath.Join(exeDir, "config/config.yaml"),
		"/etc/crypto-bridge/config.yml",
	}

	var parseErr error
	attemptedPaths := []string{}

	for _, path := range paths {
		// Check if file exists before attempting to read
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}
		attemptedPaths = append(attemptedPaths, path)

		cfg, err := Load(path)
		if err == nil {
			return cfg, nil
		}
		parseErr = err
	}

	if len(attemptedPaths) == 0 {
		return nil, fmt.Errorf("no config file found, searched paths: %v", paths)
	}

	if parseErr != nil {
		return nil, fmt.Errorf("failed to parse config from attempted paths %v: %w", attemptedPaths, parseErr)
	}

	return nil, fmt.Errorf("no valid config file found in default paths: %v", paths)
}
