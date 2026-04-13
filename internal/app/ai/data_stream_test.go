package ai

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func extractDataLines(t *testing.T, buf *bytes.Buffer) []string {
	t.Helper()
	var lines []string
	for _, line := range strings.Split(buf.String(), "\n") {
		if strings.HasPrefix(line, "data: ") {
			lines = append(lines, strings.TrimPrefix(line, "data: "))
		}
	}
	return lines
}

func TestUIMessageStreamEncoder_TextDelta(t *testing.T) {
	var buf bytes.Buffer
	enc := NewUIMessageStreamEncoder(&buf)

	_ = enc.Encode(Event{Type: EventTypeLLMStart, Sequence: 1})
	_ = enc.Encode(Event{Type: EventTypeContentDelta, Sequence: 2, Text: "Hello "})
	_ = enc.Encode(Event{Type: EventTypeContentDelta, Sequence: 3, Text: "world"})
	_ = enc.Encode(Event{Type: EventTypeDone, InputTokens: 10, OutputTokens: 20})
	_ = enc.Close()

	lines := extractDataLines(t, &buf)
	if len(lines) < 6 {
		t.Fatalf("expected at least 6 data lines, got %d: %v", len(lines), lines)
	}

	assertJSONField(t, lines[0], "type", "start")
	assertJSONField(t, lines[1], "type", "text-start")
	assertJSONField(t, lines[2], "type", "text-delta")
	assertJSONField(t, lines[2], "delta", "Hello ")
	assertJSONField(t, lines[3], "type", "text-delta")
	assertJSONField(t, lines[3], "delta", "world")
	assertJSONField(t, lines[4], "type", "text-end")
	assertJSONField(t, lines[5], "type", "finish")
	if lines[len(lines)-1] != "[DONE]" {
		t.Errorf("expected last line to be [DONE], got %s", lines[len(lines)-1])
	}
}

func TestUIMessageStreamEncoder_Reasoning(t *testing.T) {
	var buf bytes.Buffer
	enc := NewUIMessageStreamEncoder(&buf)

	_ = enc.Encode(Event{Type: EventTypeLLMStart, Sequence: 1})
	_ = enc.Encode(Event{Type: EventTypeThinkingDelta, Sequence: 2, Text: "think"})
	_ = enc.Encode(Event{Type: EventTypeThinkingDone, Sequence: 3})
	_ = enc.Encode(Event{Type: EventTypeDone})
	_ = enc.Close()

	lines := extractDataLines(t, &buf)
	assertJSONField(t, lines[1], "type", "reasoning-start")
	assertJSONField(t, lines[2], "type", "reasoning-delta")
	assertJSONField(t, lines[3], "type", "reasoning-end")
	assertJSONField(t, lines[4], "type", "finish")
}

func TestUIMessageStreamEncoder_ToolCallAndResult(t *testing.T) {
	var buf bytes.Buffer
	enc := NewUIMessageStreamEncoder(&buf)

	_ = enc.Encode(Event{Type: EventTypeLLMStart, Sequence: 1})
	_ = enc.Encode(Event{Type: EventTypeContentDelta, Sequence: 2, Text: "Let me check"})
	_ = enc.Encode(Event{Type: EventTypeToolCall, Sequence: 3, ToolCallID: "call_1", ToolName: "search", ToolArgs: json.RawMessage(`{"q":"x"}`)})
	_ = enc.Encode(Event{Type: EventTypeToolResult, Sequence: 4, ToolCallID: "call_1", ToolName: "search", ToolOutput: "result"})
	_ = enc.Encode(Event{Type: EventTypeDone})
	_ = enc.Close()

	lines := extractDataLines(t, &buf)
	assertJSONField(t, lines[1], "type", "text-start")
	assertJSONField(t, lines[2], "type", "text-delta")
	assertJSONField(t, lines[3], "type", "text-end")
	assertJSONField(t, lines[4], "type", "tool-input-available")
	assertJSONField(t, lines[4], "toolName", "search")
	assertJSONField(t, lines[5], "type", "tool-output-available")
	assertJSONField(t, lines[5], "output", "result")
}

