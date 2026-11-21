package model

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	// ErrLedgerChainBroken is returned when a verification block's hash link does not match.
	ErrLedgerChainBroken = errors.New("model: verification ledger integrity hash mismatch")

	// ErrEmptyLedger is returned when inspecting an empty verification ledger.
	ErrEmptyLedger = errors.New("model: ledger contains no evidence blocks")
)

// EvidenceBlock represents a single chained block in the verification ledger.
type EvidenceBlock struct {
	Index     int                    `json:"index"`
	Timestamp time.Time              `json:"timestamp"`
	TaskID    string                 `json:"task_id"`
	CommitSHA string                 `json:"commit_sha"`
	PrevHash  string                 `json:"prev_hash"`
	Gates     []VerificationEvidence `json:"gates"`
	AllPassed bool                   `json:"all_passed"`
	BlockHash string                 `json:"block_hash"`
}

// ComputeHash computes the SHA-256 block hash linking PrevHash and all gate evidence.
func (b *EvidenceBlock) ComputeHash() string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%d", b.Index)))
	h.Write([]byte(b.Timestamp.UTC().Format(time.RFC3339Nano)))
	h.Write([]byte(b.TaskID))
	h.Write([]byte(b.CommitSHA))
	h.Write([]byte(b.PrevHash))
	for _, g := range b.Gates {
		h.Write([]byte(g.GateID))
		h.Write([]byte(g.Command))
		h.Write([]byte(fmt.Sprintf("%d", g.ExitCode)))
		h.Write([]byte(g.Status))
		h.Write([]byte(g.EvidenceHash))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// VerificationLedger maintains an append-only, chained audit record of verification evidence.
type VerificationLedger struct {
	mu     sync.RWMutex
	TaskID string          `json:"task_id"`
	Blocks []EvidenceBlock `json:"blocks"`
}

// NewVerificationLedger initializes a new verification ledger for a task.
func NewVerificationLedger(taskID string) *VerificationLedger {
	return &VerificationLedger{
		TaskID: taskID,
		Blocks: make([]EvidenceBlock, 0),
	}
}

// AppendBlock records a new verification run into the ledger with cryptographic hash chaining.
func (l *VerificationLedger) AppendBlock(commitSHA string, gates []VerificationEvidence) (*EvidenceBlock, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	prevHash := "0000000000000000000000000000000000000000000000000000000000000000"
	index := len(l.Blocks)
	if index > 0 {
		prevHash = l.Blocks[index-1].BlockHash
	}

	allPassed := len(gates) > 0
	for _, g := range gates {
		if g.Required && g.Status != GatePassed {
			allPassed = false
			break
		}
	}

	block := EvidenceBlock{
		Index:     index,
		Timestamp: time.Now().UTC(),
		TaskID:    l.TaskID,
		CommitSHA: commitSHA,
		PrevHash:  prevHash,
		Gates:     gates,
		AllPassed: allPassed,
	}
	block.BlockHash = block.ComputeHash()

	l.Blocks = append(l.Blocks, block)
	return &block, nil
}

// VerifyIntegrity cryptographically validates the entire evidence chain.
func (l *VerificationLedger) VerifyIntegrity() (bool, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if len(l.Blocks) == 0 {
		return false, ErrEmptyLedger
	}

	for i, b := range l.Blocks {
		expectedHash := b.ComputeHash()
		if b.BlockHash != expectedHash {
			return false, fmt.Errorf("%w: block %d hash mismatch", ErrLedgerChainBroken, i)
		}
		if i > 0 {
			if b.PrevHash != l.Blocks[i-1].BlockHash {
				return false, fmt.Errorf("%w: block %d prev_hash mismatch", ErrLedgerChainBroken, i)
			}
		} else {
			if b.PrevHash != "0000000000000000000000000000000000000000000000000000000000000000" {
				return false, fmt.Errorf("%w: genesis block prev_hash invalid", ErrLedgerChainBroken)
			}
		}
	}
	return true, nil
}

// LatestBlock returns the most recent evidence block.
func (l *VerificationLedger) LatestBlock() (*EvidenceBlock, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if len(l.Blocks) == 0 {
		return nil, ErrEmptyLedger
	}
	b := l.Blocks[len(l.Blocks)-1]
	return &b, nil
}
