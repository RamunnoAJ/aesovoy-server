-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS unaccent;

ALTER TABLE clients
  ADD COLUMN IF NOT EXISTS search_tsv tsvector;

CREATE OR REPLACE FUNCTION clients_tsv_update() RETURNS trigger AS $$
BEGIN
  NEW.search_tsv :=
    setweight(to_tsvector('spanish'::regconfig, unaccent(coalesce(NEW.name,''))), 'A') ||
    setweight(to_tsvector('simple'::regconfig,  unaccent(coalesce(NEW.email,''))), 'B') ||
    setweight(to_tsvector('simple'::regconfig,  unaccent(coalesce(NEW.reference,''))), 'B') ||
    setweight(to_tsvector('simple'::regconfig,  unaccent(coalesce(NEW.cuit,''))), 'B');
  RETURN NEW;
END
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_clients_tsv ON clients;
CREATE TRIGGER trg_clients_tsv
  BEFORE INSERT OR UPDATE OF name, email, reference, cuit
  ON clients
  FOR EACH ROW
  EXECUTE FUNCTION clients_tsv_update();

-- inicializar filas existentes
UPDATE clients SET name = name;

CREATE INDEX IF NOT EXISTS idx_clients_search
  ON clients USING GIN (search_tsv);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_clients_search;
DROP TRIGGER IF EXISTS trg_clients_tsv ON clients;
DROP FUNCTION IF EXISTS clients_tsv_update();
ALTER TABLE clients DROP COLUMN IF EXISTS search_tsv;
-- +goose StatementEnd
