package handler

import (
	"log/slog"
	"net/http"
	"sort"
	"strings"

	"github.com/veggiemonk/cloud-run-iap/internal/components"
	"github.com/veggiemonk/cloud-run-iap/internal/render"
)

// Headers returns a handler that displays all request headers.
func Headers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var entries []components.HeaderEntry

		for name, values := range r.Header {
			entries = append(entries, components.HeaderEntry{
				Name:  name,
				Value: strings.Join(values, ", "),
				IsIAP: strings.HasPrefix(strings.ToLower(name), "x-goog-"),
			})
		}

		sort.Slice(entries, func(i, j int) bool {
			// IAP headers first, then alphabetical.
			if entries[i].IsIAP != entries[j].IsIAP {
				return entries[i].IsIAP
			}
			return entries[i].Name < entries[j].Name
		})

		data := components.HeadersData{Headers: entries}

		if render.WantsJSON(r) {
			render.JSON(w, data)
			return
		}

		if err := components.HeadersPage(data).Render(r.Context(), w); err != nil {
			slog.Error("failed to render headers", "error", err)
		}
	}
}
