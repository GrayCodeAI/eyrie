package router

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/GrayCodeAI/eyrie/catalog"
	"github.com/GrayCodeAI/eyrie/client"
)

type DeploymentChoice struct {
	DeploymentID string `json:"deployment_id"`
	Weight       int    `json:"weight"`
}

type RoutingStage struct {
	Deployments []DeploymentChoice `json:"deployments"`
	Retries     int                `json:"retries,omitempty"`
}

type RoutingPolicy struct {
	Default   []RoutingStage            `json:"default,omitempty"`
	Providers map[string][]RoutingStage `json:"providers,omitempty"`
	Models    map[string][]RoutingStage `json:"models,omitempty"`
}

type DeploymentAdapter struct {
	DeploymentID  string
	Provider      client.Provider
	ModelMappings map[string]string
}

type DeploymentRouterOptions struct {
	Catalog     *catalog.CompiledCatalogV1
	Deployments map[string]DeploymentAdapter
	Routing     RoutingPolicy
}

type DeploymentRouter struct {
	catalog     *catalog.CompiledCatalogV1
	deployments map[string]DeploymentAdapter
	routing     RoutingPolicy
	statsMu     sync.RWMutex
	stats       map[string]*atomic.Int64
}

var _ client.Provider = (*DeploymentRouter)(nil)

func NewDeploymentRouter(opts DeploymentRouterOptions) (*DeploymentRouter, error) {
	if opts.Catalog == nil {
		return nil, fmt.Errorf("deployment router: catalog is required")
	}
	if len(opts.Deployments) == 0 {
		return nil, fmt.Errorf("deployment router: at least one deployment is required")
	}
	deployments := make(map[string]DeploymentAdapter, len(opts.Deployments))
	stats := make(map[string]*atomic.Int64, len(opts.Deployments))
	for id, adapter := range opts.Deployments {
		if adapter.DeploymentID == "" {
			adapter.DeploymentID = id
		}
		if adapter.DeploymentID != id {
			return nil, fmt.Errorf("deployment router: deployment key %q does not match adapter %q", id, adapter.DeploymentID)
		}
		if adapter.Provider == nil {
			return nil, fmt.Errorf("deployment router: deployment %q has nil provider", id)
		}
		if opts.Catalog.DeploymentsByID[id].ID == "" {
			return nil, fmt.Errorf("deployment router: deployment %q is not in catalog", id)
		}
		adapter.ModelMappings = cloneStrings(adapter.ModelMappings)
		deployments[id] = adapter
		stats[id] = &atomic.Int64{}
	}
	router := &DeploymentRouter{
		catalog:     opts.Catalog,
		deployments: deployments,
		routing:     cloneRoutingPolicy(opts.Routing),
		stats:       stats,
	}
	return router, nil
}

func (r *DeploymentRouter) Name() string {
	return "deployment-router"
}

