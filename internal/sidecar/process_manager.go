package sidecar

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"text/template"
	"time"
)

type ProcessDef struct {
	ID            uint              `json:"id"`
	Name          string            `json:"name"`
	StartCommand  string            `json:"startCommand"`
	StopCommand   string            `json:"stopCommand"`
	ReloadCommand string            `json:"reloadCommand"`
	Env           map[string]string `json:"env"`
	RestartPolicy string            `json:"restartPolicy"`
	MaxRestarts   int               `json:"maxRestarts"`
	ProbeType     string            `json:"probeType"`
	ProbeConfig   json.RawMessage   `json:"probeConfig"`
	ConfigFiles   json.RawMessage   `json:"configFiles"`
}

type ManagedProcess struct {
	Def           ProcessDef
	cmd           *exec.Cmd
	Status        string
	PID           int
	ConfigVersion string
	RestartCount  int
	OverrideVars  json.RawMessage
	Probe         *ProbeConfig
	LastProbe     *ProbeResult
	StartedAt     time.Time
	stdoutWriter  *LogWriter
	stderrWriter  *LogWriter
	stopCh        chan struct{}
	mu            sync.Mutex
}

type ProcessManager struct {
	processes map[uint]*ManagedProcess // keyed by ProcessDef ID
	baseDir   string                   // base directory for config/data output
	logDir    string                   // base directory for log files
	mu        sync.RWMutex
}

func NewProcessManager() *ProcessManager {
	return &ProcessManager{
		processes: make(map[uint]*ManagedProcess),
	}
}

// SetBaseDir sets the base directory used for resolving {{.ConfigDir}} and {{.DataDir}} placeholders.
func (pm *ProcessManager) SetBaseDir(dir string) {
	pm.baseDir = dir
}

// SetLogDir sets the base directory for log files.
func (pm *ProcessManager) SetLogDir(dir string) {
	pm.logDir = dir
}

func (pm *ProcessManager) Start(def ProcessDef) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if existing, ok := pm.processes[def.ID]; ok {
		existing.mu.Lock()
		if existing.Status == "running" {
			existing.mu.Unlock()
			return fmt.Errorf("process %s already running", def.Name)
		}
		existing.mu.Unlock()
	}

	mp := &ManagedProcess{
		Def:    def,
		Status: "stopped",
		stopCh: make(chan struct{}),
	}
	pm.processes[def.ID] = mp

	return pm.startProcess(mp)
}

func (pm *ProcessManager) startProcess(mp *ManagedProcess) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	if mp.Def.StartCommand == "" {
		mp.Status = "error"
		return fmt.Errorf("no start command defined for process %s", mp.Def.Name)
	}

	// Render the start command with template placeholders
	rendered := pm.renderCommand(mp.Def.StartCommand, mp.Def.Name, 0)

	cmd := exec.Command("sh", "-c", rendered)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set up log writers if logDir is configured
	if pm.logDir != "" {
		if sw, err := NewLogWriter(pm.logDir, mp.Def.Name, "stdout"); err == nil {
			mp.stdoutWriter = sw
			cmd.Stdout = io.MultiWriter(os.Stdout, sw)
		}
		if ew, err := NewLogWriter(pm.logDir, mp.Def.Name, "stderr"); err == nil {
			mp.stderrWriter = ew
			cmd.Stderr = io.MultiWriter(os.Stderr, ew)
		}
	}

	// Set environment
	cmd.Env = os.Environ()
	for k, v := range mp.Def.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	// Set process group for clean termination
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		mp.Status = "error"
		return fmt.Errorf("failed to start process %s: %w", mp.Def.Name, err)
	}

	mp.cmd = cmd
	mp.Status = "running"
	mp.PID = cmd.Process.Pid
	mp.StartedAt = time.Now()

	slog.Info("process started", "name", mp.Def.Name, "pid", mp.PID, "command", rendered)

	// Monitor process in background
	go pm.monitor(mp)

	return nil
}

