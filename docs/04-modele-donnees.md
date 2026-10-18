# 04 — Modèle de données PostgreSQL

## Conventions

- UUID pour les entités principales.
- Clés étrangères + contraintes d'unicité systématiques.
- `created_at`, `updated_at` quand pertinent.
- Index sur les colonnes de recherche et de jointure.
- Médias dans le stockage objet ; seules les métadonnées en base.
- Migrations versionnées (dans `backend/migrations/`).

## Tables (issues du cahier des charges)

| Table | Champs principaux |
|---|---|
| `users` | id, display_name, phone_number (unique, E.164 `+221…`), phone_verified_at, pin_hash (nullable), failed_pin_attempts, locked_until, preferred_language (`fr`/`wo`), learning_mode (`text`/`audio`), status, created_at |
| `otp_codes` | id, phone_number, code_hash, purpose (`register`/`login`/`reset_pin`), attempts, expires_at, consumed_at, created_at |
| `refresh_tokens` | id, user_id, token_hash, device_id, expires_at, revoked_at |
| `roles` | id, code, name |
| `user_roles` | user_id, role_id |
| `license_categories` | id, code, name, questions_per_series, max_faults, exam_duration_minutes (défaut 20), is_active |
| `course_categories` | id, name, description, sort_order |
| `courses` | id, category_id, license_category_id (nullable), title, status |
| `lessons` | id, course_id, title, sort_order, status |
| `media_assets` | id, media_type, storage_key, mime_type, size_bytes, checksum |
| `lesson_contents` | id, lesson_id, language_code, content_type, text_content, media_id (nullable) |
| `questions` | id, question_type, status, version |
| `question_contents` | id, question_id, language_code, text_content, audio_media_id, image_media_id (nullable), explanation_text, explanation_audio_media_id |
| `answer_options` | id, question_id, option_code, is_correct, sort_order |
| `option_contents` | id, option_id, language_code, text_content, media_id (nullable) |
| `exam_series` | id, license_category_id, series_number, status, version |
| `series_questions` | series_id, question_id, position |
| `exam_series_snapshots` | id, series_id, version, payload (JSONB immuable), published_at |
| `exam_sessions` | id, user_id, series_id, snapshot_id, status (`in_progress`/`submitted`/`expired`), origin (`online`/`offline`), client_session_id (unique, nullable), duration_minutes, started_at, expires_at, submitted_at, synced_at |
| `candidate_answers` | id, session_id, question_id, answered_at |
| `selected_options` | candidate_answer_id, option_id |
| `exam_results` | id, session_id, correct_count, fault_count, unanswered_count, passed, evaluated_at |
| `lesson_progress` | user_id, lesson_id, status, completed_at |
| `audit_logs` | id, actor_user_id, action, entity_type, entity_id, metadata, created_at |

## Écarts entre le schéma réel et le CDC

Le schéma réel est [backend/migrations/0001_initial_schema.sql](../backend/migrations/0001_initial_schema.sql).
Il s'écarte du CDC sur les points suivants :

- **Titres traduits** : `course_category_translations`, `course_translations`,
  `lesson_translations` (fr / wo + audio). Un illettré doit entendre le nom d'un thème.
- `license_categories.series_count` (12 pour B, 5 pour C) ajouté.
- `exam_series` est la série **éditable** (`UNIQUE (license_category_id, series_number)`,
  `current_version`) ; les versions publiées vivent dans `exam_series_snapshots`.
- `questions.related_lesson_id` pour « reprendre le cours associé ».
- `selected_options.question_id` + FK composites : la base garantit qu'une proposition
  cochée appartient bien à la question.
- `practice_answers` pour les résultats d'entraînement et les thèmes à revoir.
- `exam_results.max_faults` fige le seuil appliqué.

## Relations

- users 1—N exam_sessions
- license_categories 1—N exam_series
- exam_series N—N questions via series_questions
- questions 1—N answer_options
- exam_sessions 1—N candidate_answers
- candidate_answers N—N answer_options via selected_options
- exam_sessions 1—0..1 exam_results
- courses 1—N lessons ; lessons 1—N lesson_contents

## Contraintes à implémenter

- `UNIQUE (series_id, position)` et `UNIQUE (series_id, question_id)` sur `series_questions`.
- `UNIQUE (license_category_id, series_number, version)` sur `exam_series`.
- `UNIQUE (session_id, question_id)` sur `candidate_answers`.
- `UNIQUE (session_id)` sur `exam_results` (un seul résultat par session).
- Une proposition sélectionnée doit appartenir à la question de la réponse.
- Une question publiée a au moins une proposition correcte ; une question `single`
  en a exactement une.
- Une série publiée contient exactement `questions_per_series` questions.
- Contenu publié → modification = nouvelle version, jamais de réécriture.

- `UNIQUE (series_id, version)` sur `exam_series_snapshots`.
- Index partiel : une seule session `in_progress` par `user_id`.
- `UNIQUE (client_session_id)` pour rendre la synchronisation hors ligne idempotente.
- `UNIQUE (phone_number)` sur `users`.

## Versionnement (décision D15)

Une session doit rester fidèle au contenu **tel qu'il était** au moment de l'examen.
Référencer seulement un numéro de version ne suffit pas si les questions, propositions
ou bonnes réponses peuvent être modifiées ensuite.

À chaque publication d'une série, on crée un **instantané immuable**
(`exam_series_snapshots.payload`, JSONB) contenant : questions dans l'ordre, type,
contenus `fr` et `wo` (texte + références médias), propositions, bonnes réponses et
explications. `exam_sessions.snapshot_id` pointe vers cet instantané ; l'évaluation et
la correction se font contre lui.

C'est aussi ce payload (sans les bonnes réponses en clair) que le mobile télécharge
pour passer l'examen hors ligne.

## Authentification (décision D11)

- Inscription : téléphone → OTP SMS (6 chiffres, valable 5 min, 5 essais max) →
  création du PIN.
- Connexion : téléphone + PIN, ou téléphone + OTP.
- PIN oublié : OTP SMS → nouveau PIN.
- **PIN à 4 chiffres.** Refuser les PIN trop évidents (`0000`, `1234`, `1111`…).
- Codes OTP et PIN stockés **hachés** uniquement (argon2id).
- 5 PIN erronés consécutifs → compte verrouillé 15 min (`failed_pin_attempts`,
  `locked_until`) ; déblocage immédiat possible par OTP SMS. Seuils paramétrables.
- Un PIN à 4 chiffres n'a que 10 000 combinaisons : le verrouillage et le rate limiting
  par numéro et par IP sont **obligatoires**, pas optionnels.
