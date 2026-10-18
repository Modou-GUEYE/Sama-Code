# 03 — Architecture technique

## Vue d'ensemble

```
 Flutter (mobile, MVVM, SQLite)      React + TS (admin)
              \                         /
               \      HTTPS /api/v1    /
                ----> API Go (monolithe modulaire) ----> PostgreSQL (non exposé)
                                  |
                                  +--> Stockage objet S3 (audio, images)
```

Organisation cible du dépôt (monorepo) :

```
SAMA CODE/
├── CLAUDE.md
├── docs/
├── backend/   # Go
├── mobile/    # Flutter
└── admin/     # React + TypeScript
```

## Backend Go — monolithe modulaire

Choisi pour limiter la complexité initiale ; réplicable horizontalement plus tard.

```
backend/
├── cmd/api/main.go
├── internal/
│   ├── auth/         # connexion, jetons, refresh, déconnexion
│   ├── users/        # profils et préférences (langue, mode)
│   ├── licenses/     # catégories de permis, paramètres d'évaluation
│   ├── courses/      # thèmes, cours, leçons
│   ├── media/        # métadonnées audio/images, URLs signées
│   ├── questions/    # questions, propositions, bonnes réponses, explications
│   ├── examseries/   # séries fixes, versions, ordre
│   ├── examengine/   # démarrage, sauvegarde, soumission
│   ├── scoring/      # calcul des fautes et du résultat (pur, testé unitairement)
│   ├── progress/     # historique et progression
│   ├── sync/         # synchronisation hors connexion
│   ├── admin/        # opérations d'administration
│   └── audit/        # journal des actions sensibles
├── migrations/
├── api/openapi.yaml
└── tests/
```

Principes :
- `scoring` est une fonction pure (entrée : questions + réponses + `max_faults`),
  couverte par les cas de test de [02-regles-examen.md](02-regles-examen.md).
- Les modules ne lisent pas les tables des autres modules directement ; ils passent par
  leurs services.
- Erreurs API normalisées (format JSON unique + codes HTTP documentés).

## Mobile Flutter — MVVM

| Couche | Rôle |
|---|---|
| View | Écrans/widgets ; affiche l'état, transmet les interactions. |
| ViewModel | Chargement, état de l'écran, validation des interactions, erreurs. |
| Model | Objets métier et structures de données. |
| Repository | Accès unique aux données distantes (API) et locales (SQLite). |
| Services | Audio, téléchargements, synchronisation. |

```
mobile/lib/
├── core/        # api, database, audio, downloads, localization
├── data/        # models, repositories, datasources
├── features/    # onboarding, authentication, home, courses, lessons,
│                # quizzes, exams, results, progress, settings
│                # (chacune avec views/ et viewmodels/)
├── shared/widgets/
└── main.dart
```

Les règles critiques d'évaluation restent côté serveur.

## Administration React

Séparée du mobile, organisée par fonctionnalités : `dashboard`, `users`,
`license-categories`, `courses`, `lessons`, `media`, `questions`, `exam-series`,
`results`, `audit`. Utilise les mêmes API versionnées, avec des autorisations admin.

## Sécurité

- HTTPS obligatoire ; mots de passe hachés (argon2id ou bcrypt), jamais en clair.
- Rôles et autorisations vérifiés côté serveur ; isolation stricte des sessions
  et résultats entre candidats.
- Rate limiting sur les routes sensibles (login, register, submit).
- Journalisation des opérations admin (`audit_logs`).
- Secrets en variables d'environnement / gestionnaire de secrets.
- Collecte minimale ; conformité à la loi sénégalaise sur les données personnelles (CDP).

## Déploiement et exploitation

- Backend conteneurisé (Docker), configuration par environnement (dev / test / prod).
- PostgreSQL jamais exposé à Internet.
- Migrations contrôlées avant chaque déploiement.
- Logs, alertes d'erreur, health checks, surveillance de disponibilité.
- Sauvegardes automatisées + procédure de restauration testée.

## Exigences non fonctionnelles

| Domaine | Exigence |
|---|---|
| Performance | Chargement fluide, pagination, index sur recherches/jointures. |
| Résilience | Reprise de session, gestion réseau instable, sync idempotente. |
| Compatibilité | Tests sur Android d'entrée de gamme à haut de gamme ; iOS selon publication. |
| Auditabilité | Actions sensibles journalisées, version des contenus utilisée tracée. |
| Maintenabilité | Modules séparés, conventions, documentation, revues. |