func (pm *ProcessManager) monitor(mp *ManagedProcess) {
	if mp.cmd == nil || mp.cmd.Process == nil {
		return
	}

	// Reset restart count if process runs stably for 5 minutes
	stableTimer := time.NewTimer(5 * time.Minute)
	waitDone := make(chan error, 1)
	go func() {
		waitDone <- mp.cmd.Wait()
	}()

	var exitErr error
	select {
	case <-stableTimer.C:
		// Process has been running stably — reset restart count
		mp.mu.Lock()
		mp.RestartCount = 0
		mp.mu.Unlock()
		slog.Debug("process stable, restart count reset", "name", mp.Def.Name)
		exitErr = <-waitDone
	case exitErr = <-waitDone:
		stableTimer.Stop()
	}

	mp.mu.Lock()
	wasRunning := mp.Status == "running"
	mp.Status = "stopped"
	mp.PID = 0
	mp.mu.Unlock()

	if !wasRunning {
		return
	}

	select {
	case <-mp.stopCh:
		// Intentional stop, don't restart
		slog.Info("process stopped intentionally", "name", mp.Def.Name)
		return
	default:
	}

	if exitErr != nil {
		slog.Warn("process exited with error", "name", mp.Def.Name, "error", exitErr)
	} else {
		slog.Info("process exited", "name", mp.Def.Name)
	}

	// Auto-restart based on policy
	shouldRestart := false
	switch mp.Def.RestartPolicy {
	case "always":
		shouldRestart = true
	case "on_failure":
		shouldRestart = exitErr != nil
	case "never":
		shouldRestart = false
	}

	if shouldRestart && mp.RestartCount < mp.Def.MaxRestarts {
		mp.RestartCount++
		backoff := time.Duration(mp.RestartCount) * 3 * time.Second
		if backoff > 30*time.Second {
			backoff = 30 * time.Second
		}
		slog.Info("restarting process", "name", mp.Def.Name, "attempt", mp.RestartCount, "backoff", backoff)
		time.Sleep(backoff)

		select {
		case <-mp.stopCh:
			return
		default:
		}

		if err := pm.startProcess(mp); err != nil {
			slog.Error("failed to restart process", "name", mp.Def.Name, "error", err)
			mp.mu.Lock()
			mp.Status = "error"
			mp.mu.Unlock()
		}
	} else if shouldRestart {
		slog.Error("max restarts exceeded", "name", mp.Def.Name, "max", mp.Def.MaxRestarts)
		mp.mu.Lock()
		mp.Status = "error"
		mp.mu.Unlock()
	}
}

func (pm *ProcessManager) Stop(defID uint) error {
	pm.mu.RLock()
	mp, ok := pm.processes[defID]
	pm.mu.RUnlock()

	if !ok {
		return fmt.Errorf("process not found")
	}

	return pm.stopProcess(mp)
}

