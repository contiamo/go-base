-- citext provides a case-insensitive text field
CREATE EXTENSION IF NOT EXISTS citext;

-- trgm provides trigram indexes for searching
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- fuzzystrmatch is needed for levenshtein
CREATE EXTENSION IF NOT EXISTS fuzzystrmatch;


CREATE TABLE IF NOT EXISTS resources (
    resource_id uuid PRIMARY KEY,
    parent_id uuid REFERENCES resources ON DELETE CASCADE,
    kind CITEXT NOT NULL,
    name text NOT NULL
);