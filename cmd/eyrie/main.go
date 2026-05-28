//nolint:gocritic
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/GrayCodeAI/eyrie/catalog"
	"github.com/GrayCodeAI/eyrie/client"
	"github.com/GrayCodeAI/eyrie/config"
	"github.com/GrayCodeAI/eyrie/conversation"
	"github.com/GrayCodeAI/eyrie/internal/api"
	"github.com/GrayCodeAI/eyrie/setup"
	"github.com/GrayCodeAI/eyrie/storage"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "prompt":
		runPrompt(os.Args[2:])
	case "ls":
		runList()
	case "show":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: eyrie show <node-id>")
			os.Exit(1)
		}
		runShow(os.Args[2])
	case "rm":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: eyrie rm <node-id>")
			os.Exit(1)
		}
		runDelete(os.Args[2])
	case "serve":
		port := "8080"
		if len(os.Args) > 2 {
			port = os.Args[2]
		}
		runServe(port)
	case "version":
		fmt.Println("eyrie " + client.Version)
	case "catalog":
		runCatalog(os.Args[2:])
	case "routing":
		runRouting(os.Args[2:])
	case "status":
		runStatus(os.Args[2:])
	case "help", "--help", "-h":
		printUsage()
	default:
		_, _ = fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`eyrie — conversation DAG engine for LLMs

Usage:
  eyrie prompt "message"            Start a new conversation
  eyrie prompt <node-id> "message"  Continue from a node (branch)
  eyrie ls                          List conversations
  eyrie show <node-id>              Show node tree
  eyrie rm <node-id>                Delete node and children
  eyrie serve [port]                Start REST API server (default: 8080)
  eyrie catalog refresh             Fetch published catalog only (no live provider APIs)
  eyrie catalog discover            Remote catalog + live provider APIs (env API keys) → ~/.eyrie/model_catalog.json
  eyrie catalog status              Show cached catalog metadata
  eyrie routing preview <model>     Show effective routing JSON for a model
  eyrie status [model]              Deployment routing status (optional model for route preview)
  eyrie version                     Print version`)
}

func runCatalog(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: eyrie catalog refresh|discover|status")
		os.Exit(1)
	}
	switch args[0] {
	case "refresh", "update":
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		result, err := catalog.RefreshCatalogV1(ctx, catalog.LoadCatalogV1Options{
			CachePath: catalog.DefaultCachePath(),
		})
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(result.Summary())
	case "discover":
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		result, err := setup.DiscoverModelCatalog(ctx, config.DiscoveryCredentials(ctx))
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(result.DiscoverReport())
	case "status":
		path := catalog.DefaultCachePath()
		exists, mod, size, err := catalog.CacheInfo(path)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if !exists {
			fmt.Printf("Catalog cache: not found at %s (embedded catalog used at runtime)\n", path)
			return
		}
		compiled, err := catalog.LoadCatalogV1(context.Background(), catalog.LoadCatalogV1Options{CachePath: path})
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Catalog cache: %s\n", path)
		fmt.Printf("  modified: %s (%d bytes)\n", mod.UTC().Format(time.RFC3339), size)
		fmt.Printf("  models: %d  deployments: %d  offerings: %d\n",
			len(compiled.ModelsByID), len(compiled.DeploymentsByID), len(compiled.OfferingsByID))
		if time.Now().UTC().After(compiled.Catalog.StaleAfter) {
			fmt.Println("  stale: yes — run `eyrie catalog discover`")
		}
	default:
		fmt.Fprintln(os.Stderr, "usage: eyrie catalog refresh|discover|status")
		os.Exit(1)
	}
}

func runRouting(args []string) {
	if len(args) < 2 || args[0] != "preview" {
		fmt.Fprintln(os.Stderr, "usage: eyrie routing preview <model>")
		os.Exit(1)
	}
	model := strings.Join(args[1:], " ")
	out, err := setup.RoutingPreview(context.Background(), model)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(out)
}

func runStatus(args []string) {
	model := ""
	if len(args) > 0 {
		model = strings.Join(args, " ")
	}
	report, err := setup.DeploymentStatus(context.Background(), model)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(setup.FormatStatus(report))
}

func openStore() storage.Store {
	home, err := os.UserHomeDir()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: cannot determine home directory: %v\n", err)
		os.Exit(1)
	}
	dir := filepath.Join(home, ".eyrie")
	_ = os.MkdirAll(dir, 0o700)
	store, err := storage.Open(filepath.Join(dir, "conversations.db"))
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	return store
}

