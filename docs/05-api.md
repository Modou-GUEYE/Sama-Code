# 05 — Contrat API REST

Préfixe `/api/v1`, JSON (sauf upload de fichiers). À formaliser dans
`backend/api/openapi.yaml`.

## Authentification et profil

| Méthode | Endpoint | Rôle |
|---|---|---|
| POST | `/auth/otp/request` | Envoyer un code SMS (`phone_number`, `purpose`) |
| POST | `/auth/otp/verify` | Vérifier le code ; renvoie un jeton temporaire ou des jetons de session |
| POST | `/auth/register` | Créer le compte après OTP vérifié (nom, langue, mode, PIN, `terms_version` acceptée) |
| POST | `/auth/login` | Se connecter avec téléphone + PIN |
| POST | `/auth/pin/reset` | Définir un nouveau PIN après OTP vérifié |
| POST | `/auth/refresh` | Renouveler la session |
| POST | `/auth/logout` | Fermer la session |
| GET | `/me` | Consulter son profil |
| PATCH | `/me` | Modifier ses préférences (langue d'écoute, mode écrit/audio) |
| POST | `/me/terms-acceptance` | Accepter la version en vigueur des CGU (`{"version"}`, idempotent ; 409 `terms_outdated` si ancienne) |

## Cours et médias

Toutes ces routes exigent un compte connecté (D19).

| Méthode | Endpoint | Rôle |
|---|---|---|
| GET | `/course-categories` | Lister les thèmes |
| GET | `/courses` | Lister les cours publiés |
| GET | `/courses/{courseId}` | Consulter un cours |
| GET | `/lessons/{lessonId}` | Leçon + contenus (texte, audio, images) |
| GET | `/media/{mediaId}` | Infos d'un média autorisé (URL signée) |
| POST | `/lessons/{lessonId}/completion` | Marquer une leçon terminée |

## Permis et séries

| Méthode | Endpoint | Rôle |
|---|---|---|
| GET | `/license-categories` | Lister les catégories de permis |
| GET | `/license-categories/{id}/series` | Séries publiées d'une catégorie |
| GET | `/exam-series/{id}` | Infos d'une série |
| GET | `/exam-series/{id}/questions` | Questions (entraînement), **sans** bonnes réponses |
| POST | `/exam-series/{id}/questions/{questionId}/check` | Entraînement : valider une réponse, renvoie correction + explication de **cette** question uniquement |

## Entraînement (implémenté)

| Méthode | Endpoint | Rôle |
|---|---|---|
| GET | `/practice` | Permis, séries publiées, progression (questions travaillées / réussies) et nombre d'erreurs à revoir |
| GET | `/practice/series/{id}/questions` | Questions de la série, dans l'ordre, **sans** bonnes réponses |
| GET | `/practice/license-categories/{id}/mistakes` | Questions dont la dernière réponse d'entraînement est fausse |
| POST | `/practice/series/{id}/questions/{questionId}/check` | Corriger une réponse `{option_ids}` : résultat, bonnes réponses, explications ; enregistrée dans `practice_answers` |

## Sessions d'examen

| Méthode | Endpoint | Rôle |
|---|---|---|
| POST | `/exam-sessions` | Créer une session pour la série choisie |
| GET | `/exam-sessions/{id}` | Reprendre / consulter sa session |
| PUT | `/exam-sessions/{id}/answers/{questionId}` | Enregistrer ou modifier une réponse |
| POST | `/exam-sessions/{id}/submit` | Soumettre (idempotent) |
| GET | `/exam-sessions/{id}/result` | Résultat (après soumission) |
| GET | `/exam-sessions/{id}/review` | Corrections (après soumission) |
| GET | `/exam-sessions` | Historique de ses tentatives (paginé) |
| GET *(V2, D24)* | `/exam-series/{id}/offline-package` | Télécharger l'instantané pour l'examen hors ligne |
| POST *(V2, D24)* | `/sync/exam-sessions` | Envoyer des sessions passées hors ligne (idempotent via `client_session_id`) ; le serveur recalcule et renvoie les résultats officiels |

Chronomètre : la réponse de création/reprise de session contient `expires_at` et
`server_time` pour que le mobile calcule le temps restant sans dépendre de l'horloge
du téléphone. Après `expires_at`, `PUT .../answers` renvoie 409 et la session est
soumise automatiquement.

Démarrage :

```json
POST /api/v1/exam-sessions
{
  "license_category_id": "uuid-categorie",
  "series_id": "uuid-serie"
}
```

Le serveur vérifie catégorie, série publiée, appartenance série ↔ catégorie et
autorisation, puis renvoie la session et les questions dans l'ordre, sans les
bonnes réponses.

## Administration (implémentée en phase 3)

Toutes les routes `/admin/*` exigent un rôle `editor`, `validator` ou `admin` ;
celles marquées **admin** exigent le rôle `admin`. Les changements de rôle prennent
effet au renouvellement du jeton d'accès (15 min au plus).

**Cycle de publication** (`POST …/status {"status": …}`) : `draft → in_review`
(éditeur), `in_review → published` (validateur), `published → draft | archived`
(validateur), `archived → draft` (admin). Soumettre ou publier exige le texte
**français** complet. Le wolof n'a que des audios (D25) : un texte wolof est refusé
(400) ; les audios manquants sont signalés dans l'admin (obligatoires avant le pilote).

| Méthode | Endpoint | Rôle |
|---|---|---|
| GET | `/admin/statistics` | Tableau de bord : comptes, contenus par statut, séries publiées par permis |
| GET | `/admin/license-categories` | Permis, y compris inactifs |
| PATCH | `/admin/license-categories/{id}` | **admin** — nom, séries, questions par série, fautes max, durée |
| GET | `/admin/users?search=` | **admin** — comptes et rôles |
| PUT | `/admin/users/{id}/roles` | **admin** — remplacer les rôles (pas d'auto-rétrogradation) |
| POST | `/admin/media` | Envoyer un fichier (multipart `file`) : PNG/JPEG/WEBP ≤ 5 Mo, MP3/M4A/OGG/WAV ≤ 15 Mo |
| GET | `/admin/media?type=audio\|image` | Médias récents |
| GET | `/media/{id}/file` | **public** — lire un fichier (cache immuable, lecture partielle) |
| GET | `/admin/questions?status=&search=&page=` | Liste paginée |
| POST | `/admin/questions` | Créer (texte fr obligatoire ; wo : audios seulement) |
| GET / PUT | `/admin/questions/{id}` | Lire / remplacer. Modifier une question publiée la repasse en brouillon (version + 1) ; refusé si elle est dans une série publiée |
| POST | `/admin/questions/{id}/status` | Cycle de publication |
| GET | `/admin/exam-series?license_category_id=` | Séries |
| POST | `/admin/exam-series` | Créer `{license_category_id, series_number}` |
| GET | `/admin/exam-series/{id}` | Série + questions dans l'ordre |
| PUT | `/admin/exam-series/{id}/questions` | Composition et ordre `{question_ids: […]}` ; repasse la série en brouillon |
| POST | `/admin/exam-series/{id}/status` | Publier = nombre exact de questions, toutes publiées → **instantané immuable versionné** |
| GET / POST | `/admin/course-categories` | Thèmes |
| PUT | `/admin/course-categories/{id}` | Modifier un thème |
| GET / POST | `/admin/courses?category_id=` | Cours |
| GET / PUT | `/admin/courses/{id}` | Cours + leçons |
| POST | `/admin/courses/{id}/status` | Cycle de publication |
| POST | `/admin/courses/{id}/lessons` | Créer une leçon |
| GET / PUT | `/admin/lessons/{id}` | Leçon + blocs |
| PUT | `/admin/lessons/{id}/contents` | Blocs `{fr: [{content_type, text, media_id}], wo: [audios seulement]}` |
| POST | `/admin/lessons/{id}/status` | Cycle de publication |

**Suppression définitive** (`DELETE`) : `/admin/questions/{id}`, `/admin/exam-series/{id}`,
`/admin/course-categories/{id}`, `/admin/courses/{id}` (avec ses leçons), `/admin/lessons/{id}`.
Un brouillon peut être supprimé par un éditeur ; sinon validateur ou admin (thème : validateur ou admin).
Refusée (409) si elle casserait l'historique : question dans une série ou déjà répondue,
série déjà passée par un candidat, thème contenant des cours. Il faut alors archiver.

Premier administrateur : `go run ./cmd/grant-role -phone "77 123 45 67" -role admin`
(le compte doit exister : inscription depuis l'application).

## Exigences transverses

- Authentifier et autoriser chaque opération protégée.
- Vérifier que la session appartient au candidat (sinon 404, pour ne pas révéler l'existence).
- Vérifier que la question appartient à la série de la session.
- Refuser toute modification après soumission (409).
- Soumission idempotente (même résultat renvoyé).
- Jamais de `is_correct` ni d'explication avant correction autorisée.
- Format d'erreur unique, codes HTTP documentés.
- Pagination des listes volumineuses.
