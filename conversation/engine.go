package conversation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/GrayCodeAI/eyrie/client"
	"github.com/GrayCodeAI/eyrie/storage"
	"github.com/google/uuid"
)

type Engine struct {
	store    storage.Store
	provider client.Provider
}

func New(store storage.Store, provider client.Provider) *Engine {
	return &Engine{store: store, provider: provider}
}

type PromptOpts struct {
	Model        string
	SystemPrompt string
	Tools        []client.EyrieTool
	MaxTokens    int
	Temperature  *float64
}

type Event struct {
	Type    string `json:"type"`
	Content string `json:"content,omitempty"`
	NodeID  string `json:"node_id,omitempty"`
	Error   string `json:"error,omitempty"`
}

const (
	EventDelta     = "delta"
	EventDone      = "done"
	EventNodeSaved = "node_saved"
	EventError     = "error"
)

func (e *Engine) Prompt(ctx context.Context, message string, opts PromptOpts) (<-chan Event, error) {
	rootID := uuid.New().String()
	rootNode := &storage.Node{
		ID:           rootID,
		RootID:       rootID,
		Sequence:     0,
		NodeType:     storage.NodeTypeUser,
		Content:      message,
		Status:       "completed",
		Title:        generateTitle(message),
		SystemPrompt: opts.SystemPrompt,
		CreatedAt:    time.Now(),
	}
	if err := e.store.CreateNode(ctx, rootNode); err != nil {
		return nil, fmt.Errorf("conversation: create root: %w", err)
	}

	messages := []client.EyrieMessage{{Role: "user", Content: message}}
	return e.streamAndSave(ctx, rootNode, messages, opts)
}

func (e *Engine) PromptFrom(ctx context.Context, parentNodeID, message string, opts PromptOpts) (<-chan Event, error) {
	ancestors, err := e.store.GetAncestors(ctx, parentNodeID)
	if err != nil {
		return nil, fmt.Errorf("conversation: get ancestors: %w", err)
	}
	if len(ancestors) == 0 {
		return nil, fmt.Errorf("conversation: node not found: %s", parentNodeID)
	}

	root := ancestors[0]
	lastNode := ancestors[len(ancestors)-1]

	if opts.Model == "" {
		opts.Model = root.Model
	}
	if opts.SystemPrompt == "" {
		opts.SystemPrompt = root.SystemPrompt
	}

	userNode := &storage.Node{
		ID:       uuid.New().String(),
		ParentID: parentNodeID,
		RootID:   root.ID,
		Sequence: lastNode.Sequence + 1,
		NodeType: storage.NodeTypeUser,
		Content:  message,
		Status:   "completed",
	}
	if err := e.store.CreateNode(ctx, userNode); err != nil {
		return nil, fmt.Errorf("conversation: create user node: %w", err)
	}

	messages := buildMessages(ancestors)
	messages = append(messages, client.EyrieMessage{Role: "user", Content: message})

	return e.streamAndSave(ctx, userNode, messages, opts)
}

func (e *Engine) ResolveNode(ctx context.Context, ref string) (*storage.Node, error) {
	node, err := e.store.GetNode(ctx, ref)
	if err == nil {
		return node, nil
	}
	node, err = e.store.GetNodeByPrefix(ctx, ref)
	if err == nil {
		return node, nil
	}
	return e.store.GetNodeByAlias(ctx, ref)
}

func (e *Engine) ListConversations(ctx context.Context) ([]*storage.Node, error) {
	return e.store.ListRootNodes(ctx)
}

func (e *Engine) GetSubtree(ctx context.Context, id string) ([]*storage.Node, error) {
	return e.store.GetSubtree(ctx, id)
}

func (e *Engine) DeleteNode(ctx context.Context, id string) error {
	return e.store.DeleteNode(ctx, id)
}

const defaultGroupBudgetMultiplier = 4

