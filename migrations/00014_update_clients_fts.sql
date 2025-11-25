-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION clients_tsv_update() RETURNS trigger AS $$
BEGIN
  NEW.search_tsv :=
    setweight(to_tsvector('spanish'::regconfig, unaccent(coalesce(NEW.name,''))), 'A') ||
    setweight(to_tsvector('spanish'::regconfig, unaccent(coalesce(NEW.address,''))), 'B') ||
    setweight(to_tsvector('simple'::regconfig,  unaccent(coalesce(NEW.phone,''))), 'C') ||
    setweight(to_tsvector('simple'::regconfig,  unaccent(coalesce(NEW.email,''))), 'B') ||
    setweight(to_tsvector('simple'::regconfig,  unaccent(coalesce(NEW.reference,''))), 'B') ||
    setweight(to_tsvector('simple'::regconfig,  unaccent(coalesce(NEW.cuit,''))), 'B');
  RETURN NEW;
END
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_clients_tsv ON clients;
CREATE TRIGGER trg_clients_tsv
  BEFORE INSERT OR UPDATE OF name, address, phone, email, reference, cuit
  ON clients
  FOR EACH ROW
  EXECUTE FUNCTION clients_tsv_update();

UPDATE clients SET name = name;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
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

UPDATE clients SET name = name;
-- +goose StatementEnd
