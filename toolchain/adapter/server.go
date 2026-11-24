package adapter

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"sync"
	"time"

	"github.com/krewire/krewire/toolchain/contract"
)

// MaxMessageBytes bounds one protocol line (contexts can be large).
const MaxMessageBytes = 32 << 20

// message is a JSON-RPC 2.0 request, notification, or response.
type message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *contract.Error `json:"error,omitempty"`
}

// writer serializes messages, one per line.
type writer struct {
	mu sync.Mutex
	w  io.Writer
}

func (w *writer) send(m message) error {
	m.JSONRPC = "2.0"
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	_, err = w.w.Write(append(data, '\n'))
	return err
}

// Serve runs eng behind the stdio protocol until the peer sends shutdown,
// closes r, or ctx ends. Requests run concurrently; each gets a context
// cancelled by $/cancelRequest, by its meta.timeout_ms, or by shutdown.
// Engines call it from main with os.Stdin and os.Stdout and log to stderr.
func Serve(ctx context.Context, eng Engine, r io.Reader, w io.Writer) error {
	ctx, cancelAll := context.WithCancel(ctx)
	defer cancelAll()
	out := &writer{w: w}
	var (
		mu          sync.Mutex
		inflight    = map[int64]context.CancelFunc{}
		wg          sync.WaitGroup
		initialized bool
	)
	defer wg.Wait()

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64<<10), MaxMessageBytes)
	for sc.Scan() {
		var m message
		if err := json.Unmarshal(sc.Bytes(), &m); err != nil || m.JSONRPC != "2.0" {
			_ = out.send(message{ID: m.ID, Error: contract.Errorf(contract.CodeParseError, "malformed message")})
			continue
		}
		if m.ID == nil { // notification
			if m.Method == NotifyCancel {
				var p struct {
					ID int64 `json:"id"`
				}
				if json.Unmarshal(m.Params, &p) == nil {
					mu.Lock()
					if cancel := inflight[p.ID]; cancel != nil {
						cancel()
					}
					mu.Unlock()
				}
			}
			continue
		}
		id := *m.ID
		switch {
		case m.Method == MethodShutdown:
			cancelAll()
			wg.Wait()
			return out.send(message{ID: &id, Result: json.RawMessage("null")})
		case m.Method != MethodInitialize && !initialized:
			_ = out.send(message{ID: &id, Error: contract.Errorf(contract.CodeInvalidRequest, "initialize first")})
			continue
		case m.Method == MethodInitialize:
			initialized = true
		}
		var meta struct {
			Meta Meta `json:"meta"`
		}
		_ = json.Unmarshal(m.Params, &meta)
		reqCtx, cancel := requestContext(ctx, meta.Meta)
		mu.Lock()
		inflight[id] = cancel
		mu.Unlock()
		wg.Add(1)
		go func(m message) {
			defer wg.Done()
			result, err := dispatch(reqCtx, eng, m.Method, m.Params)
			if err == nil && reqCtx.Err() != nil {
				err = reqCtx.Err()
			}
			mu.Lock()
			delete(inflight, id)
			mu.Unlock()
			cancel()
			resp := message{ID: &id}
			if err != nil {
				resp.Error = contract.AsError(err)
			} else if resp.Result, err = json.Marshal(result); err != nil {
				resp.Error = contract.AsError(err)
			}
			_ = out.send(resp)
		}(m)
	}
	return sc.Err()
}

// requestContext bounds a request by its meta timeout, if any.
func requestContext(ctx context.Context, m Meta) (context.Context, context.CancelFunc) {
	if m.TimeoutMs > 0 {
		return context.WithTimeout(ctx, time.Duration(m.TimeoutMs)*time.Millisecond)
	}
	return context.WithCancel(ctx)
}

// decode unmarshals params strictly.
func decode[T any](raw json.RawMessage) (T, error) {
	var v T
	if err := json.Unmarshal(raw, &v); err != nil {
		return v, contract.Errorf(contract.CodeInvalidParams, "%v", err)
	}
	return v, nil
}

func dispatch(ctx context.Context, eng Engine, method string, raw json.RawMessage) (any, error) {
	switch method {
	case MethodInitialize:
		p, err := decode[InitializeParams](raw)
		if err != nil {
			return nil, err
		}
		if err := contract.CompatibleContract(p.ContractVersion); err != nil {
			return nil, err
		}
		return eng.Initialize(ctx, p)
	case MethodPlan:
		p, err := decode[PlanParams](raw)
		if err != nil {
			return nil, err
		}
		return eng.Plan(ctx, p)
	case MethodImplement:
		p, err := decode[ImplementParams](raw)
		if err != nil {
			return nil, err
		}
		return eng.Implement(ctx, p)
	case MethodReview:
		p, err := decode[ReviewParams](raw)
		if err != nil {
			return nil, err
		}
		return eng.Review(ctx, p)
	}
	return nil, contract.Errorf(contract.CodeMethodNotFound, "method %q is not part of contract %s", method, contract.ContractVersion)
}
