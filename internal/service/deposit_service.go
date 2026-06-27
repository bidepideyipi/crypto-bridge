package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"crypto-bridge/internal/models"
)

// DepositInfo represents deposit information from blockchain
type DepositInfo struct {
	TxHash        string
	FromAddress   string
	ToAddress     string
	Amount        int64
	BlockHeight   int64
	BlockHash     string
	Timestamp     int64
	Confirmations int
	Status        string
}

// DepositService handles deposit operations
type DepositService struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewDepositService creates a new deposit service
func NewDepositService(db *gorm.DB, logger *zap.Logger) *DepositService {
	return &DepositService{
		db:     db,
		logger: logger.Named("deposit-service"),
	}
}

// HandleDeposit processes a deposit from the blockchain
func (s *DepositService) HandleDeposit(ctx context.Context, deposit *DepositInfo) error {
	if deposit == nil {
		return fmt.Errorf("nil deposit")
	}

	s.logger.Info("Processing deposit",
		zap.String("tx_hash", deposit.TxHash),
		zap.String("to_address", deposit.ToAddress),
		zap.Int64("amount", deposit.Amount))

	// Get user ID for this address
	var userAddr models.UserAddress
	if err := s.db.Where("address = ? AND chain = ? AND status = ?",
		deposit.ToAddress, models.ChainBTC, models.AddressStatusActive).
		First(&userAddr).Error; err != nil {

		s.logger.Warn("Address not found in database",
			zap.String("address", deposit.ToAddress),
			zap.Error(err))
		return nil // Not an error - just not our address
	}

	// Check if deposit already exists
	var existingDeposit models.Deposit
	if err := s.db.Where("tx_hash = ?", deposit.TxHash).First(&existingDeposit).Error; err == nil {
		// Already exists, update confirmations
		s.updateDepositConfirmations(ctx, &existingDeposit, deposit.Confirmations, deposit.Status)
		return nil
	}

	// Create new deposit record
	depositID := s.generateDepositID(deposit.Timestamp)

	newDeposit := &models.Deposit{
		DepositID:             depositID,
		UserID:                userAddr.UserID,
		Chain:                 models.ChainBTC,
		TxHash:                deposit.TxHash,
		FromAddress:           deposit.FromAddress,
		ToAddress:             deposit.ToAddress,
		Amount:                fmt.Sprintf("%d", deposit.Amount),
		Status:                models.DepositStatusPending,
		Confirmations:         deposit.Confirmations,
		RequiredConfirmations: 6, // Default for BTC
		CreatedAt:             time.Now().Unix(),
		UpdatedAt:             time.Now().Unix(),
	}

	if deposit.Confirmations >= 6 {
		newDeposit.Status = models.DepositStatusConfirmed
	}

	if err := s.db.Create(newDeposit).Error; err != nil {
		s.logger.Error("Failed to create deposit record",
			zap.String("tx_hash", deposit.TxHash),
			zap.Error(err))
		return fmt.Errorf("failed to create deposit: %w", err)
	}

	s.logger.Info("Deposit record created",
		zap.String("deposit_id", depositID),
		zap.String("tx_hash", deposit.TxHash),
		zap.String("user_id", userAddr.UserID))

	// If confirmed, process balance update
	if newDeposit.Status == models.DepositStatusConfirmed {
		if err := s.processConfirmedDeposit(ctx, newDeposit); err != nil {
			s.logger.Error("Failed to process confirmed deposit",
				zap.String("deposit_id", depositID),
				zap.Error(err))
			return err
		}
	}

	return nil
}

// updateDepositConfirmations updates the confirmations for a deposit
func (s *DepositService) updateDepositConfirmations(ctx context.Context, deposit *models.Deposit, confirmations int, status string) error {
	updates := map[string]interface{}{
		"confirmations": confirmations,
		"updated_at":    time.Now().Unix(),
	}

	// Determine if we need to process balance update
	shouldProcessBalance := false

	// Update status based on confirmations
	if confirmations >= deposit.RequiredConfirmations && deposit.Status != models.DepositStatusCompleted {
		if deposit.Status == models.DepositStatusPending {
			// Transition from pending to confirmed - need to process balance
			shouldProcessBalance = true
		}
		updates["status"] = models.DepositStatusConfirmed
	}

	// Apply updates to database
	if err := s.db.Model(deposit).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update deposit: %w", err)
	}

	// Process balance if transitioning to confirmed
	if shouldProcessBalance {
		deposit.Status = models.DepositStatusConfirmed
		if err := s.processConfirmedDeposit(ctx, deposit); err != nil {
			return err
		}
	}

	return nil
}

