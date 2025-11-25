-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS unaccent;

ALTER TABLE providers
  ADD COLUMN IF NOT EXISTS search_tsv tsvector;

CREATE OR REPLACE FUNCTION providers_tsv_update() RETURNS trigger AS $$
BEGIN
  NEW.search_tsv :=
    setweight(to_tsvector('spanish'::regconfig, unaccent(coalesce(NEW.name,''))), 'A') ||
    setweight(to_tsvector('simple'::regconfig,  unaccent(coalesce(NEW.email,''))), 'B') ||
    setweight(to_tsvector('simple'::regconfig,  unaccent(coalesce(NEW.reference,''))), 'B') ||
    setweight(to_tsvector('simple'::regconfig,  unaccent(coalesce(NEW.cuit,''))), 'B');
  RETURN NEW;
END
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_providers_tsv ON providers;
CREATE TRIGGER trg_providers_tsv
  BEFORE INSERT OR UPDATE OF name, email, reference, cuit
  ON providers
  FOR EACH ROW
  EXECUTE FUNCTION providers_tsv_update();

-- inicializar filas existentes
UPDATE providers SET name = name;

CREATE INDEX IF NOT EXISTS idx_providers_search
  ON providers USING GIN (search_tsv);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_providers_search;
DROP TRIGGER IF EXISTS trg_providers_tsv ON providers;
DROP FUNCTION IF EXISTS providers_tsv_update();
ALTER TABLE providers DROP COLUMN IF EXISTS search_tsv;
-- +goose StatementEnd