func (e *Engine) streamAndSave(ctx context.Context, parentNode *storage.Node, messages []client.EyrieMessage, opts PromptOpts) (<-chan Event, error) {
	maxTokens := opts.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	chatOpts := client.ChatOptions{
		Model:       opts.Model,
		System:      opts.SystemPrompt,
		MaxTokens:   maxTokens,
		Temperature: opts.Temperature,
		Tools:       opts.Tools,
	}

	sr, err := e.provider.StreamChat(ctx, messages, chatOpts)
	if err != nil {
		return nil, fmt.Errorf("conversation: stream: %w", err)
	}

	groupBudget := maxTokens * defaultGroupBudgetMultiplier
	events := make(chan Event, 100)

	go func() {
		defer close(events)

		var (
			groupID         string
			accumulatedText string
			cumulativeOut   int
			currentParent   = parentNode
			currentStream   = sr.Events
		)

		for {
			var fullText string
			var usage *client.EyrieUsage
			var stopReason string
			start := time.Now()

			for evt := range currentStream {
				switch evt.Type {
				case "content":
					fullText += evt.Content
					events <- Event{Type: EventDelta, Content: evt.Content}
				case "done":
					stopReason = evt.StopReason
					usage = evt.Usage
				case "error":
					events <- Event{Type: EventError, Error: evt.Error}
					return
				}
			}

			if fullText == "" && stopReason == "" {
				if accumulatedText != "" {
					events <- Event{Type: EventNodeSaved, NodeID: currentParent.ID}
				}
				return
			}

			accumulatedText += fullText
			if usage != nil {
				cumulativeOut += usage.CompletionTokens
			}

			shouldContinue := stopReason == "max_tokens" && cumulativeOut < groupBudget
			if shouldContinue && groupID == "" {
				groupID = uuid.New().String()
			}

			assistantNode := &storage.Node{
				ID:            uuid.New().String(),
				ParentID:      currentParent.ID,
				RootID:        currentParent.RootID,
				Sequence:      currentParent.Sequence + 1,
				NodeType:      storage.NodeTypeAssistant,
				Content:       accumulatedText,
				OutputGroupID: groupID,
				Model:         opts.Model,
				Provider:      e.provider.Name(),
				StopReason:    stopReason,
				LatencyMs:     int(time.Since(start).Milliseconds()),
				Status:        "completed",
				CreatedAt:     time.Now(),
			}
			if usage != nil {
				assistantNode.TokensIn = usage.PromptTokens
				assistantNode.TokensOut = usage.CompletionTokens
				assistantNode.TokensCacheRead = usage.CacheReadTokens
				assistantNode.TokensCacheCreation = usage.CacheCreationTokens
			}
			if err := e.store.CreateNode(ctx, assistantNode); err != nil {
				events <- Event{Type: EventError, Error: err.Error()}
				return
			}

			if !shouldContinue {
				events <- Event{Type: EventDone, NodeID: assistantNode.ID}
				return
			}

			currentParent = assistantNode

			contMessages := make([]client.EyrieMessage, len(messages), len(messages)+1)
			copy(contMessages, messages)
			contMessages = append(contMessages, client.EyrieMessage{Role: "assistant", Content: accumulatedText})

			contSR, contErr := e.provider.StreamChat(ctx, contMessages, chatOpts)
			if contErr != nil {
				events <- Event{Type: EventDone, NodeID: assistantNode.ID}
				return
			}
			currentStream = contSR.Events
		}
	}()

	return events, nil
}

func buildMessages(nodes []*storage.Node) []client.EyrieMessage {
	seen := map[string]bool{}
	var messages []client.EyrieMessage
	for _, n := range nodes {
		if n.OutputGroupID != "" {
			if seen[n.OutputGroupID] {
				continue
			}
			seen[n.OutputGroupID] = true
		}
		role := string(n.NodeType)
		if role == "tool_call" || role == "tool_result" {
			role = "user"
		}
		if role != "user" && role != "assistant" && role != "system" {
			continue
		}
		messages = append(messages, client.EyrieMessage{Role: role, Content: n.Content})
	}
	return messages
}

func generateTitle(msg string) string {
	msg = strings.TrimSpace(msg)
	if len(msg) > 50 {
		return msg[:50] + "..."
	}
	return msg
}