// processConfirmedDeposit processes a confirmed deposit by updating user balance
func (s *DepositService) processConfirmedDeposit(ctx context.Context, deposit *models.Deposit) error {
	s.logger.Info("Processing confirmed deposit",
		zap.String("deposit_id", deposit.DepositID),
		zap.String("user_id", deposit.UserID),
		zap.String("amount", deposit.Amount))

	// Start transaction
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Get or create user balance with row lock to prevent concurrent updates
		var balance models.UserBalance
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ? AND chain = ?", deposit.UserID, deposit.Chain).
			First(&balance).Error; err != nil {

			if err == gorm.ErrRecordNotFound {
				// Create new balance record
				balance = models.UserBalance{
					UserID:        deposit.UserID,
					Chain:         deposit.Chain,
					Balance:       "0",
					LockedBalance: "0",
					CreatedAt:     time.Now().Unix(),
					UpdatedAt:     time.Now().Unix(),
				}
				if err := tx.Create(&balance).Error; err != nil {
					return fmt.Errorf("failed to create balance: %w", err)
				}
			} else {
				return fmt.Errorf("failed to query balance: %w", err)
			}
		}

		// Parse amounts
		var currentBalance, depositAmount int64
		fmt.Sscanf(balance.Balance, "%d", &currentBalance)
		fmt.Sscanf(deposit.Amount, "%d", &depositAmount)

		// Update balance
		newBalance := currentBalance + depositAmount

		// Create balance transaction
		txnID := s.generateTransactionID()

		balanceTxn := &models.BalanceTransaction{
			TransactionID: txnID,
			UserID:        deposit.UserID,
			Chain:         deposit.Chain,
			Type:          models.TransactionTypeDeposit,
			Amount:        deposit.Amount,
			BalanceBefore: balance.Balance,
			BalanceAfter:  fmt.Sprintf("%d", newBalance),
			RelatedID:     deposit.DepositID,
			CreatedAt:     time.Now().Unix(),
		}

		if err := tx.Create(balanceTxn).Error; err != nil {
			return fmt.Errorf("failed to create balance transaction: %w", err)
		}

		// Update user balance
		if err := tx.Model(&balance).
			Updates(map[string]interface{}{
				"balance":    fmt.Sprintf("%d", newBalance),
				"updated_at": time.Now().Unix(),
			}).Error; err != nil {
			return fmt.Errorf("failed to update balance: %w", err)
		}

		// Update deposit status to completed
		now := time.Now().Unix()
		if err := tx.Model(deposit).
			Updates(map[string]interface{}{
				"status":       models.DepositStatusCompleted,
				"updated_at":   now,
				"completed_at": &now,
			}).Error; err != nil {
			return fmt.Errorf("failed to update deposit status: %w", err)
		}

		s.logger.Info("Deposit processed successfully",
			zap.String("deposit_id", deposit.DepositID),
			zap.String("user_id", deposit.UserID),
			zap.Int64("new_balance", newBalance))

		return nil
	})
}

// GetDeposits gets deposits for a user
func (s *DepositService) GetDeposits(ctx context.Context, userID string, chain models.ChainType, page, pageSize int) ([]models.Deposit, int64, error) {
	var deposits []models.Deposit
	var total int64

	query := s.db.Model(&models.Deposit{}).Where("user_id = ?", userID)
	if chain != "" {
		query = query.Where("chain = ?", chain)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count deposits: %w", err)
	}

	// Get paginated results
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&deposits).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get deposits: %w", err)
	}

	return deposits, total, nil
}

// generateDepositID generates a unique deposit ID
func (s *DepositService) generateDepositID(timestamp int64) string {
	// Format: dep_{date}_{random}
	date := time.Unix(timestamp, 0).Format("20060102")
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)
	random := fmt.Sprintf("%x", randomBytes)[:8]
	return fmt.Sprintf("dep_%s_%s", date, random)
}

// generateTransactionID generates a unique transaction ID
func (s *DepositService) generateTransactionID() string {
	randomBytes := make([]byte, 8)
	rand.Read(randomBytes)
	return fmt.Sprintf("txn_%x", randomBytes)
}
