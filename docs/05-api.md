# 05 — Contrat API REST

Préfixe `/api/v1`, JSON (sauf upload de fichiers). À formaliser dans
`backend/api/openapi.yaml`.

## Authentification et profil

| Méthode | Endpoint | Rôle |
|---|---|---|
| POST | `/auth/otp/request` | Envoyer un code SMS (`phone_number`, `purpose`) |
| POST | `/auth/otp/verify` | Vérifier le code ; renvoie un jeton temporaire ou des jetons de session |
| POST | `/auth/register` | Créer le compte après OTP vérifié (nom, langue, mode, PIN) |
| POST | `/auth/login` | Se connecter avec téléphone + PIN |
| POST | `/auth/pin/reset` | Définir un nouveau PIN après OTP vérifié |
| POST | `/auth/refresh` | Renouveler la session |
| POST | `/auth/logout` | Fermer la session |
| GET | `/me` | Consulter son profil |
| PATCH | `/me` | Modifier ses préférences (langue, mode écrit/audio) |

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
| GET | `/exam-series/{id}/offline-package` | Télécharger l'instantané pour l'examen hors ligne |
| POST | `/sync/exam-sessions` | Envoyer des sessions passées hors ligne (idempotent via `client_session_id`) ; le serveur recalcule et renvoie les résultats officiels |

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

## Administration

| Méthode | Endpoint | Rôle |
|---|---|---|
| POST | `/admin/courses` | Créer un cours |
| POST | `/admin/questions` | Créer une question |
| PATCH | `/admin/questions/{id}` | Modifier une question |
| POST | `/admin/exam-series` | Créer une série |
| PUT | `/admin/exam-series/{id}/questions` | Définir questions et ordre |
| POST | `/admin/exam-series/{id}/publish` | Publier une série validée |
| PATCH | `/admin/license-categories/{id}` | Modifier les paramètres d'une catégorie (dont `exam_duration_minutes`, `max_faults`) |
| GET | `/admin/statistics` | Statistiques |

Liste initiale : les CRUD complets (leçons, médias, utilisateurs, rôles, audit,
résultats) seront à ajouter pour couvrir les modules de l'admin.

## Exigences transverses

- Authentifier et autoriser chaque opération protégée.
- Vérifier que la session appartient au candidat (sinon 404, pour ne pas révéler l'existence).
- Vérifier que la question appartient à la série de la session.
- Refuser toute modification après soumission (409).
- Soumission idempotente (même résultat renvoyé).
- Jamais de `is_correct` ni d'explication avant correction autorisée.
- Format d'erreur unique, codes HTTP documentés.
- Pagination des listes volumineuses.
