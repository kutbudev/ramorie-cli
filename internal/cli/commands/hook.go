package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kutbudev/ramorie-cli/internal/api"
	mcpcontext "github.com/kutbudev/ramorie-cli/internal/mcp"
	"github.com/kutbudev/ramorie-cli/internal/protocol"
	"github.com/urfave/cli/v2"
)

// NewHookCommand exposes `ramorie hook install|uninstall|status|context`.
// It drives Claude Code PreToolUse integration so that editing a file
// automatically surfaces relevant memories/decisions into the model's
// context — no manual recall() needed.
//
// DEPRECATED (v7.1.0): prefer `ramorie setup-hooks install` (or the new
// `ramorie setup hooks install` alias) which covers Claude Code, Codex,
// Cursor, and Windsurf in one command. This single-client legacy command
// is kept for backward compatibility with existing scripts.
func NewHookCommand() *cli.Command {
	return &cli.Command{
		Name:  "hook",
		Usage: "[legacy — prefer `ramorie setup-hooks install`] Manage Claude Code integration (PreToolUse hook)",
		Description: "DEPRECATED in v7.1.0: prefer `ramorie setup-hooks install` (covers Claude Code,\n" +
			"   Codex, Cursor, and Windsurf in one call). This single-client command is kept\n" +
			"   for scripts that pinned to the older surface.",
		Subcommands: []*cli.Command{
			{
				Name:   "install",
				Usage:  "Install the PreToolUse hook into ~/.claude/settings.json",
				Action: hookInstall,
			},
			{
				Name:   "uninstall",
				Usage:  "Remove the Ramorie hook from ~/.claude/settings.json",
				Action: hookUninstall,
			},
			{
				Name:   "status",
				Usage:  "Check whether the hook is installed and wired correctly",
				Action: hookStatus,
			},
			{
				Name:   "session-start",
				Usage:  "Hook shim: emit SessionStart protocol plus live Ramorie startup context",
				Hidden: true,
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "full", Usage: "Include larger context injection payload"},
				},
				Action: hookSessionStart,
			},
			{
				Name:   "before-action",
				Usage:  "Hook shim: surface before-action runbooks for build/test/deploy commands",
				Hidden: true,
				Flags: []cli.Flag{
					&cli.IntFlag{Name: "budget", Value: 1200},
					&cli.IntFlag{Name: "limit", Value: 3},
					&cli.StringFlag{Name: "project", Usage: "Optional project name/UUID override"},
				},
				Action: hookBeforeAction,
			},
			{
				Name:   "context",
				Usage:  "Hook shim: read PreToolUse JSON from stdin, emit system-reminder",
				Hidden: true, // called from the shim, not humans
				Flags: []cli.Flag{
					&cli.IntFlag{Name: "budget", Value: 500},
					&cli.IntFlag{Name: "limit", Value: 2},
				},
				Action: hookContext,
			},
		},
	}
}

const (
	hookMatcher                 = "Edit|Write|Read"
	beforeActionHookMatcher     = "Bash|Shell"
	hookIdentifier              = "ramorie-autocontext"
	beforeActionHookIdentifier  = "ramorie-before-action-runbook"
	beforeActionQueryCandidateN = 8
	beforeActionMaxCommandRunes = 180
	beforeActionMaxRunbookRunes = 2200
	hookCooldownSecs            = 30
)

type claudeSettings struct {
	Hooks map[string][]hookGroup `json:"hooks,omitempty"`
	// Preserve unknown fields so we don't clobber user config.
	Rest map[string]json.RawMessage `json:"-"`
}

type hookGroup struct {
	Matcher string     `json:"matcher,omitempty"`
	Hooks   []hookSpec `json:"hooks"`
}

type hookSpec struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	// Marker field we set so we can identify our own entries on uninstall.
	ID string `json:"id,omitempty"`
}

func claudeSettingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "settings.json"), nil
}

func ramorieBinary() string {
	if exe, err := os.Executable(); err == nil {
		return exe
	}
	return "ramorie"
}

