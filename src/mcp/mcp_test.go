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

package mcp

import (
	"context"
	"customidp/config"
	"customidp/idp"
	sessionmgmt "customidp/session"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestListTools(t *testing.T) {
	MCPServerEnabled = true
	ctx := context.Background()
	config.SetGlobalConfig(&config.DefaultConfig)
	handler, err := initMcpHandler()
	if err != nil {
		t.Fatalf("Failed to initialize handler: %v", err)
	}

	ts := httptest.NewServer(http.HandlerFunc(handler))
	defer ts.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "mcp-client", Version: "v1.0.0"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: ts.URL}, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer session.Close()

	toolsRes, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Errorf("Failed to ListTools: %v", err)
	}

	actualTools := []string{}
	for _, t := range toolsRes.Tools {
		actualTools = append(actualTools, t.Name)
	}

	wantedTools := []string{"get_config", "set_config", "list_logs"}
	slices.Sort(wantedTools)
	slices.Sort(actualTools)
	if !slices.Equal(wantedTools, actualTools) {
		t.Errorf("expected %v, got %v", wantedTools, actualTools)
	}
}

func TestGetConfig(t *testing.T) {
	MCPServerEnabled = true
	ctx := context.Background()
	config.SetGlobalConfig(&config.DefaultConfig)
	handler, err := initMcpHandler()
	if err != nil {
		t.Fatalf("Failed to initialize handler: %v", err)
	}

	ts := httptest.NewServer(http.HandlerFunc(handler))
	defer ts.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "mcp-client", Version: "v1.0.0"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: ts.URL}, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer session.Close()

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_config",
		Arguments: map[string]interface{}{},
	})
	if err != nil {
		t.Errorf("Failed to GetConfig: %v", err)
	}
	if res.IsError {
		t.Errorf("Failed to GetConfig Tool Error")
	}

	for _, content := range res.Content {
		if text, ok := content.(*mcp.TextContent); ok {
			actualConfig := config.Config{}
			if err := json.Unmarshal([]byte(text.Text), &actualConfig); err != nil {
				t.Errorf("Failed to unmarshal config: %v", err)
			}
			if diff := cmp.Diff(config.DefaultConfig, actualConfig); diff != "" {
				t.Errorf("User mismatch (-want +got):\n%s", diff)
			}
		}
	}
}

func TestSetConfig(t *testing.T) {
	MCPServerEnabled = true
	ctx := context.Background()
	config.SetGlobalConfig(&config.DefaultConfig)
	handler, err := initMcpHandler()
	if err != nil {
		t.Fatalf("Failed to initialize handler: %v", err)
	}

	ts := httptest.NewServer(http.HandlerFunc(handler))
	defer ts.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "mcp-client", Version: "v1.0.0"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: ts.URL}, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer session.Close()

	updatedConfig := &config.DefaultConfig
	updatedConfig.IDTokenConfig.Algorithm = "ES256"

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "set_config",
		Arguments: updatedConfig,
	})
	if err != nil {
		t.Errorf("Failed to SetConfig: %v", err)
	}
	if res.IsError {
		t.Errorf("Failed to SetConfig Tool Error")
	}

	if !reflect.DeepEqual(updatedConfig, config.GetGlobalConfig()) {
		t.Errorf("JSON objects are not equal")
	}
}

func TestListLogs(t *testing.T) {
	MCPServerEnabled = true
	ctx := context.Background()
	config.SetGlobalConfig(&config.DefaultConfig)
	handler, err := initMcpHandler()
	if err != nil {
		t.Fatalf("Failed to initialize handler: %v", err)
	}

	ts := httptest.NewServer(http.HandlerFunc(handler))
	defer ts.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "mcp-client", Version: "v1.0.0"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: ts.URL}, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer session.Close()

	wantedEmptyResult := idp.ListLogsResponse{
		Entries: []idp.RequestEntry{},
	}
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_logs",
		Arguments: map[string]interface{}{},
	})
	if err != nil {
		t.Fatalf("Failed to ListLogs: %v", err)
	}
	if res.IsError {
		t.Fatalf("Failed to ListLogs Tool Error")
	}

	for _, content := range res.Content {
		if text, ok := content.(*mcp.TextContent); ok {
			actualResult := idp.ListLogsResponse{}
			if err := json.Unmarshal([]byte(text.Text), &actualResult); err != nil {
				t.Fatalf("Failed to unmarshal config: %v", err)
			}
			if !reflect.DeepEqual(actualResult, wantedEmptyResult) {
				t.Errorf("Expected empty logs are not equal")
			}
		}
	}

	wantedResponse := idp.ListLogsResponse{
		Entries: []idp.RequestEntry{
			{
				Input: &sessionmgmt.RequestInput{
					Domain: "test.com",
					Path:   "abcd/efg",
					Headers: http.Header{
						"host": {"test.com"},
					},
					URLParams: url.Values{
						"p": {"1"},
					},
					FormParams: url.Values{
						"f": {"1"},
					},
				},
				Resp: &idp.ResponseEntry{
					Body: "result",
					Headers: http.Header{
						"test": {"header"},
					},
				},
			},
		},
	}
	if err = idp.AddRequestLogEntry(wantedResponse.Entries[0]); err != nil {
		t.Fatalf("Failed to add entry: %v", err)
	}

	res, err = session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_logs",
		Arguments: map[string]interface{}{},
	})
	if err != nil {
		t.Fatalf("Failed to ListLogs: %v", err)
	}
	if res.IsError {
		t.Fatalf("Failed to ListLogs Tool Error")
	}

	for _, content := range res.Content {
		if text, ok := content.(*mcp.TextContent); ok {
			actualResult := idp.ListLogsResponse{}
			if err := json.Unmarshal([]byte(text.Text), &actualResult); err != nil {
				t.Fatalf("Failed to unmarshal config: %v", err)
			}
			if diff := cmp.Diff(wantedResponse, actualResult); diff != "" {
				t.Errorf("User mismatch (-want +got):\n%s", diff)
			}
		}
	}
}

func TestFailureWhenDisabled(t *testing.T) {
	MCPServerEnabled = false
	ctx := context.Background()
	config.SetGlobalConfig(&config.DefaultConfig)
	handler, err := initMcpHandler()
	if err != nil {
		t.Fatalf("Failed to initialize handler: %v", err)
	}

	ts := httptest.NewServer(http.HandlerFunc(handler))
	defer ts.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "mcp-client", Version: "v1.0.0"}, nil)
	_, err = client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: ts.URL}, nil)
	if err == nil {
		t.Errorf("Successful connection when expecting failure due to server disablement: %v", err)
	}
}
