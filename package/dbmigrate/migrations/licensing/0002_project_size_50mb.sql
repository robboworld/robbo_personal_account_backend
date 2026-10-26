-- Max .sb3 size per project: 50 MB for every paid tariff (was Individual 100, Class 500).
-- Larger uploads do not fit the API read timeout on typical school uplinks; such projects
-- are kept on local storage for now. Already-issued licenses are lowered as well.
-- The backend also caps uploads at projectPage.maxUploadMB (default 50).
-- Same as robbo_licensing_db/init/09_project_size_50mb.sql: a fresh volume ran it already, so it must stay
-- idempotent; existing databases get it here at backend start.

UPDATE products
SET
  description = 'Индивидуальный: автообновление, проект до 50 МБ, 2 устройства, 2 сессии',
  cloud_quota_mb = 50
WHERE sku = 'rs3-premium-1y';

UPDATE products
SET
  description = 'Класс: автообновление, проект до 50 МБ, 20 устройств, 20 сессий',
  cloud_quota_mb = 50
WHERE sku = 'rs3-class-1y';

UPDATE products
SET cloud_quota_mb = 50
WHERE cloud_quota_mb > 50;

UPDATE licenses
SET
  cloud_quota_mb = 50,
  updated_at = now()
WHERE cloud_quota_mb > 50;