func loadSettings(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]interface{}{}, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return map[string]interface{}{}, nil
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if raw == nil {
		return map[string]interface{}{}, nil
	}
	return raw, nil
}

func saveSettings(path string, raw map[string]interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func hookInstall(c *cli.Context) error {
	path, err := claudeSettingsPath()
	if err != nil {
		return err
	}
	raw, err := loadSettings(path)
	if err != nil {
		return err
	}

	bin := ramorieBinary()
	contextEntry := map[string]interface{}{
		"type":    "command",
		"command": fmt.Sprintf("%s hook context --budget 500 --limit 2", bin),
		"id":      hookIdentifier,
	}
	beforeActionEntry := map[string]interface{}{
		"type":    "command",
		"command": fmt.Sprintf("%s hook before-action --budget 1200 --limit 3", bin),
		"id":      beforeActionHookIdentifier,
	}

	hooks, _ := raw["hooks"].(map[string]interface{})
	if hooks == nil {
		hooks = map[string]interface{}{}
	}
	preUse, _ := hooks["PreToolUse"].([]interface{})

	// Remove any prior ramorie entry so reinstalling is idempotent.
	preUse = pruneHookEntries(preUse, hookIdentifier)
	preUse = pruneHookEntries(preUse, beforeActionHookIdentifier)

	preUse = append(preUse, map[string]interface{}{
		"matcher": hookMatcher,
		"hooks":   []interface{}{contextEntry},
	})
	preUse = append(preUse, map[string]interface{}{
		"matcher": beforeActionHookMatcher,
		"hooks":   []interface{}{beforeActionEntry},
	})
	hooks["PreToolUse"] = preUse
	raw["hooks"] = hooks

	if err := saveSettings(path, raw); err != nil {
		return err
	}
	fmt.Printf("✅ Installed PreToolUse hook into %s\n", path)
	fmt.Printf("   Matcher: %s\n", hookMatcher)
	fmt.Printf("   Command: %s hook context ...\n", bin)
	fmt.Printf("   Matcher: %s\n", beforeActionHookMatcher)
	fmt.Printf("   Command: %s hook before-action ...\n", bin)
	fmt.Println("   Restart Claude Code to activate.")
	return nil
}

func hookUninstall(c *cli.Context) error {
	path, err := claudeSettingsPath()
	if err != nil {
		return err
	}
	raw, err := loadSettings(path)
	if err != nil {
		return err
	}
	hooks, _ := raw["hooks"].(map[string]interface{})
	if hooks == nil {
		fmt.Println("No hooks configured — nothing to remove.")
		return nil
	}
	preUse, _ := hooks["PreToolUse"].([]interface{})
	before := len(preUse)
	preUse = pruneHookEntries(preUse, hookIdentifier)
	preUse = pruneHookEntries(preUse, beforeActionHookIdentifier)
	if len(preUse) == 0 {
		delete(hooks, "PreToolUse")
	} else {
		hooks["PreToolUse"] = preUse
	}
	if len(hooks) == 0 {
		delete(raw, "hooks")
	} else {
		raw["hooks"] = hooks
	}
	if err := saveSettings(path, raw); err != nil {
		return err
	}
	removed := before - len(preUse)
	fmt.Printf("✅ Removed %d hook entrie(s) from %s\n", removed, path)
	return nil
}

func hookStatus(c *cli.Context) error {
	path, err := claudeSettingsPath()
	if err != nil {
		return err
	}
	raw, err := loadSettings(path)
	if err != nil {
		return err
	}
	hooks, _ := raw["hooks"].(map[string]interface{})
	preUse, _ := hooks["PreToolUse"].([]interface{})
	found := 0
	for _, entry := range preUse {
		group, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		inner, _ := group["hooks"].([]interface{})
		for _, h := range inner {
			hmap, ok := h.(map[string]interface{})
			if !ok {
				continue
			}
			id, _ := hmap["id"].(string)
			if id == hookIdentifier || id == beforeActionHookIdentifier {
				found++
				fmt.Printf("✅ Ramorie hook installed\n")
				fmt.Printf("   Path:    %s\n", path)
				fmt.Printf("   Matcher: %v\n", group["matcher"])
				fmt.Printf("   Command: %v\n", hmap["command"])
			}
		}
	}
	if found == 0 {
		fmt.Printf("❌ Ramorie hook not installed.\n")
		fmt.Println("   Run: ramorie hook install")
	}
	return nil
}

func hookSessionStart(c *cli.Context) error {
	additional := protocol.SessionStartText

	client := api.NewClient()
	if strings.TrimSpace(client.APIKey) != "" {
		// Source the agent identity so per-agent agent_policy is injected at
		// startup. The hook runs in a separate process from the MCP server, so
		// prefer an explicit env override, then fall back to the agent name the
		// MCP server persisted via setup_agent.
		agentName := strings.TrimSpace(os.Getenv("RAMORIE_AGENT_NAME"))
		if agentName == "" {
			agentName = mcpcontext.ReadPersistedAgentName()
		}
		if ctx, err := mcpcontext.BuildSessionStartContext(client, c.Bool("full"), agentName); err == nil && len(ctx) > 0 {
			if b, err := json.MarshalIndent(ctx, "", "  "); err == nil {
				additional += "\n\n<ramorie_startup_context>\n"
				additional += string(b)
				additional += "\n</ramorie_startup_context>"
			}
		}
	}

	return emitHookAdditionalContext("SessionStart", additional)
}

// pruneHookEntries removes any PreToolUse groups whose inner hook matches
// the given identifier. Preserves foreign (non-ramorie) entries untouched.
func pruneHookEntries(preUse []interface{}, identifier string) []interface{} {
	filtered := make([]interface{}, 0, len(preUse))
	for _, entry := range preUse {
		group, ok := entry.(map[string]interface{})
		if !ok {
			filtered = append(filtered, entry)
			continue
		}
		inner, _ := group["hooks"].([]interface{})
		kept := make([]interface{}, 0, len(inner))
		for _, h := range inner {
			hmap, ok := h.(map[string]interface{})
			if !ok {
				kept = append(kept, h)
				continue
			}
			if hmap["id"] == identifier {
				continue // drop ramorie-owned entry
			}
			kept = append(kept, h)
		}
		if len(kept) == 0 {
			continue // drop group if it becomes empty
		}
		group["hooks"] = kept
		filtered = append(filtered, group)
	}
	return filtered
}

// hookContext is invoked by Claude Code as a PreToolUse shim. Reads the
// tool-call JSON from stdin, extracts a file path, calls the backend surface
// endpoint and writes a Claude Code compatible hook response on stdout.
//
// Output schema (per Claude Code hooks spec):
//
//	{"hookSpecificOutput": {"hookEventName":"PreToolUse","additionalContext":"..."}}
//
// Any failure is silent (exit 0, empty output) so hook errors never block
// the user's tool call.
func hookContext(c *cli.Context) error {
	payload := map[string]interface{}{}
	dec := json.NewDecoder(os.Stdin)
	_ = dec.Decode(&payload) // non-fatal; empty stdin is fine

	filePath := extractFilePathFromPayload(payload)
	if filePath == "" {
		return nil
	}

	// Cooldown: don't repeat the same file within 30s. File mtime is cheap.
	if wasRecentlyProcessed(filePath) {
		return nil
	}
	markProcessed(filePath)

	// Shell out to `ramorie hook-context-call` via the find-related helper so
	// this function stays focused on I/O shape.
	budget := c.Int("budget")
	limit := c.Int("limit")
	cmd := exec.Command(ramorieBinary(), "find-related",
		"--file", filePath,
		"--budget", fmt.Sprintf("%d", budget),
		"--limit", fmt.Sprintf("%d", limit))
	out, err := cmd.Output()
	if err != nil {
		return nil // silent
	}
	additional := strings.TrimSpace(string(out))
	if additional == "" {
		return nil
	}

	return emitHookAdditionalContext("PreToolUse", additional)
}

type beforeActionIntent struct {
	Key      string
	Label    string
	Terms    []string
	Evidence []string
}

type beforeActionRunbook struct {
	ID      string
	Name    string
	Trigger string
	Body    string
	Preview string
}

func hookBeforeAction(c *cli.Context) error {
	payload := map[string]interface{}{}
	dec := json.NewDecoder(os.Stdin)
	_ = dec.Decode(&payload) // hook failures must never block the tool call

	actionText := extractActionTextFromPayload(payload)
	if actionText == "" {
		return nil
	}
	intents := classifyBeforeActionIntents(actionText)
	if len(intents) == 0 {
		return nil
	}

	cooldownKey := "before-action:" + beforeActionIntentKeys(intents) + ":" + actionText
	if wasRecentlyProcessed(cooldownKey) {
		return nil
	}
	markProcessed(cooldownKey)

	client := api.NewClient()
	if strings.TrimSpace(client.APIKey) == "" {
		return nil
	}

	project := strings.TrimSpace(c.String("project"))
	projectHint := ""
	if project == "" {
		if cwd, err := os.Getwd(); err == nil {
			projectHint = filepath.Base(cwd)
		}
	}

	resp, err := client.FindMemories(api.FindMemoriesOptions{
		Term:             buildBeforeActionQuery(intents, actionText),
		Project:          project,
		ProjectHint:      projectHint,
		Types:            []string{"skill", "preference"},
		Limit:            beforeActionQueryCandidateN,
		BudgetTokens:     900,
		IncludeDecisions: false,
		Purpose:          "coding",
		Intent:           "how_to",
		HyDE:             "off",
		Rerank:           "off",
		FastMode:         true,
	})
	if err != nil || resp == nil || len(resp.Items) == 0 {
		return nil
	}

	// Split the candidate set by type: skills feed the existing (trigger-gated)
	// runbook pipeline untouched; preferences get a separate, tightly-gated
	// surfacing so an unrelated rule never spams an unrelated command.
	skillItems, prefItems := splitBeforeActionItems(resp.Items)

	runbooks := loadBeforeActionRunbooks(client, skillItems, intents, c.Int("limit"))
	preferences := selectBeforeActionPreferences(prefItems, intents)
	if len(runbooks) == 0 && len(preferences) == 0 {
		return nil
	}

	additional := formatBeforeActionRunbooks(actionText, intents, runbooks, preferences, c.Int("budget"))
	if strings.TrimSpace(additional) == "" {
		return nil
	}
	return emitHookAdditionalContext("PreToolUse", additional)
}

func emitHookAdditionalContext(eventName, additional string) error {
	resp := map[string]interface{}{
		"hookSpecificOutput": map[string]interface{}{
			"hookEventName":     eventName,
			"additionalContext": additional,
		},
	}
	enc := json.NewEncoder(os.Stdout)
	_ = enc.Encode(resp)
	return nil
}

func extractActionTextFromPayload(p map[string]interface{}) string {
	var parts []string
	appendPayloadString(&parts, p["tool_name"])
	appendPayloadString(&parts, p["command"])
	appendPayloadString(&parts, p["description"])

	ti, _ := p["tool_input"].(map[string]interface{})
	if ti != nil {
		for _, key := range []string{"command", "cmd", "script", "description", "prompt", "input"} {
			appendPayloadString(&parts, ti[key])
		}
	}

	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func appendPayloadString(parts *[]string, v interface{}) {
	switch x := v.(type) {
	case string:
		if s := strings.TrimSpace(x); s != "" {
			*parts = append(*parts, s)
		}
	case []interface{}:
		for _, item := range x {
			appendPayloadString(parts, item)
		}
	}
}

func classifyBeforeActionIntents(text string) []beforeActionIntent {
	lower := strings.ToLower(text)
	var out []beforeActionIntent

	if containsAny(lower, "xcodebuild", "react-native run-ios", "expo run:ios", "eas build --platform ios", "pod install", "npx pod-install") ||
		(strings.Contains(lower, "ios") && containsAny(lower, " build", " run", "test", "archive")) {
		out = append(out, beforeActionIntent{
			Key:   "ios-build",
			Label: "iOS build/run",
			Terms: []string{"before:ios-build", "ios build", "xcodebuild", "pod install", "swift", "compatibility"},
		})
	}

	if containsAny(lower, "gradlew", "react-native run-android", "expo run:android", "eas build --platform android") ||
		(strings.Contains(lower, "android") && containsAny(lower, " build", " run", "test", "assemble")) {
		out = append(out, beforeActionIntent{
			Key:   "android-build",
			Label: "Android build/run",
			Terms: []string{"before:android-build", "android build", "gradle", "gradlew", "assemble"},
		})
	}

	if containsAny(lower, "eas build", "expo prebuild", "expo run", "react-native run") {
		out = append(out, beforeActionIntent{
			Key:   "mobile-build",
			Label: "mobile build/run",
			Terms: []string{"before:mobile-build", "mobile build", "expo", "react native", "eas build"},
		})
	}

	if containsAny(lower, "docker build", "docker compose build", "docker-compose build") {
		out = append(out, beforeActionIntent{
			Key:   "docker-build",
			Label: "Docker build",
			Terms: []string{"before:docker-build", "docker build", "dockerfile", "container build"},
		})
	}

	if containsAny(lower, "railway up", "railway deploy", "vercel deploy", "fly deploy", "render deploy") ||
		(strings.Contains(lower, " deploy") && !strings.Contains(lower, "deployment")) {
		out = append(out, beforeActionIntent{
			Key:   "deploy",
			Label: "deploy",
			Terms: []string{"before:deploy", "deploy", "railway", "vercel", "release"},
		})
	}

	if containsAny(lower, "go test", "yarn test", "pnpm test", "npm test", "pytest", "vitest", "jest", "playwright test", "cargo test", "swift test") {
		evidence := testCommandEvidence(lower)
		out = append(out, beforeActionIntent{
			Key:      "test",
			Label:    "test",
			Terms:    []string{"before:test", "test", "verification", "e2e", "unit test"},
			Evidence: evidence,
		})
	}

	if containsAny(lower, "goose ", "prisma migrate", "supabase db", "db:migrate", "migrate up", "migrate down") {
		out = append(out, beforeActionIntent{
			Key:   "migration",
			Label: "database migration",
			Terms: []string{"before:migration", "migration", "database", "schema", "goose", "prisma"},
		})
	}

	if containsAny(lower, "yarn install", "yarn add", "pnpm install", "pnpm add", "npm install", "npm add", "bundle install", "pod install") {
		out = append(out, beforeActionIntent{
			Key:   "install",
			Label: "dependency install",
			Terms: []string{"before:install", "dependency install", "package manager", "yarn", "pnpm", "npm", "pods"},
		})
	}

	if containsAny(lower, "git commit", "git push") {
		out = append(out, beforeActionIntent{
			Key:   "git-publish",
			Label: "git commit/push",
			Terms: []string{"before:git-publish", "commit", "push", "pre-commit", "verification"},
		})
	}

	return dedupeBeforeActionIntents(out)
}

func containsAny(s string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(s, needle) {
			return true
		}
	}
	return false
}

func dedupeBeforeActionIntents(in []beforeActionIntent) []beforeActionIntent {
	seen := map[string]struct{}{}
	out := make([]beforeActionIntent, 0, len(in))
	for _, intent := range in {
		if _, ok := seen[intent.Key]; ok {
			continue
		}
		seen[intent.Key] = struct{}{}
		out = append(out, intent)
	}
	return out
}

func beforeActionIntentKeys(intents []beforeActionIntent) string {
	keys := make([]string, 0, len(intents))
	for _, intent := range intents {
		keys = append(keys, intent.Key)
	}
	return strings.Join(keys, ",")
}

func buildBeforeActionQuery(intents []beforeActionIntent, actionText string) string {
	parts := []string{"before-action", "runbook", "checklist", "type:skill"}
	seen := map[string]struct{}{}
	for _, intent := range intents {
		for _, term := range append([]string{intent.Key, intent.Label}, intent.Terms...) {
			term = strings.TrimSpace(term)
			if term == "" {
				continue
			}
			if _, ok := seen[term]; ok {
				continue
			}
			seen[term] = struct{}{}
			parts = append(parts, term)
		}
	}
	if clipped := clipRunes(actionText, beforeActionMaxCommandRunes); clipped != "" {
		parts = append(parts, clipped)
	}
	return strings.Join(parts, " ")
}

// splitBeforeActionItems separates retrieval candidates into skills and
// preferences by type. Anything that is not explicitly a preference stays in
// the skill bucket so the existing runbook pipeline behaves exactly as before.
func splitBeforeActionItems(items []api.FindItem) (skills, prefs []api.FindItem) {
	for _, it := range items {
		if strings.EqualFold(strings.TrimSpace(it.Type), "preference") {
			prefs = append(prefs, it)
			continue
		}
		skills = append(skills, it)
	}
	return skills, prefs
}

// selectBeforeActionPreferences surfaces only the active preferences whose text
// is relevant to the detected command family. Preferences carry no before:*
// trigger, so they are gated purely on command-family term overlap and capped
// hard — this hook has a history of false-positive spam, so an unrelated rule
// must never attach to a build/test/deploy/install command.
func selectBeforeActionPreferences(items []api.FindItem, intents []beforeActionIntent) []string {
	if len(items) == 0 || len(intents) == 0 {
		return nil
	}
	const maxPrefs = 2
	out := make([]string, 0, maxPrefs)
	seen := map[string]struct{}{}
	for _, it := range items {
		if len(out) >= maxPrefs {
			break
		}
		haystack := strings.ToLower(strings.Join([]string{it.Title, it.Preview}, "\n"))
		if !beforeActionPreferenceRelevant(haystack, intents) {
			continue
		}
		text := strings.TrimSpace(firstUsefulLine(it.Preview))
		if text == "" {
			text = strings.TrimSpace(it.Title)
		}
		if text == "" {
			continue
		}
		key := strings.ToLower(text)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, clipInlineRunes(text, 160))
	}
	return out
}

