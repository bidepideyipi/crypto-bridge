package models

import (
	"time"
)

// ChainType represents supported blockchain types
type ChainType string

const (
	ChainBTC ChainType = "BTC"
	ChainETH ChainType = "ETH"
	ChainTRX ChainType = "TRX"
	ChainSOL ChainType = "SOL"
)

// AddressType represents the type of address
type AddressType string

const (
	AddressTypeDeposit    AddressType = "deposit"
	AddressTypeHotWallet  AddressType = "hot_wallet"
	AddressTypeColdWallet AddressType = "cold_wallet"
)

// AddressStatus represents the status of an address
type AddressStatus string

const (
	AddressStatusActive   AddressStatus = "active"
	AddressStatusInactive AddressStatus = "inactive"
)

// DepositStatus represents the status of a deposit
type DepositStatus string

const (
	DepositStatusPending   DepositStatus = "pending"
	DepositStatusConfirmed DepositStatus = "confirmed"
	DepositStatusCompleted DepositStatus = "completed"
)

// WithdrawStatus represents the status of a withdrawal
type WithdrawStatus string

const (
	WithdrawStatusPending   WithdrawStatus = "pending"
	WithdrawStatusApproved  WithdrawStatus = "approved"
	WithdrawStatusRejected  WithdrawStatus = "rejected"
	WithdrawStatusCompleted WithdrawStatus = "completed"
	WithdrawStatusFailed    WithdrawStatus = "failed"
)

// TransactionType represents the type of balance transaction
type TransactionType string

const (
	TransactionTypeDeposit   TransactionType = "deposit"
	TransactionTypeWithdraw  TransactionType = "withdraw"
	TransactionTypeFreeze    TransactionType = "freeze"
	TransactionTypeUnfreeze  TransactionType = "unfreeze"
	TransactionTypeTransfer  TransactionType = "transfer"
)

// ChainTransactionStatus represents the status of a chain transaction
type ChainTransactionStatus string

const (
	ChainTxStatusPending   ChainTransactionStatus = "pending"
	ChainTxStatusConfirmed ChainTransactionStatus = "confirmed"
	ChainTxStatusFailed    ChainTransactionStatus = "failed"
)

// UserAddress represents a user's address mapping
type UserAddress struct {
	ID           int64        `json:"id" db:"id"`
	UserID       string       `json:"user_id" db:"user_id"`
	Chain        ChainType    `json:"chain" db:"chain"`
	Address      string       `json:"address" db:"address"`
	AddressType  AddressType  `json:"address_type" db:"address_type"`
	Status       AddressStatus `json:"status" db:"status"`
	CreatedAt    int64        `json:"created_at" db:"created_at"`
	UpdatedAt    int64        `json:"updated_at" db:"updated_at"`
}

// Deposit represents a deposit record
type Deposit struct {
	ID                   int64         `json:"id" db:"id"`
	DepositID            string        `json:"deposit_id" db:"deposit_id"`
	UserID               string        `json:"user_id" db:"user_id"`
	Chain                ChainType     `json:"chain" db:"chain"`
	TxHash               string        `json:"tx_hash" db:"tx_hash"`
	FromAddress          string        `json:"from_address" db:"from_address"`
	ToAddress            string        `json:"to_address" db:"to_address"`
	Amount               string        `json:"amount" db:"amount"`
	Status               DepositStatus `json:"status" db:"status"`
	Confirmations       int           `json:"confirmations" db:"confirmations"`
	RequiredConfirmations int         `json:"required_confirmations" db:"required_confirmations"`
	CreatedAt            int64         `json:"created_at" db:"created_at"`
	UpdatedAt            int64         `json:"updated_at" db:"updated_at"`
	CompletedAt          *int64        `json:"completed_at,omitempty" db:"completed_at"`
}

// Withdrawal represents a withdrawal record
type Withdrawal struct {
	ID                   int64          `json:"id" db:"id"`
	WithdrawID           string         `json:"withdraw_id" db:"withdraw_id"`
	UserID               string         `json:"user_id" db:"user_id"`
	Chain                ChainType      `json:"chain" db:"chain"`
	ToAddress            string         `json:"to_address" db:"to_address"`
	Amount               string         `json:"amount" db:"amount"`
	Fee                  string         `json:"fee" db:"fee"`
	Status               WithdrawStatus `json:"status" db:"status"`
	TxHash               string         `json:"tx_hash,omitempty" db:"tx_hash"`
	Confirmations       int            `json:"confirmations" db:"confirmations"`
	RequiredConfirmations int          `json:"required_confirmations" db:"required_confirmations"`
	Memo                 string        `json:"memo,omitempty" db:"memo"`
	RejectReason         string        `json:"reject_reason,omitempty" db:"reject_reason"`
	CreatedAt            int64         `json:"created_at" db:"created_at"`
	UpdatedAt            int64         `json:"updated_at" db:"updated_at"`
	ApprovedAt           *int64        `json:"approved_at,omitempty" db:"approved_at"`
	CompletedAt          *int64        `json:"completed_at,omitempty" db:"completed_at"`
}

