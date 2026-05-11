package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"

	"metis/internal/app"
	"metis/internal/llm"
)

// ReactExecutor implements the ReAct (Reason + Act) loop.
type ReactExecutor struct {
	llmClient    llm.Client
	toolExecutor ToolExecutor
}

func NewReactExecutor(client llm.Client, toolExec ToolExecutor) *ReactExecutor {
	return &ReactExecutor{
		llmClient:    client,
		toolExecutor: toolExec,
	}
}

func (e *ReactExecutor) Execute(ctx context.Context, req ExecuteRequest) (<-chan Event, error) {
	ch := make(chan Event, 64)

	go func() {
		defer close(ch)

		var seq atomic.Int32
		emit := func(evt Event) {
			evt.Sequence = int(seq.Add(1))
			select {
			case ch <- evt:
				return
			default:
			}
			select {
			case ch <- evt:
			case <-ctx.Done():
			}
		}

		messages := buildLLMMessages(req)
		maxTurns := req.MaxTurns
		if maxTurns <= 0 {
			maxTurns = 10
		}

		tools := buildLLMTools(req.Tools)
		var totalInput, totalOutput int
		var currentITSMServiceEngine string
		silentRetryCount := 0

		for turn := 1; turn <= maxTurns; turn++ {
			select {
			case <-ctx.Done():
				emit(stoppedEvent(ctx.Err(), "LLM stream"))
				return
			default:
			}

			emit(Event{Type: EventTypeLLMStart, Turn: turn, Model: req.AgentConfig.ModelName})
			slog.Info("react executor: starting LLM turn", "turn", turn, "maxTurns", maxTurns, "model", req.AgentConfig.ModelName)

			chatReq := llm.ChatRequest{
				Model:       req.AgentConfig.ModelName,
				Messages:    messages,
				Tools:       tools,
				MaxTokens:   req.AgentConfig.MaxTokens,
				Temperature: req.AgentConfig.Temperature,
			}

			streamCh, turnCtx, turnCancel, err := openChatStreamWithTimeout(ctx, e.llmClient, chatReq)
			if err != nil {
				slog.Warn("react executor: failed to open LLM stream", "turn", turn, "error", err)
				if ctx.Err() != nil {
					emit(stoppedEvent(ctx.Err(), "LLM stream"))
				} else {
					emit(Event{Type: EventTypeError, Message: llmCallErrorMessage("LLM stream", err)})
				}
				return
			}
			slog.Info("react executor: LLM stream established", "turn", turn, "model", req.AgentConfig.ModelName)

			var assistantContent string
			var toolCalls []llm.ToolCall
			var usage llm.Usage
			sawAnyEvent := false
			sawThinking := false

			streamDone := false
			for !streamDone {
				select {
				case evt, ok := <-streamCh:
					if !ok {
						if ctx.Err() != nil {
							turnCancel()
							emit(stoppedEvent(ctx.Err(), "LLM stream"))
							return
						}
						if turnCtx.Err() != nil {
							turnCancel()
							slog.Warn("react executor: LLM stream closed after timeout", "turn", turn, "error", turnCtx.Err())
							emit(Event{Type: EventTypeError, Message: llmCallErrorMessage("LLM stream", turnCtx.Err())})
							return
						}
						streamDone = true
						break
					}
					sawAnyEvent = true
					switch evt.Type {
					case "content_delta":
						assistantContent += evt.Content
						emit(Event{Type: EventTypeContentDelta, Text: evt.Content})
					case "thinking_delta":
						sawThinking = true
						emit(Event{Type: EventTypeThinkingDelta, Text: evt.Content})
					case "tool_call":
						if evt.ToolCall != nil {
							toolCalls = append(toolCalls, *evt.ToolCall)
							emit(Event{
								Type:       EventTypeToolCall,
								ToolCallID: evt.ToolCall.ID,
								ToolName:   evt.ToolCall.Name,
								ToolArgs:   json.RawMessage(evt.ToolCall.Arguments),
							})
						}
					case "done":
						if evt.Usage != nil {
							usage = *evt.Usage
						}
					case "error":
						turnCancel()
						slog.Warn("react executor: LLM stream returned error", "turn", turn, "error", evt.Error)
						emit(Event{Type: EventTypeError, Message: evt.Error})
						return
					}
				case <-ctx.Done():
					turnCancel()
					emit(stoppedEvent(ctx.Err(), "LLM stream"))
					return
				case <-turnCtx.Done():
					turnCancel()
					slog.Warn("react executor: LLM stream timed out", "turn", turn, "error", turnCtx.Err())
					emit(Event{Type: EventTypeError, Message: llmCallErrorMessage("LLM stream", turnCtx.Err())})
					return
				}
			}
			turnCancel()
			if sawThinking {
				emit(Event{Type: EventTypeThinkingDone})
			}
			if ctx.Err() != nil {
				emit(stoppedEvent(ctx.Err(), "LLM stream"))
				return
			}
			slog.Info("react executor: completed LLM turn", "turn", turn, "toolCalls", len(toolCalls))

			select {
			case <-ctx.Done():
				emit(stoppedEvent(ctx.Err(), "LLM stream"))
				return
			default:
			}
			if usage == (llm.Usage{}) && len(toolCalls) == 0 && assistantContent == "" {
				// An empty closed stream usually means the underlying provider ended
				// because the context deadline fired before it could emit an event.
				if ctx.Err() != nil {
					emit(stoppedEvent(ctx.Err(), "LLM stream"))
					return
				}
			}

			totalInput += usage.InputTokens
			totalOutput += usage.OutputTokens

			if sawAnyEvent && len(toolCalls) == 0 && strings.TrimSpace(assistantContent) == "" {
				resp, err := chatWithTimeout(ctx, e.llmClient, chatReq)
				if err == nil && resp != nil {
					if strings.TrimSpace(resp.Content) != "" || len(resp.ToolCalls) > 0 {
						assistantContent = resp.Content
						toolCalls = append(toolCalls, resp.ToolCalls...)
						totalInput += resp.Usage.InputTokens
						totalOutput += resp.Usage.OutputTokens
						if assistantContent != "" {
							emit(Event{Type: EventTypeContentDelta, Text: assistantContent})
						}
						for _, tc := range resp.ToolCalls {
							emit(Event{
								Type:       EventTypeToolCall,
								ToolCallID: tc.ID,
								ToolName:   tc.Name,
								ToolArgs:   json.RawMessage(tc.Arguments),
							})
						}
					}
				} else if err != nil && !errors.Is(err, llm.ErrNotSupported) {
					slog.Warn("react executor: non-streaming fallback failed", "turn", turn, "error", err)
				}
			}

			if sawAnyEvent && len(toolCalls) == 0 && strings.TrimSpace(assistantContent) == "" {
				if silentRetryCount == 0 && turn < maxTurns {
					silentRetryCount++
					messages = append(messages, llm.Message{
						Role:    llm.RoleUser,
						Content: "请继续处理，并在这一轮给出一条简短的中文答复；如果需要，也可以调用合适的工具。",
					})
					continue
				}
				emit(Event{Type: EventTypeError, Message: emptyDisplayOutputMessage})
				return
			}

			// If no tool calls, we're done
			if len(toolCalls) == 0 {
				// Check for memory extraction in the content
				e.extractMemoryUpdates(assistantContent, emit)

				emit(Event{
					Type:         EventTypeDone,
					TotalTurns:   turn,
					InputTokens:  totalInput,
					OutputTokens: totalOutput,
				})
				return
			}

			// Add assistant message with tool calls
			messages = append(messages, llm.Message{
				Role:      llm.RoleAssistant,
				Content:   assistantContent,
				ToolCalls: toolCalls,
			})

			// Execute each tool call
			for _, tc := range toolCalls {
				if tc.Name == "itsm.draft_prepare" && currentITSMServiceEngine == "smart" {
					emit(makeITSMDraftLoadingSurface(tc.ID))
				}

				start := time.Now()
				toolCtx := context.WithValue(ctx, app.UserMessageKey, latestLLMUserMessage(messages))
				result, execErr := e.toolExecutor.ExecuteTool(toolCtx, ToolCall{
					ID:   tc.ID,
					Name: tc.Name,
					Args: json.RawMessage(tc.Arguments),
				})
				durationMs := int(time.Since(start).Milliseconds())

				output := result.Output
				isError := result.IsError
				if execErr != nil {
					output = fmt.Sprintf("Error: %v", execErr)
					isError = true
				}

				emit(Event{
					Type:        EventTypeToolResult,
					ToolCallID:  tc.ID,
					ToolOutput:  output,
					DurationMs:  durationMs,
					ToolIsError: isError,
				})
				if !isError && tc.Name == "itsm.service_load" {
					currentITSMServiceEngine = parseITSMServiceEngine(output)
				}
				if !isError && tc.Name == "itsm.draft_prepare" {
					if surface, ok := makeITSMDraftReadySurface(tc.ID, output); ok {
						emit(surface)
					}
				}

				// Add tool result to messages
				messages = append(messages, llm.Message{
					Role:       llm.RoleTool,
					Content:    output,
					ToolCallID: tc.ID,
				})
			}
			// Continue loop — next turn will send messages with tool results back to LLM
		}

		emit(Event{
			Type:    EventTypeError,
			Message: fmt.Sprintf("max turns (%d) exceeded", maxTurns),
		})
	}()

	return ch, nil
}