// beforeActionPreferenceRelevant requires a concrete command-family term from
// the detected intent (e.g. "yarn", "npm", "docker build", "gradle", "deploy")
// to appear in the preference text. The "before:*" trigger tokens are skipped
// since preferences never carry them — this keeps gating as strict as the
// no-trigger skill path and avoids generic-word spam.
func beforeActionPreferenceRelevant(haystack string, intents []beforeActionIntent) bool {
	for _, intent := range intents {
		for _, term := range intent.Terms {
			term = strings.ToLower(strings.TrimSpace(term))
			if term == "" || strings.HasPrefix(term, "before:") {
				continue
			}
			if strings.Contains(haystack, term) {
				return true
			}
		}
	}
	return false
}

func loadBeforeActionRunbooks(client *api.Client, items []api.FindItem, intents []beforeActionIntent, limit int) []beforeActionRunbook {
	if limit <= 0 {
		limit = 3
	}
	seen := map[string]struct{}{}
	runbooks := make([]beforeActionRunbook, 0, limit)
	for _, item := range items {
		if len(runbooks) >= limit {
			break
		}
		if item.ID == "" {
			continue
		}
		if _, ok := seen[item.ID]; ok {
			continue
		}
		seen[item.ID] = struct{}{}

		rb := beforeActionRunbook{
			ID:      item.ID,
			Name:    item.Title,
			Preview: item.Preview,
		}
		if rendered, err := client.LoadSkill(item.ID); err == nil && rendered != nil {
			if rendered.Skill.Name != "" {
				rb.Name = rendered.Skill.Name
			}
			rb.Trigger = rendered.Skill.Trigger
			rb.Body = rendered.Body
		}
		if strings.TrimSpace(rb.Body) == "" {
			rb.Body = item.Preview
		}
		if !beforeActionRunbookLooksRelevant(item, rb, intents) {
			continue
		}
		runbooks = append(runbooks, rb)
	}
	return runbooks
}

