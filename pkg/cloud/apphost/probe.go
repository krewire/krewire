package apphost

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strconv"
	"time"
)

// probeOnce runs one readiness check against res (ports allocated).
func probeOnce(ctx context.Context, res *Resource, env []string) error {
	p := res.Readiness
	attempt := p.Interval.D() * 5
	if attempt < time.Second {
		attempt = time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, attempt)
	defer cancel()
	switch {
	case p.TCP != "":
		ep, _ := res.endpoint(p.TCP)
		var d net.Dialer
		c, err := d.DialContext(ctx, "tcp", net.JoinHostPort(ep.Host, strconv.Itoa(ep.Port)))
		if err != nil {
			return err
		}
		return c.Close()
	case p.HTTP != nil:
		ep, _ := res.endpoint(p.HTTP.Endpoint)
		u := ep.URL
		if u == "" {
			u = fmt.Sprintf("http://%s:%d", ep.Host, ep.Port)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u+p.HTTP.Path, nil)
		if err != nil {
			return err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		_ = resp.Body.Close()
		if resp.StatusCode >= 400 {
			return fmt.Errorf("GET %s: status %d", req.URL, resp.StatusCode)
		}
		return nil
	case p.Exec != nil:
		cmd := exec.CommandContext(ctx, p.Exec.Command, p.Exec.Args...)
		cmd.Dir = p.Exec.Dir
		cmd.Env = env
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("%w: %s", err, out)
		}
		return nil
	}
	return nil
}

// errExitedBeforeReady is wrapped when the process dies while being probed.
var errExitedBeforeReady = errors.New("process exited before becoming ready")

// waitReady polls the readiness probe until it passes, its timeout elapses,
// the process exits (done closed) or stop is closed. onFail sees every failed
// attempt so the orchestrator can record the latest probe error.
func waitReady(ctx context.Context, res *Resource, env []string, done <-chan struct{}, stop <-chan struct{}, onFail func(error)) error {
	p := res.Readiness
	if p == nil {
		return nil
	}
	deadline := time.NewTimer(p.Timeout.D())
	defer deadline.Stop()
	tick := time.NewTicker(p.Interval.D())
	defer tick.Stop()
	var last error
	for {
		select {
		case <-done:
			return errExitedBeforeReady
		default:
		}
		if last = probeOnce(ctx, res, env); last == nil {
			return nil
		}
		if onFail != nil {
			onFail(last)
		}
		select {
		case <-done:
			return errExitedBeforeReady
		case <-stop:
			return errStopping
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return fmt.Errorf("readiness probe (%s) did not pass within %s: %w", describeProbe(p), p.Timeout, last)
		case <-tick.C:
		}
	}
}

var errStopping = errors.New("orchestrator is stopping")
