package service

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"crypto-bridge/internal/blockchain/btc"
)

// BTCDepositAdapter adapts BTC DepositTransaction to service DepositInfo
type BTCDepositAdapter struct {
	depositService *DepositService
	logger         *zap.Logger
}

// NewBTCDepositAdapter creates a new BTC deposit adapter
func NewBTCDepositAdapter(depositService *DepositService, logger *zap.Logger) *BTCDepositAdapter {
	return &BTCDepositAdapter{
		depositService: depositService,
		logger:         logger.Named("btc-deposit-adapter"),
	}
}

// HandleDeposit implements the btc.DepositHandler interface
func (a *BTCDepositAdapter) HandleDeposit(ctx context.Context, deposit *btc.DepositTransaction) error {
	if deposit == nil {
		return fmt.Errorf("nil deposit")
	}

	// Convert BTC deposit to service deposit info
	depositInfo := &DepositInfo{
		TxHash:        deposit.TxHash,
		FromAddress:   deposit.FromAddress,
		ToAddress:     deposit.ToAddress,
		Amount:        deposit.Amount,
		BlockHeight:   deposit.BlockHeight,
		BlockHash:     deposit.BlockHash,
		Timestamp:     deposit.Timestamp,
		Confirmations: deposit.Confirmations,
		Status:        deposit.Status,
	}

	return a.depositService.HandleDeposit(ctx, depositInfo)
}
