package btc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// DepositTransaction represents a deposit transaction
type DepositTransaction struct {
	TxHash        string
	FromAddress   string
	ToAddress     string
	Amount        int64  // in satoshis
	BlockHeight   int64
	BlockHash     string
	Timestamp     int64
	Confirmations int
	Status        string // "pending", "confirmed", "completed"
}

// DepositHandler handles deposit events
type DepositHandler interface {
	HandleDeposit(ctx context.Context, deposit *DepositTransaction) error
}

// ListenService listens for Bitcoin transactions
type ListenService struct {
	adapter         *Adapter
	depositHandler  DepositHandler
	watchAddresses  map[string]string // address -> user_id
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	logger          *zap.Logger
	checkInterval   time.Duration
	lastBlockHeight int64
	requiredConfirms int
}

// NewListenService creates a new Bitcoin listen service
func NewListenService(adapter *Adapter, handler DepositHandler, requiredConfirms int,
	checkInterval time.Duration, logger *zap.Logger) *ListenService {

	ctx, cancel := context.WithCancel(context.Background())

	return &ListenService{
		adapter:         adapter,
		depositHandler:  handler,
		watchAddresses:  make(map[string]string),
		ctx:             ctx,
		cancel:          cancel,
		logger:          logger.Named("btc-listen-service"),
		checkInterval:   checkInterval,
		requiredConfirms: requiredConfirms,
	}
}

// AddWatchAddress adds an address to watch
func (s *ListenService) AddWatchAddress(address, userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.watchAddresses[address] = userID
	s.logger.Info("Added watch address",
		zap.String("address", address),
		zap.String("user_id", userID))
}

// RemoveWatchAddress removes an address from watching
func (s *ListenService) RemoveWatchAddress(address string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.watchAddresses, address)
	s.logger.Info("Removed watch address", zap.String("address", address))
}

// GetWatchAddresses returns all watch addresses
func (s *ListenService) GetWatchAddresses() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]string)
	for addr, userID := range s.watchAddresses {
		result[addr] = userID
	}
	return result
}

// Start starts the listen service
func (s *ListenService) Start() error {
	// Get initial block height
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()

	height, err := s.adapter.GetLatestBlockHeight(ctx)
	if err != nil {
		return fmt.Errorf("failed to get initial block height: %w", err)
	}

	s.lastBlockHeight = height
	s.logger.Info("Starting BTC listen service",
		zap.Int64("initial_height", height),
		zap.Int("watch_addresses", len(s.watchAddresses)))

	s.wg.Add(1)
	go s.run()

	return nil
}

// Stop stops the listen service
func (s *ListenService) Stop() {
	s.logger.Info("Stopping BTC listen service")
	s.cancel()
	s.wg.Wait()
}

// run runs the main listen loop
func (s *ListenService) run() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.checkInterval)
	defer ticker.Stop()

	// Also check immediately on start
	s.checkDeposits(s.ctx)

	for {
		select {
		case <-s.ctx.Done():
			s.logger.Info("Listen service stopped")
			return
		case <-ticker.C:
			s.checkDeposits(s.ctx)
		}
	}
}

// checkDeposits checks for new deposits
func (s *ListenService) checkDeposits(ctx context.Context) {
	// Get current block height
	currentHeight, err := s.adapter.GetLatestBlockHeight(ctx)
	if err != nil {
		s.logger.Error("Failed to get current block height", zap.Error(err))
		return
	}

	// Check for new blocks
	if currentHeight > s.lastBlockHeight {
		s.logger.Debug("New blocks detected",
			zap.Int64("last_height", s.lastBlockHeight),
			zap.Int64("current_height", currentHeight))
	}

	// Get watch addresses
	addresses := s.GetWatchAddresses()
	if len(addresses) == 0 {
		s.logger.Debug("No watch addresses, skipping deposit check")
		s.lastBlockHeight = currentHeight
		return
	}

	// Check each address for new transactions
	for address, userID := range addresses {
		if err := s.checkAddressDeposits(ctx, address, userID); err != nil {
			s.logger.Error("Failed to check address deposits",
				zap.String("address", address),
				zap.Error(err))
		}
	}

	s.lastBlockHeight = currentHeight
}

// checkAddressDeposits checks for deposits to a specific address
func (s *ListenService) checkAddressDeposits(ctx context.Context, address, userID string) error {
	// Get confirmed transactions
	txs, err := s.adapter.GetTransactions(ctx, address, "")
	if err != nil {
		return fmt.Errorf("failed to get transactions: %w", err)
	}

	// Get mempool transactions
	mempoolTxs, err := s.adapter.GetMempoolTransactions(ctx, address)
	if err != nil {
		s.logger.Warn("Failed to get mempool transactions",
			zap.String("address", address),
			zap.Error(err))
	} else {
		txs = append(mempoolTxs, txs...)
	}

	// Process each transaction
	for _, tx := range txs {
		// Check if this is an incoming transaction (deposit)
		deposit := s.parseDepositTransaction(&tx, address, userID)
		if deposit != nil {
			s.logger.Debug("Found deposit transaction",
				zap.String("tx_hash", deposit.TxHash),
				zap.String("to_address", deposit.ToAddress),
				zap.Int64("amount", deposit.Amount),
				zap.String("status", deposit.Status))

			// Handle the deposit
			if err := s.depositHandler.HandleDeposit(ctx, deposit); err != nil {
				s.logger.Error("Failed to handle deposit",
					zap.String("tx_hash", deposit.TxHash),
					zap.Error(err))
			}
		}
	}

	return nil
}

// parseDepositTransaction parses a transaction into a deposit if it's incoming
func (s *ListenService) parseDepositTransaction(tx *MempoolTx, watchAddress, userID string) *DepositTransaction {
	var fromAddress string

	// Get from address (simplified - in production, properly parse inputs)
	if len(tx.Inputs) > 0 {
		fromAddress = "unknown" // Would need to decode prevout properly
	}

	// Check outputs for our watch address
	var amount int64
	for _, output := range tx.Outputs {
		// In production, properly decode scriptPubKey to extract address
		// For now, this is a simplified check
		// The actual implementation would need to properly decode the Bitcoin script
		// output.Value is in satoshis as a string number
		var val int64
		if output.Value != "" {
			fmt.Sscanf(output.Value, "%d", &val)
			amount += val
		}
	}

	if amount <= 0 {
		return nil
	}

	// Determine status based on confirmations
	confirmations := 0
	status := "pending"

	if tx.Status.Confirmed {
		// Calculate confirmations (need current block height)
		// For now, use block height
		confirmations = 1
		status = "confirmed"
	}

	return &DepositTransaction{
		TxHash:        tx.TxID,
		FromAddress:   fromAddress,
		ToAddress:     watchAddress,
		Amount:        amount,
		BlockHeight:   int64(tx.Status.BlockHeight),
		BlockHash:     tx.Status.BlockHash,
		Timestamp:     tx.Status.BlockTime,
		Confirmations: confirmations,
		Status:        status,
	}
}

// GetStatus returns the current status of the listen service
func (s *ListenService) GetStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"chain":            "BTC",
		"running":          s.ctx.Err() == nil,
		"watch_addresses":  len(s.watchAddresses),
		"last_block_height": s.lastBlockHeight,
	}
}
