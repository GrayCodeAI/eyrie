//nolint:gocritic
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/GrayCodeAI/eyrie/catalog"
	"github.com/GrayCodeAI/eyrie/client"
	"github.com/GrayCodeAI/eyrie/config"
	"github.com/GrayCodeAI/eyrie/conversation"
	"github.com/GrayCodeAI/eyrie/internal/api"
	eyrieversion "github.com/GrayCodeAI/eyrie/internal/version"
	"github.com/GrayCodeAI/eyrie/runtime"
	"github.com/GrayCodeAI/eyrie/setup"
	"github.com/GrayCodeAI/eyrie/storage"
)

var errHelpShown = errors.New("help shown")

type cli struct {
	stdout io.Writer
	stderr io.Writer
}

func main() {
	app := cli{
		stdout: os.Stdout,
		stderr: os.Stderr,
	}
	if err := app.run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func (c cli) run(args []string) error {
	if len(args) == 0 {
		c.printRootUsage()
		return nil
	}

	switch args[0] {
	case "help", "--help", "-h":
		c.printHelp(args[1:])
		return nil
	case "prompt", "chat":
		return c.runPrompt(args[1:])
	case "ls", "list":
		return c.runList(args[1:])
	case "show":
		return c.runShow(args[1:])
	case "rm", "delete":
		return c.runDelete(args[1:])
	case "serve":
		return c.runServe(args[1:])
	case "version":
		fmt.Fprintln(c.stdout, "eyrie", eyrieversion.Version)
		return nil
	case "catalog":
		return c.runCatalog(args[1:])
	case "routing":
		return c.runRouting(args[1:])
	case "status":
		return c.runStatus(args[1:])
	case "preflight":
		return c.runPreflight(args[1:])
	case "providers":
		return c.runProviders(args[1:])
	case "models":
		return c.runModels(args[1:])
	case "select":
		return c.runSelect(args[1:])
	default:
		return fmt.Errorf("unknown command %q: run `eyrie help` to see available commands", args[0])
	}
}

func (c cli) printRootUsage() {
	fmt.Fprintf(c.stdout, `eyrie — universal LLM provider runtime

One interface for every model. Authentication, routing, streaming, retries, caching, and deployment selection are handled here.

Quick start:
  eyrie preflight
  eyrie providers
  eyrie models anthropic
  eyrie select provider anthropic
  eyrie select model claude-sonnet-4-6
  eyrie status

Usage:
  eyrie <command> [arguments]

Core commands:
  prompt [node-id] <message>   Start or continue a conversation DAG
  ls                           List saved conversations
  show <node-id>               Show a conversation subtree
  rm <node-id>                 Delete a node and its descendants
  serve [--port 8080]          Start the HTTP API server

Setup and inspection:
  preflight                    Check catalog, credentials, and active selection
  providers                    List supported provider gateways
  models [provider]            List models for a provider
  select provider <id>         Set the active provider in provider.json
  select model <id>            Set the active model in provider.json
  status [model]               Show deployment routing status
  catalog refresh              Refresh the published catalog cache
  catalog discover             Discover live models from configured providers
  catalog status               Show catalog cache metadata
  routing preview <model>      Print effective routing JSON for a model

Other:
  version                      Print eyrie version
  help [command]               Show command-specific help

Config:
  provider.json path: %s

Examples:
  eyrie prompt "Explain this repository"
  eyrie prompt 7f9a8c1e "Try a different approach"
  eyrie models anthropic --source cache
  eyrie models openai --source live --json
  eyrie serve --port 8080
`, config.GetProviderConfigPath())
}

func (c cli) printHelp(path []string) {
	if len(path) == 0 {
		c.printRootUsage()
		return
	}
	switch strings.Join(path, " ") {
	case "prompt", "chat":
		fmt.Fprint(c.stdout, promptUsage)
	case "ls", "list":
		fmt.Fprint(c.stdout, listUsage)
	case "show":
		fmt.Fprint(c.stdout, showUsage)
	case "rm", "delete":
		fmt.Fprint(c.stdout, deleteUsage)
	case "serve":
		fmt.Fprint(c.stdout, serveUsage)
	case "catalog":
		fmt.Fprint(c.stdout, catalogUsage)
	case "routing":
		fmt.Fprint(c.stdout, routingUsage)
	case "status":
		fmt.Fprint(c.stdout, statusUsage)
	case "preflight":
		fmt.Fprint(c.stdout, preflightUsage)
	case "providers":
		fmt.Fprint(c.stdout, providersUsage)
	case "models":
		fmt.Fprint(c.stdout, modelsUsage)
	case "select":
		fmt.Fprint(c.stdout, selectUsage)
	default:
		c.printRootUsage()
	}
}

func (c cli) runPrompt(args []string) error {
	if len(args) == 0 || isHelpArg(args[0]) {
		fmt.Fprint(c.stdout, promptUsage)
		return nil
	}

	store, err := openStore()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	provider := getProvider()
	engine := conversation.New(store, provider)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var events <-chan conversation.Event
	if len(args) == 1 {
		events, err = engine.Prompt(ctx, args[0], conversation.PromptOpts{})
	} else {
		events, err = engine.PromptFrom(ctx, args[0], strings.Join(args[1:], " "), conversation.PromptOpts{})
	}
	if err != nil {
		return err
	}

	for ev := range events {
		switch ev.Type {
		case conversation.EventDelta:
			fmt.Fprint(c.stdout, ev.Content)
		case conversation.EventDone:
			fmt.Fprintf(c.stdout, "\n\n[node: %s]\n", ev.NodeID)
		case conversation.EventError:
			return errors.New(ev.Error)
		}
	}
	return nil
}

func (c cli) runList(args []string) error {
	if hasHelpArg(args) {
		fmt.Fprint(c.stdout, listUsage)
		return nil
	}
	if len(args) != 0 {
		return fmt.Errorf("usage: eyrie ls")
	}

	store, err := openStore()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	nodes, err := store.ListRootNodes(context.Background())
	if err != nil {
		return err
	}
	if len(nodes) == 0 {
		fmt.Fprintln(c.stdout, "No conversations yet.")
		return nil
	}
	for _, n := range nodes {
		title := n.Title
		if title == "" {
			title = truncate(n.Content, 50)
		}
		fmt.Fprintf(c.stdout, "  %s  %s  %s\n", n.ID[:8], n.CreatedAt.Format("2006-01-02 15:04"), title)
	}
	return nil
}

func (c cli) runShow(args []string) error {
	if len(args) == 0 || isHelpArg(args[0]) {
		fmt.Fprint(c.stdout, showUsage)
		return nil
	}
	if len(args) != 1 {
		return fmt.Errorf("usage: eyrie show <node-id>")
	}

	store, err := openStore()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	nodes, err := store.GetSubtree(context.Background(), args[0])
	if err != nil || len(nodes) == 0 {
		return fmt.Errorf("node not found: %s", args[0])
	}
	for _, n := range nodes {
		role := string(n.NodeType)
		content := truncate(n.Content, 80)
		fmt.Fprintf(c.stdout, "  [%s] %s: %s\n", n.ID[:8], role, content)
	}
	return nil
}

func (c cli) runDelete(args []string) error {
	if len(args) == 0 || isHelpArg(args[0]) {
		fmt.Fprint(c.stdout, deleteUsage)
		return nil
	}
	if len(args) != 1 {
		return fmt.Errorf("usage: eyrie rm <node-id>")
	}

	store, err := openStore()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	if err := store.DeleteNode(context.Background(), args[0]); err != nil {
		return err
	}
	fmt.Fprintf(c.stdout, "Deleted node %s and children.\n", args[0])
	return nil
}

func (c cli) runServe(args []string) error {
	fs := newFlagSet("serve")
	port := fs.String("port", "8080", "port to listen on")
	listen := fs.String("listen", "", "full listen address override, e.g. 127.0.0.1:8080")
	rest, err := c.parseFlags(fs, args, serveUsage)
	if err != nil {
		if errors.Is(err, errHelpShown) {
			return nil
		}
		return err
	}
	if len(rest) > 1 {
		return fmt.Errorf("usage: eyrie serve [--port 8080]")
	}
	if len(rest) == 1 && *port == "8080" && *listen == "" {
		*port = rest[0]
	}
	addr := *listen
	if strings.TrimSpace(addr) == "" {
		addr = ":" + strings.TrimSpace(*port)
	}

	apiKey := strings.TrimSpace(os.Getenv("EYRIE_API_KEY"))
	if apiKey == "" {
		if strings.EqualFold(strings.TrimSpace(os.Getenv("EYRIE_ALLOW_INSECURE_PUBLIC_API")), "true") {
			fmt.Fprintln(c.stderr, "warning: EYRIE_ALLOW_INSECURE_PUBLIC_API=true — HTTP API is unauthenticated (not recommended)")
		} else {
			return fmt.Errorf("set EYRIE_API_KEY before starting the HTTP API, or set EYRIE_ALLOW_INSECURE_PUBLIC_API=true for explicit local insecure use")
		}
	}

	store, err := openStore()
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			fmt.Fprintf(c.stderr, "error closing store: %v\n", closeErr)
		}
	}()

	srv := api.NewServer(api.Config{
		Store:    store,
		Provider: getProvider(),
		APIKey:   apiKey,
	})
	fmt.Fprintf(c.stdout, "eyrie server listening on %s\n", addr)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		fmt.Fprintln(c.stdout, "shutting down gracefully...")
		if err := srv.Shutdown(); err != nil {
			fmt.Fprintf(c.stderr, "error during shutdown: %v\n", err)
		}
	}()

	if err := srv.ListenAndServe(addr); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (c cli) runCatalog(args []string) error {
	if len(args) == 0 || isHelpArg(args[0]) {
		fmt.Fprint(c.stdout, catalogUsage)
		return nil
	}

	switch args[0] {
	case "refresh", "update":
		fs := newFlagSet("catalog refresh")
		timeout := fs.Duration("timeout", 15*time.Second, "request timeout")
		_, err := c.parseFlags(fs, args[1:], catalogUsage)
		if err != nil {
			if errors.Is(err, errHelpShown) {
				return nil
			}
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), *timeout)
		defer cancel()
		result, err := catalog.RefreshCatalogV1(ctx, catalog.LoadCatalogV1Options{
			CachePath: catalog.DefaultCachePath(),
		})
		if err != nil {
			return err
		}
		fmt.Fprintln(c.stdout, result.Summary())
		return nil
	case "discover":
		fs := newFlagSet("catalog discover")
		timeout := fs.Duration("timeout", 30*time.Second, "request timeout")
		_, err := c.parseFlags(fs, args[1:], catalogUsage)
		if err != nil {
			if errors.Is(err, errHelpShown) {
				return nil
			}
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), *timeout)
		defer cancel()
		result, err := setup.DiscoverModelCatalog(ctx, config.DiscoveryCredentials(ctx))
		if err != nil {
			return err
		}
		fmt.Fprint(c.stdout, result.DiscoverReport())
		return nil
	case "status":
		fs := newFlagSet("catalog status")
		jsonOut := fs.Bool("json", false, "print machine-readable JSON")
		_, err := c.parseFlags(fs, args[1:], catalogUsage)
		if err != nil {
			if errors.Is(err, errHelpShown) {
				return nil
			}
			return err
		}
		report, err := catalogStatusReport(context.Background())
		if err != nil {
			return err
		}
		if *jsonOut {
			return writeJSON(c.stdout, report)
		}
		c.printCatalogStatus(report)
		return nil
	default:
		return fmt.Errorf("unknown catalog command %q\n\n%s", args[0], strings.TrimRight(catalogUsage, "\n"))
	}
}

