package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// Runtime owns the local model process: it starts it, keeps it warm, watches
// its health, restarts it when it dies, and refuses work when it cannot serve
// any. It is the only component that knows a model file exists.
//
// Two deployment shapes are supported and both are first class:
//
//   - managed (Managed=true): the API process supervises llama-server as a
//     child. One unit to deploy, and the model's lifetime is exactly the API's.
//   - external (Managed=false): a separate systemd unit (or a developer's
//     terminal) runs the server; the API only probes and uses it. This is the
//     better shape when the model should survive API restarts, or when the
//     inference box is a different machine on the private network.
//
// Whichever is used, the guarantees the rest of the application relies on are
// the same: the model is loaded once and stays warm, concurrency is bounded,
// and a dead model degrades the assistant without touching anything else in
// ShiftMaster.
type Runtime struct {
	cfg RuntimeConfig

	mu     sync.RWMutex
	state  string
	detail string

	// slots bounds simultaneous inference. Sized to the server's own slot
	// count: queueing here rather than inside llama.cpp lets us apply a wait
	// budget and cancel on client disconnect.
	slots chan struct{}

	http *http.Client

	// process supervision
	procMu   sync.Mutex
	cmd      *exec.Cmd
	logFile  *os.File
	restarts atomic.Int64

	stop     context.CancelFunc
	stopped  chan struct{}
	stopOnce sync.Once
}

// RuntimeConfig describes how to reach — and optionally how to start — the
// local model server.
type RuntimeConfig struct {
	// BaseURL is where the server listens: http://127.0.0.1:PORT or
	// unix:///run/shiftmaster/llm.sock.
	BaseURL string
	APIKey  string

	// Managed makes this process responsible for the server's lifetime.
	Managed bool
	// ServerBin is the llama.cpp server executable (looked up on PATH when a
	// bare name). Required when Managed.
	ServerBin string
	// ModelPath is the local GGUF file. Required when Managed. Never leaves
	// this package.
	ModelPath string
	// ModelName is the label used in logs and request records.
	ModelName string

	ContextSize int
	GPULayers   int // -1 offloads everything the device can hold
	Threads     int // 0 lets llama.cpp choose
	Parallel    int // server slots == our concurrency budget
	ExtraArgs   []string

	// LogPath receives the server's stdout/stderr when managed. Model logs can
	// contain prompt fragments, so this file is written with 0600.
	LogPath string

	// StartupTimeout bounds model loading before the runtime is declared
	// unavailable. Large models on cold page cache genuinely take minutes.
	StartupTimeout time.Duration
	// HealthInterval is the polling period once running.
	HealthInterval time.Duration
	// QueueWait is how long a request may wait for a free slot.
	QueueWait time.Duration
	// Warm issues one tiny generation after load so the first real user turn
	// does not pay for graph construction and cache allocation.
	Warm bool
}

func (c *RuntimeConfig) applyDefaults() {
	if c.ContextSize <= 0 {
		c.ContextSize = 8192
	}
	if c.Parallel <= 0 {
		c.Parallel = 1
	}
	if c.StartupTimeout <= 0 {
		c.StartupTimeout = 5 * time.Minute
	}
	if c.HealthInterval <= 0 {
		c.HealthInterval = 10 * time.Second
	}
	if c.QueueWait <= 0 {
		c.QueueWait = 20 * time.Second
	}
	if c.ServerBin == "" {
		c.ServerBin = "llama-server"
	}
}

// NewRuntime creates a runtime in the starting state. Nothing happens until
// Start is called.
func NewRuntime(cfg RuntimeConfig) *Runtime {
	cfg.applyDefaults()
	return &Runtime{
		cfg:     cfg,
		state:   StateStarting,
		detail:  "initialising",
		slots:   make(chan struct{}, cfg.Parallel),
		http:    newLocalHTTPClient(cfg.BaseURL),
		stopped: make(chan struct{}),
	}
}

// Health is the safe public view, for GET /assistant/status.
func (r *Runtime) Health() Health {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return Health{State: r.state, Detail: r.detail}
}

// Ready reports whether a turn can be attempted right now.
func (r *Runtime) Ready() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.state == StateReady
}