func assertNestedJSONField(t *testing.T, line, nestedKey, key, expected string) {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(line), &m); err != nil {
		t.Fatalf("invalid json line %q: %v", line, err)
	}
	nested, ok := m[nestedKey].(map[string]any)
	if !ok {
		t.Fatalf("missing or invalid nested key %q in %s", nestedKey, line)
	}
	v, ok := nested[key]
	if !ok {
		t.Fatalf("missing key %q in nested %q of %s", key, nestedKey, line)
	}
	if got := v.(string); got != expected {
		t.Errorf("%s.%s: expected %q, got %q", nestedKey, key, expected, got)
	}
}

func TestUIMessageStreamEncoder_PlanAndSteps(t *testing.T) {
	var buf bytes.Buffer
	enc := NewUIMessageStreamEncoder(&buf)

	_ = enc.Encode(Event{Type: EventTypeLLMStart, Sequence: 1})
	_ = enc.Encode(Event{Type: EventTypePlan, Sequence: 2, Steps: []PlanStep{{Index: 1, Description: "step1"}}})
	_ = enc.Encode(Event{Type: EventTypeStepStart, Sequence: 3, StepIndex: 1, Description: "step1"})
	_ = enc.Encode(Event{Type: EventTypeStepDone, Sequence: 4, StepIndex: 1, DurationMs: 100})
	_ = enc.Encode(Event{Type: EventTypeDone})
	_ = enc.Close()

	lines := extractDataLines(t, &buf)
	assertJSONField(t, lines[1], "type", "data-plan")
	assertJSONField(t, lines[2], "type", "data-step")
	assertNestedJSONField(t, lines[2], "data", "state", "start")
	assertJSONField(t, lines[3], "type", "data-step")
	assertNestedJSONField(t, lines[3], "data", "state", "done")
}

func TestUIMessageStreamEncoder_Error(t *testing.T) {
	var buf bytes.Buffer
	enc := NewUIMessageStreamEncoder(&buf)

	_ = enc.Encode(Event{Type: EventTypeLLMStart, Sequence: 1})
	_ = enc.Encode(Event{Type: EventTypeError, Message: "boom"})
	_ = enc.Close()

	lines := extractDataLines(t, &buf)
	assertJSONField(t, lines[1], "type", "error")
	assertJSONField(t, lines[1], "errorText", "boom")
}

func TestUIMessageStreamEncoder_CancelledClosesBlocks(t *testing.T) {
	var buf bytes.Buffer
	enc := NewUIMessageStreamEncoder(&buf)

	_ = enc.Encode(Event{Type: EventTypeLLMStart, Sequence: 1})
	_ = enc.Encode(Event{Type: EventTypeContentDelta, Sequence: 2, Text: "partial"})
	_ = enc.Encode(Event{Type: EventTypeCancelled, Sequence: 3})
	_ = enc.Close()

	lines := extractDataLines(t, &buf)
	assertJSONField(t, lines[1], "type", "text-start")
	assertJSONField(t, lines[2], "type", "text-delta")
	assertJSONField(t, lines[3], "type", "text-end")
	assertJSONField(t, lines[4], "type", "finish")
	assertJSONField(t, lines[4], "finishReason", "other")
}

func assertJSONField(t *testing.T, line, key, expected string) {
	t.Helper()
	if line == "[DONE]" {
		if expected == "[DONE]" {
			return
		}
		t.Fatalf("unexpected [DONE] line looking for %s=%s", key, expected)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(line), &m); err != nil {
		t.Fatalf("invalid json line %q: %v", line, err)
	}
	v, ok := m[key]
	if !ok {
		t.Fatalf("missing key %q in %s", key, line)
	}
	if got := v.(string); got != expected {
		t.Errorf("%s: expected %q, got %q", key, expected, got)
	}
}
