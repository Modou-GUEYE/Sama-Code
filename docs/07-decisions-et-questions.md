# 07 — Décisions, questions ouvertes et phasage

## Décisions prises

### Cahier des charges v1.0

| # | Décision |
|---|---|
| D1 | Nom du projet : **Sama Code**. |
| D2 | Cours en format écrit **et** audio, avec illustrations. |
| D3 | QCM à réponse unique ou multiple. |
| D4 | Le candidat choisit lui-même sa série d'examen. |
| D5 | Séries et ordre des questions fixes. |
| D6 | Permis B : 12 séries × 25 questions. Permis C : 5 séries × 10 questions. |
| D7 | 0–4 fautes = réussi ; ≥ 5 fautes = ajourné (B et C). |
| D8 | Stack : Flutter, Go, React, PostgreSQL. |
| D9 | Mobile en MVVM ; backend en monolithe modulaire. |

### Cadrage du 18/10/2026

| # | Décision | Détail |
|---|---|---|
| D10 | Langues du MVP : **français + wolof** (`fr`, `wo`). | Tous les contenus (cours, questions, propositions, explications) existent dans les deux langues, en texte et en audio. |
| D11 | Authentification par **numéro de téléphone** : code SMS (OTP) **ou** code PIN. | Inscription et récupération par OTP SMS ; connexion quotidienne par PIN (moins coûteux en SMS) ou par OTP. Voir [05-api.md](05-api.md). |
| D12 | ~~Examen blanc hors connexion autorisé.~~ **Remplacée par D24 (19/10/2026).** | — |
| D13 | **Durée de l'examen : 20 minutes**, paramétrable par l'admin. | Champ `exam_duration_minutes` par catégorie de permis. À l'expiration, soumission automatique. |
| D14 | **Un candidat peut refaire un examen** autant de fois qu'il le souhaite. | Chaque tentative est une nouvelle session ; l'historique conserve toutes les tentatives. |
| D15 | Versionnement par **instantané immuable** à la publication d'une série. | Retenu par défaut (voir [04-modele-donnees.md](04-modele-donnees.md)) : indispensable pour que les tentatives et examens hors ligne passés restent corrigeables. |
| D16 | En mode audio, le **temps d'écoute compte** dans la durée de l'examen. | Même chronomètre de 20 min pour tous les modes. |
| D17 | **PIN à 4 chiffres.** Blocage après **5 erreurs** consécutives, pendant **15 min** (valeurs par défaut, paramétrables). | Déblocage possible immédiatement par OTP SMS. Seuil à confirmer. |
| D18 | **Tentatives illimitées**, sans limite par jour. | — |
| D19 | **Compte obligatoire** : aucun contenu accessible sans connexion. | Toutes les routes de contenu exigent un jeton. L'onboarding se termine par l'inscription. |
| D20 | En examen audio, chaque question **et ses propositions sont lues automatiquement** à l'affichage. | Bouton « réécouter » toujours disponible ; la lecture compte dans le temps (D16). |
| D21 | Les contenus (cours, audios fr + wo, illustrations, questions, séries) sont **produits et validés dans l'administration**. | Via les rôles éditeur pédagogique et validateur ; cycle brouillon → validation → publication. |
| D22 | **Les questions d'entraînement sont celles des séries d'examen.** | Mode entraînement = même série, mais correction après chaque question. Une banque de questions unique. |
| D24 | **Hors connexion allégé pour le MVP** (19/10/2026) : cours, illustrations et audios consultables hors ligne une fois téléchargés ; examen tolérant aux coupures courtes (réponses renvoyées automatiquement). **L'examen complet sans réseau (résultat provisoire + synchronisation) est reporté en version 2**, selon les retours du pilote. | Moins de risque (bonnes réponses stockées sur le téléphone, synchronisation) ; on garde l'essentiel pour le public : apprendre sans réseau et économiser les données. |
| D23 | **Explications traduites et enregistrées en audio** (fr + wo). | `explanation` déplacée de `questions` vers `question_contents`. |

## Questions ouvertes

À trancher avant ou pendant la phase concernée. Noter la réponse ici avec la date.

| # | Question | Impact |
|---|---|---|
| Q8 | Hébergement cible (cloud, fournisseur local) et stockage objet retenu ? **À discuter au moment venu.** En attendant : Docker en local, MinIO pour simuler le stockage S3. | Déploiement |
| Q12 | Fournisseur SMS pour les OTP (Orange, Free, agrégateur…) et budget SMS ? **À discuter au moment venu** (phase 2). En attendant : interface `SMSSender` avec une implémentation de dev qui écrit le code dans les logs. | Module `auth`, coûts |

## Phasage

| Phase | Livrables | Statut |
|---|---|---|
| 1. Cadrage et conception | Parcours UX, règles détaillées, modèle de données, architecture, validation des contenus | Terminé (maquettes UX à faire) |
| 2. Socle technique | Dépôts, environnements, API Go, PostgreSQL, migrations, auth, CI/CD | Terminé le 18/10/2026 (fournisseur SMS et hébergement en attente) |
| 3. Administration pédagogique | Catégories, cours, médias, questions, séries, publication | API faite (18/10/2026) ; interface React en cours |
| 4. Moteur d'examen | Choix de série, sessions, chronomètre, réponses, soumission, calcul, résultats | Fait (19/10/2026) : API + écrans mobiles ; 13 critères d'acceptation testés |
| 5. Application Flutter | MVVM, cours écrits/audio, QCM, examens, résultats, progression | À faire |
| 6. Cours hors ligne (D24) | Cours, illustrations et audios téléchargés et consultables sans réseau | À faire — examen hors ligne reporté en V2 |
| 7. Validation pilote | Tests fonctionnels, accessibilité, contenus, sécurité, pilote | À faire |

## Risques principaux

| Risque | Mesure |
|---|---|
| Contenu non conforme aux règles locales | Validation par des experts, vérification des textes |
| Audio mal compris / traduction wolof approximative | Validation par des locuteurs, tests avec le public |
| Réseau instable / perte de session | Sauvegarde progressive, reprise, sync idempotente |
| Fuite des bonnes réponses | Exclues des API avant correction, tests dédiés |
| Fuite des réponses via le mode hors ligne (V2) | Évitée dans le MVP : aucune bonne réponse n'est stockée sur le téléphone (D24) |
| Modification des séries historiques | Versionnement immuable |
| Interface difficile pour certains | Tests d'usage avec les deux profils |
| Coût de production audio (2 langues) | Prioriser les leçons essentielles |
| Coût des SMS OTP | PIN pour la connexion quotidienne, OTP limité à l'inscription et à la récupération |

## Livrables attendus

CDC validé · maquettes UX/UI · spec OpenAPI · schéma + migrations PostgreSQL · code
Flutter MVVM · code Go · code React · jeu initial de contenus validés · tests
automatisés + rapport · documentation de déploiement et d'exploitation.
