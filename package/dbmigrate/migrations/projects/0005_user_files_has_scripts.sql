-- "Мои файлы": a zip or FreeCAD model with Python scripts inside (.py, .FCMacro…) is accepted
-- and flagged; the ЛК warns the uploader and asks before every download.
ALTER TABLE user_files
  ADD COLUMN IF NOT EXISTS has_scripts BOOLEAN NOT NULL DEFAULT FALSE;
