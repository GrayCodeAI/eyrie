package client

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// GuardrailType classifies a guardrail rule.
type GuardrailType string

const (
	// GuardrailPII detects personally identifiable information.
	GuardrailPII GuardrailType = "pii"
	// GuardrailPromptInjection detects prompt injection attempts in responses.
	GuardrailPromptInjection GuardrailType = "prompt_injection"
	// GuardrailHarmfulContent detects harmful or dangerous content patterns.
	GuardrailHarmfulContent GuardrailType = "harmful_content"
	// GuardrailSecretLeak detects leaked secrets, API keys, tokens, and passwords.
	GuardrailSecretLeak GuardrailType = "secret_leak"
	// GuardrailCustom is a user-defined rule with a custom pattern.
	GuardrailCustom GuardrailType = "custom"
)

// GuardrailAction determines what happens when a rule matches.
type GuardrailAction string

const (
	// GuardrailBlock prevents the response from being returned to the caller.
	GuardrailBlock GuardrailAction = "block"
	// GuardrailRedact replaces the matched content with a redaction marker.
	GuardrailRedact GuardrailAction = "redact"
	// GuardrailWarn allows the response but records the violation.
	GuardrailWarn GuardrailAction = "warn"
)

// GuardrailSeverity indicates how critical a violation is.
type GuardrailSeverity string

const (
	SeverityLow      GuardrailSeverity = "low"
	SeverityMedium   GuardrailSeverity = "medium"
	SeverityHigh     GuardrailSeverity = "high"
	SeverityCritical GuardrailSeverity = "critical"
)

// GuardrailRule defines a single guardrail check.
type GuardrailRule struct {
	Type     GuardrailType    `json:"type"`
	Name     string           `json:"name"`
	Pattern  string           `json:"pattern"`
	Action   GuardrailAction  `json:"action"`
	Severity GuardrailSeverity `json:"severity"`
	compiled *regexp.Regexp   // lazily compiled
}

// GuardrailViolation records a single rule match in the response.
type GuardrailViolation struct {
	Rule           GuardrailRule `json:"rule"`
	MatchedText    string        `json:"matched_text"`
	RedactedResult string        `json:"redacted_result,omitempty"`
}

// GuardrailError is returned when a guardrail blocks a response.
type GuardrailError struct {
	Violations []GuardrailViolation `json:"violations"`
	Message    string               `json:"message"`
}

func (e *GuardrailError) Error() string {
	return fmt.Sprintf("eyrie: guardrail blocked: %s (%d violation(s))", e.Message, len(e.Violations))
}

// Guardrails holds registered rules and runs them against LLM responses.
type Guardrails struct {
	mu    sync.RWMutex
	rules []GuardrailRule
}

// NewGuardrails creates a Guardrails instance with the given rules.
func NewGuardrails(rules ...GuardrailRule) *Guardrails {
	g := &Guardrails{}
	for _, r := range rules {
		g.AddRule(r)
	}
	return g
}

