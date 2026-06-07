package httpapi

import (
	"log/slog"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func (api *API) signup(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name        string `json:"name"`
		Email       string `json:"email"`
		Password    string `json:"password"`
		InviteToken string `json:"invite_token"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	if len(input.Name) < 3 || len(input.Name) > 80 {
		writeError(w, http.StatusBadRequest, "name must be between 3 and 80 characters")
		return
	}
	if !validEmail(input.Email) {
		writeError(w, http.StatusBadRequest, "invalid email")
		return
	}
	if len(input.Password) < 6 || len(input.Password) > 255 {
		writeError(w, http.StatusBadRequest, "password must be between 6 and 255 characters")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}
	user, err := api.pg.CreateUser(r.Context(), randomID(15), input.Name, input.Email, string(hash))
	if err != nil {
		writeError(w, http.StatusConflict, "email already exists")
		return
	}
	if input.InviteToken != "" {
		if _, err := api.pg.AcceptInvite(r.Context(), input.InviteToken, user.ID, user.Email); err != nil {
			slog.Warn("failed to accept invite during signup", "error", err)
		}
	}
	api.createSession(w, r, user)
}

func (api *API) signin(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email       string `json:"email"`
		Password    string `json:"password"`
		InviteToken string `json:"invite_token"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	user, hash, err := api.pg.UserPasswordHash(r.Context(), strings.ToLower(strings.TrimSpace(input.Email)))
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(input.Password)) != nil {
		writeError(w, http.StatusUnauthorized, "incorrect email or password")
		return
	}
	if input.InviteToken != "" {
		if _, err := api.pg.AcceptInvite(r.Context(), input.InviteToken, user.ID, user.Email); err != nil {
			slog.Warn("failed to accept invite during signin", "error", err)
		}
	}
	api.createSession(w, r, user)
}

func (api *API) signout(w http.ResponseWriter, r *http.Request) {
	sessionID := sessionIDFromRequest(r)
	if sessionID != "" {
		_ = api.pg.DeleteSession(r.Context(), sessionID)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   api.cfg.CookieSecure,
	})
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (api *API) me(w http.ResponseWriter, r *http.Request) {
	user, ok := api.requireUser(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, user)
}
