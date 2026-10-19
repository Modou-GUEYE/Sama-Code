# Sama Code — administration

Interface web (React + TypeScript + Vite) pour créer, valider et publier les
contenus : questions fr/wo, séries d'examen, cours et leçons, médias, comptes.

## Lancer en développement

L'API doit tourner (`cd ../backend && make run`). Ensuite :

```bash
npm install        # une seule fois
npm run dev        # http://localhost:5173 — /api est relayé vers l'API (port 8080)
```

Connexion avec un compte Sama Code (téléphone + code secret) ayant le rôle
`editor`, `validator` ou `admin`. Premier administrateur :

```bash
cd ../backend && go run ./cmd/grant-role -phone "77 123 45 67" -role admin
```

## Rôles

| Rôle | Peut |
|---|---|
| Éditeur | Créer et modifier les contenus, les soumettre à validation |
| Validateur | Publier, dépublier, archiver |
| Administrateur | Tout, plus les comptes et les paramètres des permis |

## Vérifications

```bash
npx tsc -b && npm run build
npm run lint
```
