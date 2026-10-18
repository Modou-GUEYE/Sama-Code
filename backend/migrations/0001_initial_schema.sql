-- Sama Code — schéma initial.
-- Compatible PostgreSQL 14+ (gen_random_uuid est natif depuis PG 13).

CREATE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ---------------------------------------------------------------------------
-- Utilisateurs, rôles, authentification
-- ---------------------------------------------------------------------------

CREATE TABLE roles (
    id   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code text NOT NULL UNIQUE,
    name text NOT NULL
);

CREATE TABLE users (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    display_name        text NOT NULL,
    phone_number        text NOT NULL UNIQUE,          -- E.164, ex. +221771234567
    phone_verified_at   timestamptz,
    pin_hash            text,                          -- argon2id
    failed_pin_attempts integer NOT NULL DEFAULT 0 CHECK (failed_pin_attempts >= 0),
    locked_until        timestamptz,
    preferred_language  text NOT NULL DEFAULT 'fr' CHECK (preferred_language IN ('fr', 'wo')),
    learning_mode       text NOT NULL DEFAULT 'text' CHECK (learning_mode IN ('text', 'audio')),
    status              text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now()
);
CREATE TRIGGER users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE user_roles (
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role_id uuid NOT NULL REFERENCES roles (id) ON DELETE RESTRICT,
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE otp_codes (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    phone_number  text NOT NULL,
    purpose       text NOT NULL CHECK (purpose IN ('register', 'login', 'reset_pin')),
    code_hash     text NOT NULL,                       -- HMAC-SHA256, jamais le code en clair
    attempts      integer NOT NULL DEFAULT 0,
    expires_at    timestamptz NOT NULL,
    consumed_at   timestamptz,                         -- code vérifié ou remplacé
    token_used_at timestamptz,                         -- jeton de vérification utilisé (usage unique)
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX otp_codes_phone_created_idx ON otp_codes (phone_number, created_at DESC);

CREATE TABLE refresh_tokens (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash text NOT NULL UNIQUE,                   -- SHA-256 du jeton
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX refresh_tokens_user_idx ON refresh_tokens (user_id);

-- ---------------------------------------------------------------------------
-- Catégories de permis
-- ---------------------------------------------------------------------------

CREATE TABLE license_categories (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code                  text NOT NULL UNIQUE,
    name                  text NOT NULL,
    series_count          integer NOT NULL CHECK (series_count > 0),
    questions_per_series  integer NOT NULL CHECK (questions_per_series > 0),
    max_faults            integer NOT NULL CHECK (max_faults >= 0),
    exam_duration_minutes integer NOT NULL DEFAULT 20 CHECK (exam_duration_minutes > 0),
    sort_order            integer NOT NULL DEFAULT 0,
    is_active             boolean NOT NULL DEFAULT true,
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now()
);
CREATE TRIGGER license_categories_updated_at BEFORE UPDATE ON license_categories
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ---------------------------------------------------------------------------
-- Médias (fichiers dans le stockage objet, métadonnées ici)
-- ---------------------------------------------------------------------------

CREATE TABLE media_assets (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    media_type  text NOT NULL CHECK (media_type IN ('audio', 'image')),
    storage_key text NOT NULL UNIQUE,
    mime_type   text NOT NULL,
    size_bytes  bigint NOT NULL CHECK (size_bytes >= 0),
    checksum    text NOT NULL,                         -- SHA-256
    duration_ms integer,                               -- audio uniquement
    created_by  uuid REFERENCES users (id) ON DELETE SET NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);

-- ---------------------------------------------------------------------------
-- Cours : thèmes → cours → leçons → contenus (fr / wo, texte / audio / image)
-- ---------------------------------------------------------------------------

CREATE TABLE course_categories (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code          text NOT NULL UNIQUE,
    icon_media_id uuid REFERENCES media_assets (id) ON DELETE SET NULL,
    sort_order    integer NOT NULL DEFAULT 0,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);
CREATE TRIGGER course_categories_updated_at BEFORE UPDATE ON course_categories
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE course_category_translations (
    category_id    uuid NOT NULL REFERENCES course_categories (id) ON DELETE CASCADE,
    language_code  text NOT NULL CHECK (language_code IN ('fr', 'wo')),
    name           text NOT NULL,
    description    text,
    audio_media_id uuid REFERENCES media_assets (id) ON DELETE SET NULL,
    PRIMARY KEY (category_id, language_code)
);

CREATE TABLE courses (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id         uuid NOT NULL REFERENCES course_categories (id) ON DELETE RESTRICT,
    license_category_id uuid REFERENCES license_categories (id) ON DELETE RESTRICT,  -- NULL = tous permis
    status              text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'in_review', 'published', 'archived')),
    sort_order          integer NOT NULL DEFAULT 0,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX courses_category_idx ON courses (category_id);
CREATE INDEX courses_license_category_idx ON courses (license_category_id);
CREATE TRIGGER courses_updated_at BEFORE UPDATE ON courses
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE course_translations (
    course_id      uuid NOT NULL REFERENCES courses (id) ON DELETE CASCADE,
    language_code  text NOT NULL CHECK (language_code IN ('fr', 'wo')),
    title          text NOT NULL,
    audio_media_id uuid REFERENCES media_assets (id) ON DELETE SET NULL,
    PRIMARY KEY (course_id, language_code)
);

CREATE TABLE lessons (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id  uuid NOT NULL REFERENCES courses (id) ON DELETE CASCADE,
    sort_order integer NOT NULL DEFAULT 0,
    status     text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'in_review', 'published', 'archived')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX lessons_course_idx ON lessons (course_id);
CREATE TRIGGER lessons_updated_at BEFORE UPDATE ON lessons
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE lesson_translations (
    lesson_id      uuid NOT NULL REFERENCES lessons (id) ON DELETE CASCADE,
    language_code  text NOT NULL CHECK (language_code IN ('fr', 'wo')),
    title          text NOT NULL,
    audio_media_id uuid REFERENCES media_assets (id) ON DELETE SET NULL,
    PRIMARY KEY (lesson_id, language_code)
);

-- Blocs ordonnés d'une leçon, par langue.
CREATE TABLE lesson_contents (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    lesson_id     uuid NOT NULL REFERENCES lessons (id) ON DELETE CASCADE,
    language_code text NOT NULL CHECK (language_code IN ('fr', 'wo')),
    content_type  text NOT NULL CHECK (content_type IN ('text', 'audio', 'image')),
    sort_order    integer NOT NULL DEFAULT 0,
    text_content  text,
    media_id      uuid REFERENCES media_assets (id) ON DELETE RESTRICT,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CHECK ((content_type = 'text' AND text_content IS NOT NULL)
        OR (content_type <> 'text' AND media_id IS NOT NULL))
);
CREATE INDEX lesson_contents_lesson_idx ON lesson_contents (lesson_id, language_code, sort_order);
CREATE TRIGGER lesson_contents_updated_at BEFORE UPDATE ON lesson_contents
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE lesson_progress (
    user_id      uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    lesson_id    uuid NOT NULL REFERENCES lessons (id) ON DELETE CASCADE,
    status       text NOT NULL CHECK (status IN ('started', 'completed')),
    completed_at timestamptz,
    updated_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, lesson_id)
);

-- ---------------------------------------------------------------------------
-- Questions et propositions
-- ---------------------------------------------------------------------------

CREATE TABLE questions (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    question_type     text NOT NULL CHECK (question_type IN ('single', 'multiple')),
    related_lesson_id uuid REFERENCES lessons (id) ON DELETE SET NULL,  -- « revoir le cours associé »
    status            text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'in_review', 'published', 'archived')),
    version           integer NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);
