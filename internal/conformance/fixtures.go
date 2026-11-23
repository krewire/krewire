package conformance

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func siteFixture(r *run) {
	if !r.prepare("init/static") {
		return
	}
	r.gate("compile", gateBuild, "static site generation into .krewire/build",
		fixtureBin+" build .krewire/build", func() (string, error) { return r.fixture("build", ".krewire/build") })
	r.file("output", ".krewire/build/index.html", "Conformance Site")
	addr, _ := freeAddr()
	r.serve([]string{"KREWIRE_ADDR=" + addr}, func(_ string, _ *proc) {
		r.probe("probe.page", "preview server renders the page", "http://"+addr+"/", 200, "Conformance Site")
		r.probe("probe.notfound", "unknown route is 404", "http://"+addr+"/no-such-page", 404, "")
	})
	r.diagnose("invalid-page", "a malformed .kiw page must be reported, not silently dropped",
		writeRel("pages/broken.kiw", "---\ntitle: [unclosed\n---\n{#if open}\n<p>never closed\n"), "broken.kiw")
}

func bookFixture(r *run) {
	if !r.prepare("init/book") {
		return
	}
	r.gate("compile", gateBuild, "compile content/ into a static book",
		fixtureBin+" build .krewire/book", func() (string, error) { return r.fixture("build", ".krewire/book") })
	r.file("output", ".krewire/book/index.html", "Getting Started")
	addr, _ := freeAddr()
	r.serve([]string{"KREWIRE_ADDR=" + addr}, func(_ string, _ *proc) {
		r.probe("probe.index", "book server renders the table of contents", "http://"+addr+"/", 200, "Getting Started")
		r.probe("probe.notfound", "unknown chapter is 404", "http://"+addr+"/no-such-chapter/", 404, "")
	})
	r.diagnose("missing-content", "a missing content directory is diagnosed",
		func(dir string) error { return os.RemoveAll(filepath.Join(dir, "content")) }, "content")
}

func appFixture(r *run) {
	if !r.prepare("init/app") {
		return
	}
	addr, _ := freeAddr()
	r.serve([]string{"KREWIRE_ADDR=" + addr}, func(_ string, p *proc) {
		r.probe("probe.ssr", "server renders HTML", "http://"+addr+"/", 200, "<h1>Conformance App</h1>")
		r.probe("probe.api", "JSON API", "http://"+addr+"/api/greet?name=krewire", 200, `"hello krewire"`)
		r.probe("probe.health", "liveness probe", "http://"+addr+"/healthz", 200, "")
		r.assert("probe.worker", gateTest, "in-process background worker runs under the same lifecycle",
			"await JOB line", func() (string, error) { return p.waitLine("JOB done", 10*time.Second) })
	})
	r.exclude("wasm-islands", "client WebAssembly island compilation is not exercised by this fixture; it is covered by platform_test.go (TestPlatform_WasmCompile) and pkg/runtime/conformance", "#141")
	r.assert("diagnose.port-in-use", gateTest, "an unavailable listen address fails startup with a diagnostic",
		fixtureBin+" serve (port taken)", func() (string, error) {
			l, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				return "", err
			}
			defer l.Close()
			out, err := r.exec(30*time.Second, []string{"KREWIRE_ADDR=" + l.Addr().String()}, filepath.Join(r.dir, fixtureBin), "serve")
			if err == nil || !strings.Contains(out, "address already in use") && !strings.Contains(out, "bind") {
				return out, fmt.Errorf("want non-zero exit with a bind diagnostic, got err=%v", err)
			}
			return out, nil
		})
}

func cliFixture(r *run) {
	if !r.prepare("init/cli") {
		return
	}
	r.assert("reproducible", gateBuild, "second clean build is bit-for-bit identical",
		"go build -trimpath -o bin/fixture2 . && sha256 compare", func() (string, error) {
			if out, err := r.build("bin/fixture2"); err != nil {
				return out, err
			}
			a, b := r.res.Artifacts[0], r.res.Artifacts[len(r.res.Artifacts)-1]
			if a.SHA256 != b.SHA256 {
				return a.SHA256 + " != " + b.SHA256, fmt.Errorf("non-reproducible build")
			}
			return "sha256 " + a.SHA256, nil
		})
	r.assert("run", gateTest, "representative command", fixtureBin+" greet -name krewire", func() (string, error) {
		out, err := r.fixture("greet", "-name", "krewire")
		if err == nil && !strings.Contains(out, "Hello, krewire") {
			err = fmt.Errorf("unexpected output")
		}
		return out, err
	})
	r.assert("exit", gateTest, "command completes and exits 0", fixtureBin+" version", func() (string, error) { return r.fixture("version") })
	r.diagnose("unknown-command", "unknown subcommand exits non-zero with usage",
		func(string) error { return nil }, "unknown command", "frobnicate")
	r.diagnose("bad-flag", "invalid flag exits non-zero with a flag diagnostic",
		func(string) error { return nil }, "flag provided but not defined", "greet", "-bogus")
}