// AddRule registers a guardrail rule. It panics if the pattern is invalid.
func (g *Guardrails) AddRule(r GuardrailRule) {
	if r.Pattern != "" {
		compiled, err := regexp.Compile(r.Pattern)
		if err != nil {
			panic(fmt.Sprintf("eyrie: guardrails: invalid regex %q in rule %q: %v", r.Pattern, r.Name, err))
		}
		r.compiled = compiled
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rules = append(g.rules, r)
}

// Rules returns a snapshot of the currently registered rules.
func (g *Guardrails) Rules() []GuardrailRule {
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := make([]GuardrailRule, len(g.rules))
	copy(out, g.rules)
	return out
}

// Check evaluates all rules against the response text.
// It returns violations and an error only if a rule with Action=Block matches.
// For Redact actions, the redacted result is populated in the violation.
// For Warn actions, the violation is recorded but no error is returned.
func (g *Guardrails) Check(ctx context.Context, response string) ([]GuardrailViolation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	g.mu.RLock()
	rules := make([]GuardrailRule, len(g.rules))
	copy(rules, g.rules)
	g.mu.RUnlock()

	var violations []GuardrailViolation
	hasBlock := false

	for _, rule := range rules {
		if rule.compiled == nil {
			continue
		}
		matches := rule.compiled.FindAllString(response, -1)
		if len(matches) == 0 {
			continue
		}
		for _, match := range matches {
			v := GuardrailViolation{
				Rule:        rule,
				MatchedText: match,
			}
			if rule.Action == GuardrailRedact {
				v.RedactedResult = strings.Repeat("*", len(match))
			}
			violations = append(violations, v)
			if rule.Action == GuardrailBlock {
				hasBlock = true
			}
		}
	}

	if hasBlock {
		return violations, &GuardrailError{
			Violations: violations,
			Message:    "response blocked by guardrail",
		}
	}

	return violations, nil
}

// ApplyRedactions takes the response text and violations, replacing redacted
// matches with their redaction markers. Non-redact violations are left intact.
func ApplyRedactions(response string, violations []GuardrailViolation) string {
	result := response
	for _, v := range violations {
		if v.Rule.Action != GuardrailRedact {
			continue
		}
		result = strings.ReplaceAll(result, v.MatchedText, v.RedactedResult)
	}
	return result
}

// ---------------------------------------------------------------------------
// Built-in rule constructors
// ---------------------------------------------------------------------------

// DefaultPIIRules returns built-in rules for detecting PII in responses.
func DefaultPIIRules() []GuardrailRule {
	return []GuardrailRule{
		{
			Type:     GuardrailPII,
			Name:     "ssn",
			Pattern:  `\b\d{3}-\d{2}-\d{4}\b`,
			Action:   GuardrailRedact,
			Severity: SeverityCritical,
		},
		{
			Type:     GuardrailPII,
			Name:     "credit_card",
			Pattern:  `\b(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|3[47][0-9]{13}|3(?:0[0-5]|[68][0-9])[0-9]{11}|6(?:011|5[0-9]{2})[0-9]{12}|(?:2131|1800|35\d{3})\d{11})\b`,
			Action:   GuardrailRedact,
			Severity: SeverityCritical,
		},
		{
			Type:     GuardrailPII,
			Name:     "phone_number",
			Pattern:  `\b(?:\+?1[-.\s]?)?(?:\(?\d{3}\)?[-.\s]?)?\d{3}[-.\s]?\d{4}\b`,
			Action:   GuardrailRedact,
			Severity: SeverityMedium,
		},
	}
}

// DefaultSecretLeakRules returns built-in rules for detecting leaked secrets.
func DefaultSecretLeakRules() []GuardrailRule {
	return []GuardrailRule{
		{
			Type:     GuardrailSecretLeak,
			Name:     "api_key_generic",
			Pattern:  `(?i)(?:api[_-]?key|apikey)\s*[:=]\s*['"` + "`" + `]?([A-Za-z0-9_\-]{20,})['"` + "`" + `]?`,
			Action:   GuardrailBlock,
			Severity: SeverityCritical,
		},
		{
			Type:     GuardrailSecretLeak,
			Name:     "bearer_token",
			Pattern:  `(?i)bearer\s+[A-Za-z0-9_\-\.]{20,}`,
			Action:   GuardrailBlock,
			Severity: SeverityCritical,
		},
		{
			Type:     GuardrailSecretLeak,
			Name:     "password_assignment",
			Pattern:  `(?i)(?:password|passwd|pwd)\s*[:=]\s*['"` + "`" + `]?[^\s'"` + "`" + `]{8,}['"` + "`" + `]?`,
			Action:   GuardrailBlock,
			Severity: SeverityHigh,
		},
		{
			Type:     GuardrailSecretLeak,
			Name:     "aws_secret_key",
			Pattern:  `(?i)(?:aws_secret_access_key)\s*[:=]\s*['"` + "`" + `]?([A-Za-z0-9/+=]{40})['"` + "`" + `]?`,
			Action:   GuardrailBlock,
			Severity: SeverityCritical,
		},
		{
			Type:     GuardrailSecretLeak,
			Name:     "private_key_block",
			Pattern:  `-----BEGIN (?:RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`,
			Action:   GuardrailBlock,
			Severity: SeverityCritical,
		},
	}
}

// DefaultPromptInjectionRules returns built-in rules for detecting prompt injection in responses.
func DefaultPromptInjectionRules() []GuardrailRule {
	return []GuardrailRule{
		{
			Type:     GuardrailPromptInjection,
			Name:     "ignore_instructions",
			Pattern:  `(?i)ignore\s+(?:all\s+)?(?:previous|prior|above)\s+(?:instructions|prompts|rules|constraints)`,
			Action:   GuardrailBlock,
			Severity: SeverityHigh,
		},
		{
			Type:     GuardrailPromptInjection,
			Name:     "you_are_now",
			Pattern:  `(?i)you\s+are\s+now\s+(?:a|an|the)\s+\w+`,
			Action:   GuardrailWarn,
			Severity: SeverityMedium,
		},
		{
			Type:     GuardrailPromptInjection,
			Name:     "disregard_above",
			Pattern:  `(?i)disregard\s+(?:all\s+)?(?:the\s+)?(?:above|previous|prior)\s+(?:instructions|prompts|context)`,
			Action:   GuardrailBlock,
			Severity: SeverityHigh,
		},
		{
			Type:     GuardrailPromptInjection,
			Name:     "system_prompt_leak",
			Pattern:  `(?i)(?:reveal|show|print|output|display)\s+(?:your|the)\s+(?:system|initial)\s+(?:prompt|instructions|message)`,
			Action:   GuardrailBlock,
			Severity: SeverityHigh,
		},
		{
			Type:     GuardrailPromptInjection,
			Name:     "new_instructions",
			Pattern:  `(?i)\[(?:new|updated|revised)\s+(?:system\s+)?(?:instructions|rules|prompt)\]`,
			Action:   GuardrailBlock,
			Severity: SeverityHigh,
		},
	}
}

// DefaultHarmfulContentRules returns built-in rules for detecting harmful content patterns.
func DefaultHarmfulContentRules() []GuardrailRule {
	return []GuardrailRule{
		{
			Type:     GuardrailHarmfulContent,
			Name:     "bomb_making",
			Pattern:  `(?i)(?:how\s+to\s+make|instructions\s+for\s+(?:making|building))\s+(?:a\s+)?(?:bomb|explosive|detonator|grenade|ied)`,
			Action:   GuardrailBlock,
			Severity: SeverityCritical,
		},
		{
			Type:     GuardrailHarmfulContent,
			Name:     "synthesis_drugs",
			Pattern:  `(?i)(?:how\s+to\s+(?:synthesize|make|cook|manufacture))\s+(?:methamphetamine|cocaine|heroin|fentanyl|lsd|mdma|ecstasy)`,
			Action:   GuardrailBlock,
			Severity: SeverityCritical,
		},
		{
			Type:     GuardrailHarmfulContent,
			Name:     "harm_self",
			Pattern:  `(?i)(?:ways?\s+to\s+(?:hurt|harm|kill)\s+(?:your)?self|methods?\s+of\s+suicide)`,
			Action:   GuardrailBlock,
			Severity: SeverityCritical,
		},
		{
			Type:     GuardrailHarmfulContent,
			Name:     "weapon_instructions",
			Pattern:  `(?i)(?:step[- ]by[- ]step|detailed)\s+(?:instructions?|guide)\s+(?:for\s+)?(?:building|making|assembling)\s+(?:a\s+)?(?:firearm|gun|weapon|silencer|suppressor)`,
			Action:   GuardrailBlock,
			Severity: SeverityCritical,
		},
	}
}

// ---------------------------------------------------------------------------
// Rule lookup helpers
// ---------------------------------------------------------------------------

// AllDefaultRules returns all built-in guardrail rules (PII, secrets,
// prompt injection, and harmful content).
func AllDefaultRules() []GuardrailRule {
	var rules []GuardrailRule
	rules = append(rules, DefaultPIIRules()...)
	rules = append(rules, DefaultSecretLeakRules()...)
	rules = append(rules, DefaultPromptInjectionRules()...)
	rules = append(rules, DefaultHarmfulContentRules()...)
	return rules
}

// RulesForType returns the built-in rules for a single GuardrailType.
func RulesForType(t GuardrailType) []GuardrailRule {
	switch t {
	case GuardrailPII:
		return DefaultPIIRules()
	case GuardrailSecretLeak:
		return DefaultSecretLeakRules()
	case GuardrailPromptInjection:
		return DefaultPromptInjectionRules()
	case GuardrailHarmfulContent:
		return DefaultHarmfulContentRules()
	default:
		return nil
	}
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// applyGuardrails runs guardrail checks on the response and applies redactions.
// This is called by provider Chat methods after receiving the LLM response.
func applyGuardrails(ctx context.Context, resp *EyrieResponse, g *Guardrails) error {
	if g == nil || resp == nil || resp.Content == "" {
		return nil
	}
	violations, err := g.Check(ctx, resp.Content)
	if err != nil {
		return err
	}
	if len(violations) > 0 {
		resp.Content = ApplyRedactions(resp.Content, violations)
	}
	return nil
}

// ---------------------------------------------------------------------------
// GuardrailProvider: standalone Provider wrapper
// ---------------------------------------------------------------------------

// GuardrailProvider wraps any Provider and runs guardrail checks on LLM
// responses before returning them to the caller. When a guardrail with
// Action=Block matches, the response is replaced with an error. When
// Action=Redact, matched content is scrubbed. When Action=Warn, the
// violation is logged but the response passes through unchanged.
//
// GuardrailProvider is safe for concurrent use.
type GuardrailProvider struct {
	inner      Provider
	guardrails *Guardrails
}

// Compile-time check that GuardrailProvider implements Provider.
var _ Provider = (*GuardrailProvider)(nil)

// NewGuardrailProvider wraps the given provider with output guardrails.
// The inner provider must not be nil. The guardrails parameter may be nil
// (in which case the wrapper is a no-op).
func NewGuardrailProvider(inner Provider, g *Guardrails) *GuardrailProvider {
	if inner == nil {
		panic("eyrie: NewGuardrailProvider inner provider must not be nil")
	}
	return &GuardrailProvider{
		inner:      inner,
		guardrails: g,
	}
}

// Name returns the inner provider name suffixed with "/guardrails".
func (gp *GuardrailProvider) Name() string {
	return gp.inner.Name() + "/guardrails"
}

// Ping delegates to the inner provider.
func (gp *GuardrailProvider) Ping(ctx context.Context) error {
	return gp.inner.Ping(ctx)
}

// Inner returns the wrapped provider.
func (gp *GuardrailProvider) Inner() Provider {
	return gp.inner
}

// Chat sends a chat request and validates the response against guardrails.
func (gp *GuardrailProvider) Chat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*EyrieResponse, error) {
	resp, err := gp.inner.Chat(ctx, messages, opts)
	if err != nil {
		return nil, err
	}
	if err := applyGuardrails(ctx, resp, gp.guardrails); err != nil {
		return nil, err
	}
	return resp, nil
}

// StreamChat sends a streaming request and validates content events.
// Blocked violations cause the stream to be cancelled and an error event emitted.
// Redactions are applied to individual content chunks.
func (gp *GuardrailProvider) StreamChat(ctx context.Context, messages []EyrieMessage, opts ChatOptions) (*StreamResult, error) {
	result, err := gp.inner.StreamChat(ctx, messages, opts)
	if err != nil {
		return nil, err
	}

	if gp.guardrails == nil {
		return result, nil
	}

	origEvents := result.Events
	wrappedEvents := make(chan EyrieStreamEvent, cap(origEvents))

	go func() {
		defer close(wrappedEvents)
		for evt := range origEvents {
			if evt.Type == "content" && gp.guardrails != nil {
				violations, checkErr := gp.guardrails.Check(ctx, evt.Content)
				if checkErr != nil {
					select {
					case wrappedEvents <- EyrieStreamEvent{
						Type:  "error",
						Error: checkErr.Error(),
					}:
					case <-ctx.Done():
					}
					result.Close()
					return
				}
				if len(violations) > 0 {
					evt.Content = ApplyRedactions(evt.Content, violations)
				}
			}
			select {
			case wrappedEvents <- evt:
			case <-ctx.Done():
				result.Close()
				return
			}
		}
	}()

	return &StreamResult{
		Events:    wrappedEvents,
		RequestID: result.RequestID,
	}, nil
}
