package router

import (
	"context"
	"fmt"
	"testing"

	"github.com/GrayCodeAI/eyrie/catalog"
	"github.com/GrayCodeAI/eyrie/client"
)

type deploymentMockProvider struct {
	name       string
	err        error
	streamErr  error
	lastModel  string
	lastTools  []client.EyrieTool
	streamDone bool
}

func (m *deploymentMockProvider) Chat(_ context.Context, _ []client.EyrieMessage, opts client.ChatOptions) (*client.EyrieResponse, error) {
	m.lastModel = opts.Model
	m.lastTools = opts.Tools
	if m.err != nil {
		return nil, m.err
	}
	return &client.EyrieResponse{Content: "from " + m.name}, nil
}

func (m *deploymentMockProvider) StreamChat(_ context.Context, _ []client.EyrieMessage, opts client.ChatOptions) (*client.StreamResult, error) {
	m.lastModel = opts.Model
	if m.err != nil {
		return nil, m.err
	}
	ch := make(chan client.EyrieStreamEvent, 2)
	if m.streamErr != nil {
		ch <- client.EyrieStreamEvent{Type: "error", Error: m.streamErr.Error()}
	} else {
		ch <- client.EyrieStreamEvent{Type: "content", Content: "from " + m.name}
		ch <- client.EyrieStreamEvent{Type: "done"}
		m.streamDone = true
	}
	close(ch)
	return &client.StreamResult{Events: ch}, nil
}

func (m *deploymentMockProvider) Ping(_ context.Context) error { return m.err }
func (m *deploymentMockProvider) Name() string                 { return m.name }

func testCompiledCatalog(t *testing.T) *catalog.CompiledCatalogV1 {
	t.Helper()
	c := catalog.DefaultCatalogV1()
	compiled, err := catalog.CompileCatalogV1(&c)
	if err != nil {
		t.Fatalf("compile catalog: %v", err)
	}
	return compiled
}

