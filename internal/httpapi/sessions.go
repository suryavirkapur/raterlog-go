package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"raterlog-go/internal/postgres"
)

func (api *API) createSession(w http.ResponseWriter, r *http.Request, user postgres.User) {
	sessionID := randomID(32)
	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	if err := api.pg.CreateSession(r.Context(), sessionID, user.ID, expiresAt); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   api.cfg.CookieSecure,
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"user":    user,
		"session": sessionID,
	})
}

func (api *API) requireUser(w http.ResponseWriter, r *http.Request) (postgres.User, bool) {
	user, err := api.userFromRequest(r.Context(), r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return postgres.User{}, false
	}
	return user, true
}

func (api *API) userFromRequest(ctx context.Context, r *http.Request) (postgres.User, error) {
	sessionID := sessionIDFromRequest(r)
	if sessionID == "" {
		return postgres.User{}, errors.New("missing session")
	}
	return api.pg.UserBySession(ctx, sessionID)
}

func (api *API) requireCompanyMember(w http.ResponseWriter, r *http.Request) bool {
	user, ok := api.requireUser(w, r)
	if !ok {
		return false
	}
	member, err := api.pg.IsCompanyMember(r.Context(), chi.URLParam(r, "companyID"), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check company access")
		return false
	}
	if !member {
		writeError(w, http.StatusForbidden, "company access denied")
		return false
	}
	return true
}
