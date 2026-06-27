package btc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	// BTC mainnet constants
	SatoshiPerBTC = 100000000

	// Blockstream API endpoints
	BlockstreamAPI = "https://blockstream.info/api"
)

// RPCClient represents a Bitcoin RPC client
type RPCClient struct {
	endpoint   string
	httpClient *http.Client
	logger     *zap.Logger
}

// Adapter represents the Bitcoin blockchain adapter
type Adapter struct {
	clients   []*RPCClient
	currentIdx int
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	logger    *zap.Logger
	network   string
	timeout   time.Duration
	maxRetries int
}

// TxOutput represents a transaction output
type TxOutput struct {
	Scriptpubkey string `json:"scriptpubkey"`
	Value        string `json:"value"`
}

// TxInput represents a transaction input
type TxInput struct {
	Scriptsig string `json:"scriptsig"`
}

// MempoolTx represents a mempool transaction
type MempoolTx struct {
	TxID     string     `json:"txid"`
	Version  int        `json:"version"`
	Inputs   []TxInput  `json:"vin"`
	Outputs  []TxOutput `json:"vout"`
	Status   Status     `json:"status"`
	Fee      int        `json:"fee"`
}

// Status represents transaction status
type Status struct {
	Confirmed   bool   `json:"confirmed"`
	BlockHeight int    `json:"block_height"`
	BlockHash   string `json:"block_hash"`
	BlockTime   int64  `json:"block_time"`
}

// Block represents a Bitcoin block
type Block struct {
	ID     string `json:"id"`
	Height int    `json:"height"`
	Timestamp int64 `json:"timestamp"`
}

// AddressInfo represents address information
type AddressInfo struct {
	Address       string   `json:"address"`
	ChainStats    ChainStats `json:"chain_stats"`
	MempoolStats  ChainStats `json:"mempool_stats"`
}

// ChainStats represents chain statistics
type ChainStats struct {
	FundedTxoSum int `json:"funded_txo_sum"`
	SpentTxoSum  int `json:"spent_txo_sum"`
	TxCount      int `json:"tx_count"`
}

// TxResult represents a transaction query result
type TxResult struct {
	TxID     string     `json:"txid"`
	Version  int        `json:"version"`
	Inputs   []TxInput  `json:"vin"`
	Outputs  []TxOutput `json:"vout"`
	Status   Status     `json:"status"`
	Fee      int        `json:"fee"`
}

// NewAdapter creates a new Bitcoin blockchain adapter
func NewAdapter(endpoints []string, network string, timeout time.Duration, maxRetries int, logger *zap.Logger) *Adapter {
	ctx, cancel := context.WithCancel(context.Background())

	clients := make([]*RPCClient, 0, len(endpoints))
	for _, endpoint := range endpoints {
		clients = append(clients, &RPCClient{
			endpoint: endpoint,
			httpClient: &http.Client{
				Timeout: timeout,
			},
			logger: logger.Named("rpc-client"),
		})
	}

	return &Adapter{
		clients:    clients,
		currentIdx: 0,
		ctx:        ctx,
		cancel:     cancel,
		logger:     logger.Named("btc-adapter"),
		network:    network,
		timeout:    timeout,
		maxRetries: maxRetries,
	}
}

// GetCurrentClient returns the current RPC client with failover
func (a *Adapter) GetCurrentClient() *RPCClient {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if len(a.clients) == 0 {
		return nil
	}

	// Round-robin through clients
	idx := a.currentIdx % len(a.clients)
	a.currentIdx++
	return a.clients[idx]
}

// GetAddressBalance gets the balance of an address
func (r *RPCClient) GetAddressBalance(ctx context.Context, address string) (int64, error) {
	url := fmt.Sprintf("%s/address/%s", r.endpoint, address)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to get address info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var addrInfo AddressInfo
	if err := json.NewDecoder(resp.Body).Decode(&addrInfo); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	// Balance = funded - spent
	balance := int64(addrInfo.ChainStats.FundedTxoSum - addrInfo.ChainStats.SpentTxoSum)
	balance += int64(addrInfo.MempoolStats.FundedTxoSum - addrInfo.MempoolStats.SpentTxoSum)

	return balance, nil
}