func (r *Runtime) set(state, detail string) {
	r.mu.Lock()
	changed := r.state != state
	r.state, r.detail = state, detail
	r.mu.Unlock()
	if changed {
		log.Printf("assistant: runtime %s (%s)", state, detail)
	}
}

// Enter implements Gate: bounded queueing in front of a finite resource. A
// caller that gives up (client disconnect, turn deadline) frees its place
// immediately instead of holding a slot no one is waiting on.
func (r *Runtime) Enter(ctx context.Context) (func(), error) {
	if !r.Ready() {
		h := r.Health()
		return nil, fmt.Errorf("%w: runtime %s", ErrUnavailable, h.State)
	}
	waitCtx, cancel := context.WithTimeout(ctx, r.cfg.QueueWait)
	defer cancel()

	select {
	case r.slots <- struct{}{}:
		var once sync.Once
		return func() { once.Do(func() { <-r.slots }) }, nil
	case <-waitCtx.Done():
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%w: %v", ErrUnavailable, ctx.Err())
		}
		return nil, ErrBusy
	}
}

// Start brings the runtime up and keeps it up. It returns as soon as
// supervision has begun; readiness is reported through Health, so a slow model
// load never delays the API's own startup and never blocks any other feature.
func (r *Runtime) Start(parent context.Context) {
	ctx, cancel := context.WithCancel(context.WithoutCancel(parent))
	r.stop = cancel
	go r.supervise(ctx)
}

// Close stops supervision and terminates a managed server.
func (r *Runtime) Close() error {
	r.stopOnce.Do(func() {
		if r.stop != nil {
			r.stop()
		}
		select {
		case <-r.stopped:
		case <-time.After(15 * time.Second):
			log.Printf("assistant: runtime shutdown timed out")
		}
	})
	return nil
}

// supervise is the lifecycle loop: (re)start when managed, wait for health,
// warm, then poll. A crash-looping model backs off instead of spinning.
func (r *Runtime) supervise(ctx context.Context) {
	defer close(r.stopped)
	defer r.terminate()

	backoff := 2 * time.Second
	for {
		if ctx.Err() != nil {
			return
		}

		if r.cfg.Managed {
			if err := r.spawn(ctx); err != nil {
				r.set(StateUnavailable, "start_failed")
				log.Printf("assistant: cannot start local model runtime: %v", err)
				if !sleepCtx(ctx, backoff) {
					return
				}
				backoff = nextBackoff(backoff)
				continue
			}
		}

		if err := r.waitHealthy(ctx); err != nil {
			if ctx.Err() != nil {
				return
			}
			r.set(StateUnavailable, unavailableDetail(r.cfg.Managed))
			r.terminate()
			if !sleepCtx(ctx, backoff) {
				return
			}
			backoff = nextBackoff(backoff)
			continue
		}

		backoff = 2 * time.Second
		if r.cfg.Warm {
			r.warm(ctx)
		}
		r.set(StateReady, "")

		// Steady state: poll until the server stops answering or we are asked
		// to stop.
		if r.monitor(ctx) {
			return
		}
		r.restarts.Add(1)
		r.set(StateDegraded, "restarting")
		r.terminate()
		if !sleepCtx(ctx, backoff) {
			return
		}
		backoff = nextBackoff(backoff)
	}
}

func unavailableDetail(managed bool) string {
	if managed {
		return "model_load_failed"
	}
	return "runtime_not_running"
}

func nextBackoff(d time.Duration) time.Duration {
	d *= 2
	if d > time.Minute {
		return time.Minute
	}
	return d
}

func sleepCtx(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}

