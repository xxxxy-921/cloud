package sidecar

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

// SSEEvent represents a parsed Server-Sent Event.
type SSEEvent struct {
	Event string
	Data  string
}

// SSEClient connects to an SSE endpoint and delivers events.
type SSEClient struct {
	url        string
	token      string
	httpClient *http.Client
}

func NewSSEClient(serverURL, token string) *SSEClient {
	return &SSEClient{
		url:   strings.TrimRight(serverURL, "/") + "/api/v1/nodes/sidecar/stream",
		token: token,
		httpClient: &http.Client{
			// No timeout — SSE connections are long-lived
		},
	}
}

// Connect establishes an SSE connection and sends events to the channel.
// Blocks until the connection is closed or stopCh is closed.
// Returns an error if the connection fails.
func (s *SSEClient) Connect(eventCh chan<- SSEEvent, stopCh <-chan struct{}) error {
	req, err := http.NewRequest("GET", s.url, nil)
	if err != nil {
		return fmt.Errorf("create SSE request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("SSE connect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("SSE connect failed: HTTP %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	var currentEvent SSEEvent

	// Read events in a goroutine so we can check stopCh
	lineCh := make(chan string, 64)
	errCh := make(chan error, 1)
	go func() {
		for scanner.Scan() {
			lineCh <- scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			errCh <- err
		} else {
			errCh <- fmt.Errorf("SSE connection closed by server")
		}
	}()

	for {
		select {
		case <-stopCh:
			return nil
		case err := <-errCh:
			return err
		case line := <-lineCh:
			if line == "" {
				// Empty line = event boundary
				if currentEvent.Data != "" {
					select {
					case eventCh <- currentEvent:
					case <-stopCh:
						return nil
					}
				}
				currentEvent = SSEEvent{}
				continue
			}

			if strings.HasPrefix(line, "event:") {
				currentEvent.Event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			} else if strings.HasPrefix(line, "data:") {
				currentEvent.Data = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			}
			// Ignore other fields (id, retry, comments)
		}
	}
}

// ParseCommandEvent parses an SSE event data as a Command.
func ParseCommandEvent(data string) (Command, error) {
	var cmd Command
	if err := json.Unmarshal([]byte(data), &cmd); err != nil {
		return cmd, fmt.Errorf("parse command event: %w", err)
	}
	return cmd, nil
}

// ReconnectJitter returns a random duration between 1-5 seconds.
func ReconnectJitter() time.Duration {
	return time.Duration(1000+rand.Intn(4000)) * time.Millisecond
}

// BackoffDuration returns an exponential backoff duration capped at maxBackoff.
func BackoffDuration(attempt int, maxBackoff time.Duration) time.Duration {
	d := time.Duration(1<<uint(attempt)) * time.Second
	if d > maxBackoff {
		return maxBackoff
	}
	return d
}

func init() {
	// Seed random for jitter (Go 1.20+ auto-seeds, but be explicit)
	slog.Debug("SSE client initialized")
}
