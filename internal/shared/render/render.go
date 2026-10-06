// Package render owns the response content-negotiation helpers used by
// every diagnostic handler: WantsJSON (decide HTML vs JSON from query
// param or Accept header) and JSON (write a well-formed JSON response).
//
// Exists so each handler can offer both an HTMX-rendered page and a
// scriptable JSON view from the same code path, without each handler
// reinventing the negotiation rules.
package render

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"net/http"
	"strings"
)

// WantsJSON returns true if the request prefers a JSON response,
// either via ?format=json query parameter or Accept header.
func WantsJSON(r *http.Request) bool {
	if r.URL.Query().Get("format") == "json" {
		return true
	}
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "application/json")
}

// JSON writes data as indented JSON with map keys sorted and a trailing
// newline, with proper content-type.
func JSON(w http.ResponseWriter, data any) {
	b, err := json.Marshal(data, jsontext.WithIndent("  "), json.Deterministic(true))
	if err != nil {
		http.Error(w, `{"error":"json encoding failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(append(b, '\n'))
}
