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
	"fmt"
	"net/http"
	"net/url"
)

// CheckCSRF checks for CSRF protections.
func CheckCSRF(r *http.Request) error {
	// Custom header defense.
	if r.Header.Get("X-Pseudo-IDP-CSRF-Protection") != "1" {
		return fmt.Errorf("invalid Request")
	}

	// Ensure a correct content type. Don't allow form-data or test/plain.
	if r.Header.Get("Content-Type") != "application/json" {
		return fmt.Errorf("invalid Request")
	}

	// As a defense in depth, validate the Origin.
	host := r.Host
	if host == "" {
		return fmt.Errorf("invalid Request")
	}

	originURL, err := url.ParseRequestURI(r.Header.Get("Origin"))
	if err != nil {
		return fmt.Errorf("invalid Request")
	}

	if host != originURL.Host {
		return fmt.Errorf("invalid Request")
	}

	return nil
}
