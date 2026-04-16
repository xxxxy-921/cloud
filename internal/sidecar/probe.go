package sidecar

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"time"
)

type ProbeConfig struct {
	Type     string `json:"type"`     // http, tcp, exec
	Endpoint string `json:"endpoint"` // URL for http, host:port for tcp
	Command  string `json:"command"`  // shell command for exec
	Timeout  int    `json:"timeout"`  // seconds, default 5
	Interval int    `json:"interval"` // seconds, default 30
}

type ProbeResult struct {
	Status   string `json:"status"`   // healthy, unhealthy
	Message  string `json:"message"`
	Duration int64  `json:"duration"` // milliseconds
}

func RunProbe(cfg ProbeConfig) ProbeResult {
	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	start := time.Now()
	var result ProbeResult

	switch cfg.Type {
	case "http":
		result = probeHTTP(cfg.Endpoint, timeout)
	case "tcp":
		result = probeTCP(cfg.Endpoint, timeout)
	case "exec":
		result = probeExec(cfg.Command, timeout)
	default:
		result = ProbeResult{Status: "healthy", Message: "no probe configured"}
	}

	result.Duration = time.Since(start).Milliseconds()
	return result
}

func probeHTTP(url string, timeout time.Duration) ProbeResult {
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(url)
	if err != nil {
		return ProbeResult{Status: "unhealthy", Message: fmt.Sprintf("http error: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return ProbeResult{Status: "healthy", Message: fmt.Sprintf("HTTP %d", resp.StatusCode)}
	}
	return ProbeResult{Status: "unhealthy", Message: fmt.Sprintf("HTTP %d", resp.StatusCode)}
}

func probeTCP(addr string, timeout time.Duration) ProbeResult {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return ProbeResult{Status: "unhealthy", Message: fmt.Sprintf("tcp error: %v", err)}
	}
	conn.Close()
	return ProbeResult{Status: "healthy", Message: "tcp connection ok"}
}

func probeExec(command string, timeout time.Duration) ProbeResult {
	cmd := exec.Command("sh", "-c", command)

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			return ProbeResult{Status: "unhealthy", Message: fmt.Sprintf("exec error: %v", err)}
		}
		return ProbeResult{Status: "healthy", Message: "exec ok"}
	case <-time.After(timeout):
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return ProbeResult{Status: "unhealthy", Message: "exec timeout"}
	}
}

func MarshalProbeResult(r ProbeResult) json.RawMessage {
	data, _ := json.Marshal(r)
	return data
}
