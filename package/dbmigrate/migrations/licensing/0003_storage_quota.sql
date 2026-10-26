-- Account storage quota (all projects together; later also cloud files):
-- Free 100 MB (backend constant), Individual 1 GB, Class 5 GB. Over the quota nothing
-- is deleted: only actions that grow usage are refused.
-- Class becomes 30 devices / 30 sessions (a whole school class).
-- Same as robbo_licensing_db/init/10_storage_quota.sql: a fresh volume ran it already, so it must stay
-- idempotent; existing databases get it here at backend start.

ALTER TABLE products
  ADD COLUMN IF NOT EXISTS storage_quota_mb INT NOT NULL DEFAULT 1024;
ALTER TABLE licenses
  ADD COLUMN IF NOT EXISTS storage_quota_mb INT NOT NULL DEFAULT 1024;

COMMENT ON COLUMN products.storage_quota_mb IS 'Total storage per account in MB for licenses issued from this product';
COMMENT ON COLUMN licenses.storage_quota_mb IS 'Total storage per account in MB granted by this license';
COMMENT ON COLUMN products.cloud_quota_mb IS 'Max size of one project in MB for licenses issued from this product';
COMMENT ON COLUMN licenses.cloud_quota_mb IS 'Max size of one project in MB granted by this license';

UPDATE products
SET
  description = 'Индивидуальный: автообновление, проект до 50 МБ, 1 ГБ на аккаунт, 2 устройства, 2 сессии',
  storage_quota_mb = 1024
WHERE sku = 'rs3-premium-1y';

UPDATE products
SET
  description = 'Класс: автообновление, проект до 50 МБ, 5 ГБ на аккаунт, 30 устройств, 30 сессий',
  storage_quota_mb = 5120,
  seat_limit = 30,
  session_limit = 30
WHERE sku = 'rs3-class-1y';

-- Sync already-issued licenses with their product.
UPDATE licenses AS l
SET
  storage_quota_mb = p.storage_quota_mb,
  seat_limit = p.seat_limit,
  session_limit = p.session_limit,
  updated_at = now()
FROM products AS p
WHERE l.product_id = p.id
  AND (
    l.storage_quota_mb IS DISTINCT FROM p.storage_quota_mb
    OR l.seat_limit IS DISTINCT FROM p.seat_limit
    OR l.session_limit IS DISTINCT FROM p.session_limit
  );