func beforeActionRunbookLooksRelevant(item api.FindItem, rb beforeActionRunbook, intents []beforeActionIntent) bool {
	trigger := strings.ToLower(strings.TrimSpace(rb.Trigger))
	haystack := strings.ToLower(strings.Join([]string{
		item.Title,
		item.Preview,
		item.Kind,
		rb.Name,
		rb.Trigger,
		rb.Body,
	}, "\n"))
	for _, intent := range intents {
		intentKey := strings.ToLower(intent.Key)
		if (strings.Contains(trigger, "before:"+intentKey) || trigger == intentKey) && beforeActionIntentEvidenceMatches(intent, haystack) {
			return true
		}
		for _, term := range intent.Terms {
			term = strings.ToLower(strings.TrimSpace(term))
			if term == "" {
				continue
			}
			if strings.Contains(trigger, term) && beforeActionIntentEvidenceMatches(intent, haystack) {
				return true
			}
		}
	}
	// A structured skill with a trigger is an explicit before-action contract.
	// If that trigger did not match the detected intent, do not let generic
	// words buried in the body (e.g. "Validation") pull it into an unrelated
	// build/test/deploy command.
	if trigger != "" {
		return false
	}
	for _, intent := range intents {
		for _, term := range intent.Terms {
			term = strings.ToLower(strings.TrimSpace(term))
			if term != "" && strings.Contains(haystack, term) {
				return true
			}
		}
	}
	return false
}

