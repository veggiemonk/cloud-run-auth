package iaphandler

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"log/slog"
	"net/http"

	"github.com/veggiemonk/cloud-run-auth/internal/components/iapui"
	"github.com/veggiemonk/cloud-run-auth/internal/iap"
	"github.com/veggiemonk/cloud-run-auth/internal/shared/render"
)

// prettyJSON indents the decoded JWT parts and sorts their keys.
var prettyJSON = json.JoinOptions(jsontext.WithIndent("  "), json.Deterministic(true))

// JWT returns a handler that inspects the IAP JWT.
func JWT(verifier *iap.Verifier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		det := iap.DetectionResultFromContext(r)

		data := iapui.JWTData{
			Present: det.HasJWT,
		}

		if det.HasJWT {
			var result *iap.VerificationResult
			if verifier.ExpectedAudience() != "" {
				result = verifier.Verify(r.Context(), det.RawJWT)
			} else {
				result = verifier.Decode(det.RawJWT)
			}

			data.Valid = result.Valid
			data.ValidationError = result.Error
			data.SignatureB64 = result.SignatureB64
			data.Claims = result.Claims

			if result.Header != nil {
				if b, err := json.Marshal(result.Header, prettyJSON); err != nil {
					slog.Error("failed to marshal JWT header", "error", err)
				} else {
					data.HeaderJSON = string(b)
				}
			}
			if result.Payload != nil {
				if b, err := json.Marshal(result.Payload, prettyJSON); err != nil {
					slog.Error("failed to marshal JWT payload", "error", err)
				} else {
					data.PayloadJSON = string(b)
				}
			}
		}

		if render.WantsJSON(r) {
			render.JSON(w, data)
			return
		}

		if err := iapui.JWTPage(data).Render(r.Context(), w); err != nil {
			slog.Error("failed to render jwt", "error", err)
		}
	}
}
