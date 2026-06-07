package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	DB *pgxpool.Pool
}

type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Session struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
}

type Company struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Billing bool   `json:"billing"`
}

type Channel struct {
	ID        string `json:"id"`
	CompanyID string `json:"company_id"`
	Name      string `json:"name"`
	Icon      string `json:"icon"`
}

type APIToken struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Token     string `json:"token"`
	CompanyID string `json:"company_id"`
}

type Member struct {
	UserID string `json:"user_id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}

type Invite struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	CompanyID   string    `json:"company_id"`
	CompanyName string    `json:"company_name,omitempty"`
	Token       string    `json:"token"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type CompanyDetail struct {
	Company  Company    `json:"company"`
	Channels []Channel  `json:"channels"`
	Tokens   []APIToken `json:"tokens"`
	Members  []Member   `json:"members"`
	Invites  []Invite   `json:"invites"`
}

func Connect(ctx context.Context, url string) (*Store, error) {
	db, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(ctx); err != nil {
		db.Close()
		return nil, err
	}
	store := &Store{DB: db}
	if err := store.Migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() {
	s.DB.Close()
}

func (s *Store) Migrate(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id text PRIMARY KEY,
			name text NOT NULL,
			email text NOT NULL UNIQUE,
			password_hash text NOT NULL,
			created_at timestamptz NOT NULL DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id text PRIMARY KEY,
			user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			expires_at timestamptz NOT NULL,
			created_at timestamptz NOT NULL DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS companies (
			id text PRIMARY KEY,
			name text NOT NULL,
			billing boolean NOT NULL DEFAULT false,
			created_at timestamptz NOT NULL DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS company_members (
			company_id text NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
			user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			role text NOT NULL DEFAULT 'member',
			created_at timestamptz NOT NULL DEFAULT now(),
			PRIMARY KEY (company_id, user_id)
		)`,
		`CREATE INDEX IF NOT EXISTS company_members_user_id_idx ON company_members(user_id)`,
		`CREATE TABLE IF NOT EXISTS channels (
			id text PRIMARY KEY,
			company_id text NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
			name text NOT NULL,
			icon text NOT NULL,
			created_at timestamptz NOT NULL DEFAULT now()
		)`,
		`CREATE INDEX IF NOT EXISTS channels_company_id_idx ON channels(company_id)`,
		`CREATE TABLE IF NOT EXISTS api_tokens (
			id bigserial PRIMARY KEY,
			name text NOT NULL,
			token text NOT NULL UNIQUE,
			company_id text NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
			created_at timestamptz NOT NULL DEFAULT now()
		)`,
		`CREATE INDEX IF NOT EXISTS api_tokens_company_id_idx ON api_tokens(company_id)`,
		`CREATE TABLE IF NOT EXISTS invites (
			id text PRIMARY KEY,
			email text NOT NULL,
			company_id text NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
			token text NOT NULL UNIQUE,
			status text NOT NULL DEFAULT 'pending',
			created_at timestamptz NOT NULL DEFAULT now(),
			expires_at timestamptz NOT NULL
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS invites_pending_email_company_idx
			ON invites (lower(email), company_id)
			WHERE status = 'pending'`,
	}
	for _, statement := range statements {
		if _, err := s.DB.Exec(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) CreateUser(ctx context.Context, id, name, email, passwordHash string) (User, error) {
	var user User
	err := s.DB.QueryRow(ctx,
		`INSERT INTO users (id, name, email, password_hash)
		 VALUES ($1, $2, lower($3), $4)
		 RETURNING id, name, email`,
		id, name, email, passwordHash,
	).Scan(&user.ID, &user.Name, &user.Email)
	return user, err
}

func (s *Store) UserPasswordHash(ctx context.Context, email string) (User, string, error) {
	var user User
	var hash string
	err := s.DB.QueryRow(ctx,
		`SELECT id, name, email, password_hash FROM users WHERE email = lower($1)`,
		email,
	).Scan(&user.ID, &user.Name, &user.Email, &hash)
	return user, hash, err
}

func (s *Store) UserBySession(ctx context.Context, sessionID string) (User, error) {
	var user User
	err := s.DB.QueryRow(ctx,
		`SELECT u.id, u.name, u.email
		 FROM sessions s
		 JOIN users u ON u.id = s.user_id
		 WHERE s.id = $1 AND s.expires_at > now()`,
		sessionID,
	).Scan(&user.ID, &user.Name, &user.Email)
	return user, err
}

func (s *Store) CreateSession(ctx context.Context, id, userID string, expiresAt time.Time) error {
	_, err := s.DB.Exec(ctx,
		`INSERT INTO sessions (id, user_id, expires_at) VALUES ($1, $2, $3)`,
		id, userID, expiresAt,
	)
	return err
}

func (s *Store) DeleteSession(ctx context.Context, id string) error {
	_, err := s.DB.Exec(ctx, `DELETE FROM sessions WHERE id = $1`, id)
	return err
}

func (s *Store) CreateCompany(ctx context.Context, id, name, ownerID string) (Company, error) {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return Company{}, err
	}
	defer tx.Rollback(ctx)

	var company Company
	err = tx.QueryRow(ctx,
		`INSERT INTO companies (id, name, billing) VALUES ($1, $2, false)
		 RETURNING id, name, billing`,
		id, name,
	).Scan(&company.ID, &company.Name, &company.Billing)
	if err != nil {
		return Company{}, err
	}
	_, err = tx.Exec(ctx,
		`INSERT INTO company_members (company_id, user_id, role) VALUES ($1, $2, 'owner')`,
		id, ownerID,
	)
	if err != nil {
		return Company{}, err
	}
	return company, tx.Commit(ctx)
}

func (s *Store) CompaniesForUser(ctx context.Context, userID string) ([]Company, error) {
	rows, err := s.DB.Query(ctx,
		`SELECT c.id, c.name, c.billing
		 FROM companies c
		 JOIN company_members cm ON cm.company_id = c.id
		 WHERE cm.user_id = $1
		 ORDER BY c.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCompanies(rows)
}

func (s *Store) IsCompanyMember(ctx context.Context, companyID, userID string) (bool, error) {
	var exists bool
	err := s.DB.QueryRow(ctx,
		`SELECT EXISTS (
			SELECT 1 FROM company_members WHERE company_id = $1 AND user_id = $2
		)`,
		companyID, userID,
	).Scan(&exists)
	return exists, err
}

func (s *Store) UpdateCompany(ctx context.Context, companyID, name string) (Company, error) {
	var company Company
	err := s.DB.QueryRow(ctx,
		`UPDATE companies SET name = $2 WHERE id = $1 RETURNING id, name, billing`,
		companyID, name,
	).Scan(&company.ID, &company.Name, &company.Billing)
	return company, err
}

func (s *Store) CompanyDetail(ctx context.Context, companyID string) (CompanyDetail, error) {
	var detail CompanyDetail
	err := s.DB.QueryRow(ctx,
		`SELECT id, name, billing FROM companies WHERE id = $1`,
		companyID,
	).Scan(&detail.Company.ID, &detail.Company.Name, &detail.Company.Billing)
	if err != nil {
		return detail, err
	}
	if detail.Channels, err = s.Channels(ctx, companyID); err != nil {
		return detail, err
	}
	if detail.Tokens, err = s.Tokens(ctx, companyID); err != nil {
		return detail, err
	}
	if detail.Members, err = s.Members(ctx, companyID); err != nil {
		return detail, err
	}
	if detail.Invites, err = s.Invites(ctx, companyID); err != nil {
		return detail, err
	}
	return detail, nil
}

func (s *Store) CreateChannel(ctx context.Context, id, companyID, name, icon string) (Channel, error) {
	var channel Channel
	err := s.DB.QueryRow(ctx,
		`INSERT INTO channels (id, company_id, name, icon)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, company_id, name, icon`,
		id, companyID, name, icon,
	).Scan(&channel.ID, &channel.CompanyID, &channel.Name, &channel.Icon)
	return channel, err
}

func (s *Store) Channels(ctx context.Context, companyID string) ([]Channel, error) {
	rows, err := s.DB.Query(ctx,
		`SELECT id, company_id, name, icon FROM channels WHERE company_id = $1 ORDER BY created_at DESC`,
		companyID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	channels := []Channel{}
	for rows.Next() {
		var channel Channel
		if err := rows.Scan(&channel.ID, &channel.CompanyID, &channel.Name, &channel.Icon); err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}
	return channels, rows.Err()
}

func (s *Store) CreateToken(ctx context.Context, companyID, name, token string) (APIToken, error) {
	var apiToken APIToken
	err := s.DB.QueryRow(ctx,
		`INSERT INTO api_tokens (company_id, name, token)
		 VALUES ($1, $2, $3)
		 RETURNING id, name, token, company_id`,
		companyID, name, token,
	).Scan(&apiToken.ID, &apiToken.Name, &apiToken.Token, &apiToken.CompanyID)
	return apiToken, err
}

func (s *Store) Tokens(ctx context.Context, companyID string) ([]APIToken, error) {
	rows, err := s.DB.Query(ctx,
		`SELECT id, name, token, company_id FROM api_tokens WHERE company_id = $1 ORDER BY created_at DESC`,
		companyID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tokens := []APIToken{}
	for rows.Next() {
		var token APIToken
		if err := rows.Scan(&token.ID, &token.Name, &token.Token, &token.CompanyID); err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}
	return tokens, rows.Err()
}

func (s *Store) DeleteToken(ctx context.Context, companyID string, tokenID int64) error {
	tag, err := s.DB.Exec(ctx, `DELETE FROM api_tokens WHERE id = $1 AND company_id = $2`, tokenID, companyID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *Store) VerifyAPITokenForChannel(ctx context.Context, token, channelID string) (bool, string, error) {
	var companyID string
	err := s.DB.QueryRow(ctx,
		`SELECT c.company_id
		 FROM api_tokens t
		 JOIN channels c ON c.company_id = t.company_id
		 WHERE t.token = $1 AND c.id = $2`,
		token, channelID,
	).Scan(&companyID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}
	return true, companyID, nil
}

func (s *Store) UserCanReadChannel(ctx context.Context, userID, channelID string) (bool, error) {
	var exists bool
	err := s.DB.QueryRow(ctx,
		`SELECT EXISTS (
			SELECT 1
			FROM channels c
			JOIN company_members cm ON cm.company_id = c.company_id
			WHERE c.id = $1 AND cm.user_id = $2
		)`,
		channelID, userID,
	).Scan(&exists)
	return exists, err
}

func (s *Store) Members(ctx context.Context, companyID string) ([]Member, error) {
	rows, err := s.DB.Query(ctx,
		`SELECT u.id, u.name, u.email, cm.role
		 FROM company_members cm
		 JOIN users u ON u.id = cm.user_id
		 WHERE cm.company_id = $1
		 ORDER BY cm.created_at ASC`,
		companyID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := []Member{}
	for rows.Next() {
		var member Member
		if err := rows.Scan(&member.UserID, &member.Name, &member.Email, &member.Role); err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	return members, rows.Err()
}

func (s *Store) CreateInvite(ctx context.Context, id, email, companyID, token string, expiresAt time.Time) (Invite, error) {
	var invite Invite
	err := s.DB.QueryRow(ctx,
		`INSERT INTO invites (id, email, company_id, token, status, expires_at)
		 VALUES ($1, lower($2), $3, $4, 'pending', $5)
		 RETURNING id, email, company_id, token, status, created_at, expires_at`,
		id, email, companyID, token, expiresAt,
	).Scan(&invite.ID, &invite.Email, &invite.CompanyID, &invite.Token, &invite.Status, &invite.CreatedAt, &invite.ExpiresAt)
	return invite, err
}

func (s *Store) Invites(ctx context.Context, companyID string) ([]Invite, error) {
	rows, err := s.DB.Query(ctx,
		`SELECT id, email, company_id, token, status, created_at, expires_at
		 FROM invites
		 WHERE company_id = $1
		 ORDER BY created_at DESC`,
		companyID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	invites := []Invite{}
	for rows.Next() {
		var invite Invite
		if err := rows.Scan(&invite.ID, &invite.Email, &invite.CompanyID, &invite.Token, &invite.Status, &invite.CreatedAt, &invite.ExpiresAt); err != nil {
			return nil, err
		}
		invites = append(invites, invite)
	}
	return invites, rows.Err()
}

func (s *Store) InviteByToken(ctx context.Context, token string) (Invite, error) {
	var invite Invite
	err := s.DB.QueryRow(ctx,
		`SELECT i.id, i.email, i.company_id, c.name, i.token, i.status, i.created_at, i.expires_at
		 FROM invites i
		 JOIN companies c ON c.id = i.company_id
		 WHERE i.token = $1`,
		token,
	).Scan(&invite.ID, &invite.Email, &invite.CompanyID, &invite.CompanyName, &invite.Token, &invite.Status, &invite.CreatedAt, &invite.ExpiresAt)
	return invite, err
}

func (s *Store) DeleteInvite(ctx context.Context, companyID, inviteID string) error {
	tag, err := s.DB.Exec(ctx, `DELETE FROM invites WHERE id = $1 AND company_id = $2`, inviteID, companyID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *Store) AcceptInvite(ctx context.Context, token, userID, userEmail string) (Invite, error) {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return Invite{}, err
	}
	defer tx.Rollback(ctx)

	var invite Invite
	err = tx.QueryRow(ctx,
		`SELECT id, email, company_id, token, status, created_at, expires_at
		 FROM invites
		 WHERE token = $1
		 FOR UPDATE`,
		token,
	).Scan(&invite.ID, &invite.Email, &invite.CompanyID, &invite.Token, &invite.Status, &invite.CreatedAt, &invite.ExpiresAt)
	if err != nil {
		return Invite{}, err
	}
	if invite.Status != "pending" {
		return Invite{}, fmt.Errorf("invite is %s", invite.Status)
	}
	if time.Now().After(invite.ExpiresAt) {
		return Invite{}, errors.New("invite expired")
	}
	if invite.Email != userEmail {
		return Invite{}, errors.New("invite email does not match current user")
	}
	_, err = tx.Exec(ctx,
		`INSERT INTO company_members (company_id, user_id, role)
		 VALUES ($1, $2, 'member')
		 ON CONFLICT (company_id, user_id) DO NOTHING`,
		invite.CompanyID, userID,
	)
	if err != nil {
		return Invite{}, err
	}
	_, err = tx.Exec(ctx, `UPDATE invites SET status = 'accepted' WHERE id = $1`, invite.ID)
	if err != nil {
		return Invite{}, err
	}
	invite.Status = "accepted"
	return invite, tx.Commit(ctx)
}

func scanCompanies(rows pgx.Rows) ([]Company, error) {
	companies := []Company{}
	for rows.Next() {
		var company Company
		if err := rows.Scan(&company.ID, &company.Name, &company.Billing); err != nil {
			return nil, err
		}
		companies = append(companies, company)
	}
	return companies, rows.Err()
}
