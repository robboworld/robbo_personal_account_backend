-- Interest in the paid editions while tariffs are not approved: a signed-in user names their
-- role and the feature they need first ("Хочу узнать первым" in the ЛК). One row per account;
-- a repeat answer replaces the previous one.
-- Same as robbo_licensing_db/init/13_feature_interest.sql: a fresh volume ran it already, so it must stay
-- idempotent; existing databases get it here at backend start.

CREATE TABLE IF NOT EXISTS feature_interest (
  lms_user_id TEXT PRIMARY KEY,
  role        TEXT NOT NULL,
  feature     TEXT NOT NULL,
  other       TEXT NOT NULL DEFAULT '',
  notify      BOOLEAN NOT NULL DEFAULT false,
  source      TEXT NOT NULL DEFAULT '',
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE feature_interest IS 'Which paid-edition feature a user needs first (interest survey; tariffs not approved)';
COMMENT ON COLUMN feature_interest.source IS 'Where the survey was opened: landing, licenses, rs-desktop, rs-web';