func (c cli) runRouting(args []string) error {
	if len(args) == 0 || isHelpArg(args[0]) {
		fmt.Fprint(c.stdout, routingUsage)
		return nil
	}
	if len(args) < 2 || args[0] != "preview" {
		return fmt.Errorf("usage: eyrie routing preview <model>")
	}
	out, err := setup.RoutingPreview(context.Background(), strings.Join(args[1:], " "))
	if err != nil {
		return err
	}
	fmt.Fprintln(c.stdout, out)
	return nil
}

func (c cli) runStatus(args []string) error {
	fs := newFlagSet("status")
	jsonOut := fs.Bool("json", false, "print machine-readable JSON")
	rest, err := c.parseFlags(fs, args, statusUsage)
	if err != nil {
		if errors.Is(err, errHelpShown) {
			return nil
		}
		return err
	}
	if len(rest) > 1 {
		return fmt.Errorf("usage: eyrie status [model] [--json]")
	}
	model := ""
	if len(rest) > 0 {
		model = strings.Join(rest, " ")
	}
	report, err := setup.DeploymentStatus(context.Background(), model)
	if err != nil {
		return err
	}
	if *jsonOut {
		return writeJSON(c.stdout, report)
	}
	fmt.Fprintln(c.stdout, setup.FormatStatus(report))
	return nil
}

