-- +goose Up
-- +goose StatementBegin
ALTER TABLE products
  ADD COLUMN IF NOT EXISTS search_tsv tsvector;

CREATE OR REPLACE FUNCTION products_tsv_update() RETURNS trigger AS $$
BEGIN
  NEW.search_tsv :=
    setweight(to_tsvector('spanish'::regconfig, unaccent(coalesce(NEW.name,''))), 'A') ||
    setweight(to_tsvector('spanish'::regconfig, unaccent(coalesce(NEW.description,''))), 'B');
  RETURN NEW;
END
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_products_tsv ON products;
CREATE TRIGGER trg_products_tsv
  BEFORE INSERT OR UPDATE OF name, description
  ON products
  FOR EACH ROW
  EXECUTE FUNCTION products_tsv_update();

-- Update existing rows to populate search_tsv
UPDATE products SET name = name;

CREATE INDEX IF NOT EXISTS idx_products_search
  ON products USING GIN (search_tsv);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_products_search;
DROP TRIGGER IF EXISTS trg_products_tsv ON products;
DROP FUNCTION IF EXISTS products_tsv_update();
ALTER TABLE products DROP COLUMN IF EXISTS search_tsv;
-- +goose StatementEnd
