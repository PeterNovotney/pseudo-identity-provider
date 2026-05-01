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
	sessionmgmt "customidp/session"
	"net/http"
	"strings"
)

// callbackHandler does whatever you'd like.
func callbackHandler(w http.ResponseWriter, r *http.Request) {
	action := config.GetGlobalConfig().CallbackAction
	input := getInputData(r)
	addRequestLogEntry(input, action.Action)

	switch action.Action {
	case "respond":
		callbackRespond(w, input)
	case "redirect":
		http.Redirect(w, r, config.GetGlobalConfig().CallbackAction.Redirect.Target, http.StatusFound)
	case "error":
		errorResponse(w, r, &action.Error)
	case "block":
		blockResponse(w)
	}
	// Default is to return empty 200.
}

// callbackRespond responds with JSON content as configured.
func callbackRespond(w http.ResponseWriter, input *sessionmgmt.RequestInput) {
	c := config.GetGlobalConfig().CallbackAction.Respond
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")

	// Set headers from config.
	for _, header := range c.Headers {
		w.Header().Set(header.Key, header.Value)
	}

	vals, err := bodyToParameter(c.Body).Get(input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Write([]byte(strings.Join(vals, "")))
}

// bodyToParameter converts a Body config to a Parameter for easy processing.
func bodyToParameter(body config.Body) config.Parameter {
	return config.Parameter{
		Action:    body.Action,
		Values:    []string{body.Value},
		CustomKey: body.CustomKey,
	}
}