func latestLLMUserMessage(messages []llm.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == llm.RoleUser || messages[i].Role == MessageRoleUser {
			return messages[i].Content
		}
	}
	return ""
}

// buildLLMMessages converts ExecuteRequest messages to llm.Message format,
// prepending the system prompt.
func buildLLMMessages(req ExecuteRequest) []llm.Message {
	msgs := make([]llm.Message, 0, len(req.Messages)+1)
	if req.SystemPrompt != "" {
		msgs = append(msgs, llm.Message{Role: llm.RoleSystem, Content: req.SystemPrompt})
	}
	for _, m := range req.Messages {
		content := normalizeLLMMessageContent(m)
		msgs = append(msgs, llm.Message{
			Role:       m.Role,
			Content:    content,
			Images:     m.Images,
			ToolCalls:  m.ToolCalls,
			ToolCallID: m.ToolCallID,
		})
	}
	return msgs
}

func normalizeLLMMessageContent(m ExecuteMessage) string {
	if m.Role == llm.RoleAssistant && strings.TrimSpace(m.Content) == "" {
		return " "
	}
	return m.Content
}

// buildLLMTools converts ToolDefinitions to llm.ToolDef format.
func buildLLMTools(tools []ToolDefinition) []llm.ToolDef {
	if len(tools) == 0 {
		return nil
	}
	defs := make([]llm.ToolDef, len(tools))
	for i, t := range tools {
		var params any
		if len(t.Parameters) > 0 {
			_ = json.Unmarshal(t.Parameters, &params)
		}
		defs[i] = llm.ToolDef{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  params,
		}
	}
	return defs
}

// extractMemoryUpdates looks for memory extraction markers in content.
// In practice, the system prompt instructs the LLM to output <memory> tags.
func (e *ReactExecutor) extractMemoryUpdates(content string, emit func(Event)) {
	// Simple pattern: LLM is instructed to emit JSON memory blocks
	// This is a placeholder — full implementation will use structured output
	// or a dedicated memory extraction prompt.
	_ = content
	_ = emit
}
