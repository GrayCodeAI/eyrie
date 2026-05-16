package client

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
)

// SSEEvent represents a single Server-Sent Event.
type SSEEvent struct {
	Event string
	Data  string
}

// SSE stream constants.
const (
	sseChannelBuffer    = 128
	sseScannerInitBuf   = 64 * 1024
	sseScannerMaxBuf    = 2 * 1024 * 1024
	streamChannelBuffer = 128
)

// parseSSEStream reads an SSE stream and sends events to a channel.
// The goroutine closes the channel and body when done or context is cancelled.
// Scanner errors are emitted as SSEEvent with Event="error" so callers can detect truncation.
func parseSSEStream(ctx context.Context, body io.ReadCloser, logger *slog.Logger) <-chan SSEEvent {
	ch := make(chan SSEEvent, sseChannelBuffer)
	go func() {
		defer close(ch)
		defer func() { _ = body.Close() }()

		scanner := bufio.NewScanner(body)
		scanner.Buffer(make([]byte, 0, sseScannerInitBuf), sseScannerMaxBuf)

		var event, data strings.Builder
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
			}

			line := scanner.Text()
			if line == "" {
				if data.Len() > 0 {
					select {
					case ch <- SSEEvent{Event: strings.TrimSpace(event.String()), Data: strings.TrimSpace(data.String())}:
					case <-ctx.Done():
						return
					}
				}
				event.Reset()
				data.Reset()
				continue
			}
			if strings.HasPrefix(line, "event:") {
				event.WriteString(strings.TrimPrefix(line, "event:"))
			} else if strings.HasPrefix(line, "data:") {
				if data.Len() > 0 {
					data.WriteByte('\n')
				}
				data.WriteString(strings.TrimPrefix(line, "data:"))
			}
		}
		if err := scanner.Err(); err != nil {
			logger.Warn("SSE stream read error", "error", err)
			select {
			case ch <- SSEEvent{Event: "error", Data: fmt.Sprintf("stream read error: %v", err)}:
			case <-ctx.Done():
			}
		}
	}()
	return ch
}

// --- Anthropic streaming ---