func testCommandEvidence(lower string) []string {
	switch {
	case strings.Contains(lower, "go test"):
		return []string{"go test", "golang", "go toolchain"}
	case containsAny(lower, "yarn test", "pnpm test", "npm test", "vitest", "jest", "playwright test"):
		return []string{"yarn test", "pnpm test", "npm test", "vitest", "jest", "playwright", "node", "javascript", "typescript"}
	case strings.Contains(lower, "pytest"):
		return []string{"pytest", "python"}
	case strings.Contains(lower, "cargo test"):
		return []string{"cargo test", "rust"}
	case strings.Contains(lower, "swift test"):
		return []string{"swift test", "swift"}
	default:
		return nil
	}
}

func beforeActionIntentEvidenceMatches(intent beforeActionIntent, haystack string) bool {
	if intent.Key != "test" || len(intent.Evidence) == 0 {
		return true
	}
	for _, evidence := range intent.Evidence {
		evidence = strings.ToLower(strings.TrimSpace(evidence))
		if evidence != "" && strings.Contains(haystack, evidence) {
			return true
		}
	}
	return false
}

func formatBeforeActionRunbooks(command string, intents []beforeActionIntent, runbooks []beforeActionRunbook, preferences []string, budgetTokens int) string {
	if len(runbooks) == 0 && len(preferences) == 0 {
		return ""
	}
	if budgetTokens <= 0 {
		budgetTokens = 1200
	}

	labels := make([]string, 0, len(intents))
	for _, intent := range intents {
		labels = append(labels, intent.Label)
	}

	maxChars := budgetTokens * 4
	if maxChars < 1200 {
		maxChars = 1200
	}
	var b strings.Builder
	b.WriteString("Ramorie BEFORE-ACTION RUNBOOK\n")
	b.WriteString("Detected intent: ")
	b.WriteString(strings.Join(labels, ", "))
	b.WriteString("\n")
	if clipped := clipRunes(command, beforeActionMaxCommandRunes); clipped != "" {
		b.WriteString("Command: ")
		b.WriteString(clipped)
		b.WriteString("\n")
	}
	if len(runbooks) > 0 {
		b.WriteString("Apply the relevant checklist before running this command. If it is stale or unsafe, verify first; do not silently skip it.\n")
	}

	for i, rb := range runbooks {
		if b.Len() >= maxChars {
			break
		}
		b.WriteString("\n")
		fmt.Fprintf(&b, "Runbook %d: %s\n", i+1, friendlyRunbookName(rb))
		if strings.TrimSpace(rb.Trigger) != "" {
			b.WriteString("Trigger: ")
			b.WriteString(strings.TrimSpace(rb.Trigger))
			b.WriteString("\n")
		}
		body := clipRunes(formatRunbookChecklist(rb.Body), beforeActionMaxRunbookRunes)
		if body != "" {
			b.WriteString(body)
			b.WriteString("\n")
		}
	}

	// Compact, relevance-gated user preferences (e.g. "always use yarn"). One
	// short line each so they never dominate the runbook payload.
	if len(preferences) > 0 && b.Len() < maxChars {
		b.WriteString("\nActive preferences (apply if relevant):\n")
		for _, p := range preferences {
			if b.Len() >= maxChars {
				break
			}
			b.WriteString("- ")
			b.WriteString(p)
			b.WriteString("\n")
		}
	}

	return clipRunes(b.String(), maxChars)
}

