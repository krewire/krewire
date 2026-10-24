package mail

import (
	"context"
	"sync"
)

// MemoryDriver stores sent messages in memory for assertions during testing.
type MemoryDriver struct {
	mu       sync.RWMutex
	messages []*Message
}

// NewMemoryDriver creates an in-memory mail delivery driver.
func NewMemoryDriver() *MemoryDriver {
	return &MemoryDriver{}
}

func (d *MemoryDriver) Name() string {
	return "memory"
}

func (d *MemoryDriver) Close() error {
	d.Clear()
	return nil
}

func (d *MemoryDriver) Send(_ context.Context, msg *Message) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.messages = append(d.messages, msg)
	return nil
}

// SentMessages returns a snapshot of all sent messages.
func (d *MemoryDriver) SentMessages() []*Message {
	d.mu.RLock()
	defer d.mu.RUnlock()
	res := make([]*Message, len(d.messages))
	copy(res, d.messages)
	return res
}

// LastMessage returns the most recently dispatched message or nil.
func (d *MemoryDriver) LastMessage() *Message {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if len(d.messages) == 0 {
		return nil
	}
	return d.messages[len(d.messages)-1]
}

// Count returns the number of captured messages.
func (d *MemoryDriver) Count() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.messages)
}

// Clear resets the sent message history.
func (d *MemoryDriver) Clear() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.messages = nil
}

var _ Driver = (*MemoryDriver)(nil)
