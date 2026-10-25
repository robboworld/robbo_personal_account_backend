-- Tariff limits: max size of one project (cloud_quota_mb, per project — not a total)
-- + concurrent web-session cap, per product/license. 0 = unlimited. Free/Standard has no product row; its
-- defaults live as Go constants (licensing.FreeCloudQuotaMB / FreeSessionLimit /
-- FreeSeatLimit / FreeMaxProjects — currently 10 MB per project / 2 sessions /
-- 1 device / 20 projects). Paid tariffs have no project-count limit.
-- Applied automatically on first container start via docker-entrypoint-initdb.d.
-- Existing volumes: apply this file manually (init/ only runs on fresh volumes);
-- it is idempotent and brings existing rows to the current limits.

ALTER TABLE products
  ADD COLUMN IF NOT EXISTS cloud_quota_mb INT NOT NULL DEFAULT 10,
  ADD COLUMN IF NOT EXISTS session_limit INT NOT NULL DEFAULT 0;

ALTER TABLE licenses
  ADD COLUMN IF NOT EXISTS cloud_quota_mb INT NOT NULL DEFAULT 10,
  ADD COLUMN IF NOT EXISTS session_limit INT NOT NULL DEFAULT 0;

COMMENT ON COLUMN products.cloud_quota_mb IS 'Cloud project storage quota in MB for licenses issued from this product; 0 = unlimited';
COMMENT ON COLUMN products.session_limit IS 'Max concurrent web login sessions for this tariff; 0 = unlimited';
COMMENT ON COLUMN licenses.cloud_quota_mb IS 'Cloud project storage quota in MB granted by this license; 0 = unlimited';
COMMENT ON COLUMN licenses.session_limit IS 'Max concurrent web login sessions granted by this license; 0 = unlimited';

-- Individual: repurpose the existing dev seed (seat_limit=2 already matches).
UPDATE products
SET
  title = 'Individual',
  description = 'Индивидуальный: автообновление, проект до 100 МБ, 2 устройства, 2 сессии',
  cloud_quota_mb = 100,
  session_limit = 2
WHERE sku = 'rs3-premium-1y';

-- Class: same-account fixed tariff. Later intended as one teacher account +
-- linked student accounts with distinct functionality — not implemented yet.
INSERT INTO products (
  sku, title, description, amount, currency, seat_limit, capabilities,
  cloud_quota_mb, session_limit, duration_days, is_active
)
VALUES (
  'rs3-class-1y',
  'Class',
  'Класс: автообновление, проект до 500 МБ, 20 устройств, 20 сессий',
  4990.00,
  'RUB',
  20,
  ARRAY['premium.auto_update']::TEXT[],
  500,
  20,
  365,
  true
)
ON CONFLICT (sku) DO UPDATE SET
  description = EXCLUDED.description,
  seat_limit = EXCLUDED.seat_limit,
  cloud_quota_mb = EXCLUDED.cloud_quota_mb,
  session_limit = EXCLUDED.session_limit,
  updated_at = now();

-- Sync already-issued licenses with current product tariff limits.
UPDATE licenses AS l
SET
  cloud_quota_mb = p.cloud_quota_mb,
  session_limit = p.session_limit,
  updated_at = now()
FROM products AS p
WHERE l.product_id = p.id
  AND (
    l.cloud_quota_mb IS DISTINCT FROM p.cloud_quota_mb
    OR l.session_limit IS DISTINCT FROM p.session_limit
  );

