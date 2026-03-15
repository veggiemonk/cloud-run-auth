package oauthhandler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/veggiemonk/cloud-run-auth/internal/components/oauthui"
	"github.com/veggiemonk/cloud-run-auth/internal/oauth"
	"github.com/veggiemonk/cloud-run-auth/internal/shared/render"
)

// Token returns a handler that shows OAuth token details.
func Token() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := oauth.UserFromContext(r.Context())
		if user == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		data := oauthui.TokenData{}

		if user.Token != nil {
			accessToken := user.Token.AccessToken
			if len(accessToken) > 10 {
				data.AccessTokenMasked = accessToken[:10] + "..."
			} else if accessToken != "" {
				data.AccessTokenMasked = "***"
			}

			data.HasRefreshToken = user.Token.RefreshToken != ""
			data.TokenType = user.Token.TokenType

			if !user.Token.Expiry.IsZero() {
				data.Expiry = user.Token.Expiry.Format(time.RFC3339)
			}

			// Extract scopes from the token's extra data if available.
			if extra := user.Token.Extra("scope"); extra != nil {
				if scopeStr, ok := extra.(string); ok && scopeStr != "" {
					// Scopes are space-separated.
					data.Scopes = splitScopes(scopeStr)
				}
			}
		}

		if render.WantsJSON(r) {
			render.JSON(w, data)
			return
		}

		if err := oauthui.TokenPage(data).Render(r.Context(), w); err != nil {
			slog.Error("failed to render token page", "error", err)
		}
	}
}

// splitScopes splits a space-separated scope string into individual scopes.
func splitScopes(s string) []string {
	var scopes []string
	current := ""
	for _, c := range s {
		if c == ' ' {
			if current != "" {
				scopes = append(scopes, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		scopes = append(scopes, current)
	}
	return scopes
}