func (c cli) runPreflight(args []string) error {
	fs := newFlagSet("preflight")
	jsonOut := fs.Bool("json", false, "print machine-readable JSON")
	rest, err := c.parseFlags(fs, args, preflightUsage)
	if err != nil {
		if errors.Is(err, errHelpShown) {
			return nil
		}
		return err
	}
	if len(rest) > 0 {
		return fmt.Errorf("usage: eyrie preflight [--json]")
	}
	report := runtime.Preflight(context.Background())
	if *jsonOut {
		if err := writeJSON(c.stdout, report); err != nil {
			return err
		}
	} else {
		fmt.Fprintln(c.stdout, runtime.FormatPreflightReport(report))
	}
	if !report.Ready {
		return fmt.Errorf("preflight failed — see the checks above")
	}
	return nil
}

func (c cli) runProviders(args []string) error {
	fs := newFlagSet("providers")
	jsonOut := fs.Bool("json", false, "print machine-readable JSON")
	rest, err := c.parseFlags(fs, args, providersUsage)
	if err != nil {
		if errors.Is(err, errHelpShown) {
			return nil
		}
		return err
	}
	if len(rest) > 0 {
		return fmt.Errorf("usage: eyrie providers [--json]")
	}

	providers := runtime.ListCredentialProviders()
	if *jsonOut {
		return writeJSON(c.stdout, providers)
	}

	active := strings.TrimSpace(runtime.ActiveProvider(context.Background()))
	tw := tabwriter.NewWriter(c.stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ACTIVE\tID\tMODE\tENV\tNAME")
	for _, p := range providers {
		marker := ""
		if p.ProviderID == active {
			marker = "*"
		}
		mode := "api-key"
		if !p.RequiresKey {
			mode = "local"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", marker, p.ProviderID, mode, p.EnvVar, p.DisplayName)
	}
	return tw.Flush()
}

func (c cli) runModels(args []string) error {
	fs := newFlagSet("models")
	source := fs.String("source", string(runtime.ListSourceCache), "model source: cache, live, or auto")
	refresh := fs.Bool("refresh", false, "refresh the catalog before listing")
	jsonOut := fs.Bool("json", false, "print machine-readable JSON")
	rest, err := c.parseFlags(fs, args, modelsUsage)
	if err != nil {
		if errors.Is(err, errHelpShown) {
			return nil
		}
		return err
	}

	providerID := ""
	if len(rest) > 0 {
		providerID = strings.TrimSpace(rest[0])
	}
	if providerID == "" {
		providerID = strings.TrimSpace(runtime.ActiveProvider(context.Background()))
	}
	if len(rest) > 1 {
		return fmt.Errorf("usage: eyrie models [provider] [--source cache|live|auto] [--refresh] [--json]")
	}
	if providerID == "" {
		return fmt.Errorf("provider required — pass one explicitly or run `eyrie select provider <id>`")
	}

	opts := runtime.ListModelsOpts{
		ProviderID: providerID,
		Refresh:    *refresh,
	}
	switch strings.TrimSpace(*source) {
	case "", string(runtime.ListSourceCache):
		opts.Source = runtime.ListSourceCache
	case string(runtime.ListSourceAuto):
		opts.Source = runtime.ListSourceAuto
	case string(runtime.ListSourceLive):
		opts.Source = runtime.ListSourceLive
	default:
		return fmt.Errorf("unsupported --source %q (expected cache, live, or auto)", *source)
	}

	entries, err := runtime.ListModels(context.Background(), opts)
	if err != nil {
		return err
	}
	if *jsonOut {
		return writeJSON(c.stdout, entries)
	}
	if len(entries) == 0 {
		fmt.Fprintf(c.stdout, "No models found for %s.\n", providerID)
		return nil
	}

	fmt.Fprintf(c.stdout, "Provider: %s\nModels: %d\n\n", providerID, len(entries))
	tw := tabwriter.NewWriter(c.stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tSOURCE\tCONTEXT\tINPUT $/1M\tOUTPUT $/1M")
	for _, entry := range entries {
		fmt.Fprintf(
			tw,
			"%s\t%s\t%d\t%s\t%s\n",
			entry.ID,
			entry.Source,
			entry.ContextWindow,
			formatPrice(entry.InputPricePer1M),
			formatPrice(entry.OutputPricePer1M),
		)
	}
	return tw.Flush()
}

func (c cli) runSelect(args []string) error {
	if len(args) == 0 || isHelpArg(args[0]) {
		fmt.Fprint(c.stdout, selectUsage)
		return nil
	}

	switch args[0] {
	case "provider":
		if len(args) != 2 {
			return fmt.Errorf("usage: eyrie select provider <provider-id>")
		}
		if err := runtime.SetActiveProvider(context.Background(), args[1]); err != nil {
			return err
		}
		fmt.Fprintf(c.stdout, "Active provider set to %s in %s\n", args[1], config.GetProviderConfigPath())
		return nil
	case "model":
		if len(args) < 2 {
			return fmt.Errorf("usage: eyrie select model <model-id>")
		}
		modelID := strings.Join(args[1:], " ")
		if err := runtime.SetActiveModel(context.Background(), modelID); err != nil {
			return err
		}
		fmt.Fprintf(c.stdout, "Active model set to %s in %s\n", modelID, config.GetProviderConfigPath())
		return nil
	case "clear":
		if len(args) != 1 {
			return fmt.Errorf("usage: eyrie select clear")
		}
		if err := runtime.ClearActiveSelection(context.Background()); err != nil {
			return err
		}
		fmt.Fprintf(c.stdout, "Active provider/model cleared in %s\n", config.GetProviderConfigPath())
		return nil
	default:
		return fmt.Errorf("unknown select target %q\n\n%s", args[0], strings.TrimRight(selectUsage, "\n"))
	}
}

func (c cli) parseFlags(fs *flag.FlagSet, args []string, usage string) ([]string, error) {
	var parseErr bytes.Buffer
	fs.SetOutput(&parseErr)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(c.stdout, usage)
			return nil, errHelpShown
		}
		msg := strings.TrimSpace(parseErr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("%s\n\n%s", msg, strings.TrimRight(usage, "\n"))
	}
	return fs.Args(), nil
}

type catalogStatus struct {
	CachePath     string    `json:"cache_path"`
	Exists        bool      `json:"exists"`
	ModifiedAt    time.Time `json:"modified_at,omitempty"`
	SizeBytes     int64     `json:"size_bytes,omitempty"`
	Models        int       `json:"models,omitempty"`
	Deployments   int       `json:"deployments,omitempty"`
	Offerings     int       `json:"offerings,omitempty"`
	Stale         bool      `json:"stale,omitempty"`
	UsingEmbedded bool      `json:"using_embedded"`
}

func catalogStatusReport(ctx context.Context) (catalogStatus, error) {
	path := catalog.DefaultCachePath()
	exists, mod, size, err := catalog.CacheInfo(path)
	if err != nil {
		return catalogStatus{}, err
	}

	report := catalogStatus{
		CachePath:  path,
		Exists:     exists,
		ModifiedAt: mod,
		SizeBytes:  size,
	}
	compiled, err := catalog.LoadCatalogV1(ctx, catalog.LoadCatalogV1Options{CachePath: path})
	if err != nil {
		return report, err
	}
	report.Models = len(compiled.ModelsByID)
	report.Deployments = len(compiled.DeploymentsByID)
	report.Offerings = len(compiled.OfferingsByID)
	report.Stale = time.Now().UTC().After(compiled.Catalog.StaleAfter)
	report.UsingEmbedded = !exists
	return report, nil
}

func (c cli) printCatalogStatus(report catalogStatus) {
	if !report.Exists {
		fmt.Fprintf(c.stdout, "Catalog cache: not found at %s (embedded catalog used at runtime)\n", report.CachePath)
		return
	}
	fmt.Fprintf(c.stdout, "Catalog cache: %s\n", report.CachePath)
	fmt.Fprintf(c.stdout, "  modified: %s (%d bytes)\n", report.ModifiedAt.UTC().Format(time.RFC3339), report.SizeBytes)
	fmt.Fprintf(c.stdout, "  models: %d  deployments: %d  offerings: %d\n", report.Models, report.Deployments, report.Offerings)
	if report.Stale {
		fmt.Fprintln(c.stdout, "  stale: yes — run `eyrie catalog discover`")
	}
}

func writeJSON(w io.Writer, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(data))
	return err
}

func openStore() (storage.Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("determine home directory: %w", err)
	}
	dir := filepath.Join(home, ".eyrie")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create eyrie directory: %w", err)
	}
	store, err := storage.Open(filepath.Join(dir, "conversations.db"))
	if err != nil {
		return nil, fmt.Errorf("open conversations store: %w", err)
	}
	return store, nil
}

