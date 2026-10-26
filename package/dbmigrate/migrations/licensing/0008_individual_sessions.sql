-- Individual: 10 concurrent sign-ins (was 2). A paid tariff had fewer than Free (5), so buying it
-- signed people out more often. Over the limit the oldest session ends rather than the sign-in
-- failing, and every school computer is a new device: 10 covers one family at home and at school,
-- while one account shared by a class still keeps pushing itself out. Class stays at 30.
-- Same as robbo_licensing_db/init/14_individual_sessions.sql: a fresh volume ran it already, so it
-- must stay idempotent; existing databases get it here at backend start.

UPDATE products
SET
  session_limit = 10,
  description = 'Индивидуальный: автообновление, проект до 50 МБ, 1 ГБ на аккаунт, 2 устройства, 10 сессий',
  updated_at = now()
WHERE sku = 'rs3-premium-1y'
  AND (session_limit IS DISTINCT FROM 10
    OR description IS DISTINCT FROM 'Индивидуальный: автообновление, проект до 50 МБ, 1 ГБ на аккаунт, 2 устройства, 10 сессий');

-- Licences already issued from the product get the new limit too.
UPDATE licenses AS l
SET session_limit = p.session_limit, updated_at = now()
FROM products AS p
WHERE l.product_id = p.id
  AND p.sku = 'rs3-premium-1y'
  AND l.session_limit IS DISTINCT FROM p.session_limit;