func workerFixture(r *run) {
	if !r.prepare("new/bootstrap") {
		return
	}
	r.serve(nil, func(_ string, p *proc) {
		r.assert("probe.jobs", gateTest, "all enqueued jobs are processed", "await PROCESSED 3",
			func() (string, error) { return p.waitLine("PROCESSED 3", 15*time.Second) })
		r.assert("probe.retry", gateTest, "a transiently failing job is retried with backoff and succeeds", "await RETRIED",
			func() (string, error) { return p.waitLine("RETRIED", 15*time.Second) })
	})
	r.exclude("external-backends", "Redis/PostgreSQL queue backends need external services and are not exercised; the in-memory queue is", "#141")
	r.diagnose("invalid-cron", "an invalid cron schedule in krewire.yaml is diagnosed",
		replaceIn("krewire.yaml", `"*/5 * * * *"`, `"61 * * * *"`), "cron")
}

func serviceFixture(r *run) {
	if !r.prepare("new/bootstrap") {
		return
	}
	addr, _ := freeAddr()
	r.serve([]string{"KREWIRE_ADDR=" + addr, "SERVICE_STORE=memory"}, func(_ string, p *proc) {
		r.probe("probe.ready", "readiness reflects started providers", "http://"+addr+"/readyz", 200, "")
		r.probe("probe.api", "REST resource", "http://"+addr+"/v1/items", 200, `"items"`)
		r.probe("probe.notfound", "unknown resource is 404", "http://"+addr+"/v1/nope", 404, "")
		r.assert("probe.provider", gateTest, "provider Start ran before serving", "await PROVIDER started",
			func() (string, error) { return p.waitLine("PROVIDER started", 5*time.Second) })
	})
	r.assert("diagnose.missing-dependency", gateTest, "a missing provider dependency fails bootstrap with a diagnostic",
		fixtureBin+" serve (SERVICE_STORE unset)", func() (string, error) {
			out, err := r.exec(30*time.Second, []string{"KREWIRE_ADDR=" + addr, "SERVICE_STORE="}, filepath.Join(r.dir, fixtureBin), "serve")
			if err == nil || !strings.Contains(out, "SERVICE_STORE") {
				return out, fmt.Errorf("want non-zero exit naming SERVICE_STORE, got err=%v", err)
			}
			return out, nil
		})
}

func gatewayFixture(r *run) {
	if !r.prepare("new/bootstrap") {
		return
	}
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		fmt.Fprintf(w, "upstream %s", q.URL.Path)
	}))
	defer up.Close()
	addr, _ := freeAddr()
	r.serve([]string{"KREWIRE_ADDR=" + addr, "UPSTREAM_ADDR=" + up.Listener.Addr().String()}, func(_ string, _ *proc) {
		r.probe("probe.proxy", "prefix route proxies with strip-prefix", "http://"+addr+"/api/hello", 200, "upstream /hello")
		r.probe("probe.health", "gateway liveness", "http://"+addr+"/healthz", 200, "")
		r.probe("probe.noroute", "unmatched path is 404", "http://"+addr+"/nope", 404, "")
		r.probe("diagnose.unresolved-upstream", "route to an unregistered service is 502", "http://"+addr+"/ghost", 502, "")
		r.assert("diagnose.upstream-down", gateTest, "an unreachable upstream is reported as 502", "close upstream; GET /api/hello",
			func() (string, error) {
				up.Close()
				return expect("http://"+addr+"/api/hello", 502, "")
			})
	})
}

func infraFixture(r *run) {
	if !r.prepare("new/bootstrap") {
		return
	}
	r.gate("state", gateTest, "lock, save, reload and unlock local file state",
		fixtureBin+" build .krewire/state.json", func() (string, error) { return r.fixture("build", ".krewire/state.json") })
	r.file("output", ".krewire/state.json", `"web"`)
	r.exclude("plan", "core ships provider/state/schema contracts only; no planner or diff engine exists to produce a plan", "#138")
	r.exclude("apply", "no provider driver (Docker, systemd or cloud) is implemented in core", "#138")
	r.exclude("drift", "drift detection is not implemented", "#138")
	r.diagnose("invalid-resource", "an out-of-range resource declaration is diagnosed",
		replaceIn("krewire.yaml", "replicas: 2", "replicas: 0"), "Replicas")
}
