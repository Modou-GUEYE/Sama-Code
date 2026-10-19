-- Entraînement : progression comptée par série (une même question peut figurer
-- dans plusieurs séries). Les réponses antérieures, sans série, ne comptent plus.
ALTER TABLE practice_answers
    ADD COLUMN series_id uuid REFERENCES exam_series (id) ON DELETE CASCADE;

CREATE INDEX practice_answers_series_idx
    ON practice_answers (user_id, series_id, question_id, answered_at DESC);
