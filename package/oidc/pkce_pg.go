package oidc

import (
	"database/sql"
	"time"

	_ "github.com/lib/pq"
)

const pkceEnsureSQL = `
CREATE TABLE IF NOT EXISTS lk_oidc_pkce (
  state TEXT PRIMARY KEY,
  nonce TEXT NOT NULL DEFAULT '',
  code_verifier TEXT NOT NULL DEFAULT '',
  code_challenge TEXT NOT NULL DEFAULT '',
  return_to TEXT NOT NULL DEFAULT '',
  prompt TEXT NOT NULL DEFAULT '',
  kick_other_sessions BOOLEAN NOT NULL DEFAULT FALSE,
  expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_lk_oidc_pkce_expires ON lk_oidc_pkce (expires_at);
`

type postgresPKCEStore struct {
	db *sql.DB
}

func newPostgresPKCEStore(dsn string) (*postgresPKCEStore, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(8)
	db.SetConnMaxLifetime(30 * time.Minute)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := db.Exec(pkceEnsureSQL); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &postgresPKCEStore{db: db}, nil
}

func (s *postgresPKCEStore) Save(state string, entry PKCEEntry) error {
	_, _ = s.db.Exec(`DELETE FROM lk_oidc_pkce WHERE expires_at < NOW()`)
	expires := time.Now().Add(10 * time.Minute)
	_, err := s.db.Exec(`
		INSERT INTO lk_oidc_pkce
		  (state, nonce, code_verifier, code_challenge, return_to, prompt, kick_other_sessions, expires_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (state) DO UPDATE SET
		  nonce = EXCLUDED.nonce,
		  code_verifier = EXCLUDED.code_verifier,
		  code_challenge = EXCLUDED.code_challenge,
		  return_to = EXCLUDED.return_to,
		  prompt = EXCLUDED.prompt,
		  kick_other_sessions = EXCLUDED.kick_other_sessions,
		  expires_at = EXCLUDED.expires_at`,
		state, entry.Nonce, entry.CodeVerifier, entry.CodeChallenge,
		entry.ReturnTo, entry.Prompt, entry.KickOtherSessions, expires,
	)
	return err
}

func (s *postgresPKCEStore) scan(row *sql.Row) (PKCEEntry, bool) {
	var e PKCEEntry
	err := row.Scan(
		&e.State, &e.Nonce, &e.CodeVerifier, &e.CodeChallenge,
		&e.ReturnTo, &e.Prompt, &e.KickOtherSessions, &e.ExpiresAt,
	)
	if err != nil {
		return PKCEEntry{}, false
	}
	if time.Now().After(e.ExpiresAt) {
		return PKCEEntry{}, false
	}
	return e, true
}

func (s *postgresPKCEStore) Peek(state string) (PKCEEntry, bool) {
	row := s.db.QueryRow(`
		SELECT state, nonce, code_verifier, code_challenge, return_to, prompt, kick_other_sessions, expires_at
		FROM lk_oidc_pkce WHERE state = $1 AND expires_at > NOW()`, state)
	return s.scan(row)
}

func (s *postgresPKCEStore) Consume(state string) (PKCEEntry, bool) {
	row := s.db.QueryRow(`
		DELETE FROM lk_oidc_pkce
		WHERE state = $1 AND expires_at > NOW()
		RETURNING state, nonce, code_verifier, code_challenge, return_to, prompt, kick_other_sessions, expires_at`,
		state)
	return s.scan(row)
}