func getProvider() client.Provider {
	cfg := config.LoadProviderConfig("")
	if setup.UseDeploymentRouting(cfg) {
		provider, err := setup.DeploymentProvider(context.Background(), cfg)
		if err == nil {
			return provider
		}
		fmt.Fprintf(os.Stderr, "warning: deployment routing unavailable, using legacy provider: %v\n", err)
	}
	detected := client.DetectProvider()
	c := client.Client(&client.EyrieConfig{Provider: detected})
	return client.NewTracingProvider(&clientProviderAdapter{c: c, provider: detected})
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

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.Usage = func() {}
	return fs
}

func formatPrice(value float64) string {
	if value == 0 {
		return "-"
	}
	return fmt.Sprintf("%.4f", value)
}

func isHelpArg(arg string) bool {
	return arg == "--help" || arg == "-h" || arg == "help"
}

func hasHelpArg(args []string) bool {
	return len(args) == 1 && isHelpArg(args[0])
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	runes := []rune(s)
	if len(runes) > n {
		return string(runes[:n]) + "..."
	}
	return s
}

const promptUsage = `Usage:
  eyrie prompt <message>
  eyrie prompt <node-id> <message>

Start a new conversation, or branch from an existing node.

Examples:
  eyrie prompt "Explain this repository"
  eyrie prompt 7f9a8c1e "Take a different route"
`

