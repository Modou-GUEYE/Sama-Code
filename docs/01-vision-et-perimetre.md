# 01 — Vision et périmètre

## Problème

Les supports classiques du Code de la route (manuels, questions écrites) excluent une
grande partie de la population sénégalaise peu ou pas alphabétisée.

**Question centrale** : comment permettre à des personnes de niveaux de lecture,
d'écriture et d'aisance numérique différents d'apprendre les mêmes règles et de se
préparer au même examen théorique ?

## Objectif

Une application mobile inclusive pour **apprendre, réviser et simuler** l'examen du Code
de la route, avec des contenus écrits, audio et visuels.

## Public cible

| Profil | Besoins |
|---|---|
| **Lettré** | Lire cours, questions, propositions et explications ; consulter résultats et erreurs passées. |
| **Non scolarisé / difficultés de lecture** | Écouter cours, questions, propositions et explications ; répondre via des boutons visuels ; réécouter ; naviguer sans dépendre du texte. |

L'utilisateur choisit son **mode d'apprentissage** (écrit ou audio) à l'onboarding et
peut en changer dans les paramètres.

## Inclus (v1)

- App Flutter Android + iOS (iOS sous réserve de publication).
- Mode écrit et mode audio.
- Cours par thèmes et par catégorie de permis.
- QCM illustrés (réponse unique ou multiple) avec corrections.
- Examens blancs avec choix manuel de la série.
- Historique et suivi de progression, révision des erreurs.
- Administration web React (contenus, séries, publication, statistiques, audit).
- API REST Go + PostgreSQL + stockage objet pour les médias.
- Hors connexion pour les contenus téléchargés.

## Thèmes de cours (exemples)

Panneaux de signalisation · priorités · intersections · limitations de vitesse ·
dépassement · feux · arrêt et stationnement · sécurité routière · obligations du
conducteur · usagers vulnérables.

## Hors périmètre (ou à confirmer)

- Inscription officielle à l'examen, délivrance du permis.
- Certification officielle des simulations.
- Paiement / abonnement (modèle économique non décidé).
- Reconnaissance vocale des réponses.
- Toutes les langues nationales dès la v1.
- Gestion administrative des auto-écoles.

## Hypothèses

- Contenus, règles et seuils à valider par des personnes compétentes et à confronter
  aux textes en vigueur au Sénégal.
- Le seuil de 5 fautes est une **décision fonctionnelle du projet**, pas une
  certification réglementaire.
- La réussite dépend autant de la qualité des contenus (et des audios validés
  linguistiquement) que de la technique.
