package apphost

// The test binary doubles as the child processes the orchestrator runs: when
// KREWIRE_APPHOST_HELPER names a role, TestMain runs that tiny program instead
// of the tests. This gives real processes, real ports and real signals with no
// Docker or external binaries.

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

const helperEnv = "KREWIRE_APPHOST_HELPER"

func TestMain(m *testing.M) {
	if role := os.Getenv(helperEnv); role != "" {
		os.Exit(runHelper(role))
	}
	os.Exit(m.Run())
}

// helper builds a process resource that runs role.
func helper(kind Kind, role string, env map[string]string) *Resource {
	e := map[string]string{helperEnv: role}
	for k, v := range env {
		e[k] = v
	}
	return &Resource{Kind: kind, Run: &RunSpec{Command: os.Args[0]}, Env: e}
}

func runHelper(role string) int {
	term := make(chan os.Signal, 1)
	signal.Notify(term, syscall.SIGTERM, os.Interrupt)
	if d, err := time.ParseDuration(os.Getenv("HELPER_DELAY")); err == nil {
		time.Sleep(d)
	}
	switch role {
	case "listen":
		return serveTCP(os.Getenv("KREWIRE_SELF_TCP_PORT"), term)
	case "crashonce":
		marker := os.Getenv("HELPER_MARKER")
		if _, err := os.Stat(marker); err != nil {
			_ = os.WriteFile(marker, []byte("1"), 0o600)
			go func() {
				time.Sleep(300 * time.Millisecond)
				fmt.Fprintln(os.Stderr, "crashing after ready")
				os.Exit(1)
			}()
		}
		return serveTCP(os.Getenv("KREWIRE_SELF_TCP_PORT"), term)
	case "api":
		return serveAPI(term)
	case "worker":
		c, err := net.Dial("tcp", os.Getenv("CACHE_ADDR"))
		if err != nil {
			fmt.Println("worker cannot reach cache:", err)
			return 2
		}
		_ = c.Close()
		fmt.Println("worker connected to cache")
		<-term
		fmt.Println("worker draining")
		return 0
	case "crash":
		fmt.Fprintln(os.Stderr, "boom: cannot open database")
		return 3
	case "crashlater":
		time.Sleep(100 * time.Millisecond)
		return 4
	case "exit0":
		fmt.Println("job done")
		return 0
	case "silent":
		<-term
		return 0
	case "stubborn":
		signal.Ignore(syscall.SIGTERM, os.Interrupt)
		fmt.Println("ignoring SIGTERM")
		time.Sleep(time.Hour)
		return 0
	case "spawn":
		child := exec.Command(os.Args[0])
		child.Env = append(os.Environ(), helperEnv+"=stubborn")
		if err := child.Start(); err != nil {
			fmt.Println("spawn failed:", err)
			return 2
		}
		fmt.Printf("child=%d\n", child.Process.Pid)
		<-term
		fmt.Println("parent exiting, leaving child behind")
		return 0
	}
	fmt.Fprintln(os.Stderr, "unknown helper role", role)
	return 99
}

func serveTCP(port string, term <-chan os.Signal) int {
	l, err := net.Listen("tcp", "127.0.0.1:"+port)
	if err != nil {
		fmt.Fprintln(os.Stderr, "listen:", err)
		return 2
	}
	fmt.Println("listening on", l.Addr())
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			_, _ = c.Write([]byte("ok\n"))
			_ = c.Close()
		}
	}()
	<-term
	fmt.Println("graceful shutdown")
	_ = l.Close()
	return 0
}

func serveAPI(term <-chan os.Signal) int {
	for _, key := range []string{"DB_ADDR", "CACHE_ADDR"} {
		c, err := net.Dial("tcp", os.Getenv(key))
		if err != nil {
			fmt.Println("api cannot reach", key, err)
			return 2
		}
		line, _ := bufio.NewReader(c).ReadString('\n')
		_ = c.Close()
		fmt.Printf("api reached %s: %s", key, line)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, "db=%s cache=%s", os.Getenv("KREWIRE_DB_URL"), os.Getenv("KREWIRE_CACHE_URL"))
	})
	srv := &http.Server{Addr: "127.0.0.1:" + os.Getenv("PORT"), Handler: mux}
	go func() { _ = srv.ListenAndServe() }()
	<-term
	fmt.Println("api graceful shutdown")
	_ = srv.Close()
	return 0
}