func getProvider() client.Provider {
	cfg := config.LoadProviderConfig("")
	if setup.UseDeploymentRouting(cfg) {
		provider, err := setup.DeploymentProvider(context.Background(), cfg)
		if err == nil {
			return provider
		}
		_, _ = fmt.Fprintf(os.Stderr, "warning: deployment routing unavailable, using legacy provider: %v\n", err)
	}
	detected := client.DetectProvider()
	c := client.Client(&client.EyrieConfig{Provider: detected})
	return &clientProviderAdapter{c: c, provider: detected}
}

type clientProviderAdapter struct {
	c        *client.EyrieClient
	provider string
}

func (a *clientProviderAdapter) Chat(ctx context.Context, messages []client.EyrieMessage, opts client.ChatOptions) (*client.EyrieResponse, error) {
	opts.Provider = a.provider
	return a.c.Chat(ctx, messages, opts)
}

func (a *clientProviderAdapter) StreamChat(ctx context.Context, messages []client.EyrieMessage, opts client.ChatOptions) (*client.StreamResult, error) {
	opts.Provider = a.provider
	return a.c.StreamChat(ctx, messages, opts)
}

func (a *clientProviderAdapter) Ping(ctx context.Context) error {
	return a.c.Ping(ctx, a.provider)
}
func (a *clientProviderAdapter) Name() string { return a.provider }

func runPrompt(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: eyrie prompt [node-id] \"message\"")
		os.Exit(1)
	}

	store := openStore()
	defer func() { _ = store.Close() }()
	provider := getProvider()
	engine := conversation.New(store, provider)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var events <-chan conversation.Event
	var err error

	if len(args) == 1 {
		events, err = engine.Prompt(ctx, args[0], conversation.PromptOpts{})
	} else {
		events, err = engine.PromptFrom(ctx, args[0], strings.Join(args[1:], " "), conversation.PromptOpts{})
	}
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	for ev := range events {
		switch ev.Type {
		case conversation.EventDelta:
			fmt.Print(ev.Content)
		case conversation.EventDone:
			fmt.Printf("\n\n[node: %s]\n", ev.NodeID)
		case conversation.EventError:
			_, _ = fmt.Fprintf(os.Stderr, "\nerror: %s\n", ev.Error)
			os.Exit(1)
		}
	}
}

func runList() {
	store := openStore()
	defer func() { _ = store.Close() }()
	ctx := context.Background()
	nodes, err := store.ListRootNodes(ctx)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if len(nodes) == 0 {
		fmt.Println("No conversations yet.")
		return
	}
	for _, n := range nodes {
		title := n.Title
		if title == "" {
			title = truncate(n.Content, 50)
		}
		fmt.Printf("  %s  %s  %s\n", n.ID[:8], n.CreatedAt.Format("2006-01-02 15:04"), title)
	}
}

func runShow(id string) {
	store := openStore()
	defer func() { _ = store.Close() }()
	ctx := context.Background()
	nodes, err := store.GetSubtree(ctx, id)
	if err != nil || len(nodes) == 0 {
		_, _ = fmt.Fprintf(os.Stderr, "node not found: %s\n", id)
		os.Exit(1)
	}
	for _, n := range nodes {
		role := string(n.NodeType)
		content := truncate(n.Content, 80)
		fmt.Printf("  [%s] %s: %s\n", n.ID[:8], role, content)
	}
}

func runDelete(id string) {
	store := openStore()
	defer func() { _ = store.Close() }()
	if err := store.DeleteNode(context.Background(), id); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Deleted node %s and children.\n", id)
}

func runServe(port string) {
	apiKey := strings.TrimSpace(os.Getenv("EYRIE_API_KEY"))
	if apiKey == "" {
		if strings.EqualFold(strings.TrimSpace(os.Getenv("EYRIE_ALLOW_INSECURE_PUBLIC_API")), "true") {
			_, _ = fmt.Fprintf(os.Stderr, "warning: EYRIE_ALLOW_INSECURE_PUBLIC_API=true — HTTP API is unauthenticated (not recommended)\n")
		} else {
			_, _ = fmt.Fprintf(os.Stderr, "error: set EYRIE_API_KEY (Bearer / X-API-Key) before starting the HTTP API, or set EYRIE_ALLOW_INSECURE_PUBLIC_API=true for explicit insecure local use\n")
			os.Exit(1)
		}
	}
	store := openStore()
	defer func() {
		if err := store.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "error closing store: %v\n", err)
		}
	}()
	provider := getProvider()
	srv := api.NewServer(api.Config{
		Store:    store,
		Provider: provider,
		APIKey:   apiKey,
	})
	fmt.Printf("eyrie server listening on :%s\n", port)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		fmt.Println("shutting down gracefully...")
		if err := srv.Shutdown(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "error during shutdown: %v\n", err)
		}
	}()

	if err := srv.ListenAndServe(":" + port); err != nil && err != http.ErrServerClosed {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	runes := []rune(s)
	if len(runes) > n {
		return string(runes[:n]) + "..."
	}
	return s
}
