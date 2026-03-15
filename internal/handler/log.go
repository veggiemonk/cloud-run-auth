package handler

import (
	"log/slog"
	"net/http"

	"github.com/veggiemonk/cloud-run-iap/internal/components"
	"github.com/veggiemonk/cloud-run-iap/internal/render"
	"github.com/veggiemonk/cloud-run-iap/internal/reqlog"
)

// Log returns a handler that displays the request log.
func Log(buf *reqlog.Buffer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := components.LogData{
			Entries: buf.Entries(),
		}

		if render.WantsJSON(r) {
			render.JSON(w, data)
			return
		}

		if err := components.LogPage(data).Render(r.Context(), w); err != nil {
			slog.Error("failed to render log", "error", err)
		}
	}
}
