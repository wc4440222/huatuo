// Copyright 2025 The HuaTuo Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pod

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// errBody is an io.ReadCloser that always fails on Read.
type errBody struct{}

func (errBody) Read(p []byte) (int, error) {
	return 0, errors.New("simulated body read error")
}

func (errBody) Close() error { return nil }

// errorRoundTripper returns a 200 OK response whose Body fails on Read.
type errorRoundTripper struct{}

func (errorRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       errBody{},
		Header:     make(http.Header),
	}, nil
}

// okRoundTripper returns a 200 OK response with the given body content.
type okRoundTripper struct {
	body string
}

func (r *okRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(r.body)),
		Header:     make(http.Header),
	}, nil
}

// nonOKRoundTripper returns a response with a non-200 status code.
type nonOKRoundTripper struct {
	statusCode int
	body       string
}

func (r *nonOKRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: r.statusCode,
		Body:       io.NopCloser(strings.NewReader(r.body)),
		Header:     make(http.Header),
	}, nil
}

func TestHTTPDoRequestBodyReadError(t *testing.T) {
	client := &http.Client{Transport: errorRoundTripper{}}

	body, err := httpDoRequest(client, "http://127.0.0.1:12345/pods")
	if err == nil {
		t.Fatalf("expected error when body read fails, got nil; body=%q", string(body))
	}

	// The error message should mention "read body" and the URL.
	if !strings.Contains(err.Error(), "read body") {
		t.Errorf("expected error to contain 'read body', got: %v", err)
	}
	if !strings.Contains(err.Error(), "simulated body read error") {
		t.Errorf("expected error to contain the underlying read error, got: %v", err)
	}
}

func TestHTTPDoRequestSuccess(t *testing.T) {
	client := &http.Client{Transport: &okRoundTripper{body: `{"items": []}`}}

	body, err := httpDoRequest(client, "http://127.0.0.1:12345/pods")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	expected := `{"items": []}`
	if string(body) != expected {
		t.Errorf("expected body %q, got %q", expected, string(body))
	}
}

func TestHTTPDoRequestNonOKStatus(t *testing.T) {
	client := &http.Client{Transport: &nonOKRoundTripper{statusCode: http.StatusInternalServerError, body: "internal error"}}

	body, err := httpDoRequest(client, "http://127.0.0.1:12345/pods")
	if err == nil {
		t.Fatalf("expected error for non-200 status, got nil; body=%q", string(body))
	}

	if !strings.Contains(err.Error(), "status: 500") {
		t.Errorf("expected error to contain 'status: 500', got: %v", err)
	}
	if !strings.Contains(err.Error(), "internal error") {
		t.Errorf("expected error to contain body 'internal error', got: %v", err)
	}
}