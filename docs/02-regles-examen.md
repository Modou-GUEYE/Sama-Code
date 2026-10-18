# 02 — Règles fonctionnelles des examens

## Séries configurées

| Paramètre | Permis B | Permis C |
|---|---|---|
| Nombre de séries | 12 | 5 |
| Questions par série | 25 | 10 |
| Emplacements de questions | 300 | 50 |
| Choix de la série | Manuel (candidat) | Manuel (candidat) |
| Ordre des questions | Fixe | Fixe |
| Fautes autorisées | 4 max | 4 max |
| À partir de 5 fautes | Ajourné | Ajourné |

Ces valeurs sont des **paramètres** de `license_categories` (`questions_per_series`,
`max_faults`), modifiables par un administrateur — pas des constantes dans le code.

## Durée

- **20 minutes** par défaut, paramétrable par l'admin pour chaque catégorie de permis
  (`license_categories.exam_duration_minutes`).
- À la création de la session, le serveur enregistre `started_at` et
  `expires_at = started_at + durée`. Le chronomètre affiché sur le mobile se cale sur
  `expires_at`, pas sur l'horloge du téléphone.
- Fermer l'app **n'arrête pas** le chronomètre.
- En mode audio, le temps d'écoute **compte** dans la durée : même chronomètre pour
  tous les modes (décision D16).
- À l'expiration, la session est **soumise automatiquement** avec les réponses
  enregistrées (questions sans réponse = fautes). Toute réponse envoyée après
  `expires_at` est refusée.
- La durée est figée dans la session : la changer ensuite n'affecte pas les sessions
  en cours.
- Le chronomètre doit être compréhensible sans lire (barre ou cercle qui se vide,
  alerte sonore à 5 min et à 1 min).

## Tentatives

- Le candidat peut **refaire un examen autant de fois qu'il veut**, sur la même série
  ou une autre, sans limite par jour (décision D18).
- Chaque tentative est une **nouvelle session** ; les précédentes restent dans
  l'historique avec leur résultat.
- Une seule session `in_progress` à la fois par candidat : en démarrer une nouvelle
  propose de reprendre ou d'abandonner l'ancienne (abandon = soumission en l'état).

## Démarrage d'un examen

1. Le candidat choisit la catégorie de permis.
2. L'app affiche les séries **publiées** de cette catégorie.
3. Le candidat choisit une série.
4. Le backend vérifie : série publiée, appartenant à la catégorie, utilisateur autorisé.
5. Création d'une session liée à la série **et à sa version**.
6. Les questions sont renvoyées dans l'ordre de la série, **sans les bonnes réponses**.

## Évaluation

- Type de question : `single` (une bonne réponse) ou `multiple` (plusieurs).
- Réponse multiple correcte **uniquement** si l'ensemble coché = l'ensemble des bonnes
  réponses (ni manque, ni choix en trop), sauf barème officiel configuré plus tard.
- Question sans réponse à la soumission = **faute**.
- Nombre de fautes = nombre de **questions** incorrectes (pas de propositions).
- `passed = fault_count <= max_faults` → 0–4 fautes réussi, ≥ 5 ajourné.
- Résultat affiché : bonnes réponses, fautes, sans réponse, réussi/ajourné, puis
  corrections.

```
fault_count = incorrect_count + unanswered_count
correct_count + fault_count = questions_per_series
```

## Mode entraînement vs mode examen

Les deux modes utilisent **les mêmes questions** : celles des séries (D22). Seuls la
correction et le chronomètre changent.

| | Entraînement (QCM) | Examen blanc |
|---|---|---|
| Correction | Après chaque réponse validée | Seulement après soumission |
| Explication | Oui, immédiatement | Après soumission |
| Modifier sa réponse | Avant validation | Jusqu'à la soumission |
| Chronomètre | Non | Oui (20 min par défaut) |
| Enregistré | Résultats + thèmes à revoir | Session, réponses, résultat |

## Intégrité des sessions

- Une session conserve la série choisie et sa version.
- Fermer l'app ne change ni la série ni l'ordre → session reprenable.
- Soumission **idempotente** : une 2ᵉ soumission renvoie le même résultat.
- Session soumise = non modifiable.
- Corrections accessibles uniquement après soumission.
- Les changements de contenu ne modifient jamais les sessions passées.

## Hors connexion

L'examen blanc **hors ligne est autorisé** (décision D12).

- Le candidat télécharge au préalable l'instantané de la série (questions, propositions,
  médias, bonnes réponses chiffrées sur l'appareil).
- Le chronomètre de 20 min tourne localement.
- Le mobile calcule un résultat affiché comme **provisoire** (libellé + icône distincts).
- À la reconnexion, le mobile envoie la session (id généré côté client, réponses,
  horodatages). Le serveur **recalcule** le résultat contre l'instantané : c'est lui qui
  fait foi. L'envoi est idempotent (même id client = pas de doublon).
- Le serveur rejette une session hors ligne dont la version de série est inconnue ou
  dont la durée est incohérente.

## Critères d'acceptation (tests obligatoires)

| Cas | Attendu |
|---|---|
| Permis B, 4 fautes | Réussi |
| Permis B, 5 fautes | Ajourné |
| Permis C, 4 fautes | Réussi |
| Permis C, 5 fautes | Ajourné |
| Réponse multiple exacte | Correcte |
| Réponse multiple incomplète ou avec un choix erroné | Incorrecte |
| Question non répondue | Faute |
| Série d'une autre catégorie | Requête rejetée |
| Fermeture pendant une session | Session récupérable, même série et même ordre |
| Nouvelle version d'une série | Sessions historiques inchangées |
| Accès au résultat d'un autre candidat | Refusé |
| Correction demandée avant soumission | Bonnes réponses non exposées |
| Soumission répétée | Pas de double évaluation ni de résultat contradictoire |
| Temps écoulé (20 min) | Soumission automatique, questions sans réponse = fautes |
| Réponse envoyée après `expires_at` | Refusée |
| Durée modifiée par l'admin pendant une session | La session garde sa durée d'origine |
| Refaire la même série | Nouvelle session, l'ancienne reste dans l'historique |
| Examen hors ligne synchronisé | Résultat recalculé par le serveur, plus marqué provisoire |
| Même session hors ligne envoyée deux fois | Une seule session enregistrée |
