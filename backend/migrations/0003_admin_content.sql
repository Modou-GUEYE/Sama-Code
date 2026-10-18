-- Phase 3 : administration des contenus.

-- Nom du fichier tel qu'envoyé, pour le retrouver dans l'administration.
ALTER TABLE media_assets ADD COLUMN original_name text;

CREATE INDEX questions_status_idx ON questions (status, updated_at DESC);
CREATE INDEX exam_series_license_idx ON exam_series (license_category_id, series_number);
