CREATE TABLE IF NOT EXISTS users (
    id text PRIMARY KEY,
    name text NOT NULL,
    email text NOT NULL UNIQUE,
    password_hash text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sessions (
    id text PRIMARY KEY,
    user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS companies (
    id text PRIMARY KEY,
    name text NOT NULL,
    billing boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS company_members (
    company_id text NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role text NOT NULL DEFAULT 'member',
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (company_id, user_id)
);

CREATE INDEX IF NOT EXISTS company_members_user_id_idx ON company_members(user_id);

CREATE TABLE IF NOT EXISTS channels (
    id text PRIMARY KEY,
    company_id text NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name text NOT NULL,
    icon text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS channels_company_id_idx ON channels(company_id);

CREATE TABLE IF NOT EXISTS api_tokens (
    id bigserial PRIMARY KEY,
    name text NOT NULL,
    token text NOT NULL UNIQUE,
    company_id text NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS api_tokens_company_id_idx ON api_tokens(company_id);

CREATE TABLE IF NOT EXISTS invites (
    id text PRIMARY KEY,
    email text NOT NULL,
    company_id text NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    token text NOT NULL UNIQUE,
    status text NOT NULL DEFAULT 'pending',
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS invites_pending_email_company_idx
    ON invites (lower(email), company_id)
    WHERE status = 'pending';
