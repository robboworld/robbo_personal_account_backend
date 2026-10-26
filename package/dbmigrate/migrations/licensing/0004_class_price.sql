-- Class tariff price: 35 000 ₽/year (school subscription, see Control docs/products).
-- Same as robbo_licensing_db/init/11_class_price.sql: a fresh volume ran it already, so it must stay
-- idempotent; existing databases get it here at backend start.

UPDATE products
SET amount = 35000.00
WHERE sku = 'rs3-class-1y';
