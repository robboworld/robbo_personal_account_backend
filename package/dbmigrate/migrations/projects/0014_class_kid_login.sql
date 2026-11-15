-- Class sign-in for young students (architecture/ADR_CLASS_KID_LOGIN.md). A teacher types the
-- names, the ЛК creates their LMS users; a child signs in on the class page by tapping their
-- avatar and five pictures in order, or with a printed QR card. Only classes kept in the ЛК
-- (not LMS-linked).

-- The class page: its own slug, apart from the invite slug (the page lists the children's names),
-- and the lesson window for picture sign-in. One row per class that has kid accounts.
-- room_code: six digits a teacher shows on the projector while the lesson is open; a laptop with
-- a wiped browser types the short /k address and the code instead of the class page address. A
-- new code each lesson, NULL once closed; a code left on an expired window is unused but unique.
CREATE TABLE IF NOT EXISTS class_kid_login_rooms (
  class_id UUID PRIMARY KEY REFERENCES class_invites (id) ON DELETE CASCADE,
  login_slug TEXT NOT NULL,
  open_until TIMESTAMPTZ NULL,
  room_code TEXT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT class_kid_login_rooms_code_digits CHECK (room_code ~ '^[0-9]{6}$')
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_class_kid_login_rooms_slug
  ON class_kid_login_rooms (login_slug);

CREATE UNIQUE INDEX IF NOT EXISTS uq_class_kid_login_rooms_code
  ON class_kid_login_rooms (room_code) WHERE room_code IS NOT NULL;

-- One row per kid account. user_id is the LMS user; the account belongs to this class only.
-- pictures are stored as they are: 9^5 = 59049 secrets are brute-forced offline in no time, so a
-- hash would not protect them, and teachers print them. The card token is 128 bits, only its
-- SHA-256 is kept.
CREATE TABLE IF NOT EXISTS class_kid_logins (
  user_id TEXT PRIMARY KEY,
  class_id UUID NOT NULL REFERENCES class_invites (id) ON DELETE CASCADE,
  display_name TEXT NOT NULL,
  avatar SMALLINT NOT NULL,
  pictures SMALLINT[] NOT NULL,
  card_token_hash TEXT NULL,
  failed_attempts INTEGER NOT NULL DEFAULT 0,
  locked_at TIMESTAMPTZ NULL,
  created_by TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_login_at TIMESTAMPTZ NULL,
  CONSTRAINT class_kid_logins_name_nonempty CHECK (length(trim(display_name)) > 0),
  CONSTRAINT class_kid_logins_pictures_len CHECK (array_length(pictures, 1) = 5)
);

CREATE INDEX IF NOT EXISTS idx_class_kid_logins_class
  ON class_kid_logins (class_id, created_at);

CREATE UNIQUE INDEX IF NOT EXISTS uq_class_kid_logins_card
  ON class_kid_logins (card_token_hash) WHERE card_token_hash IS NOT NULL;
