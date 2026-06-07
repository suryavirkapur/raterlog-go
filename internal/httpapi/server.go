package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"raterlog-go/internal/config"
	"raterlog-go/internal/postgres"
	"raterlog-go/internal/scylla"
)

const sessionCookieName = "raterlog_session"

type API struct {
	cfg  config.Config
	pg   *postgres.Store
	logs *scylla.Store
}

func New(cfg config.Config, pg *postgres.Store, logs *scylla.Store) http.Handler {
	api := &API{cfg: cfg, pg: pg, logs: logs}

	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Session-Token"},
		AllowCredentials: true,
		MaxAge:           3600,
	}))

	r.Get("/health", api.health)

	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/signup", api.signup)
		r.Post("/signin", api.signin)
		r.Post("/signout", api.signout)
		r.Get("/me", api.me)
	})

	r.Get("/api/companies", api.listCompanies)
	r.Post("/api/companies", api.createCompany)
	r.Get("/api/companies/{companyID}", api.companyDetail)
	r.Patch("/api/companies/{companyID}", api.updateCompany)
	r.Post("/api/companies/{companyID}/channels", api.createChannel)
	r.Get("/api/companies/{companyID}/members", api.members)
	r.Post("/api/companies/{companyID}/invites", api.createInvite)
	r.Delete("/api/companies/{companyID}/invites/{inviteID}", api.deleteInvite)
	r.Post("/api/companies/{companyID}/tokens", api.createToken)
	r.Delete("/api/companies/{companyID}/tokens/{tokenID}", api.deleteToken)

	r.Get("/api/invites/{token}", api.invite)
	r.Post("/api/invites/{token}/accept", api.acceptInvite)

	r.Post("/api/logs", api.createLog)
	r.Get("/api/logs/{channelID}", api.getLogs)
	r.Post("/log", api.createLog)
	r.Get("/log/{channelID}", api.getLogs)

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "not found")
	})
	return r
}

func (api *API) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
