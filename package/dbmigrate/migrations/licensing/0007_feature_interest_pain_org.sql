-- Interest survey: optional "what gets in the way" and "club or school, city" answers.
-- A fresh volume has them from robbo_licensing_db/init/13_feature_interest.sql.

ALTER TABLE feature_interest
  ADD COLUMN IF NOT EXISTS pain TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS org  TEXT NOT NULL DEFAULT '';

COMMENT ON COLUMN feature_interest.pain IS 'What takes the most time or gets in the way now (optional free text)';
COMMENT ON COLUMN feature_interest.org IS 'Club or school and city (optional), to tell answers of one club from many';