type anthropicStreamEvent struct {
	Type  string `json:"type"`
	Index int    `json:"index,omitempty"`
	Delta *struct {
		Type        string `json:"type"`
		Text        string `json:"text,omitempty"`
		PartialJSON string `json:"partial_json,omitempty"`
	} `json:"delta,omitempty"`
	ContentBlock *struct {
		Type string `json:"type"`
		ID   string `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Text string `json:"text,omitempty"`
	} `json:"content_block,omitempty"`
}

// processAnthropicStream converts Anthropic SSE events to EyrieStreamEvents.
// Handles text, tool_use (with input_json_delta), and thinking blocks.
func processAnthropicStream(ctx context.Context, sseEvents <-chan SSEEvent, logger *slog.Logger) <-chan EyrieStreamEvent {
	ch := make(chan EyrieStreamEvent, streamChannelBuffer)
	go func() {
		defer close(ch)

		// Track current tool call being accumulated
		type toolAccum struct {
			id, name string
			jsonBuf  strings.Builder
		}
		var currentTool *toolAccum
		var stopReason string

		for {
			select {
			case <-ctx.Done():
				return
			case evt, ok := <-sseEvents:
				if !ok {
					// Stream closed — emit partial tool call as error if incomplete
					if currentTool != nil {
						emit(ctx, ch, EyrieStreamEvent{
							Type:  "error",
							Error: fmt.Sprintf("stream closed with incomplete tool call: %s", currentTool.name),
						})
						currentTool = nil
					}
					return
				}
				// Propagate SSE-level errors
				if evt.Event == "error" {
					emit(ctx, ch, EyrieStreamEvent{Type: "error", Error: evt.Data})
					return
				}
				data := strings.TrimSpace(evt.Data)
				if data == "" || data == "[DONE]" {
					continue
				}
				var ae anthropicStreamEvent
				if err := json.Unmarshal([]byte(data), &ae); err != nil {
					logger.Debug("failed to parse anthropic event", "error", err)
					continue
				}

				switch ae.Type {
				case "content_block_start":
					if ae.ContentBlock != nil {
						switch ae.ContentBlock.Type {
						case "tool_use":
							currentTool = &toolAccum{id: ae.ContentBlock.ID, name: ae.ContentBlock.Name}
						case "thinking":
							// thinking block started, deltas will follow
						}
					}

				case "content_block_delta":
					if ae.Delta == nil {
						continue
					}
					switch ae.Delta.Type {
					case "text_delta":
						if ae.Delta.Text != "" {
							emit(ctx, ch, EyrieStreamEvent{Type: "content", Content: ae.Delta.Text})
						}
					case "input_json_delta":
						if currentTool != nil && ae.Delta.PartialJSON != "" {
							currentTool.jsonBuf.WriteString(ae.Delta.PartialJSON)
						}
					case "thinking_delta":
						if ae.Delta.Text != "" {
							emit(ctx, ch, EyrieStreamEvent{Type: "thinking", Thinking: ae.Delta.Text})
						}
					}

				case "content_block_stop":
					if currentTool != nil {
						rawJSON := currentTool.jsonBuf.String()
						var args map[string]interface{}
						if err := json.Unmarshal([]byte(rawJSON), &args); err != nil {
							logger.Warn("invalid tool call JSON accumulated", "tool", currentTool.name, "error", err)
							emit(ctx, ch, EyrieStreamEvent{
								Type:  "error",
								Error: fmt.Sprintf("invalid tool call JSON for %s: %v", currentTool.name, err),
							})
						} else {
							emit(ctx, ch, EyrieStreamEvent{
								Type:     "tool_call",
								ToolCall: &ToolCall{ID: currentTool.id, Name: currentTool.name, Arguments: args},
							})
						}
						currentTool = nil
					}

				case "message_stop":
					emit(ctx, ch, EyrieStreamEvent{Type: "done", StopReason: stopReason})
					return

				case "message_delta":
					// Contains usage (output_tokens) and stop_reason
					var md struct {
						Delta *struct {
							StopReason string `json:"stop_reason"`
						} `json:"delta"`
						Usage *struct {
							OutputTokens int `json:"output_tokens"`
						} `json:"usage"`
					}
					_ = json.Unmarshal([]byte(data), &md)
					if md.Delta != nil && md.Delta.StopReason != "" {
						stopReason = md.Delta.StopReason
					}
					if md.Usage != nil && md.Usage.OutputTokens > 0 {
						emit(ctx, ch, EyrieStreamEvent{
							Type: "usage",
							Usage: &EyrieUsage{
								CompletionTokens: md.Usage.OutputTokens,
							},
						})
					}

				case "message_start":
					// Contains input token count
					var ms struct {
						Message struct {
							Usage struct {
								InputTokens  int `json:"input_tokens"`
								OutputTokens int `json:"output_tokens"`
							} `json:"usage"`
						} `json:"message"`
					}
					_ = json.Unmarshal([]byte(data), &ms)
					if ms.Message.Usage.InputTokens > 0 {
						emit(ctx, ch, EyrieStreamEvent{
							Type: "usage",
							Usage: &EyrieUsage{
								PromptTokens: ms.Message.Usage.InputTokens,
							},
						})
					}

				case "error":
					emit(ctx, ch, EyrieStreamEvent{Type: "error", Error: data})
					return
				}
			}
		}
	}()
	return ch
}

// --- OpenAI streaming ---

type openaiStreamChoice struct {
	Delta struct {
		Content   string `json:"content,omitempty"`
		Role      string `json:"role,omitempty"`
		ToolCalls []struct {
			Index    int    `json:"index"`
			ID       string `json:"id,omitempty"`
			Function struct {
				Name      string `json:"name,omitempty"`
				Arguments string `json:"arguments,omitempty"`
			} `json:"function"`
		} `json:"tool_calls,omitempty"`
	} `json:"delta"`
	FinishReason *string `json:"finish_reason"`
}

type openaiStreamPayload struct {
	Choices []openaiStreamChoice `json:"choices"`
	Usage   *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

// processOpenAIStream converts OpenAI SSE events to EyrieStreamEvents.
// Handles text deltas and tool call streaming by index.
func processOpenAIStream(ctx context.Context, sseEvents <-chan SSEEvent, logger *slog.Logger) <-chan EyrieStreamEvent {
	ch := make(chan EyrieStreamEvent, streamChannelBuffer)
	go func() {
		defer close(ch)

		// Accumulate tool calls by index
		type toolAccum struct {
			id, name string
			argsBuf  strings.Builder
		}
		tools := make(map[int]*toolAccum)
		toolsEmitted := false

		emitTools := func() {
			if toolsEmitted {
				return
			}
			toolsEmitted = true
			for _, t := range tools {
				var args map[string]interface{}
				_ = json.Unmarshal([]byte(t.argsBuf.String()), &args)
				emit(ctx, ch, EyrieStreamEvent{
					Type:     "tool_call",
					ToolCall: &ToolCall{ID: t.id, Name: t.name, Arguments: args},
				})
			}
		}

		for {
			select {
			case <-ctx.Done():
				return
			case evt, ok := <-sseEvents:
				if !ok {
					emitTools()
					emit(ctx, ch, EyrieStreamEvent{Type: "done"})
					return
				}
				// Propagate SSE-level errors
				if evt.Event == "error" {
					emit(ctx, ch, EyrieStreamEvent{Type: "error", Error: evt.Data})
					return
				}
				data := strings.TrimSpace(evt.Data)
				if data == "" || data == "[DONE]" {
					emitTools()
					emit(ctx, ch, EyrieStreamEvent{Type: "done"})
					return
				}

				var oe openaiStreamPayload
				if err := json.Unmarshal([]byte(data), &oe); err != nil {
					logger.Debug("failed to parse openai event", "error", err)
					continue
				}

				// Emit usage if present (final chunk with stream_options.include_usage)
				if oe.Usage != nil {
					emit(ctx, ch, EyrieStreamEvent{
						Type: "usage",
						Usage: &EyrieUsage{
							PromptTokens:     oe.Usage.PromptTokens,
							CompletionTokens: oe.Usage.CompletionTokens,
							TotalTokens:      oe.Usage.TotalTokens,
						},
					})
				}

				if len(oe.Choices) == 0 {
					continue
				}
				choice := oe.Choices[0]

				// Text content
				if choice.Delta.Content != "" {
					emit(ctx, ch, EyrieStreamEvent{Type: "content", Content: choice.Delta.Content})
				}

				// Tool calls by index
				for _, tc := range choice.Delta.ToolCalls {
					t, ok := tools[tc.Index]
					if !ok {
						t = &toolAccum{}
						tools[tc.Index] = t
					}
					if tc.ID != "" {
						t.id = tc.ID
					}
					if tc.Function.Name != "" {
						t.name = tc.Function.Name
					}
					if tc.Function.Arguments != "" {
						t.argsBuf.WriteString(tc.Function.Arguments)
					}
				}

				if choice.FinishReason != nil {
					emitTools()
					emit(ctx, ch, EyrieStreamEvent{Type: "done", StopReason: *choice.FinishReason})
					return
				}
			}
		}
	}()
	return ch
}

func emit(ctx context.Context, ch chan<- EyrieStreamEvent, evt EyrieStreamEvent) {
	select {
	case ch <- evt:
	case <-ctx.Done():
	}
}

// ParseInlineToolCalls detects and extracts tool calls embedded in text content.
// Some providers (e.g., canopywave/kimi) return tool calls in a text format:
// <|tool_calls_section_begin|> <|tool_call_begin|> functions.ToolName:0 <|tool_call_argument_begin|> {"arg":"val"} <|tool_call_end|> <|tool_calls_section_end|>
func ParseInlineToolCalls(text string) (cleanText string, toolCalls []ToolCall) {
	const marker = "<|tool_calls_section_begin|>"
	idx := strings.Index(text, marker)
	if idx < 0 {
		return text, nil
	}

	cleanText = strings.TrimSpace(text[:idx])
	section := text[idx:]

	// Extract individual tool calls
	calls := strings.Split(section, "<|tool_call_begin|>")
	for _, call := range calls[1:] { // skip first empty part
		endIdx := strings.Index(call, "<|tool_call_end|>")
		if endIdx < 0 {
			continue
		}
		call = call[:endIdx]

		// Parse function name: "functions.ToolName:0"
		nameStart := strings.TrimSpace(call)
		argStart := strings.Index(nameStart, "<|tool_call_argument_begin|>")
		if argStart < 0 {
			continue
		}
		funcLine := strings.TrimSpace(nameStart[:argStart])
		argJSON := strings.TrimSpace(nameStart[argStart+len("<|tool_call_argument_begin|>"):])

		// Extract tool name from "functions.ToolName:0"
		toolName := funcLine
		if dotIdx := strings.Index(funcLine, "."); dotIdx >= 0 {
			toolName = funcLine[dotIdx+1:]
		}
		if colonIdx := strings.Index(toolName, ":"); colonIdx >= 0 {
			toolName = toolName[:colonIdx]
		}

		// Parse arguments JSON
		var args map[string]interface{}
		if err := json.Unmarshal([]byte(argJSON), &args); err != nil {
			args = map[string]interface{}{"_raw": argJSON}
		}

		toolCalls = append(toolCalls, ToolCall{
			ID:        fmt.Sprintf("inline_%s_%d", toolName, len(toolCalls)),
			Name:      toolName,
			Arguments: args,
		})
	}

	return cleanText, toolCalls
}

// parseErrorBody reads and parses an error response body (capped at 4KB).
func parseErrorBody(body io.ReadCloser) string {
	defer func() { _ = body.Close() }()
	data, err := io.ReadAll(io.LimitReader(body, 4096))
	if err != nil {
		return "failed to read error body"
	}
	var errResp struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
	}
	if json.Unmarshal(data, &errResp) == nil && errResp.Error.Message != "" {
		return fmt.Sprintf("%s: %s", errResp.Error.Type, errResp.Error.Message)
	}
	return string(data)
}
