package sidecar

import (
	"encoding/json"
	"log/slog"
	"os"
	"runtime"
	"sync"
	"time"
)

const (
	sseMaxBackoff      = 60 * time.Second
	sseStableThreshold = 30 * time.Second // reset backoff if connection lasted this long
)

type Agent struct {
	config  *Config
	client  *Client
	pm      *ProcessManager
	cm      *ConfigManager
	nodeID  uint
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

func NewAgent(config *Config) *Agent {
	client := NewClient(config.ServerURL, config.Token)
	pm := NewProcessManager()
	pm.SetBaseDir("generate")
	pm.SetLogDir("logs")
	cm := NewConfigManager(client, pm, "generate")

	return &Agent{
		config: config,
		client: client,
		pm:     pm,
		cm:     cm,
		stopCh: make(chan struct{}),
	}
}

func (a *Agent) Run() error {
	slog.Info("sidecar starting", "version", Version, "server", a.config.ServerURL)

	// Register with server (retry until success or stopped)
	sysInfo := collectSystemInfo()
	for {
		nodeID, err := a.client.Register(sysInfo, nil, Version)
		if err == nil {
			a.nodeID = nodeID
			slog.Info("registered with server", "nodeId", nodeID)
			break
		}
		slog.Warn("register failed, retrying in 5s", "error", err)
		select {
		case <-a.stopCh:
			return nil
		case <-time.After(5 * time.Second):
		}
	}

	// Start heartbeat loop
	a.wg.Add(1)
	go a.heartbeatLoop()

	// Start SSE event loop (replaces command polling)
	a.wg.Add(1)
	go a.sseLoop()

	// Start probe loop
	a.wg.Add(1)
	go a.probeLoop()

	// Start log upload loop
	a.wg.Add(1)
	go a.logUploadLoop()

	// Wait for stop signal
	<-a.stopCh
	slog.Info("sidecar shutting down")
	return nil
}

func (a *Agent) Stop() {
	close(a.stopCh)
	a.pm.StopAll()
	a.wg.Wait()
	slog.Info("sidecar stopped")
}

func (a *Agent) heartbeatLoop() {
	defer a.wg.Done()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Send initial heartbeat
	a.sendHeartbeat()

	for {
		select {
		case <-a.stopCh:
			return
		case <-ticker.C:
			a.sendHeartbeat()
		}
	}
}

func (a *Agent) sendHeartbeat() {
	reports := a.pm.GetStatus()
	if err := a.client.Heartbeat(reports); err != nil {
		slog.Warn("heartbeat failed", "error", err)
	}
}

func (a *Agent) sseLoop() {
	defer a.wg.Done()

	sseClient := NewSSEClient(a.config.ServerURL, a.config.Token)
	eventCh := make(chan SSEEvent, 64)
	attempt := 0

	// Process events in a separate goroutine
	go func() {
		for {
			select {
			case <-a.stopCh:
				return
			case evt := <-eventCh:
				a.handleSSEEvent(evt)
			}
		}
	}()

	for {
		select {
		case <-a.stopCh:
			return
		default:
		}

		slog.Info("connecting to SSE stream", "attempt", attempt)
		connStart := time.Now()
		err := sseClient.Connect(eventCh, a.stopCh)

		// Reset backoff if connection was stable for a while
		if time.Since(connStart) > sseStableThreshold {
			attempt = 0
		}

		select {
		case <-a.stopCh:
			return
		default:
		}

		if err != nil {
			slog.Warn("SSE connection lost", "error", err)
		}

		backoff := BackoffDuration(attempt, sseMaxBackoff) + ReconnectJitter()
		slog.Info("SSE reconnecting", "backoff", backoff, "attempt", attempt)
		select {
		case <-a.stopCh:
			return
		case <-time.After(backoff):
		}
		attempt++
	}
}

func (a *Agent) handleSSEEvent(evt SSEEvent) {
	switch evt.Event {
	case "command":
		cmd, err := ParseCommandEvent(evt.Data)
		if err != nil {
			slog.Warn("failed to parse SSE command event", "error", err)
			return
		}
		a.handleCommand(cmd)
	case "ping":
		// Ignore keepalive pings
	default:
		slog.Debug("unknown SSE event type", "type", evt.Event)
	}
}

func (a *Agent) probeLoop() {
	defer a.wg.Done()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-a.stopCh:
			return
		case <-ticker.C:
			a.pm.RunProbes()
		}
	}
}

func (a *Agent) logUploadLoop() {
	defer a.wg.Done()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-a.stopCh:
			// Final drain before shutdown
			a.uploadLogs()
			return
		case <-ticker.C:
			a.uploadLogs()
		}
	}
}

func (a *Agent) uploadLogs() {
	logs := a.pm.DrainLogs()
	if len(logs) == 0 {
		return
	}
	if err := a.client.UploadLogs(logs); err != nil {
		slog.Warn("log upload failed", "count", len(logs), "error", err)
	}
}

func (a *Agent) handleCommand(cmd Command) {
	slog.Info("received command", "id", cmd.ID, "type", cmd.Type)

	switch cmd.Type {
	case "config.update":
		var payload struct {
			ProcessDefID uint   `json:"process_def_id"`
			ProcessName  string `json:"process_name"`
		}
		_ = json.Unmarshal(cmd.Payload, &payload)
		go a.cm.HandleConfigUpdate(payload.ProcessDefID, payload.ProcessName, a.client, cmd.ID)
	default:
		a.pm.HandleCommand(cmd, a.client)
	}
}

func collectSystemInfo() map[string]any {
	hostname, _ := os.Hostname()
	return map[string]any{
		"hostname": hostname,
		"os":       runtime.GOOS,
		"arch":     runtime.GOARCH,
		"cpus":     runtime.NumCPU(),
	}
}
