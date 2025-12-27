package mcp

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/terzigolu/josepshbrain-go/internal/api"
)

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type initializeParams struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ClientInfo      map[string]interface{} `json:"clientInfo"`
}

type toolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type textContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func ServeStdio(client *api.Client) error {
	if client == nil {
		return errors.New("api client is required")
	}

	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()

	initialized := false
	protocolVersion := "2025-11-25"

	for in.Scan() {
		line := strings.TrimSpace(in.Text())
		if line == "" {
			continue
		}

		var req jsonRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			writeResponse(out, jsonRPCResponse{JSONRPC: "2.0", Error: &jsonRPCError{Code: -32700, Message: "Parse error"}})
			continue
		}

		switch req.Method {
		case "initialize":
			var p initializeParams
			_ = json.Unmarshal(req.Params, &p)
			if strings.TrimSpace(p.ProtocolVersion) != "" {
				protocolVersion = strings.TrimSpace(p.ProtocolVersion)
			}

			res := map[string]interface{}{
				"protocolVersion": protocolVersion,
				"capabilities": map[string]interface{}{
					"tools": map[string]interface{}{},
				},
				"serverInfo": map[string]interface{}{
					"name":    "ramorie",
					"version": "0.1.0",
				},
			}
			writeResponse(out, jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: res})

		case "notifications/initialized":
			initialized = true

		case "tools/list":
			if !initialized {
				writeResponse(out, jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Error: &jsonRPCError{Code: -32002, Message: "Server not initialized"}})
				continue
			}
			writeResponse(out, jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]interface{}{"tools": ToolDefinitions()}})

		case "tools/call":
			if !initialized {
				writeResponse(out, jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Error: &jsonRPCError{Code: -32002, Message: "Server not initialized"}})
				continue
			}

			var p toolCallParams
			if err := json.Unmarshal(req.Params, &p); err != nil {
				writeResponse(out, jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Error: &jsonRPCError{Code: -32602, Message: "Invalid params"}})
				continue
			}

			result, err := CallTool(client, p.Name, p.Arguments)
			if err != nil {
				payload := map[string]interface{}{
					"isError": true,
					"content": []interface{}{textContent{Type: "text", Text: err.Error()}},
				}
				writeResponse(out, jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: payload})
				continue
			}

			b, _ := json.Marshal(result)
			payload := map[string]interface{}{
				"isError":           false,
				"structuredContent": result,
				"content":           []interface{}{textContent{Type: "text", Text: string(b)}},
			}
			writeResponse(out, jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: payload})

		case "ping":
			writeResponse(out, jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]interface{}{}})

		default:
			writeResponse(out, jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Error: &jsonRPCError{Code: -32601, Message: "Method not found"}})
		}
	}

	if err := in.Err(); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

func writeResponse(w *bufio.Writer, resp jsonRPCResponse) {
	b, err := json.Marshal(resp)
	if err != nil {
		b = []byte(fmt.Sprintf("{\"jsonrpc\":\"2.0\",\"error\":{\"code\":-32603,\"message\":\"Internal error\"}}"))
	}
	w.Write(b)
	w.WriteByte('\n')
	w.Flush()
}