func (r *DeploymentRouter) Ping(ctx context.Context) error {
	var lastErr error
	for _, id := range sortedDeploymentIDs(r.deployments) {
		if err := r.deployments[id].Provider.Ping(ctx); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	if lastErr != nil {
		return fmt.Errorf("deployment router: all deployments failed ping: %w", lastErr)
	}
	return fmt.Errorf("deployment router: no deployments configured")
}

func (r *DeploymentRouter) Chat(ctx context.Context, messages []client.EyrieMessage, opts client.ChatOptions) (*client.EyrieResponse, error) {
	target, err := r.resolveTarget(opts.Model)
	if err != nil {
		return nil, err
	}
	var lastErr error
	for stageIndex, stage := range r.routeFor(target.canonicalModelID) {
		choices := r.eligibleChoices(target, stage, opts)
		if len(choices) == 0 {
			lastErr = fmt.Errorf("stage %d has no eligible deployments", stageIndex)
			continue
		}
		attempts := stage.Retries + 1
		if attempts < 1 {
			attempts = 1
		}
		for attempt := 0; attempt < attempts; attempt++ {
			choice := selectDeploymentChoice(choices)
			resp, err := r.chatWithDeployment(ctx, messages, opts, target, choice.DeploymentID)
			if err == nil {
				r.recordSuccess(choice.DeploymentID)
				return resp, nil
			}
			lastErr = err
			if !IsTransient(err) {
				if ShouldTryNextDeployment(err) {
					break
				}
				return nil, err
			}
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no route configured")
	}
	return nil, fmt.Errorf("deployment router: all deployments failed for %q: %w", target.canonicalModelID, lastErr)
}

func (r *DeploymentRouter) StreamChat(ctx context.Context, messages []client.EyrieMessage, opts client.ChatOptions) (*client.StreamResult, error) {
	target, err := r.resolveTarget(opts.Model)
	if err != nil {
		return nil, err
	}
	out := make(chan client.EyrieStreamEvent, 64)
	go func() {
		defer close(out)
		var lastErr error
		for stageIndex, stage := range r.routeFor(target.canonicalModelID) {
			choices := r.eligibleChoices(target, stage, opts)
			if len(choices) == 0 {
				lastErr = fmt.Errorf("stage %d has no eligible deployments", stageIndex)
				continue
			}
			attempts := stage.Retries + 1
			if attempts < 1 {
				attempts = 1
			}
			for attempt := 0; attempt < attempts; attempt++ {
				choice := selectDeploymentChoice(choices)
				fallback, err := r.streamWithDeployment(ctx, out, messages, opts, target, choice.DeploymentID)
				if err == nil {
					r.recordSuccess(choice.DeploymentID)
					return
				}
				lastErr = err
				if !fallback {
					out <- client.EyrieStreamEvent{Type: "error", Error: err.Error()}
					return
				}
				if !IsTransient(err) {
					if ShouldTryNextDeployment(err) {
						break
					}
					out <- client.EyrieStreamEvent{Type: "error", Error: err.Error()}
					return
				}
			}
		}
		if lastErr == nil {
			lastErr = fmt.Errorf("no route configured")
		}
		out <- client.EyrieStreamEvent{Type: "error", Error: fmt.Sprintf("deployment router: all deployments failed for %q: %v", target.canonicalModelID, lastErr)}
	}()
	return &client.StreamResult{Events: out}, nil
}

func (r *DeploymentRouter) Stats() map[string]int64 {
	r.statsMu.RLock()
	defer r.statsMu.RUnlock()
	out := make(map[string]int64, len(r.stats))
	for id, counter := range r.stats {
		out[id] = counter.Load()
	}
	return out
}

type deploymentTarget struct {
	canonicalModelID string
	nativeHint       string
}

func (r *DeploymentRouter) resolveTarget(requested string) (deploymentTarget, error) {
	if requested == "" {
		return deploymentTarget{}, fmt.Errorf("deployment router: model is required")
	}
	if canonical, ok := r.catalog.CanonicalModelForAliasOrID(requested); ok {
		return deploymentTarget{canonicalModelID: canonical}, nil
	}
	var matches []string
	for _, offering := range r.catalog.Catalog.Offerings {
		if offering.NativeModelID == requested || offering.ID == requested {
			matches = append(matches, offering.CanonicalModelID)
		}
	}
	for _, adapter := range r.deployments {
		for canonical, native := range adapter.ModelMappings {
			if native == requested {
				matches = append(matches, canonical)
			}
		}
	}
	matches = uniqueStrings(matches)
	if len(matches) == 1 {
		return deploymentTarget{canonicalModelID: matches[0]}, nil
	}
	if len(matches) > 1 {
		sort.Strings(matches)
		return deploymentTarget{}, fmt.Errorf("deployment router: model %q is ambiguous: %s", requested, strings.Join(matches, ", "))
	}
	if strings.Contains(requested, "/") {
		return deploymentTarget{canonicalModelID: requested, nativeHint: requested}, nil
	}
	return deploymentTarget{}, fmt.Errorf("deployment router: model %q is not in catalog", requested)
}

func (r *DeploymentRouter) routeFor(canonicalModelID string) []RoutingStage {
	var stages []RoutingStage
	if explicit, ok := r.routing.Models[canonicalModelID]; ok && len(explicit) > 0 {
		stages = cloneRoutingStages(explicit)
	} else {
		providerID := ownerProviderID(canonicalModelID)
		if model := r.catalog.ModelsByID[canonicalModelID]; model.ID != "" {
			providerID = model.ProviderID
		}
		if providerID != "" {
			if explicit, ok := r.routing.Providers[providerID]; ok && len(explicit) > 0 {
				stages = cloneRoutingStages(explicit)
			} else {
				for key, explicit := range r.routing.Providers {
					if canonicalProviderID(key) == providerID && len(explicit) > 0 {
						stages = cloneRoutingStages(explicit)
						break
					}
				}
			}
		}
		if len(stages) == 0 {
			if r.routing.Default != nil {
				stages = cloneRoutingStages(r.routing.Default)
			} else {
				return r.automaticStages(canonicalModelID)
			}
		}
	}
	return appendAutomaticFallback(stages, r.automaticStages(canonicalModelID))
}

func (r *DeploymentRouter) automaticStages(canonicalModelID string) []RoutingStage {
	var choices []DeploymentChoice
	for _, id := range sortedDeploymentIDs(r.deployments) {
		if _, _, err := r.resolveOffering(deploymentTarget{canonicalModelID: canonicalModelID}, id); err == nil {
			choices = append(choices, DeploymentChoice{DeploymentID: id, Weight: 100})
		}
	}
	if len(choices) == 0 {
		return nil
	}
	return []RoutingStage{{Deployments: choices}}
}

func (r *DeploymentRouter) eligibleChoices(target deploymentTarget, stage RoutingStage, opts client.ChatOptions) []DeploymentChoice {
	var choices []DeploymentChoice
	var toolCapable []DeploymentChoice
	requiredTools := requestedServerTools(opts.Tools)
	for _, choice := range stage.Deployments {
		if choice.DeploymentID == "" || choice.Weight <= 0 {
			continue
		}
		offering, _, err := r.resolveOffering(target, choice.DeploymentID)
		if err != nil {
			continue
		}
		choices = append(choices, choice)
		if len(requiredTools) == 0 || offeringSupportsTools(offering, requiredTools) {
			toolCapable = append(toolCapable, choice)
		}
	}
	if len(requiredTools) > 0 && len(toolCapable) > 0 {
		return toolCapable
	}
	return choices
}

func (r *DeploymentRouter) chatWithDeployment(ctx context.Context, messages []client.EyrieMessage, opts client.ChatOptions, target deploymentTarget, deploymentID string) (*client.EyrieResponse, error) {
	offering, adapter, err := r.resolveOffering(target, deploymentID)
	if err != nil {
		return nil, err
	}
	nativeOpts := optsForOffering(opts, offering)
	return adapter.Provider.Chat(ctx, messages, nativeOpts)
}

func (r *DeploymentRouter) streamWithDeployment(ctx context.Context, out chan<- client.EyrieStreamEvent, messages []client.EyrieMessage, opts client.ChatOptions, target deploymentTarget, deploymentID string) (fallback bool, err error) {
	offering, adapter, err := r.resolveOffering(target, deploymentID)
	if err != nil {
		return true, err
	}
	nativeOpts := optsForOffering(opts, offering)
	stream, err := adapter.Provider.StreamChat(ctx, messages, nativeOpts)
	if err != nil {
		return true, err
	}
	defer stream.Close()
	emitted := false
	var buffered []client.EyrieStreamEvent
	flush := func() {
		for _, event := range buffered {
			out <- event
		}
		buffered = nil
	}
	for event := range stream.Events {
		if event.Type == "error" {
			if emitted {
				out <- event
				return false, fmt.Errorf("%s", event.Error)
			}
			if event.Error == "" {
				return true, fmt.Errorf("deployment %q stream failed before output", deploymentID)
			}
			return true, fmt.Errorf("%s", event.Error)
		}
		if isOutputEvent(event) {
			emitted = true
			flush()
			out <- event
			continue
		}
		if emitted || event.Type == "done" {
			flush()
			out <- event
			return false, nil
		}
		buffered = append(buffered, event)
	}
	if emitted {
		return false, fmt.Errorf("deployment %q stream ended after output without done", deploymentID)
	}
	return true, fmt.Errorf("deployment %q stream ended before output", deploymentID)
}

func (r *DeploymentRouter) resolveOffering(target deploymentTarget, deploymentID string) (catalog.ModelOfferingV1, DeploymentAdapter, error) {
	adapter, ok := r.deployments[deploymentID]
	if !ok {
		return catalog.ModelOfferingV1{}, DeploymentAdapter{}, fmt.Errorf("deployment %q is not configured", deploymentID)
	}
	if offering, ok := r.catalog.OfferingForDeployment(target.canonicalModelID, deploymentID); ok {
		if nativeID := adapter.ModelMappings[target.canonicalModelID]; nativeID != "" {
			offering.NativeModelID = nativeID
			offering.ID = deploymentID + ":" + nativeID
		}
		return offering, adapter, nil
	}
	for _, tmpl := range r.catalog.TemplatesByCanonicalModel[target.canonicalModelID] {
		if tmpl.DeploymentID != deploymentID {
			continue
		}
		nativeID := adapter.ModelMappings[target.canonicalModelID]
		if nativeID == "" {
			return catalog.ModelOfferingV1{}, DeploymentAdapter{}, fmt.Errorf("deployment %q requires model mapping for %q", deploymentID, target.canonicalModelID)
		}
		return materializeTemplate(tmpl, nativeID), adapter, nil
	}
	if target.nativeHint != "" {
		deployment := r.catalog.DeploymentsByID[deploymentID]
		if deployment.ID != "" && deployment.NativeModelIDSource == catalog.NativeModelIDDiscovered {
			return catalog.ModelOfferingV1{
				ID:               deploymentID + ":" + target.nativeHint,
				CanonicalModelID: target.canonicalModelID,
				DeploymentID:     deploymentID,
				NativeModelID:    nativeModelHintForDeployment(target.nativeHint, deployment),
				Pricing:          catalog.PricingV1{Status: catalog.PricingUnknown},
			}, adapter, nil
		}
	}
	return catalog.ModelOfferingV1{}, DeploymentAdapter{}, fmt.Errorf("deployment %q cannot serve %q", deploymentID, target.canonicalModelID)
}

func materializeTemplate(tmpl catalog.ModelOfferingTemplateV1, nativeID string) catalog.ModelOfferingV1 {
	return catalog.ModelOfferingV1{
		ID:               tmpl.DeploymentID + ":" + nativeID,
		CanonicalModelID: tmpl.CanonicalModelID,
		DeploymentID:     tmpl.DeploymentID,
		NativeModelID:    nativeID,
		Capabilities:     tmpl.Capabilities,
		Pricing:          tmpl.Pricing,
		Provenance:       tmpl.Provenance,
	}
}

func optsForOffering(opts client.ChatOptions, offering catalog.ModelOfferingV1) client.ChatOptions {
	copied := opts
	copied.Model = offering.NativeModelID
	copied.Provider = offering.DeploymentID
	if len(copied.Tools) > 0 {
		copied.Tools = filterTools(copied.Tools, offering)
	}
	return copied
}

func filterTools(tools []client.EyrieTool, offering catalog.ModelOfferingV1) []client.EyrieTool {
	if len(offering.Capabilities.ServerTools) == 0 {
		return tools
	}
	filtered := make([]client.EyrieTool, 0, len(tools))
	for _, tool := range tools {
		if offering.Capabilities.ServerTools[tool.Name] == catalog.CapabilityUnsupported ||
			offering.Capabilities.ServerTools[tool.Name] == catalog.CapabilityUnknown {
			continue
		}
		filtered = append(filtered, tool)
	}
	return filtered
}

func requestedServerTools(tools []client.EyrieTool) []string {
	seen := map[string]bool{}
	var out []string
	for _, tool := range tools {
		if tool.Name == "" || seen[tool.Name] {
			continue
		}
		seen[tool.Name] = true
		out = append(out, tool.Name)
	}
	sort.Strings(out)
	return out
}

func offeringSupportsTools(offering catalog.ModelOfferingV1, tools []string) bool {
	for _, tool := range tools {
		if offering.Capabilities.ServerTools[tool] != catalog.CapabilitySupported {
			return false
		}
	}
	return true
}

func selectDeploymentChoice(choices []DeploymentChoice) DeploymentChoice {
	if len(choices) == 1 {
		return choices[0]
	}
	total := 0
	for _, choice := range choices {
		total += choice.Weight
	}
	if total <= 0 {
		return choices[0]
	}
	n := rand.IntN(total)
	for _, choice := range choices {
		n -= choice.Weight
		if n < 0 {
			return choice
		}
	}
	return choices[len(choices)-1]
}

func isOutputEvent(event client.EyrieStreamEvent) bool {
	return event.Content != "" || event.Thinking != "" || event.ToolCall != nil || event.Type == "content" || event.Type == "thinking" || event.Type == "tool_call"
}

func ownerProviderID(canonicalModelID string) string {
	owner, _, _ := strings.Cut(canonicalModelID, "/")
	return canonicalProviderID(owner)
}

func canonicalProviderID(providerID string) string {
	switch providerID {
	case "gemini":
		return "google"
	case "grok":
		return "xai"
	case "zai":
		return "z-ai"
	case "moonshotai":
		return "moonshotai"
	default:
		return providerID
	}
}

func nativeModelHintForDeployment(model string, deployment catalog.DeploymentV1) string {
	if deployment.ID == "openrouter" {
		if native, ok := strings.CutPrefix(model, "openrouter/"); ok {
			return native
		}
		return model
	}
	if native, ok := strings.CutPrefix(model, deployment.ProviderID+"/"); ok {
		return native
	}
	return model
}

func sortedDeploymentIDs(deployments map[string]DeploymentAdapter) []string {
	ids := make([]string, 0, len(deployments))
	for id := range deployments {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func appendAutomaticFallback(stages, automatic []RoutingStage) []RoutingStage {
	if len(automatic) == 0 {
		return stages
	}
	tried := map[string]bool{}
	for _, stage := range stages {
		for _, choice := range stage.Deployments {
			tried[choice.DeploymentID] = true
		}
	}
	var remaining []DeploymentChoice
	for _, stage := range automatic {
		for _, choice := range stage.Deployments {
			if choice.DeploymentID == "" || choice.Weight <= 0 || tried[choice.DeploymentID] {
				continue
			}
			remaining = append(remaining, choice)
			tried[choice.DeploymentID] = true
		}
	}
	if len(remaining) == 0 {
		return stages
	}
	return append(stages, RoutingStage{Deployments: remaining, Retries: 1})
}

func cloneRoutingPolicy(policy RoutingPolicy) RoutingPolicy {
	return RoutingPolicy{
		Default:   cloneRoutingStages(policy.Default),
		Providers: cloneRoutingStageMap(policy.Providers),
		Models:    cloneRoutingStageMap(policy.Models),
	}
}

func cloneRoutingStageMap(in map[string][]RoutingStage) map[string][]RoutingStage {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string][]RoutingStage, len(in))
	for key, stages := range in {
		out[key] = cloneRoutingStages(stages)
	}
	return out
}

func cloneRoutingStages(stages []RoutingStage) []RoutingStage {
	if stages == nil {
		return nil
	}
	out := make([]RoutingStage, len(stages))
	for i, stage := range stages {
		out[i].Retries = stage.Retries
		out[i].Deployments = append([]DeploymentChoice(nil), stage.Deployments...)
	}
	return out
}

func cloneStrings(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func (r *DeploymentRouter) recordSuccess(deploymentID string) {
	r.statsMu.RLock()
	counter := r.stats[deploymentID]
	r.statsMu.RUnlock()
	if counter != nil {
		counter.Add(1)
	}
}