// spawn starts llama-server. The model file is validated first so a typo in
// configuration produces one clear log line rather than a restart loop.
func (r *Runtime) spawn(ctx context.Context) error {
	r.procMu.Lock()
	defer r.procMu.Unlock()
	if r.cmd != nil && r.cmd.Process != nil {
		return nil // already running
	}

	if r.cfg.ModelPath == "" {
		return errors.New("no model path configured")
	}
	info, err := os.Stat(r.cfg.ModelPath)
	if err != nil {
		return fmt.Errorf("model file unreadable: %w", err)
	}
	if info.IsDir() || info.Size() < 1<<20 {
		return fmt.Errorf("model file is not a model: %s", filepath.Base(r.cfg.ModelPath))
	}

	args := r.serverArgs()
	cmd := exec.CommandContext(ctx, r.cfg.ServerBin, args...)
	// A managed server gets its own process group so terminate() can take down
	// the whole tree, and so a Ctrl-C in a terminal does not race us.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if r.cfg.LogPath != "" {
		if f, err := os.OpenFile(r.cfg.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600); err == nil {
			cmd.Stdout, cmd.Stderr = f, f
			r.logFile = f
		} else {
			log.Printf("assistant: cannot open runtime log %s: %v", r.cfg.LogPath, err)
		}
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("exec %s: %w", r.cfg.ServerBin, err)
	}
	r.cmd = cmd
	r.set(StateStarting, "loading_model")
	log.Printf("assistant: started local model runtime (%s, %d slots, ctx %d)",
		r.cfg.ModelName, r.cfg.Parallel, r.cfg.ContextSize)

	// Reap the child so a crashed server does not linger as a zombie, and so
	// monitor() sees the socket close.
	go func() {
		err := cmd.Wait()
		r.procMu.Lock()
		if r.cmd == cmd {
			r.cmd = nil
		}
		if r.logFile != nil {
			_ = r.logFile.Close()
			r.logFile = nil
		}
		r.procMu.Unlock()
		if err != nil && ctx.Err() == nil {
			log.Printf("assistant: local model runtime exited: %v", err)
		}
	}()
	return nil
}

// serverArgs builds the llama.cpp command line. Every choice here is a
// production decision, not a default:
//
//   - reasoning off: hybrid models must not spend the turn's token budget on
//     private thinking, and no reasoning trace may reach a user.
//   - jinja on: the model's own chat template renders tool definitions and
//     tool results in the exact form it was trained on.
//   - no webui: the box must not serve a chat UI to anyone who finds the port.
//   - cont-batching: several employees' turns interleave instead of queueing
//     head-of-line behind one long generation.
func (r *Runtime) serverArgs() []string {
	// Both branches set host, so seeding it with a loopback literal only looked
	// like a default — every path overwrote it before it was read. A Unix
	// socket has no port, which the zero value already says.
	var host, port string
	if sock, ok := unixSocketPath(r.cfg.BaseURL); ok {
		host = sock
		_ = os.Remove(sock) // a stale socket file blocks bind
	} else {
		host, port = splitHostPort(r.cfg.BaseURL)
	}

	args := []string{
		"--model", r.cfg.ModelPath,
		"--alias", r.cfg.ModelName,
		"--host", host,
		"--ctx-size", strconv.Itoa(r.cfg.ContextSize * r.cfg.Parallel),
		"--parallel", strconv.Itoa(r.cfg.Parallel),
		"--jinja",
		"--reasoning", "off",
		"--reasoning-budget", "0",
		"--no-webui",
		"--cont-batching",
		"--metrics",
	}
	if port != "" {
		args = append(args, "--port", port)
	}
	if r.cfg.GPULayers != 0 {
		args = append(args, "--gpu-layers", strconv.Itoa(r.cfg.GPULayers))
	}
	if r.cfg.Threads > 0 {
		args = append(args, "--threads", strconv.Itoa(r.cfg.Threads))
	}
	if r.cfg.APIKey != "" {
		args = append(args, "--api-key", r.cfg.APIKey)
	}
	return append(args, r.cfg.ExtraArgs...)
}

// splitHostPort pulls the listen address out of an http URL without importing
// net/url for two fields.
func splitHostPort(base string) (string, string) {
	s := base
	for _, prefix := range []string{"http://", "https://"} {
		if len(s) > len(prefix) && s[:len(prefix)] == prefix {
			s = s[len(prefix):]
		}
	}
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == ':' {
			return s[:i], s[i+1:]
		}
		if s[i] == '/' {
			s = s[:i]
		}
	}
	return s, ""
}

