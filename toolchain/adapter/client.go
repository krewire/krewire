package adapter

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/krewire/krewire/toolchain/contract"
)

// Client is the host end of the stdio protocol. It implements Engine, so the
// toolchain drives a remote engine exactly like an in-process one.
type Client struct {
	out     *writer
	closer  io.Closer
	mu      sync.Mutex
	nextID  int64
	pending map[int64]chan message
	done    chan struct{}
	err     error // set when the read loop ends
	wait    func() error
}

// NewClient speaks the protocol over r (engine stdout) and w (engine stdin).
func NewClient(r io.Reader, w io.WriteCloser) *Client {
	c := &Client{out: &writer{w: w}, closer: w, pending: map[int64]chan message{}, done: make(chan struct{})}
	go c.read(r)
	return c
}

func (c *Client) read(r io.Reader) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64<<10), MaxMessageBytes)
	for sc.Scan() {
		var m message
		if json.Unmarshal(sc.Bytes(), &m) != nil || m.ID == nil {
			continue // engines must not write anything else to stdout
		}
		c.mu.Lock()
		ch := c.pending[*m.ID]
		delete(c.pending, *m.ID)
		c.mu.Unlock()
		if ch != nil {
			ch <- m
		}
	}
	c.mu.Lock()
	c.err = contract.Errorf(contract.CodeEngineUnavailable, "engine closed the connection")
	if err := sc.Err(); err != nil {
		c.err = contract.Errorf(contract.CodeEngineUnavailable, "engine stream failed: %v", err)
	}
	c.mu.Unlock()
	close(c.done)
}

// call performs one request; ctx cancellation sends $/cancelRequest and
// returns cancelled or timeout without waiting for the engine.
func (c *Client) call(ctx context.Context, method string, params, result any) error {
	if err := ctx.Err(); err != nil {
		return contract.AsError(err) // never start work for a dead request
	}
	raw, err := json.Marshal(params)
	if err != nil {
		return contract.Errorf(contract.CodeInvalidParams, "%v", err)
	}
	ch := make(chan message, 1)
	c.mu.Lock()
	if c.err != nil {
		c.mu.Unlock()
		return c.err
	}
	c.nextID++
	id := c.nextID
	c.pending[id] = ch
	c.mu.Unlock()
	if err := c.out.send(message{ID: &id, Method: method, Params: raw}); err != nil {
		return contract.Errorf(contract.CodeEngineUnavailable, "send %s: %v", method, err)
	}
	select {
	case m := <-ch:
		if m.Error != nil {
			return m.Error
		}
		if result != nil {
			if err := json.Unmarshal(m.Result, result); err != nil {
				return contract.Errorf(contract.CodeInvalidDocument, "%s result: %v", method, err)
			}
		}
		return nil
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		cancel, _ := json.Marshal(map[string]int64{"id": id})
		_ = c.out.send(message{Method: NotifyCancel, Params: cancel})
		return contract.AsError(ctx.Err())
	case <-c.done:
		c.mu.Lock()
		defer c.mu.Unlock()
		return c.err
	}
}

// Initialize implements Engine and checks the engine's contract version.
func (c *Client) Initialize(ctx context.Context, p InitializeParams) (InitializeResult, error) {
	if p.ContractVersion == "" {
		p.ContractVersion = contract.ContractVersion
	}
	var r InitializeResult
	if err := c.call(ctx, MethodInitialize, p, &r); err != nil {
		return r, err
	}
	return r, contract.CompatibleContract(r.Engine.ContractVersion)
}

// Plan implements Engine.
func (c *Client) Plan(ctx context.Context, p PlanParams) (PlanResult, error) {
	p.Meta = MetaFor(ctx)
	var r PlanResult
	return r, c.call(ctx, MethodPlan, p, &r)
}

// Implement implements Engine.
func (c *Client) Implement(ctx context.Context, p ImplementParams) (ImplementResult, error) {
	p.Meta = MetaFor(ctx)
	var r ImplementResult
	return r, c.call(ctx, MethodImplement, p, &r)
}

// Review implements Engine.
func (c *Client) Review(ctx context.Context, p ReviewParams) (ReviewResult, error) {
	p.Meta = MetaFor(ctx)
	var r ReviewResult
	return r, c.call(ctx, MethodReview, p, &r)
}

// Close sends shutdown (bounded to two seconds), closes the engine's stdin,
// and, for spawned engines, waits for the process to exit.
func (c *Client) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = c.call(ctx, MethodShutdown, struct{}{}, nil)
	err := c.closer.Close()
	select { // drain stdout before Wait closes the pipe
	case <-c.done:
	case <-ctx.Done():
	}
	if c.wait != nil {
		if werr := c.wait(); werr != nil && !errors.Is(werr, os.ErrProcessDone) {
			err = werr
		}
	}
	return err
}

// Spawn starts an engine process speaking the protocol on stdio. stderr
// receives the engine's logs. The engine inherits env only — contract
// documents never carry credentials.
func Spawn(ctx context.Context, argv []string, env []string, stderr io.Writer) (*Client, error) {
	if len(argv) == 0 {
		return nil, contract.Errorf(contract.CodeEngineUnavailable, "no engine configured (set %s)", EnvEngine)
	}
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Env, cmd.Stderr = env, stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, contract.Errorf(contract.CodeEngineUnavailable, "start engine %q: %v", argv[0], err)
	}
	c := NewClient(stdout, stdin)
	c.wait = cmd.Wait
	return c, nil
}

// EngineCommand returns the configured engine argv from KREW_ENGINE.
func EngineCommand(getenv func(string) string) []string {
	return strings.Fields(getenv(EnvEngine))
}
