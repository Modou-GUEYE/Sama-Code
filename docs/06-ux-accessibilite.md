# 06 — UX, accessibilité et charte graphique

## Principe directeur

Un utilisateur qui **ne sait pas lire** doit pouvoir faire tout le parcours :
s'orienter, écouter un cours, répondre à un QCM, passer un examen et comprendre son
résultat.

## Règles d'accessibilité

- Boutons grands (cible tactile ≥ 48 dp) et faciles à sélectionner.
- Textes lisibles, contraste suffisant (WCAG AA), icônes compréhensibles.
- Aucune information portée **uniquement** par la couleur (ex. bonne/mauvaise réponse :
  couleur + icône ✓/✗ + son).
- Audio rejouable partout ; vitesse de lecture réglable si possible.
- Navigation courte et cohérente, peu de niveaux.
- Propositions de réponse identifiables visuellement (lettres A/B/C/D + couleur + forme)
  et énoncées à l'audio.
- Changement de mode écrit/audio possible à tout moment dans les paramètres.
- Prévoir des appareils Android modestes et une connexion faible.

## Parcours principaux

1. **Onboarding** : choix de la langue (fr / wo, avec audio) → présentation courte →
   choix du mode (écrit / audio) → numéro de téléphone → code SMS → création du PIN.
   Aucun contenu n'est accessible sans compte (D19).
2. **Cours** : thèmes → cours → leçon (texte + audio + illustrations) → marquer terminé.
3. **Entraînement QCM** : question (texte/image/audio) → sélection → validation →
   correction + explication → thèmes à revoir.
4. **Examen blanc** : catégorie de permis → liste des séries → série → questions dans
   l'ordre (en mode audio, question + propositions lues automatiquement, D20) → soumission → résultat (bonnes, fautes, sans réponse, réussi/ajourné) →
   corrections → revoir les erreurs et les cours associés.
5. **Progression** : historique des examens, leçons terminées, erreurs à revoir.
6. **Hors connexion** (D24) : cours, illustrations et audios consultables sans réseau une fois
   téléchargés ; examen en ligne tolérant aux coupures courtes. Examen hors ligne : version 2.

## Charte graphique (à partir du logo)

Logo : [Logo Sama Code.png](../Logo%20Sama%20Code.png) — carte du Sénégal aux couleurs
du drapeau, feu tricolore, voiture blanche, panneaux (passage piéton, 50, direction),
« Sama » en vert, « Code » en orange avec un volant dans le « o ».

| Rôle | Couleur (approx., à confirmer) |
|---|---|
| Primaire — vert Sénégal | `#00853F` |
| Accent — jaune | `#FDEF42` |
| Alerte / erreur — rouge | `#E31B23` |
| Secondaire — orange « Code » | `#F7931E` → `#E8650A` |
| Texte | `#2B2B2B` |
| Fond | `#FFFFFF` |

Attention : le vert et le rouge ne doivent pas être le **seul** moyen de signaler
réussi/échoué (daltonisme) — toujours ajouter icône et libellé/son.
