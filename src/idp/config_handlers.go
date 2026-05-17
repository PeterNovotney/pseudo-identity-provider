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

package idp

import (
	"customidp/config"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/invopop/jsonschema"
)

// configHandler gets or updates the instance configuration.
// Authorization is required for modifying the configuration.
// A POST request with no Body resets the configuration to the default.
func configHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if err := CheckCSRF(r); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if !CheckAuth(w, r) {
			return
		}

		if r.Body == nil {
			http.Error(w, "No config sent", http.StatusBadRequest)
			return
		}

		var newConfig config.Config
		if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		config.SetGlobalConfig(&newConfig)
	} else if r.Method == "DELETE" {
		if err := CheckCSRF(r); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if !CheckAuth(w, r) {
			return
		}

		config.SetGlobalConfig(&config.DefaultConfig)
	}

	data, err := json.Marshal(config.GetGlobalConfig())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprint(w, string(data))
}

// configSchemaHandler gets the JSON Schema of the Config object.
// This is used to generate the Frontend forms.
func configSchemaHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	r := new(jsonschema.Reflector)
	r.KeyNamer = jsonschema.ToSnakeCase
	r.DoNotReference = true
	r.RequiredFromJSONSchemaTags = true
	schema := r.Reflect(&config.Config{})
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprint(w, string(data))
}
