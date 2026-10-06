package iaphandler

import (
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/veggiemonk/cloud-run-auth/internal/components/iapui"
	"github.com/veggiemonk/cloud-run-auth/internal/shared/render"
)

// Headers returns a handler that displays all request headers.
func Headers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entries := make([]iapui.HeaderEntry, 0, len(r.Header))

		for name, values := range r.Header {
			entries = append(entries, iapui.HeaderEntry{
				Name:  name,
				Value: strings.Join(values, ", "),
				IsIAP: strings.HasPrefix(strings.ToLower(name), "x-goog-"),
			})
		}

		slices.SortFunc(entries, func(a, b iapui.HeaderEntry) int {
			// IAP headers first, then alphabetical.
			if a.IsIAP != b.IsIAP {
				if a.IsIAP {
					return -1
				}
				return 1
			}
			return strings.Compare(a.Name, b.Name)
		})

		data := iapui.HeadersData{Headers: entries}

		if render.WantsJSON(r) {
			render.JSON(w, data)
			return
		}

		if err := iapui.HeadersPage(data).Render(r.Context(), w); err != nil {
			slog.Error("failed to render headers", "error", err)
		}
	}
}
