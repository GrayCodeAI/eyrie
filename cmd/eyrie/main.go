//nolint:gocritic
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/GrayCodeAI/eyrie/client"
	"github.com/GrayCodeAI/eyrie/conversation"
	"github.com/GrayCodeAI/eyrie/internal/api"
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
  eyrie version                     Print version`)
}

func openStore() storage.Store {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".eyrie")
	_ = os.MkdirAll(dir, 0o755)
	store, err := storage.Open(filepath.Join(dir, "conversations.db"))
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	return store
}

func getProvider() client.Provider {
	detected := client.DetectProvider()
	c := client.Client(&client.EyrieConfig{Provider: detected})
	p, err := c.Chat(context.Background(), nil, client.ChatOptions{})
	_ = p
	_ = err
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
	store := openStore()
	provider := getProvider()
	srv := api.NewServer(api.Config{
		Store:    store,
		Provider: provider,
		APIKey:   os.Getenv("EYRIE_API_KEY"),
	})
	fmt.Printf("eyrie server listening on :%s\n", port)
	if err := srv.ListenAndServe(":" + port); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}