func (pm *ProcessManager) stopProcess(mp *ManagedProcess) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	if mp.Status != "running" || mp.cmd == nil || mp.cmd.Process == nil {
		mp.Status = "stopped"
		return nil
	}

	close(mp.stopCh)

	// If a StopCommand is defined, use it
	if mp.Def.StopCommand != "" {
		rendered := pm.renderCommand(mp.Def.StopCommand, mp.Def.Name, mp.PID)
		slog.Info("executing stop command", "name", mp.Def.Name, "command", rendered)
		stopCmd := exec.Command("sh", "-c", rendered)
		stopCmd.Stdout = os.Stdout
		stopCmd.Stderr = os.Stderr
		if err := stopCmd.Run(); err != nil {
			slog.Warn("stop command failed, falling back to SIGTERM", "name", mp.Def.Name, "error", err)
		} else {
			// Wait for process to exit after stop command
			done := make(chan struct{})
			go func() {
				_ = mp.cmd.Wait()
				close(done)
			}()

			select {
			case <-done:
				mp.Status = "stopped"
				mp.PID = 0
				slog.Info("process stopped via stop command", "name", mp.Def.Name)
				return nil
			case <-time.After(10 * time.Second):
				slog.Warn("process did not exit after stop command, sending SIGKILL", "name", mp.Def.Name)
				_ = mp.cmd.Process.Kill()
				mp.Status = "stopped"
				mp.PID = 0
				return nil
			}
		}
	}

	// Default: Send SIGTERM
	if err := mp.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		slog.Warn("SIGTERM failed, sending SIGKILL", "name", mp.Def.Name, "error", err)
		_ = mp.cmd.Process.Kill()
	} else {
		// Wait up to 10s for graceful shutdown
		done := make(chan struct{})
		go func() {
			_ = mp.cmd.Wait()
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(10 * time.Second):
			slog.Warn("process did not stop in time, sending SIGKILL", "name", mp.Def.Name)
			_ = mp.cmd.Process.Kill()
		}
	}

	mp.Status = "stopped"
	mp.PID = 0
	slog.Info("process stopped", "name", mp.Def.Name)
	return nil
}

func (pm *ProcessManager) Restart(defID uint) error {
	pm.mu.RLock()
	mp, ok := pm.processes[defID]
	pm.mu.RUnlock()

	if !ok {
		return fmt.Errorf("process not found")
	}

	if err := pm.stopProcess(mp); err != nil {
		return err
	}

	// Reset stop channel and restart count
	mp.stopCh = make(chan struct{})
	mp.RestartCount = 0

	return pm.startProcess(mp)
}

func (pm *ProcessManager) Reload(defID uint) error {
	pm.mu.RLock()
	mp, ok := pm.processes[defID]
	pm.mu.RUnlock()

	if !ok {
		return fmt.Errorf("process not found")
	}

	mp.mu.Lock()
	if mp.Status != "running" || mp.cmd == nil || mp.cmd.Process == nil {
		mp.mu.Unlock()
		return fmt.Errorf("process not running")
	}

	// If a ReloadCommand is defined, try it first
	if mp.Def.ReloadCommand != "" {
		rendered := pm.renderCommand(mp.Def.ReloadCommand, mp.Def.Name, mp.PID)
		mp.mu.Unlock()

		slog.Info("executing reload command", "name", mp.Def.Name, "command", rendered)
		cmd := exec.Command("sh", "-c", rendered)
		if err := cmd.Run(); err != nil {
			slog.Warn("reload command failed, falling back to restart", "name", mp.Def.Name, "error", err)
		} else {
			return nil
		}
	} else {
		mp.mu.Unlock()
	}

	// Fallback to restart (lock already released)
	slog.Info("falling back to restart for reload", "name", mp.Def.Name)
	return pm.Restart(defID)
}

// renderCommand renders a command template with standard placeholders.
func (pm *ProcessManager) renderCommand(cmdTmpl string, processName string, pid int) string {
	baseDir := pm.baseDir
	if baseDir == "" {
		baseDir = "."
	}
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		absBase = baseDir
	}

	ctx := map[string]any{
		"ConfigDir":   filepath.Join(absBase, processName),
		"DataDir":     filepath.Join(absBase, "data", processName),
		"ProcessName": processName,
		"PID":         strconv.Itoa(pid),
	}

	t, err := template.New("cmd").Parse(cmdTmpl)
	if err != nil {
		slog.Warn("failed to parse command template, using raw", "error", err)
		return cmdTmpl
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, ctx); err != nil {
		slog.Warn("failed to render command template, using raw", "error", err)
		return cmdTmpl
	}

	return buf.String()
}