CREATE TRIGGER questions_updated_at BEFORE UPDATE ON questions
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Énoncé et explication par langue, en texte et en audio (D23).
CREATE TABLE question_contents (
    id                         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    question_id                uuid NOT NULL REFERENCES questions (id) ON DELETE CASCADE,
    language_code              text NOT NULL CHECK (language_code IN ('fr', 'wo')),
    text_content               text NOT NULL,
    audio_media_id             uuid REFERENCES media_assets (id) ON DELETE RESTRICT,
    image_media_id             uuid REFERENCES media_assets (id) ON DELETE RESTRICT,
    explanation_text           text,
    explanation_audio_media_id uuid REFERENCES media_assets (id) ON DELETE RESTRICT,
    UNIQUE (question_id, language_code)
);

CREATE TABLE answer_options (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    question_id uuid NOT NULL REFERENCES questions (id) ON DELETE CASCADE,
    option_code text NOT NULL,                         -- A, B, C, D…
    is_correct  boolean NOT NULL DEFAULT false,
    sort_order  integer NOT NULL DEFAULT 0,
    UNIQUE (question_id, option_code),
    UNIQUE (id, question_id)                           -- cible de la FK composite de selected_options
);

CREATE TABLE option_contents (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    option_id      uuid NOT NULL REFERENCES answer_options (id) ON DELETE CASCADE,
    language_code  text NOT NULL CHECK (language_code IN ('fr', 'wo')),
    text_content   text NOT NULL,
    audio_media_id uuid REFERENCES media_assets (id) ON DELETE RESTRICT,
    image_media_id uuid REFERENCES media_assets (id) ON DELETE RESTRICT,
    UNIQUE (option_id, language_code)
);

-- ---------------------------------------------------------------------------
-- Séries d'examen et instantanés immuables (D15)
-- ---------------------------------------------------------------------------

