package httpapi

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

func (api *API) listCompanies(w http.ResponseWriter, r *http.Request) {
	user, ok := api.requireUser(w, r)
	if !ok {
		return
	}
	companies, err := api.pg.CompaniesForUser(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list companies")
		return
	}
	writeJSON(w, http.StatusOK, companies)
}

func (api *API) createCompany(w http.ResponseWriter, r *http.Request) {
	user, ok := api.requireUser(w, r)
	if !ok {
		return
	}
	var input struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		writeError(w, http.StatusBadRequest, "company name is required")
		return
	}
	company, err := api.pg.CreateCompany(r.Context(), randomID(15), input.Name, user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create company")
		return
	}
	writeJSON(w, http.StatusCreated, company)
}

func (api *API) companyDetail(w http.ResponseWriter, r *http.Request) {
	if !api.requireCompanyMember(w, r) {
		return
	}
	detail, err := api.pg.CompanyDetail(r.Context(), chi.URLParam(r, "companyID"))
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, pgx.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeError(w, status, "failed to load company")
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (api *API) updateCompany(w http.ResponseWriter, r *http.Request) {
	if !api.requireCompanyMember(w, r) {
		return
	}
	var input struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		writeError(w, http.StatusBadRequest, "company name is required")
		return
	}
	company, err := api.pg.UpdateCompany(r.Context(), chi.URLParam(r, "companyID"), input.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update company")
		return
	}
	writeJSON(w, http.StatusOK, company)
}

func (api *API) createChannel(w http.ResponseWriter, r *http.Request) {
	if !api.requireCompanyMember(w, r) {
		return
	}
	var input struct {
		Name string `json:"name"`
		Icon string `json:"icon"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Icon = strings.TrimSpace(input.Icon)
	if input.Name == "" || input.Icon == "" {
		writeError(w, http.StatusBadRequest, "channel name and icon are required")
		return
	}
	channel, err := api.pg.CreateChannel(r.Context(), randomID(15), chi.URLParam(r, "companyID"), input.Name, input.Icon)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create channel")
		return
	}
	writeJSON(w, http.StatusCreated, channel)
}

func (api *API) members(w http.ResponseWriter, r *http.Request) {
	if !api.requireCompanyMember(w, r) {
		return
	}
	members, err := api.pg.Members(r.Context(), chi.URLParam(r, "companyID"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list members")
		return
	}
	writeJSON(w, http.StatusOK, members)
}

func (api *API) createInvite(w http.ResponseWriter, r *http.Request) {
	if !api.requireCompanyMember(w, r) {
		return
	}
	var input struct {
		Email string `json:"email"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	if !validEmail(input.Email) {
		writeError(w, http.StatusBadRequest, "valid email is required")
		return
	}

	companyID := chi.URLParam(r, "companyID")
	token := randomID(32)
	invite, err := api.pg.CreateInvite(r.Context(), randomID(15), input.Email, companyID, token, time.Now().Add(7*24*time.Hour))
	if err != nil {
		writeError(w, http.StatusConflict, "invite already exists or could not be created")
		return
	}
	detail, err := api.pg.CompanyDetail(r.Context(), companyID)
	if err == nil {
		if err := api.sendInvite(input.Email, detail.Company.Name, token); err != nil {
			slog.Warn("failed to send invite email", "error", err)
		}
	}
	writeJSON(w, http.StatusCreated, invite)
}

func (api *API) deleteInvite(w http.ResponseWriter, r *http.Request) {
	if !api.requireCompanyMember(w, r) {
		return
	}
	err := api.pg.DeleteInvite(r.Context(), chi.URLParam(r, "companyID"), chi.URLParam(r, "inviteID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "invite not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (api *API) createToken(w http.ResponseWriter, r *http.Request) {
	if !api.requireCompanyMember(w, r) {
		return
	}
	var input struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		writeError(w, http.StatusBadRequest, "token name is required")
		return
	}
	token, err := api.pg.CreateToken(r.Context(), chi.URLParam(r, "companyID"), input.Name, randomID(24))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create token")
		return
	}
	writeJSON(w, http.StatusCreated, token)
}

func (api *API) deleteToken(w http.ResponseWriter, r *http.Request) {
	if !api.requireCompanyMember(w, r) {
		return
	}
	tokenID, err := strconv.ParseInt(chi.URLParam(r, "tokenID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid token id")
		return
	}
	if err := api.pg.DeleteToken(r.Context(), chi.URLParam(r, "companyID"), tokenID); err != nil {
		writeError(w, http.StatusNotFound, "token not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (api *API) invite(w http.ResponseWriter, r *http.Request) {
	invite, err := api.pg.InviteByToken(r.Context(), chi.URLParam(r, "token"))
	if err != nil {
		writeError(w, http.StatusNotFound, "invite not found")
		return
	}
	writeJSON(w, http.StatusOK, invite)
}

func (api *API) acceptInvite(w http.ResponseWriter, r *http.Request) {
	user, ok := api.requireUser(w, r)
	if !ok {
		return
	}
	invite, err := api.pg.AcceptInvite(r.Context(), chi.URLParam(r, "token"), user.ID, user.Email)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, invite)
}
