package apphost

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ContainerDriver runs ContainerSpec resources with a Docker-compatible CLI
// (docker, podman). It is optional: tests and topologies without container
// resources never need it, and NewOrchestrator only installs it when the
// binary is on PATH.
type ContainerDriver struct {
	// Binary is the CLI to invoke (default "docker").
	Binary string
	// Prefix namespaces container names (default "krewire").
	Prefix string
}

// DetectContainerDriver returns a ContainerDriver when docker or podman is
// installed.
func DetectContainerDriver() (Driver, bool) {
	for _, bin := range []string{"docker", "podman"} {
		if path, err := exec.LookPath(bin); err == nil {
			return ContainerDriver{Binary: path}, true
		}
	}
	return nil, false
}

// RunArgs builds the CLI arguments for one resource. Only explicitly declared
// variables are passed into the container (never the host environment).
func (d ContainerDriver) RunArgs(spec LaunchSpec) []string {
	r := spec.Resource
	args := []string{"run", "--rm", "--name", d.containerName(r)}
	for _, e := range r.Endpoints {
		args = append(args, "-p", e.Host+":"+strconv.Itoa(e.Port)+":"+strconv.Itoa(e.TargetPort))
	}
	for _, kv := range spec.Env {
		k, _, _ := strings.Cut(kv, "=")
		if _, declared := r.Env[k]; declared || strings.HasPrefix(k, discoveryPrefix) {
			args = append(args, "-e", kv)
		}
	}
	args = append(args, r.Container.Image)
	return append(args, r.Container.Args...)
}

func (d ContainerDriver) bin() string {
	if d.Binary == "" {
		return "docker"
	}
	return d.Binary
}

func (d ContainerDriver) containerName(r *Resource) string {
	p := d.Prefix
	if p == "" {
		p = "krewire"
	}
	return p + "-" + r.Name
}

// Start implements Driver. The CLI runs attached so container logs flow into
// the orchestrator's log aggregation; Stop uses `<cli> stop -t <grace>`.
func (d ContainerDriver) Start(_ context.Context, spec LaunchSpec) (Process, error) {
	if spec.Resource.Container == nil {
		return nil, fmt.Errorf("apphost: resource %q has no container block", spec.Resource.Name)
	}
	cmd := exec.Command(d.bin(), d.RunArgs(spec)...)
	p, err := startCmd(cmd, spec, 0)
	if err != nil {
		return nil, err
	}
	name := d.containerName(spec.Resource)
	p.stop = func(grace time.Duration) {
		secs := strconv.Itoa(int(grace.Seconds()))
		_ = exec.Command(d.bin(), "stop", "-t", secs, name).Run()
	}
	return p, nil
}
