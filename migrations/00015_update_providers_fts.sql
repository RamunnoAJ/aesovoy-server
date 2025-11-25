-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION providers_tsv_update() RETURNS trigger AS $$
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

DROP TRIGGER IF EXISTS trg_providers_tsv ON providers;
CREATE TRIGGER trg_providers_tsv
  BEFORE INSERT OR UPDATE OF name, address, phone, email, reference, cuit
  ON providers
  FOR EACH ROW
  EXECUTE FUNCTION providers_tsv_update();

UPDATE providers SET name = name;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
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

UPDATE providers SET name = name;
-- +goose StatementEnd