// terminate stops a managed server, politely first.
func (r *Runtime) terminate() {
	r.procMu.Lock()
	cmd := r.cmd
	r.cmd = nil
	r.procMu.Unlock()
	if cmd == nil || cmd.Process == nil {
		return
	}
	// Signal the whole process group: llama-server may have helper threads and
	// we do not want an orphan holding the port or the socket file.
	if pgid, err := syscall.Getpgid(cmd.Process.Pid); err == nil {
		_ = syscall.Kill(-pgid, syscall.SIGTERM)
	} else {
		_ = cmd.Process.Signal(syscall.SIGTERM)
	}
	done := make(chan struct{})
	go func() { _, _ = cmd.Process.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		_ = cmd.Process.Kill()
	}
	if sock, ok := unixSocketPath(r.cfg.BaseURL); ok {
		_ = os.Remove(sock)
	}
}

// waitHealthy polls until the server reports a loaded model or the startup
// budget runs out.
func (r *Runtime) waitHealthy(ctx context.Context) error {
	deadline := time.Now().Add(r.cfg.StartupTimeout)
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		state, err := r.probe(ctx)
		switch {
		case err == nil && state == StateReady:
			return nil
		case err == nil:
			r.set(StateStarting, "loading_model")
		}
		if time.Now().After(deadline) {
			return errors.New("runtime did not become healthy in time")
		}
		if !sleepCtx(ctx, time.Second) {
			return ctx.Err()
		}
	}
}

// monitor polls a healthy server. It returns true when told to stop, false
// when the server needs restarting.
func (r *Runtime) monitor(ctx context.Context) bool {
	failures := 0
	for {
		select {
		case <-ctx.Done():
			return true
		case <-time.After(r.cfg.HealthInterval):
		}
		state, err := r.probe(ctx)
		if err == nil && state == StateReady {
			failures = 0
			r.set(StateReady, "")
			continue
		}
		failures++
		if failures == 1 {
			r.set(StateDegraded, "health_check_failed")
		}
		// Three consecutive failures (~30s by default): the model is gone, not
		// merely busy.
		if failures >= 3 {
			return false
		}
	}
}

// probe asks the server how it is. llama.cpp answers 200 when the model is
// loaded and 503 while it is still loading.
func (r *Runtime) probe(ctx context.Context) (string, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, r.endpoint("/health"), nil)
	if err != nil {
		return StateUnavailable, err
	}
	if r.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+r.cfg.APIKey)
	}
	resp, err := r.http.Do(req)
	if err != nil {
		return StateUnavailable, err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))

	switch resp.StatusCode {
	case http.StatusOK:
		return StateReady, nil
	case http.StatusServiceUnavailable:
		return StateStarting, nil
	default:
		return StateUnavailable, fmt.Errorf("health HTTP %d", resp.StatusCode)
	}
}

func (r *Runtime) endpoint(path string) string {
	if _, ok := unixSocketPath(r.cfg.BaseURL); ok {
		return "http://localhost" + path
	}
	return r.cfg.BaseURL + path
}

// warm runs one trivial generation so the first employee of the day does not
// pay for graph build and cache allocation. Failure is not fatal: a cold model
// is slower, not broken.
func (r *Runtime) warm(ctx context.Context) {
	warmCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	body, _ := json.Marshal(map[string]any{
		"model":       r.cfg.ModelName,
		"messages":    []map[string]string{{"role": "user", "content": "ok"}},
		"max_tokens":  1,
		"temperature": 0,
		"stream":      false,
	})
	req, err := http.NewRequestWithContext(warmCtx, http.MethodPost,
		r.endpoint("/v1/chat/completions"), newReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if r.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+r.cfg.APIKey)
	}
	start := time.Now()
	resp, err := r.http.Do(req)
	if err != nil {
		return
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
	log.Printf("assistant: model warm in %dms", time.Since(start).Milliseconds())
}

// WaitReady blocks until the model is loaded and answering, or ctx expires.
// Callers use it to schedule work that is only worth doing once the runtime is
// up — pre-warming the prompt cache, for instance.
func (r *Runtime) WaitReady(ctx context.Context) bool {
	for {
		if r.Ready() {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-r.stopped:
			return false
		case <-time.After(time.Second):
		}
	}
}

// Restarts reports how many times the runtime has been brought back up, for
// operational metrics.
func (r *Runtime) Restarts() int64 { return r.restarts.Load() }
