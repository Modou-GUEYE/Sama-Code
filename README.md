# Sama Code

> Le Code de la route à la portée de tous.

Application d'apprentissage du Code de la route au Sénégal, pour les personnes
lettrées comme illettrées (français et wolof, texte et audio).

| Dossier | Contenu | État |
|---|---|---|
| [backend/](backend/) | API REST Go + migrations PostgreSQL | Phase 2 : socle et authentification |
| [mobile/](mobile/) | Application Flutter (MVVM) | Phase 5 |
| [admin/](admin/) | Administration React + TypeScript | Phase 3 |
| [docs/](docs/) | Cadrage fonctionnel et technique | À jour |

Contexte complet pour le développement : [CLAUDE.md](CLAUDE.md).

## Démarrer l'API en local

Prérequis : Go ≥ 1.25 et PostgreSQL ≥ 14, soit installé localement (Homebrew/pgAdmin),
soit via `docker compose up -d` (PostgreSQL sur le port 5433 + MinIO).

```bash
createdb samacode && createdb samacode_test   # une seule fois (PostgreSQL local)
cd backend
cp .env.example .env                          # puis remplacer les secrets
make run                                      # API sur http://localhost:8080
make test                                     # tests unitaires + intégration
```

En développement (`SMS_PROVIDER=log`), le code SMS s'affiche dans les journaux de l'API.

### Parcours d'inscription (curl)

```bash
API=http://localhost:8080/api/v1
curl -X POST $API/auth/otp/request -d '{"phone_number":"77 123 45 67","purpose":"register"}'
# lire le code dans les journaux de l'API
curl -X POST $API/auth/otp/verify  -d '{"phone_number":"771234567","purpose":"register","code":"XXXXXX"}'
curl -X POST $API/auth/register    -d '{"verification_token":"…","display_name":"Awa Diop","preferred_language":"wo","learning_mode":"audio","pin":"2580"}'
curl -X POST $API/auth/login       -d '{"phone_number":"771234567","pin":"2580"}'
curl $API/me -H "Authorization: Bearer <access_token>"
```

Contrat d'API : [backend/api/openapi.yaml](backend/api/openapi.yaml).

## Relancer le projet après une fermeture

PostgreSQL démarre seul à l'ouverture de session (`brew services`). Ensuite, deux terminaux :

```bash
# Terminal 1 — API (laisser ouvert)
cd "/Users/mac/Desktop/SAMA CODE/backend"
make run

# Terminal 2 — application mobile sur le téléphone branché en USB
export PATH="$PATH:$HOME/Library/Android/sdk/platform-tools"
adb devices                         # le téléphone doit apparaître en « device »
adb reverse tcp:8080 tcp:8080       # le téléphone joint l'API par le câble
cd "/Users/mac/Desktop/SAMA CODE/mobile"
flutter run --dart-define=API_BASE_URL=http://localhost:8080/api/v1
```

Dans `flutter run` : `r` = rechargement à chaud, `R` = redémarrage, `q` = quitter.
À refaire si le câble est débranché : `adb reverse tcp:8080 tcp:8080`.
