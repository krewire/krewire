package adapter_test

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/krewire/krewire/pkg/toolchain/adapter"
	"github.com/krewire/krewire/pkg/toolchain/conformance"
	"github.com/krewire/krewire/pkg/toolchain/contract"
	"github.com/krewire/krewire/pkg/toolchain/fakeengine"
)

// TestMain doubles as a reference engine executable: when spawned with
// KREW_TEST_ENGINE=1 the test binary serves the fake engine on stdio.
func TestMain(m *testing.M) {
	if os.Getenv("KREW_TEST_ENGINE") == "1" {
		if err := adapter.Serve(context.Background(), &fakeengine.Engine{}, os.Stdin, os.Stdout); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// TestSpawnedEngineProcess runs the conformance lifecycle against a real
// child process over stdin/stdout.
func TestSpawnedEngineProcess(t *testing.T) {
	conformance.Run(t, func(t *testing.T) adapter.Engine {
		c, err := adapter.Spawn(context.Background(), []string{os.Args[0]}, append(os.Environ(), "KREW_TEST_ENGINE=1"), os.Stderr)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := c.Close(); err != nil {
				t.Errorf("close: %v", err)
			}
		})
		return c
	})
}

// rawSession drives Serve with hand-written JSON lines.
func rawSession(t *testing.T, eng adapter.Engine, lines ...string) []map[string]any {
	t.Helper()
	var out strings.Builder
	if err := adapter.Serve(context.Background(), eng, strings.NewReader(strings.Join(lines, "\n")+"\n"), &out); err != nil {
		t.Fatal(err)
	}
	var msgs []map[string]any
	sc := bufio.NewScanner(strings.NewReader(out.String()))
	for sc.Scan() {
		var m map[string]any
		if err := json.Unmarshal(sc.Bytes(), &m); err != nil {
			t.Fatalf("engine wrote a non-JSON line: %q", sc.Text())
		}
		msgs = append(msgs, m)
	}
	return msgs
}

func errCode(m map[string]any) int {
	e, _ := m["error"].(map[string]any)
	if e == nil {
		return 0
	}
	return int(e["code"].(float64))
}

func TestProtocolErrors(t *testing.T) {
	eng := &fakeengine.Engine{}
	msgs := rawSession(t, eng,
		`{"jsonrpc":"2.0","id":1,"method":"plan","params":{}}`,
		`not json`,
		`{"jsonrpc":"2.0","id":2,"method":"initialize","params":{"contract_version":"9.0.0"}}`,
		`{"jsonrpc":"2.0","id":3,"method":"bogus","params":{}}`,
		`{"jsonrpc":"2.0","id":4,"method":"plan","params":{"task":"wrong-type"}}`,
		`{"jsonrpc":"2.0","method":"$/cancelRequest","params":{"id":99}}`,
	)
	want := map[float64]contract.Code{1: contract.CodeInvalidRequest, 2: contract.CodeVersionUnsupported,
		3: contract.CodeMethodNotFound, 4: contract.CodeInvalidParams}
	seenParse := false
	for _, m := range msgs {
		id, ok := m["id"].(float64)
		if !ok {
			seenParse = seenParse || errCode(m) == int(contract.CodeParseError)
			continue
		}
		if errCode(m) != int(want[id]) {
			t.Errorf("request %v: got %v, want %s", id, m["error"], want[id])
		}
	}
	if !seenParse || len(msgs) != 5 {
		t.Errorf("want 4 responses and a parse error, got %v", msgs)
	}
}

func TestTimeoutAndCancellationAcrossTheBoundary(t *testing.T) {
	pctx, task := conformance.Fixture()
	slow := conformance.Pipe(t, &fakeengine.Engine{Delay: 5 * time.Second})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := slow.Plan(ctx, adapter.PlanParams{Context: pctx, Task: task}); !errors.Is(err, contract.ErrTimeout) {
		t.Errorf("want timeout, got %v", err)
	}
	ctx2, cancel2 := context.WithCancel(context.Background())
	go func() { time.Sleep(20 * time.Millisecond); cancel2() }()
	if _, err := slow.Review(ctx2, adapter.ReviewParams{}); !errors.Is(err, contract.ErrCancelled) {
		t.Errorf("want cancelled, got %v", err)
	}
	if time.Since(start) > 2*time.Second {
		t.Error("timeouts and cancellation must not wait for the engine")
	}

	// Server side: meta.timeout_ms alone bounds the engine.
	msgs := rawSession(t, &fakeengine.Engine{Delay: 5 * time.Second},
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"contract_version":"1.0.0"}}`,
		`{"jsonrpc":"2.0","id":2,"method":"plan","params":{"meta":{"timeout_ms":20},"task":{}}}`)
	if len(msgs) != 2 || errCode(msgs[1]) != int(contract.CodeTimeout) {
		t.Errorf("server must enforce meta.timeout_ms: %v", msgs)
	}
}

func TestEngineErrorsKeepTheirCodes(t *testing.T) {
	c := conformance.Pipe(t, adapter.Unavailable{Reason: "no provider credentials"})
	_, err := c.Plan(context.Background(), adapter.PlanParams{})
	if !errors.Is(err, contract.ErrEngineUnavailable) || !strings.Contains(err.Error(), "credentials") {
		t.Errorf("want engine_unavailable across the wire, got %v", err)
	}
	if _, err := c.Implement(context.Background(), adapter.ImplementParams{}); !errors.Is(err, contract.ErrEngineUnavailable) {
		t.Error(err)
	}
	if _, err := c.Review(context.Background(), adapter.ReviewParams{}); !errors.Is(err, contract.ErrEngineUnavailable) {
		t.Error(err)
	}
}

func TestDeadEngine(t *testing.T) {
	r, w := io.Pipe()
	_, hostW := io.Pipe()
	c := adapter.NewClient(r, hostW)
	_ = w.Close() // engine exits
	time.Sleep(10 * time.Millisecond)
	if _, err := c.Plan(context.Background(), adapter.PlanParams{}); !errors.Is(err, contract.ErrEngineUnavailable) {
		t.Errorf("want engine_unavailable, got %v", err)
	}
	if _, err := adapter.Spawn(context.Background(), nil, nil, nil); !errors.Is(err, contract.ErrEngineUnavailable) {
		t.Errorf("no engine configured: %v", err)
	}
	if _, err := adapter.Spawn(context.Background(), []string{"/nonexistent/engine"}, nil, nil); !errors.Is(err, contract.ErrEngineUnavailable) {
		t.Errorf("missing binary: %v", err)
	}
	argv := adapter.EngineCommand(func(string) string { return " krew-engine  --stdio " })
	if len(argv) != 2 || argv[0] != "krew-engine" {
		t.Errorf("EngineCommand: %v", argv)
	}
	if m := adapter.MetaFor(context.Background()); m.TimeoutMs != 0 {
		t.Error("no deadline, no timeout")
	}
}
