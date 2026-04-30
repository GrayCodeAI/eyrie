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

// parseSSEStream reads an SSE stream and sends events to a channel.
// The goroutine closes the channel and body when done or context is cancelled.
func parseSSEStream(ctx context.Context, body io.ReadCloser, logger *slog.Logger) <-chan SSEEvent {
	ch := make(chan SSEEvent, 64)
	go func() {
		defer close(ch)
		defer body.Close()

		scanner := bufio.NewScanner(body)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

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
		}
	}()
	return ch
}

// --- Anthropic streaming ---

type anthropicStreamEvent struct {
	Type         string `json:"type"`
	Index        int    `json:"index,omitempty"`
	Delta        *struct {
		Type         string `json:"type"`
		Text         string `json:"text,omitempty"`
		PartialJSON  string `json:"partial_json,omitempty"`
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
	ch := make(chan EyrieStreamEvent, 64)
	go func() {
		defer close(ch)

		// Track current tool call being accumulated
		type toolAccum struct {
			id, name string
			jsonBuf  strings.Builder
		}
		var currentTool *toolAccum

		for {
			select {
			case <-ctx.Done():
				return
			case evt, ok := <-sseEvents:
				if !ok {
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
						var args map[string]interface{}
						_ = json.Unmarshal([]byte(currentTool.jsonBuf.String()), &args)
						emit(ctx, ch, EyrieStreamEvent{
							Type:     "tool_call",
							ToolCall: &ToolCall{ID: currentTool.id, Name: currentTool.name, Arguments: args},
						})
						currentTool = nil
					}

				case "message_stop":
					emit(ctx, ch, EyrieStreamEvent{Type: "done"})
					return

				case "message_delta":
					// Contains usage and stop_reason
					if ae.Delta != nil {
						var delta struct {
							StopReason string `json:"stop_reason"`
						}
						_ = json.Unmarshal([]byte(data), &struct {
							Delta *struct {
								StopReason string `json:"stop_reason"`
							} `json:"delta"`
							Usage *struct {
								OutputTokens int `json:"output_tokens"`
							} `json:"usage"`
						}{Delta: &delta})
					}
					// Usage is captured on message_stop via done event

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
}

// processOpenAIStream converts OpenAI SSE events to EyrieStreamEvents.
// Handles text deltas and tool call streaming by index.
func processOpenAIStream(ctx context.Context, sseEvents <-chan SSEEvent, logger *slog.Logger) <-chan EyrieStreamEvent {
	ch := make(chan EyrieStreamEvent, 64)
	go func() {
		defer close(ch)

		// Accumulate tool calls by index
		type toolAccum struct {
			id, name string
			argsBuf  strings.Builder
		}
		tools := make(map[int]*toolAccum)

		for {
			select {
			case <-ctx.Done():
				return
			case evt, ok := <-sseEvents:
				if !ok {
					// Emit accumulated tool calls
					for _, t := range tools {
						var args map[string]interface{}
						_ = json.Unmarshal([]byte(t.argsBuf.String()), &args)
						emit(ctx, ch, EyrieStreamEvent{
							Type:     "tool_call",
							ToolCall: &ToolCall{ID: t.id, Name: t.name, Arguments: args},
						})
					}
					emit(ctx, ch, EyrieStreamEvent{Type: "done"})
					return
				}
				data := strings.TrimSpace(evt.Data)
				if data == "" || data == "[DONE]" {
					// Emit accumulated tool calls
					for _, t := range tools {
						var args map[string]interface{}
						_ = json.Unmarshal([]byte(t.argsBuf.String()), &args)
						emit(ctx, ch, EyrieStreamEvent{
							Type:     "tool_call",
							ToolCall: &ToolCall{ID: t.id, Name: t.name, Arguments: args},
						})
					}
					emit(ctx, ch, EyrieStreamEvent{Type: "done"})
					return
				}

				var oe openaiStreamPayload
				if err := json.Unmarshal([]byte(data), &oe); err != nil {
					logger.Debug("failed to parse openai event", "error", err)
					continue
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
					// Emit accumulated tool calls before done
					for _, t := range tools {
						var args map[string]interface{}
						_ = json.Unmarshal([]byte(t.argsBuf.String()), &args)
						emit(ctx, ch, EyrieStreamEvent{
							Type:     "tool_call",
							ToolCall: &ToolCall{ID: t.id, Name: t.name, Arguments: args},
						})
					}
					emit(ctx, ch, EyrieStreamEvent{Type: "done"})
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

// parseErrorBody reads and parses an error response body (capped at 4KB).
func parseErrorBody(body io.ReadCloser) string {
	defer body.Close()
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
