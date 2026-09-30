CREATE SCHEMA IF NOT EXISTS imfohsa_private;
REVOKE ALL ON SCHEMA imfohsa_private FROM PUBLIC, anon, authenticated;
CREATE TABLE IF NOT EXISTS imfohsa_private.documents (
 key text PRIMARY KEY CHECK (key IN ('state','admins')),
 payload jsonb NOT NULL,
 version bigint NOT NULL DEFAULT 1 CHECK (version >= 1),
 updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS imfohsa_private.uploads (
 name text PRIMARY KEY CHECK (name <> '' AND position('/' in name)=0),
 content bytea NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE imfohsa_private.documents ENABLE ROW LEVEL SECURITY;
ALTER TABLE imfohsa_private.uploads ENABLE ROW LEVEL SECURITY;
REVOKE ALL ON ALL TABLES IN SCHEMA imfohsa_private FROM PUBLIC, anon, authenticated;
COMMENT ON SCHEMA imfohsa_private IS 'IMFOHSA: acceso exclusivo desde el servidor; fuera de la Data API.';