-- Série éditable. Chaque publication fige un instantané dans exam_series_snapshots.
CREATE TABLE exam_series (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    license_category_id uuid NOT NULL REFERENCES license_categories (id) ON DELETE RESTRICT,
    series_number       integer NOT NULL CHECK (series_number > 0),
    status              text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'in_review', 'published', 'archived')),
    current_version     integer NOT NULL DEFAULT 0 CHECK (current_version >= 0),  -- 0 = jamais publiée
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    UNIQUE (license_category_id, series_number)
);
CREATE TRIGGER exam_series_updated_at BEFORE UPDATE ON exam_series
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE series_questions (
    series_id   uuid NOT NULL REFERENCES exam_series (id) ON DELETE CASCADE,
    question_id uuid NOT NULL REFERENCES questions (id) ON DELETE RESTRICT,
    position    integer NOT NULL CHECK (position > 0),
    PRIMARY KEY (series_id, position),
    UNIQUE (series_id, question_id)
);

CREATE TABLE exam_series_snapshots (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    series_id    uuid NOT NULL REFERENCES exam_series (id) ON DELETE RESTRICT,
    version      integer NOT NULL CHECK (version > 0),
    payload      jsonb NOT NULL,                       -- questions, ordre, contenus, propositions, bonnes réponses
    published_by uuid REFERENCES users (id) ON DELETE SET NULL,
    published_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (series_id, version)
);

-- ---------------------------------------------------------------------------
-- Sessions d'examen, réponses, résultats
-- ---------------------------------------------------------------------------

CREATE TABLE exam_sessions (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    series_id         uuid NOT NULL REFERENCES exam_series (id) ON DELETE RESTRICT,
    snapshot_id       uuid NOT NULL REFERENCES exam_series_snapshots (id) ON DELETE RESTRICT,
    status            text NOT NULL DEFAULT 'in_progress' CHECK (status IN ('in_progress', 'submitted')),
    submit_reason     text CHECK (submit_reason IN ('manual', 'timeout', 'abandoned')),
    origin            text NOT NULL DEFAULT 'online' CHECK (origin IN ('online', 'offline')),
    client_session_id uuid UNIQUE,                     -- idempotence de la synchro hors ligne
    duration_minutes  integer NOT NULL CHECK (duration_minutes > 0),
    started_at        timestamptz NOT NULL,
    expires_at        timestamptz NOT NULL,
    submitted_at      timestamptz,
    synced_at         timestamptz,
    created_at        timestamptz NOT NULL DEFAULT now(),
    CHECK ((status = 'submitted') = (submitted_at IS NOT NULL))
);
CREATE INDEX exam_sessions_user_idx ON exam_sessions (user_id, started_at DESC);
-- Une seule session en cours par candidat.
CREATE UNIQUE INDEX exam_sessions_one_in_progress_idx ON exam_sessions (user_id) WHERE status = 'in_progress';

CREATE TABLE candidate_answers (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id  uuid NOT NULL REFERENCES exam_sessions (id) ON DELETE CASCADE,
    question_id uuid NOT NULL REFERENCES questions (id) ON DELETE RESTRICT,
    answered_at timestamptz NOT NULL,
    UNIQUE (session_id, question_id),
    UNIQUE (id, question_id)
);

-- La FK composite garantit qu'une proposition cochée appartient à la question de la réponse.
CREATE TABLE selected_options (
    candidate_answer_id uuid NOT NULL,
    question_id         uuid NOT NULL,
    option_id           uuid NOT NULL,
    PRIMARY KEY (candidate_answer_id, option_id),
    FOREIGN KEY (candidate_answer_id, question_id) REFERENCES candidate_answers (id, question_id) ON DELETE CASCADE,
    FOREIGN KEY (option_id, question_id) REFERENCES answer_options (id, question_id) ON DELETE RESTRICT
);

CREATE TABLE exam_results (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id       uuid NOT NULL UNIQUE REFERENCES exam_sessions (id) ON DELETE CASCADE,
    correct_count    integer NOT NULL CHECK (correct_count >= 0),
    fault_count      integer NOT NULL CHECK (fault_count >= 0),
    unanswered_count integer NOT NULL CHECK (unanswered_count >= 0),
    max_faults       integer NOT NULL CHECK (max_faults >= 0),   -- seuil appliqué, figé
    passed           boolean NOT NULL,
    evaluated_at     timestamptz NOT NULL DEFAULT now()
);

-- Réponses en mode entraînement (thèmes à revoir).
CREATE TABLE practice_answers (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    question_id uuid NOT NULL REFERENCES questions (id) ON DELETE CASCADE,
    is_correct  boolean NOT NULL,
    answered_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX practice_answers_user_idx ON practice_answers (user_id, answered_at DESC);

-- ---------------------------------------------------------------------------
-- Audit
-- ---------------------------------------------------------------------------

CREATE TABLE audit_logs (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_user_id uuid REFERENCES users (id) ON DELETE SET NULL,
    action        text NOT NULL,
    entity_type   text NOT NULL,
    entity_id     uuid,
    metadata      jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX audit_logs_entity_idx ON audit_logs (entity_type, entity_id);
CREATE INDEX audit_logs_created_idx ON audit_logs (created_at DESC);
