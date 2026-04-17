package mcpinstall

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

// SmokeResult captures what the stdio probe observed.
type SmokeResult struct {
	OK       bool
	Tools    int
	Version  string
	Duration time.Duration
	Err      error
}

// SmokeTest spawns `binary args...` as an MCP stdio server, performs the
// initialize + notifications/initialized + tools/list handshake, and
// validates that (a) the server responds, (b) tools include a reasonable
// count. Intended to confirm a just-installed entry is actually callable.
//
// A 5-second ceiling keeps the TUI snappy even if the client never writes.
func SmokeTest(ctx context.Context, binary string, args []string, env map[string]string) SmokeResult {
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, args...)
	if len(env) > 0 {
		cmd.Env = append(cmd.Environ(), flattenEnv(env)...)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return SmokeResult{Err: fmt.Errorf("stdin pipe: %w", err), Duration: time.Since(start)}
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return SmokeResult{Err: fmt.Errorf("stdout pipe: %w", err), Duration: time.Since(start)}
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return SmokeResult{Err: fmt.Errorf("start %s: %w", binary, err), Duration: time.Since(start)}
	}
	defer func() { _ = cmd.Process.Kill() }()

	// 1) initialize
	req := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"ramorie-smoke","version":"1"}}}`
	if _, err := io.WriteString(stdin, req+"\n"); err != nil {
		return SmokeResult{Err: err, Duration: time.Since(start)}
	}
	if _, err := io.WriteString(stdin, `{"jsonrpc":"2.0","method":"notifications/initialized"}`+"\n"); err != nil {
		return SmokeResult{Err: err, Duration: time.Since(start)}
	}
	// 2) tools/list
	if _, err := io.WriteString(stdin, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`+"\n"); err != nil {
		return SmokeResult{Err: err, Duration: time.Since(start)}
	}

	// Read lines until we see id:2 response or ctx timeout.
	scanner := bufio.NewScanner(stdout)
	buf := make([]byte, 0, 256*1024)
	scanner.Buffer(buf, 4*1024*1024) // 4MB — MCP responses can be long
	var version string
	var toolCount int
	found := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		var env struct {
			ID     int             `json:"id"`
			Result json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal([]byte(line), &env); err != nil {
			continue
		}
		if env.ID == 1 {
			// initialize response — try to extract server version for the UI
			var init struct {
				ServerInfo struct {
					Version string `json:"version"`
				} `json:"serverInfo"`
			}
			_ = json.Unmarshal(env.Result, &init)
			version = init.ServerInfo.Version
		}
		if env.ID == 2 {
			var tl struct {
				Tools []any `json:"tools"`
			}
			if err := json.Unmarshal(env.Result, &tl); err == nil {
				toolCount = len(tl.Tools)
				found = true
				break
			}
		}
		if ctx.Err() != nil {
			break
		}
	}

	_ = stdin.Close()
	_ = cmd.Wait()

	if !found {
		return SmokeResult{Err: fmt.Errorf("no tools/list response within timeout"), Duration: time.Since(start), Version: version}
	}
	if toolCount == 0 {
		return SmokeResult{Err: fmt.Errorf("server returned 0 tools — handshake likely broken"), Duration: time.Since(start), Version: version}
	}
	return SmokeResult{
		OK:       true,
		Tools:    toolCount,
		Version:  version,
		Duration: time.Since(start),
	}
}

func flattenEnv(env map[string]string) []string {
	out := make([]string, 0, len(env))
	for k, v := range env {
		out = append(out, k+"="+v)
	}
	return out
}
