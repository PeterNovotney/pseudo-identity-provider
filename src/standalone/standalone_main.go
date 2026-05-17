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

// Package standalone_main runs the IdP as a standalone server.
package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"customidp/config"
	idp "customidp/idp"
	"customidp/mcp"
)

// Static HTML file templates.
var templates = template.Must(template.ParseGlob("../src/static/browser/*.html"))

// loadConfigFromFile loads configuration from a JSON file if specified.
func loadConfigFromFile(configPath string) error {
	if configPath == "" {
		log.Println("No config file specified, using default configuration")
		return nil
	}

	log.Printf("Loading configuration from: %s", configPath)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var cfg config.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}

	config.SetGlobalConfig(&cfg)
	log.Println("Configuration loaded successfully")
	return nil
}

// watchConfigFile watches for changes to the config file and reloads it automatically.
func watchConfigFile(configPath string) {
	if configPath == "" {
		return
	}

	log.Printf("Starting config file watcher for: %s", configPath)

	// Get initial file info
	initialStat, err := os.Stat(configPath)
	if err != nil {
		log.Printf("Warning: Could not stat config file for watching: %v", err)
		return
	}
	lastModTime := initialStat.ModTime()

	// Check for file changes every second
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stat, err := os.Stat(configPath)
		if err != nil {
			// File might be temporarily unavailable during write
			continue
		}

		// Check if file was modified
		if stat.ModTime().After(lastModTime) {
			lastModTime = stat.ModTime()

			// Small delay to ensure file write is complete
			time.Sleep(100 * time.Millisecond)

			log.Printf("Configuration file changed, reloading...")
			if err := loadConfigFromFile(configPath); err != nil {
				log.Printf("Error reloading configuration: %v", err)
			} else {
				log.Println("Configuration reloaded successfully")
			}
		}
	}
}

// Main setups handlers and starts the service for AppEngine hosting.
func main() {
	idp.InitHandlers(templates, "../src/static/browser")
	mcp.InitMcpServer()

	// Load configuration from file if specified
	configPath := os.Getenv("CONFIG_FILE")
	if configPath != "" {
		// Make path absolute for watching
		if !filepath.IsAbs(configPath) {
			absPath, err := filepath.Abs(configPath)
			if err == nil {
				configPath = absPath
			}
		}

		if err := loadConfigFromFile(configPath); err != nil {
			log.Printf("Warning: Failed to load config file: %v", err)
			log.Println("Using default configuration")
		}

		// Start watching for config file changes in background
		go watchConfigFile(configPath)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Listening on port %s", port)
	http.ListenAndServe(":"+port, nil)
}