func friendlyRunbookName(rb beforeActionRunbook) string {
	name := strings.TrimSpace(rb.Name)
	if name == "" || looksLikeGeneratedSlug(name) {
		if fallback := firstUsefulLine(rb.Preview); fallback != "" {
			name = fallback
		} else if fallback := firstUsefulLine(stripFrontMatter(rb.Body)); fallback != "" {
			name = fallback
		}
	}
	name = strings.TrimPrefix(strings.TrimSpace(name), "Runbook: ")
	if name == "" {
		name = "untitled runbook"
	}
	return clipInlineRunes(name, 100)
}

func looksLikeGeneratedSlug(s string) bool {
	return !strings.Contains(s, " ") && strings.Count(s, "-") >= 6 && len(s) > 48
}

func firstUsefulLine(s string) string {
	for _, line := range strings.Split(stripFrontMatter(s), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "---") {
			continue
		}
		return strings.TrimPrefix(line, "description: ")
	}
	return ""
}

func formatRunbookChecklist(body string) string {
	body = stripFrontMatter(body)
	steps := extractMarkdownSection(body, "Steps")
	validation := extractMarkdownSection(body, "Validation")

	var b strings.Builder
	if steps != "" {
		b.WriteString("Checklist:\n")
		b.WriteString(steps)
	}
	if validation != "" {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString("Validation:\n")
		b.WriteString(validation)
	}
	if b.Len() > 0 {
		return strings.TrimSpace(b.String())
	}
	return strings.TrimSpace(body)
}

