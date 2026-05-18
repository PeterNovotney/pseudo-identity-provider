// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package mcp implements MCP server logic to control the Pseudo IdP
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strings"

	"customidp/config"
	idp "customidp/idp"

	"github.com/google/jsonschema-go/jsonschema"
	invoschema "github.com/invopop/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const MCPServerEnabledVar = "PSEUDO_IDP_ENABLE_MCP"

// MCPStatusMessage hold the message to update the MCPServerEnabled status.
type MCPStatusMessage struct {
	Enabled bool `json:"enabled"`
}

// Whether to enable the MCP server or not. We keep this seperate from the IdP config so
// MCP calls dont turn the MCP server off.
var MCPServerEnabled bool

// ListLogs is a simple MCP tool that returns request logs
func ListLogs(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
	resp := idp.ListLogs()
	safeResp := idp.ListLogsResponse{
		Entries: []idp.RequestEntry{},
	}
	for _, entry := range resp.Entries {
		initializeNilSlices(entry)
		safeResp.Entries = append(safeResp.Entries, entry)
	}
	return nil, safeResp, nil
}

// GetConfig returns the current config.
func GetConfig(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
	config := config.GetGlobalConfig()
	initializeNilSlices(config)
	return nil, *config, nil
}

// SetConfig updates the server configuration.
func SetConfig(_ context.Context, _ *mcp.CallToolRequest, input config.Config) (*mcp.CallToolResult, any, error) {
	config.SetGlobalConfig(&input)
	return nil, nil, nil
}

// InitMcpServer configures all HTTP request handlers for MCP.
func InitMcpServer() error {
	MCPServerEnabled = (strings.ToLower(os.Getenv(MCPServerEnabledVar)) == "true")

	handler, err := initMcpHandler()
	if err != nil {
		return err
	}
	http.HandleFunc("/mcp", handler)
	http.HandleFunc("/mcpstatus", mcpStatusHandler)
	return nil
}

// initMcpHandler sets up a HTTP Hander for the MCP server and
func initMcpHandler() (func(http.ResponseWriter, *http.Request), error) {
	server := mcp.NewServer(&mcp.Implementation{Name: "authenticated-pseudoidp-mcp-server"}, nil)

	out, err := jsonschema.For[idp.ListLogsResponse](nil)
	if err != nil {
		return nil, err
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:         "list_logs",
		Description:  "List request logs",
		OutputSchema: out,
	}, ListLogs)

	// modelcontextprotocol/go-sdk/mcp uses jsonschema.Schema which differs
	// from our forms system Schema handling. We have to do a little dance
	// here to convert to a  format that agrees with the MCP framework Schema
	// parsing by convering to a JSON blob and then importing into jsonschema.
	r := new(invoschema.Reflector)
	r.KeyNamer = invoschema.ToSnakeCase
	r.DoNotReference = true
	r.RequiredFromJSONSchemaTags = true
	schema := r.Reflect(&config.Config{})
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return nil, err
	}

	var s jsonschema.Schema
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:         "get_config",
		Description:  "Get Psuedo Identity Provider Configuration",
		OutputSchema: s,
	}, GetConfig)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "set_config",
		Description: "Set Psuedo Identity Provider Configuration",
		InputSchema: s,
	}, SetConfig)

	// Localhost DNS rebinding protections are disabled for AppEngine hosted
	// as there is a local reverse proxy. If you are using Pseudo IdP behind
	// a reverse proxy in another configuration, you may need to force this
	// value to true as well. See https://github.com/modelcontextprotocol/go-sdk/pull/760
	// for more information.
	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{DisableLocalhostProtection: isAppEngine()})

	// Secure wrapper to handle Basic Auth challenge and validation
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !MCPServerEnabled {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		if !idp.CheckAuth(w, r) {
			return
		}
		handler.ServeHTTP(w, r)
	}), nil
}

// isAppEngine check if we are running on AppEngine.
func isAppEngine() bool {
	// GAE_ENV is set to "standard" in the GAE Standard environment
	return os.Getenv("GAE_ENV") != ""
}

// initializeNilSlices fills in any Nil array slices, which the MCP Server's
// parsing does not like, with an empty array.
func initializeNilSlices(i interface{}) {
	v := reflect.ValueOf(i)

	// We must have a pointer to modify the underlying data
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return
	}

	initializeRecursive(v.Elem(), reflect.StructField{})
}

// initializeRecursive walks the structure to fill in the slices.
func initializeRecursive(v reflect.Value, field reflect.StructField) {
	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		initializeRecursive(v.Elem(), field)

	case reflect.Struct:
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			initializeRecursive(v.Field(i), t.Field(i))
		}

	case reflect.Slice:
		// If the slice is nil, create an empty slice of the same type
		if v.IsNil() && v.CanSet() {
			emptySlice := reflect.MakeSlice(v.Type(), 0, 0)
			v.Set(emptySlice)
		}

		// Even if it wasn't nil, we need to check its elements
		// for nested structs/slices
		for i := 0; i < v.Len(); i++ {
			initializeRecursive(v.Index(i), field)
		}

	case reflect.Map:
		// Optional: Handle maps similarly if needed
		if v.IsNil() && v.CanSet() {
			v.Set(reflect.MakeMap(v.Type()))
		}
		// Iterate through map values
		for _, key := range v.MapKeys() {
			initializeRecursive(v.MapIndex(key), field)
		}
	case reflect.String:
		if v.CanSet() && v.IsZero() {
			if tagValue := field.Tag.Get("jsonschema"); tagValue != "" {
				if defaultValue := extractDefault(tagValue); defaultValue != "" {
					v.SetString(defaultValue)
				}
			}
		}
	}
}

// extractDefault parses "fieldName,default=X" or "fieldName,omitempty,default=X"
func extractDefault(tag string) string {
	parts := strings.Split(tag, ",")
	for _, part := range parts {
		if strings.HasPrefix(part, "default=") {
			return strings.TrimPrefix(part, "default=")
		}
	}
	return ""
}

// mcpStatusHandler updates if the MCP server is on or off.
func mcpStatusHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		status := MCPStatusMessage{
			Enabled: MCPServerEnabled,
		}
		resp, err := json.Marshal(status)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to marshal content %v", err), http.StatusInternalServerError)
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprint(w, string(resp))
	case "POST":
		if !idp.CheckAuth(w, r) {
			return
		}

		if err := idp.CheckCSRF(r); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if !idp.CheckAuth(w, r) {
			return
		}

		if r.Body == nil {
			http.Error(w, "No config sent", http.StatusBadRequest)
			return
		}

		newStatus := MCPStatusMessage{}
		if err := json.NewDecoder(r.Body).Decode(&newStatus); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		MCPServerEnabled = newStatus.Enabled
	}
}