func (pm *ProcessManager) GetStatus() []ProcessReport {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	reports := make([]ProcessReport, 0, len(pm.processes))
	for _, mp := range pm.processes {
		mp.mu.Lock()
		report := ProcessReport{
			ProcessDefID:  mp.Def.ID,
			Status:        mp.Status,
			PID:           mp.PID,
			ConfigVersion: mp.ConfigVersion,
		}
		if mp.LastProbe != nil {
			report.ProbeResult = MarshalProbeResult(*mp.LastProbe)
		}
		reports = append(reports, report)
		mp.mu.Unlock()
	}
	return reports
}

func (pm *ProcessManager) SetConfigVersion(defID uint, version string) {
	pm.mu.RLock()
	mp, ok := pm.processes[defID]
	pm.mu.RUnlock()
	if ok {
		mp.mu.Lock()
		mp.ConfigVersion = version
		mp.mu.Unlock()
	}
}

// RunProbes executes probes for all running processes that have a probe configured.
func (pm *ProcessManager) RunProbes() {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	for _, mp := range pm.processes {
		mp.mu.Lock()
		if mp.Status != "running" || mp.Probe == nil || mp.Probe.Type == "" {
			mp.mu.Unlock()
			continue
		}
		probe := *mp.Probe
		mp.mu.Unlock()

		result := RunProbe(probe)

		mp.mu.Lock()
		mp.LastProbe = &result
		mp.mu.Unlock()
	}
}

// DrainLogs collects all buffered log lines from all processes.
func (pm *ProcessManager) DrainLogs() []LogLine {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var all []LogLine
	for _, mp := range pm.processes {
		mp.mu.Lock()
		if mp.stdoutWriter != nil {
			all = append(all, mp.stdoutWriter.Drain()...)
		}
		if mp.stderrWriter != nil {
			all = append(all, mp.stderrWriter.Drain()...)
		}
		mp.mu.Unlock()
	}
	return all
}

func (pm *ProcessManager) StopAll() {
	pm.mu.RLock()
	ids := make([]uint, 0, len(pm.processes))
	for id := range pm.processes {
		ids = append(ids, id)
	}
	pm.mu.RUnlock()

	for _, id := range ids {
		if err := pm.Stop(id); err != nil {
			slog.Error("failed to stop process", "id", id, "error", err)
		}
	}
}

func (pm *ProcessManager) HandleCommand(cmd Command, client *Client) {
	var payload struct {
		ProcessDefID  uint            `json:"process_def_id"`
		NodeProcessID uint            `json:"node_process_id"`
		OverrideVars  json.RawMessage `json:"override_vars"`
		ProcessDef    ProcessDef      `json:"process_def"`
	}
	_ = json.Unmarshal(cmd.Payload, &payload)

	var err error
	switch cmd.Type {
	case "process.start":
		def := payload.ProcessDef
		if def.ID == 0 {
			def.ID = payload.ProcessDefID
		}
		err = pm.Start(def)
		if err == nil {
			// Store probe config and override vars after successful start
			pm.mu.RLock()
			mp, ok := pm.processes[def.ID]
			pm.mu.RUnlock()
			if ok {
				mp.mu.Lock()
				mp.OverrideVars = payload.OverrideVars
				if def.ProbeType != "" {
					pc := ProbeConfig{Type: def.ProbeType}
					_ = json.Unmarshal(def.ProbeConfig, &pc)
					mp.Probe = &pc
				}
				mp.mu.Unlock()
			}
		}
	case "process.stop":
		err = pm.Stop(payload.ProcessDefID)
	case "process.restart":
		err = pm.Restart(payload.ProcessDefID)
	case "config.update":
		// Config update is handled by ConfigManager
		return
	default:
		err = fmt.Errorf("unknown command type: %s", cmd.Type)
	}

	success := err == nil
	result := ""
	if err != nil {
		result = err.Error()
	}

	if ackErr := client.AckCommand(cmd.ID, success, result); ackErr != nil {
		slog.Error("failed to ack command", "id", cmd.ID, "error", ackErr)
	}
}