func stripFrontMatter(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return strings.TrimSpace(s)
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return strings.TrimSpace(strings.Join(lines[i+1:], "\n"))
		}
	}
	return strings.TrimSpace(s)
}

func extractMarkdownSection(body, heading string) string {
	lines := strings.Split(body, "\n")
	target := "## " + heading
	var out []string
	inSection := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.EqualFold(trimmed, target) {
			inSection = true
			continue
		}
		if inSection && strings.HasPrefix(trimmed, "## ") {
			break
		}
		if inSection {
			out = append(out, line)
		}
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

func clipRunes(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len([]rune(s)) <= max {
		return s
	}
	r := []rune(s)
	return strings.TrimSpace(string(r[:max])) + "\n[truncated]"
}

func clipInlineRunes(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len([]rune(s)) <= max {
		return s
	}
	if max <= 3 {
		return strings.TrimSpace(string([]rune(s)[:max]))
	}
	r := []rune(s)
	return strings.TrimSpace(string(r[:max-3])) + "..."
}

// extractFilePathFromPayload tries the common shapes Claude Code sends.
// Examples:
//
//	{"tool_name":"Edit","tool_input":{"file_path":"/abs/path.go"}}
//	{"tool_name":"Read","tool_input":{"file_path":"/abs/path.go"}}
func extractFilePathFromPayload(p map[string]interface{}) string {
	ti, _ := p["tool_input"].(map[string]interface{})
	if ti == nil {
		return ""
	}
	if fp, ok := ti["file_path"].(string); ok {
		return fp
	}
	// Some tools use `path`
	if fp, ok := ti["path"].(string); ok {
		return fp
	}
	return ""
}