func TestDeploymentRouterRewritesCanonicalModelToNativeModel(t *testing.T) {
	p := &deploymentMockProvider{name: "anthropic"}
	r, err := NewDeploymentRouter(DeploymentRouterOptions{
		Catalog: testCompiledCatalog(t),
		Deployments: map[string]DeploymentAdapter{
			"anthropic-direct": {Provider: p},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := r.Chat(context.Background(), []client.EyrieMessage{{Role: "user", Content: "hi"}}, client.ChatOptions{Model: "anthropic/claude-sonnet-4-6"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "from anthropic" {
		t.Fatalf("unexpected response %q", resp.Content)
	}
	if p.lastModel != "claude-sonnet-4-6" {
		t.Fatalf("model sent to provider = %q, want native model", p.lastModel)
	}
}

func TestDeploymentRouterFallsBackAcrossStages(t *testing.T) {
	primary := &deploymentMockProvider{name: "direct", err: fmt.Errorf("HTTP 503 unavailable")}
	vertex := &deploymentMockProvider{name: "vertex"}
	bedrock := &deploymentMockProvider{name: "bedrock"}
	r, err := NewDeploymentRouter(DeploymentRouterOptions{
		Catalog: testCompiledCatalog(t),
		Deployments: map[string]DeploymentAdapter{
			"anthropic-direct":  {Provider: primary},
			"anthropic-vertex":  {Provider: vertex},
			"anthropic-bedrock": {Provider: bedrock},
		},
		Routing: RoutingPolicy{
			Providers: map[string][]RoutingStage{
				"anthropic": {
					{Deployments: []DeploymentChoice{{DeploymentID: "anthropic-direct", Weight: 100}}},
					{Deployments: []DeploymentChoice{{DeploymentID: "anthropic-vertex", Weight: 50}, {DeploymentID: "anthropic-bedrock", Weight: 50}}},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := r.Chat(context.Background(), []client.EyrieMessage{{Role: "user", Content: "hi"}}, client.ChatOptions{Model: "claude-sonnet-4-6"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "from vertex" && resp.Content != "from bedrock" {
		t.Fatalf("expected fallback deployment, got %q", resp.Content)
	}
}

func TestDeploymentRouterNonTransientDoesNotFallback(t *testing.T) {
	primary := &deploymentMockProvider{name: "direct", err: fmt.Errorf("HTTP 401 unauthorized")}
	fallback := &deploymentMockProvider{name: "vertex"}
	r, err := NewDeploymentRouter(DeploymentRouterOptions{
		Catalog: testCompiledCatalog(t),
		Deployments: map[string]DeploymentAdapter{
			"anthropic-direct": {Provider: primary},
			"anthropic-vertex": {Provider: fallback},
		},
		Routing: RoutingPolicy{Providers: map[string][]RoutingStage{"anthropic": {
			{Deployments: []DeploymentChoice{{DeploymentID: "anthropic-direct", Weight: 100}}},
			{Deployments: []DeploymentChoice{{DeploymentID: "anthropic-vertex", Weight: 100}}},
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.Chat(context.Background(), []client.EyrieMessage{{Role: "user", Content: "hi"}}, client.ChatOptions{Model: "anthropic/claude-sonnet-4-6"})
	if err == nil {
		t.Fatal("expected auth error")
	}
	if fallback.lastModel != "" {
		t.Fatal("fallback provider should not be called for non-transient errors")
	}
}

func TestDeploymentRouterMaterializesAzureModelMapping(t *testing.T) {
	azure := &deploymentMockProvider{name: "azure"}
	r, err := NewDeploymentRouter(DeploymentRouterOptions{
		Catalog: testCompiledCatalog(t),
		Deployments: map[string]DeploymentAdapter{
			"openai-azure": {
				Provider: azure,
				ModelMappings: map[string]string{
					"openai/gpt-4o": "gpt-4o-prod",
				},
			},
		},
		Routing: RoutingPolicy{Models: map[string][]RoutingStage{
			"openai/gpt-4o": {{Deployments: []DeploymentChoice{{DeploymentID: "openai-azure", Weight: 100}}}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.Chat(context.Background(), []client.EyrieMessage{{Role: "user", Content: "hi"}}, client.ChatOptions{Model: "openai/gpt-4o"})
	if err != nil {
		t.Fatal(err)
	}
	if azure.lastModel != "gpt-4o-prod" {
		t.Fatalf("azure native model = %q, want mapping", azure.lastModel)
	}
}

func TestDeploymentRouterModelMappingOverridesCatalogOffering(t *testing.T) {
	bedrock := &deploymentMockProvider{name: "bedrock"}
	r, err := NewDeploymentRouter(DeploymentRouterOptions{
		Catalog: testCompiledCatalog(t),
		Deployments: map[string]DeploymentAdapter{
			"anthropic-bedrock": {
				Provider: bedrock,
				ModelMappings: map[string]string{
					"anthropic/claude-sonnet-4-6": "anthropic.claude-sonnet-4-6-bedrock",
				},
			},
		},
		Routing: RoutingPolicy{Models: map[string][]RoutingStage{
			"anthropic/claude-sonnet-4-6": {{Deployments: []DeploymentChoice{{DeploymentID: "anthropic-bedrock", Weight: 100}}}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.Chat(context.Background(), []client.EyrieMessage{{Role: "user", Content: "hi"}}, client.ChatOptions{Model: "anthropic/claude-sonnet-4-6"})
	if err != nil {
		t.Fatal(err)
	}
	if bedrock.lastModel != "anthropic.claude-sonnet-4-6-bedrock" {
		t.Fatalf("bedrock native model = %q, want mapping override", bedrock.lastModel)
	}
}

func TestDeploymentRouterStreamFallbackBeforeOutput(t *testing.T) {
	primary := &deploymentMockProvider{name: "direct", streamErr: fmt.Errorf("HTTP 503")}
	fallback := &deploymentMockProvider{name: "vertex"}
	r, err := NewDeploymentRouter(DeploymentRouterOptions{
		Catalog: testCompiledCatalog(t),
		Deployments: map[string]DeploymentAdapter{
			"anthropic-direct": {Provider: primary},
			"anthropic-vertex": {Provider: fallback},
		},
		Routing: RoutingPolicy{Providers: map[string][]RoutingStage{"anthropic": {
			{Deployments: []DeploymentChoice{{DeploymentID: "anthropic-direct", Weight: 100}}},
			{Deployments: []DeploymentChoice{{DeploymentID: "anthropic-vertex", Weight: 100}}},
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	stream, err := r.StreamChat(context.Background(), []client.EyrieMessage{{Role: "user", Content: "hi"}}, client.ChatOptions{Model: "anthropic/claude-sonnet-4-6"})
	if err != nil {
		t.Fatal(err)
	}
	var content string
	for event := range stream.Events {
		if event.Content != "" {
			content += event.Content
		}
		if event.Type == "error" {
			t.Fatalf("unexpected stream error: %s", event.Error)
		}
	}
	if content != "from vertex" {
		t.Fatalf("content = %q, want fallback output", content)
	}
}
