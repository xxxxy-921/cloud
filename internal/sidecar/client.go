package sidecar

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewClient(serverURL, token string) *Client {
	return &Client{
		baseURL: strings.TrimRight(serverURL, "/"),
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) do(method, path string, body any) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	url := c.baseURL + path
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.httpClient.Do(req)
}

func (c *Client) parseResponse(resp *http.Response, target any) error {
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	if target != nil {
		return json.NewDecoder(resp.Body).Decode(target)
	}
	return nil
}

// APIResponse is the standard server response wrapper.
type APIResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type RegisterResponse struct {
	NodeID uint `json:"nodeId"`
}

func (c *Client) Register(sysInfo map[string]any, capabilities map[string]any, version string) (uint, error) {
	body := map[string]any{
		"systemInfo":   sysInfo,
		"capabilities": capabilities,
		"version":      version,
	}

	resp, err := c.do("POST", "/api/v1/nodes/sidecar/register", body)
	if err != nil {
		return 0, err
	}

	var apiResp APIResponse
	if err := c.parseResponse(resp, &apiResp); err != nil {
		return 0, err
	}

	var result RegisterResponse
	if err := json.Unmarshal(apiResp.Data, &result); err != nil {
		return 0, err
	}
	return result.NodeID, nil
}

func (c *Client) Heartbeat(processes []ProcessReport) error {
	body := map[string]any{
		"processes": processes,
		"version":   Version,
	}

	resp, err := c.do("POST", "/api/v1/nodes/sidecar/heartbeat", body)
	if err != nil {
		return err
	}
	return c.parseResponse(resp, nil)
}

type ProcessReport struct {
	ProcessDefID  uint            `json:"processDefId"`
	Status        string          `json:"status"`
	PID           int             `json:"pid"`
	ConfigVersion string          `json:"configVersion"`
	ProbeResult   json.RawMessage `json:"probeResult,omitempty"`
}

type Command struct {
	ID      uint            `json:"id"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

func (c *Client) AckCommand(cmdID uint, success bool, result string) error {
	body := map[string]any{
		"success": success,
		"result":  result,
	}

	resp, err := c.do("POST", fmt.Sprintf("/api/v1/nodes/sidecar/commands/%d/ack", cmdID), body)
	if err != nil {
		return err
	}
	return c.parseResponse(resp, nil)
}

type ConfigResponse struct {
	Content string
	Hash    string
}

func (c *Client) DownloadConfig(processName string, filename string) (*ConfigResponse, error) {
	path := fmt.Sprintf("/api/v1/nodes/sidecar/configs/%s", processName)
	if filename != "" {
		path += "?file=" + filename
	}
	resp, err := c.do("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return &ConfigResponse{
		Content: string(content),
		Hash:    resp.Header.Get("X-Config-Hash"),
	}, nil
}

func (c *Client) UploadLogs(logs []LogLine) error {
	if len(logs) == 0 {
		return nil
	}
	resp, err := c.do("POST", "/api/v1/nodes/sidecar/logs", map[string]any{
		"logs": logs,
	})
	if err != nil {
		return err
	}
	return c.parseResponse(resp, nil)
}

// Version is set by ldflags at build time.
var Version = "dev"
