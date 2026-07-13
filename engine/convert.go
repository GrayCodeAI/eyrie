package engine

import "github.com/GrayCodeAI/eyrie/client"

func toClientMessages(in []Message) []client.EyrieMessage {
	out := make([]client.EyrieMessage, 0, len(in))
	for _, message := range in {
		parts := make([]client.ContentPart, 0, len(message.ContentParts))
		for _, part := range message.ContentParts {
			converted := client.ContentPart{Type: part.Type, Text: part.Text}
			if part.URL != "" {
				converted.ImageURL = &client.ImageURLPart{URL: part.URL, Detail: part.Detail}
			}
			if part.AudioData != "" {
				converted.InputAudio = &client.InputAudioPart{Data: part.AudioData, Format: part.AudioFormat}
			}
			parts = append(parts, converted)
		}
		calls := make([]client.ToolCall, 0, len(message.ToolCalls))
		for _, call := range message.ToolCalls {
			calls = append(calls, client.ToolCall{ID: call.ID, Name: call.Name, Arguments: call.Arguments})
		}
		results := make([]client.ToolResult, 0, len(message.ToolResults))
		for _, result := range message.ToolResults {
			results = append(results, client.ToolResult{ToolUseID: result.ToolUseID, Content: result.Content, IsError: result.IsError})
		}
		out = append(out, client.EyrieMessage{
			Role: message.Role, Content: message.Content, Thinking: message.Thinking,
			ContentParts: parts, ToolUse: calls, ToolResults: results,
		})
	}
	return out
}

func toClientOptions(req GenerateRequest, route Route, stream bool) client.ChatOptions {
	tools := make([]client.EyrieTool, 0, len(req.Tools))
	for _, tool := range req.Tools {
		tools = append(tools, client.EyrieTool{Name: tool.Name, Description: tool.Description, Parameters: tool.Parameters})
	}
	opts := client.ChatOptions{
		Provider: route.Provider, Model: route.Model, Stream: stream,
		System: req.SystemPrompt, Tools: tools, Temperature: req.Temperature,
		MaxTokens: req.Limits.MaxOutputTokens, MetadataUserID: req.Metadata.UserID,
	}
	advanced := req.Options
	opts.EnableCaching = advanced.EnableCaching
	opts.ReasoningEffort = advanced.ReasoningEffort
	opts.ThinkingBudgetTokens = advanced.ThinkingBudgetTokens
	opts.ThinkingMode = advanced.ThinkingMode
	opts.ThinkingDisplay = advanced.ThinkingDisplay
	opts.GLMThinkingEnabled = advanced.GLMThinkingEnabled
	opts.VirtualKeyID = advanced.VirtualKeyID
	opts.KimiContextCacheID = advanced.KimiContextCacheID
	opts.KimiCacheResetTTL = advanced.KimiCacheResetTTL
	opts.TopP = advanced.TopP
	opts.TopK = advanced.TopK
	opts.StopSequences = append([]string(nil), advanced.StopSequences...)
	if advanced.ToolChoice != nil {
		opts.ToolChoice = &client.ToolChoiceOption{
			Type: advanced.ToolChoice.Type, Name: advanced.ToolChoice.Name,
			DisableParallelToolUse: advanced.ToolChoice.DisableParallelToolUse,
		}
	}
	opts.ServiceTier = advanced.ServiceTier
	opts.OutputEffort = advanced.OutputEffort
	opts.PresencePenalty = advanced.PresencePenalty
	opts.FrequencyPenalty = advanced.FrequencyPenalty
	opts.N = advanced.N
	opts.LogProbs = advanced.LogProbs
	opts.TopLogProbs = advanced.TopLogProbs
	opts.Seed = advanced.Seed
	opts.Store = advanced.Store
	opts.Metadata = cloneStringMap(advanced.Metadata)
	if opts.Metadata == nil {
		opts.Metadata = make(map[string]string)
	}
	setMetadataIfPresent(opts.Metadata, "session.id", req.Metadata.SessionID)
	setMetadataIfPresent(opts.Metadata, "turn.id", req.Metadata.TurnID)
	setMetadataIfPresent(opts.Metadata, "project.id", req.Metadata.ProjectID)
	if len(opts.Metadata) == 0 {
		opts.Metadata = nil
	}
	opts.Modalities = append([]string(nil), advanced.Modalities...)
	opts.AudioConfig = advanced.AudioConfig
	opts.Prediction = advanced.Prediction
	opts.WebSearchOptions = advanced.WebSearchOptions
	if req.OutputSchema != "" {
		opts.ResponseFormat = &client.ResponseFormat{Type: "json_schema", Schema: req.OutputSchema}
		opts.OutputSchema = req.OutputSchema
	}
	return opts
}

func setMetadataIfPresent(metadata map[string]string, key, value string) {
	if value != "" {
		metadata[key] = value
	}
}

func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func fromClientResponse(resp *client.EyrieResponse, route Route) *GenerateResponse {
	if resp == nil {
		return &GenerateResponse{Route: route}
	}
	calls := make([]ToolCall, 0, len(resp.ToolCalls))
	for _, call := range resp.ToolCalls {
		calls = append(calls, ToolCall{ID: call.ID, Name: call.Name, Arguments: call.Arguments})
	}
	return &GenerateResponse{
		Content: resp.Content, Thinking: resp.Thinking, ToolCalls: calls,
		FinishReason: resp.FinishReason, RequestID: resp.RequestID,
		Usage: fromClientUsage(resp.Usage), Route: route,
	}
}

func fromClientUsage(usage *client.EyrieUsage) *Usage {
	if usage == nil {
		return nil
	}
	return &Usage{
		InputTokens: usage.PromptTokens, OutputTokens: usage.CompletionTokens,
		TotalTokens: usage.TotalTokens, CacheCreationTokens: usage.CacheCreationTokens,
		CacheReadTokens: usage.CacheReadTokens, ThinkingTokens: usage.ThinkingTokens,
	}
}
