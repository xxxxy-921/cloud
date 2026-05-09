package llm

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestOpenAIClientBuildRequestOmitsTemperatureForGPT54(t *testing.T) {
	temp := float32(0.3)
	client := &openaiClient{}

	for _, model := range []string{"gpt-5.4", "gpt-5.4-2026-04-24"} {
		t.Run(model, func(t *testing.T) {
			req := client.buildRequest(ChatRequest{
				Model: model,
				Messages: []Message{
					{Role: RoleUser, Content: "ping"},
				},
				Temperature: &temp,
			})

			payload, err := json.Marshal(req)
			if err != nil {
				t.Fatalf("marshal request: %v", err)
			}
			if jsonContainsKey(payload, "temperature") {
				t.Fatalf("%s request should omit temperature, got %s", model, payload)
			}
		})
	}
}

func TestOpenAIClientBuildRequestKeepsTemperatureForOtherModels(t *testing.T) {
	temp := float32(0.3)
	client := &openaiClient{}

	req := client.buildRequest(ChatRequest{
		Model: "gpt-4o",
		Messages: []Message{
			{Role: RoleUser, Content: "ping"},
		},
		Temperature: &temp,
	})

	if req.Temperature != temp {
		t.Fatalf("temperature = %v, want %v", req.Temperature, temp)
	}
}

func TestOpenAIClientBuildRequestSerializesAssistantToolCallsWithStringContent(t *testing.T) {
	client := &openaiClient{}

	req := client.buildRequest(ChatRequest{
		Model: "gpt-4o",
		Messages: []Message{
			{
				Role:    RoleAssistant,
				Content: " ",
				ToolCalls: []ToolCall{{
					ID:        "call_1",
					Name:      "itsm.service_match",
					Arguments: `{"query":"VPN"}`,
				}},
			},
		},
	})

	payload, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	if !jsonContainsString(payload, `"content":" "`) {
		t.Fatalf("expected assistant tool call message to serialize string content, got %s", payload)
	}
	if !jsonContainsString(payload, `"tool_calls"`) {
		t.Fatalf("expected tool_calls to be preserved, got %s", payload)
	}
}

func jsonContainsKey(payload []byte, key string) bool {
	var data map[string]any
	if err := json.Unmarshal(payload, &data); err != nil {
		return false
	}
	_, ok := data[key]
	return ok
}

func jsonContainsString(payload []byte, token string) bool {
	return string(payload) != "" && strings.Contains(string(payload), token)
}