// UserBalance represents a user's balance
type UserBalance struct {
	ID             int64  `json:"id" db:"id"`
	UserID         string `json:"user_id" db:"user_id"`
	Chain          ChainType `json:"chain" db:"chain"`
	Balance        string `json:"balance" db:"balance"`
	LockedBalance  string `json:"locked_balance" db:"locked_balance"`
	CreatedAt      int64  `json:"created_at" db:"created_at"`
	UpdatedAt      int64  `json:"updated_at" db:"updated_at"`
}

// BalanceTransaction represents a balance change transaction
type BalanceTransaction struct {
	ID            int64           `json:"id" db:"id"`
	TransactionID string          `json:"transaction_id" db:"transaction_id"`
	UserID        string          `json:"user_id" db:"user_id"`
	Chain         ChainType      `json:"chain" db:"chain"`
	Type          TransactionType `json:"type" db:"type"`
	Amount        string          `json:"amount" db:"amount"`
	BalanceBefore string          `json:"balance_before" db:"balance_before"`
	BalanceAfter  string          `json:"balance_after" db:"balance_after"`
	RelatedID     string          `json:"related_id,omitempty" db:"related_id"`
	CreatedAt     int64           `json:"created_at" db:"created_at"`
}

// ChainTransaction represents a chain transaction tracking record
type ChainTransaction struct {
	ID                   int64                    `json:"id" db:"id"`
	Chain                ChainType                `json:"chain" db:"chain"`
	TxHash               string                   `json:"tx_hash" db:"tx_hash"`
	Type                 TransactionType          `json:"type" db:"type"`
	Status               ChainTransactionStatus   `json:"status" db:"status"`
	Confirmations       int                      `json:"confirmations" db:"confirmations"`
	RequiredConfirmations int                    `json:"required_confirmations" db:"required_confirmations"`
	RawTx                string                   `json:"raw_tx,omitempty" db:"raw_tx"`
	CreatedAt            int64                    `json:"created_at" db:"created_at"`
	UpdatedAt            int64                    `json:"updated_at" db:"updated_at"`
}

// Address represents a blockchain address
type Address struct {
	Address   string   `json:"address"`
	Chain     ChainType `json:"chain"`
	UserID    string   `json:"user_id,omitempty"`
}

// Transaction represents a blockchain transaction
type Transaction struct {
	TxHash       string    `json:"tx_hash"`
	Chain        ChainType `json:"chain"`
	FromAddress  string    `json:"from_address"`
	ToAddress    string    `json:"to_address"`
	Amount       string    `json:"amount"`
	BlockHash    string    `json:"block_hash,omitempty"`
	BlockHeight int64     `json:"block_height,omitempty"`
	Timestamp   int64     `json:"timestamp,omitempty"`
	Confirmations int     `json:"confirmations,omitempty"`
}

// Balance represents an address balance
type Balance struct {
	Address    string    `json:"address"`
	Chain      ChainType `json:"chain"`
	Amount     string    `json:"amount"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// DepositEvent represents a deposit event for MQ
type DepositEvent struct {
	EventID       string    `json:"event_id"`
	EventType     string    `json:"event_type"`
	UserID        string    `json:"user_id"`
	Chain         string    `json:"chain"`
	TxHash        string    `json:"tx_hash"`
	Amount        string    `json:"amount"`
	FromAddress   string    `json:"from_address"`
	ToAddress     string    `json:"to_address"`
	Confirmations int       `json:"confirmations"`
	Timestamp     int64     `json:"timestamp"`
}

// WithdrawalEvent represents a withdrawal event for MQ
type WithdrawalEvent struct {
	EventID    string    `json:"event_id"`
	EventType  string    `json:"event_type"`
	WithdrawID string    `json:"withdraw_id"`
	UserID     string    `json:"user_id"`
	Chain      string    `json:"chain"`
	TxHash     string    `json:"tx_hash"`
	Amount     string    `json:"amount"`
	ToAddress  string    `json:"to_address"`
	Status     string    `json:"status"`
	Timestamp  int64     `json:"timestamp"`
}