// GetAddressTransactions gets transactions for an address
func (r *RPCClient) GetAddressTransactions(ctx context.Context, address string, lastTxID string) ([]MempoolTx, error) {
	url := fmt.Sprintf("%s/address/%s/txs", r.endpoint, address)
	if lastTxID != "" {
		url = fmt.Sprintf("%s/address/%s/txs/chain/%s", r.endpoint, address, lastTxID)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get address transactions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var txs []MempoolTx
	if err := json.NewDecoder(resp.Body).Decode(&txs); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return txs, nil
}

// GetMempoolTransactions gets mempool transactions
func (r *RPCClient) GetMempoolTransactions(ctx context.Context, address string) ([]MempoolTx, error) {
	url := fmt.Sprintf("%s/address/%s/txs/mempool", r.endpoint, address)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get mempool transactions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// No mempool transactions is OK
		if resp.StatusCode == http.StatusNotFound {
			return []MempoolTx{}, nil
		}
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var txs []MempoolTx
	if err := json.NewDecoder(resp.Body).Decode(&txs); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return txs, nil
}

// GetLatestBlockHeight gets the latest block height
func (r *RPCClient) GetLatestBlockHeight(ctx context.Context) (int64, error) {
	url := fmt.Sprintf("%s/blocks/tip/height", r.endpoint)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to get tip height: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %w", err)
	}

	var height int64
	if _, err := fmt.Sscanf(string(body), "%d", &height); err != nil {
		return 0, fmt.Errorf("failed to parse height: %w", err)
	}

	return height, nil
}

// GetTransaction gets a transaction by ID
func (r *RPCClient) GetTransaction(ctx context.Context, txID string) (*TxResult, error) {
	url := fmt.Sprintf("%s/tx/%s", r.endpoint, txID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var tx TxResult
	if err := json.NewDecoder(resp.Body).Decode(&tx); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &tx, nil
}

// GetBalance gets the balance of an address with retry
func (a *Adapter) GetBalance(ctx context.Context, address string) (int64, error) {
	var lastErr error

	for i := 0; i < a.maxRetries; i++ {
		client := a.GetCurrentClient()
		if client == nil {
			return 0, fmt.Errorf("no available RPC client")
		}

		balance, err := client.GetAddressBalance(ctx, address)
		if err == nil {
			return balance, nil
		}

		lastErr = err
		a.logger.Warn("Failed to get balance, retrying",
			zap.Int("attempt", i+1),
			zap.String("address", address),
			zap.Error(err))

		// Wait before retry
		if i < a.maxRetries-1 {
			time.Sleep(time.Second * time.Duration(i+1))
		}
	}

	return 0, fmt.Errorf("failed after %d retries: %w", a.maxRetries, lastErr)
}

// GetTransactions gets transactions for an address
func (a *Adapter) GetTransactions(ctx context.Context, address string, lastTxID string) ([]MempoolTx, error) {
	client := a.GetCurrentClient()
	if client == nil {
		return nil, fmt.Errorf("no available RPC client")
	}

	return client.GetAddressTransactions(ctx, address, lastTxID)
}

// GetMempoolTransactions gets mempool transactions for an address
func (a *Adapter) GetMempoolTransactions(ctx context.Context, address string) ([]MempoolTx, error) {
	client := a.GetCurrentClient()
	if client == nil {
		return nil, fmt.Errorf("no available RPC client")
	}

	return client.GetMempoolTransactions(ctx, address)
}

// GetLatestBlockHeight gets the latest block height
func (a *Adapter) GetLatestBlockHeight(ctx context.Context) (int64, error) {
	client := a.GetCurrentClient()
	if client == nil {
		return 0, fmt.Errorf("no available RPC client")
	}

	return client.GetLatestBlockHeight(ctx)
}

// GetTransaction gets a transaction by ID
func (a *Adapter) GetTransaction(ctx context.Context, txID string) (*TxResult, error) {
	client := a.GetCurrentClient()
	if client == nil {
		return nil, fmt.Errorf("no available RPC client")
	}

	return client.GetTransaction(ctx, txID)
}

// Close closes the adapter
func (a *Adapter) Close() error {
	a.cancel()
	return nil
}

// IsAddressForWallet checks if a transaction output is for a given address
func (a *Adapter) IsAddressForWallet(scriptPubKey, address string) bool {
	// This is a simplified check. In production, you would need to
	// properly decode the scriptPubKey and compare the address
	return true
}

// GetChainType returns the chain type
func (a *Adapter) GetChainType() string {
	return "BTC"
}
