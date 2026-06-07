package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"raterlog-go/internal/scylla"
)

func (api *API) createLog(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ChannelID    string           `json:"channel_id"`
		EventName    string           `json:"event_name"`
		EventPayload string           `json:"event_payload"`
		Metadata     *json.RawMessage `json:"metadata"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	input.ChannelID = strings.TrimSpace(input.ChannelID)
	input.EventName = strings.TrimSpace(input.EventName)
	if input.ChannelID == "" || input.EventName == "" {
		writeError(w, http.StatusBadRequest, "channel_id and event_name are required")
		return
	}
	token := apiTokenFromRequest(r)
	if token == "" {
		writeError(w, http.StatusUnauthorized, "api token is required")
		return
	}
	ok, _, err := api.pg.VerifyAPITokenForChannel(r.Context(), token, input.ChannelID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to verify token")
		return
	}
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid api token")
		return
	}

	var metadata *string
	if input.Metadata != nil {
		value := string(*input.Metadata)
		metadata = &value
	}
	log := scylla.Log{
		ChannelID:    input.ChannelID,
		Timestamp:    time.Now().UTC(),
		EventName:    input.EventName,
		EventPayload: input.EventPayload,
		Metadata:     metadata,
	}
	if err := api.logs.CreateLog(r.Context(), log); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create log")
		return
	}
	writeJSON(w, http.StatusCreated, log)
}

func (api *API) getLogs(w http.ResponseWriter, r *http.Request) {
	channelID := chi.URLParam(r, "channelID")
	token := apiTokenFromRequest(r)
	if token != "" {
		ok, _, err := api.pg.VerifyAPITokenForChannel(r.Context(), token, channelID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to verify token")
			return
		}
		if !ok {
			writeError(w, http.StatusUnauthorized, "invalid api token")
			return
		}
	} else {
		user, ok := api.requireUser(w, r)
		if !ok {
			return
		}
		canRead, err := api.pg.UserCanReadChannel(r.Context(), user.ID, channelID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to check channel access")
			return
		}
		if !canRead {
			writeError(w, http.StatusForbidden, "channel access denied")
			return
		}
	}

	limit := 200
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	logs, err := api.logs.Logs(r.Context(), channelID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load logs")
		return
	}
	writeJSON(w, http.StatusOK, logs)
}
