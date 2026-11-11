-- Add optional notes/description field to monitors.
ALTER TABLE monitors ADD COLUMN IF NOT EXISTS notes TEXT NOT NULL DEFAULT '';

INSERT INTO schema_migrations (version) VALUES ('003') ON CONFLICT DO NOTHING;
