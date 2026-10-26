-- Concurrent sign-ins are counted per device (browser), not per client IP: behind a
-- school NAT every PC shares one IP, which made the session limit meaningless.
-- device_id comes from the long-lived lk_device cookie; '' = a client without the cookie,
-- matched by IP.
-- Same as robbo_licensing_db/init/12_session_devices.sql: a fresh volume ran it already, so it must stay
-- idempotent; existing databases get it here at backend start.

ALTER TABLE lk_user_sessions
  ADD COLUMN IF NOT EXISTS device_id TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_lk_user_sessions_device
  ON lk_user_sessions (lms_user_id, device_id)
  WHERE revoked_at IS NULL;

COMMENT ON COLUMN lk_user_sessions.device_id IS 'Browser id from the lk_device cookie; one active session per device counts against session_limit';
