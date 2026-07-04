package wallet

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"
)

var (
	ErrInsufficientBalance = errors.New("insufficient wallet balance")
)

type WalletStub struct {
	mu       sync.RWMutex
	balances map[uuid.UUID]int64
}

func NewWalletStub() *WalletStub {
	return &WalletStub{
		balances: make(map[uuid.UUID]int64),
	}
}

func (s *WalletStub) GetBalance(ctx context.Context, userID uuid.UUID) (int64, error) {
	s.mu.RLock()
	bal, exists := s.balances[userID]
	s.mu.RUnlock()

	if !exists {
		// Seed default balance of 500,000 IDR (or 500000 cents/units) for easy developer testing
		s.mu.Lock()
		s.balances[userID] = 500000
		bal = 500000
		s.mu.Unlock()
	}

	return bal, nil
}

func (s *WalletStub) Deduct(ctx context.Context, userID uuid.UUID, amount int64, description string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	bal, exists := s.balances[userID]
	if !exists {
		bal = 500000 // seed default
	}

	if bal < amount {
		return ErrInsufficientBalance
	}

	s.balances[userID] = bal - amount
	return nil
}

func (s *WalletStub) Refund(ctx context.Context, userID uuid.UUID, amount int64, description string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	bal, exists := s.balances[userID]
	if !exists {
		bal = 500000 // seed default
	}

	s.balances[userID] = bal + amount
	return nil
}

// SetBalance allows tests or debug endpoints to explicitly set a user's wallet balance
func (s *WalletStub) SetBalance(userID uuid.UUID, balance int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.balances[userID] = balance
}