const listUsage = `Usage:
  eyrie ls

List root conversations stored in ~/.eyrie/conversations.db.
`

const showUsage = `Usage:
  eyrie show <node-id>

Show a conversation subtree from the specified node.
`

const deleteUsage = `Usage:
  eyrie rm <node-id>

Delete a node and all of its descendants from the conversation store.
`

const serveUsage = `Usage:
  eyrie serve [--port 8080]
  eyrie serve [--listen 127.0.0.1:8080]

Start the HTTP API server.

Environment:
  EYRIE_API_KEY                     Required by default for Bearer / X-API-Key auth
  EYRIE_ALLOW_INSECURE_PUBLIC_API   Set to true only for explicit local insecure use
`

const catalogUsage = `Usage:
  eyrie catalog refresh [--timeout 15s]
  eyrie catalog discover [--timeout 30s]
  eyrie catalog status [--json]

refresh   Fetch the published catalog cache only
discover  Fetch the published catalog and query live provider APIs
status    Show cache metadata and staleness
`

const routingUsage = `Usage:
  eyrie routing preview <model>

Print the effective deployment routing JSON for a model identifier.
`

const statusUsage = `Usage:
  eyrie status [model] [--json]

Show deployment routing status. When a model is provided, include route resolution details.
`

const preflightUsage = `Usage:
  eyrie preflight [--json]

Check catalog availability, credential store health, configured credentials, active model selection, and best-effort live model reachability.
`

const providersUsage = `Usage:
  eyrie providers [--json]

List supported provider gateways, the env var each one expects, and whether it is a local or API-key-backed provider.
`

const modelsUsage = `Usage:
  eyrie models [provider] [--source cache|live|auto] [--refresh] [--json]

List models for a provider. If provider is omitted, the active provider from provider.json is used.

Examples:
  eyrie models anthropic
  eyrie models openai --source live
  eyrie models --json
`

const selectUsage = `Usage:
  eyrie select provider <provider-id>
  eyrie select model <model-id>
  eyrie select clear

Update the active provider/model in provider.json.
`
