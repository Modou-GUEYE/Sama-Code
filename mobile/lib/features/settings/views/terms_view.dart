import 'package:flutter/material.dart';

import '../../../core/config.dart';
import '../../../shared/widgets/surfaces.dart';

/// Conditions générales d'utilisation.
///
/// ⚠️ Projet rédigé pour le MVP : à faire relire et valider par un juriste avant la
/// publication sur les boutiques d'applications.
class TermsView extends StatelessWidget {
  const TermsView({super.key});

  static const updatedOn = '19 octobre 2026';

  /// Espaces insécables : le nom de l'éditeur et le numéro ne sont jamais coupés.
  static String _nb(String text) => text.replaceAll(' ', '\u00A0');

  static final _sections = <(String, List<String>)>[
    (
      '1. Objet',
      [
        'Les présentes conditions générales d’utilisation (CGU) encadrent l’utilisation de l’application ${AppConfig.appName}, qui aide à préparer l’examen du Code de la route au Sénégal grâce à des cours, des exercices d’entraînement et des examens blancs, en texte, en audio et en images.',
        'En créant un compte, vous acceptez ces conditions.',
      ],
    ),
    (
      '2. Éditeur',
      [
        'L’application est éditée par ${_nb(AppConfig.publisher)}.',
        'Contact : ${AppConfig.contactEmail}, ${_nb(AppConfig.contactPhoneDisplay)}.',
      ],
    ),
    (
      '3. Compte et sécurité',
      [
        'Un compte est nécessaire pour accéder aux contenus. Il est créé avec votre numéro de téléphone, vérifié par un code reçu par SMS.',
        'Votre code secret à 4 chiffres est personnel : ne le communiquez à personne. Après plusieurs erreurs, l’accès est bloqué pendant quelques minutes pour protéger votre compte.',
        'Vous êtes responsable de l’utilisation de votre compte. Si vous pensez qu’une autre personne l’utilise, changez votre code secret et contactez-nous.',
      ],
    ),
    (
      '4. Contenus pédagogiques',
      [
        'Les cours, questions et explications sont préparés et vérifiés avec soin. Ils servent à l’apprentissage et ne remplacent pas les textes officiels du Code de la route ni la formation d’une auto-école.',
        'Les résultats obtenus dans l’application, y compris aux examens blancs, n’ont aucune valeur officielle et ne garantissent pas la réussite à l’examen du permis de conduire.',
      ],
    ),
    (
      '5. Utilisation autorisée',
      [
        'L’application est réservée à un usage personnel et non commercial.',
        'Il est interdit de copier, revendre ou diffuser les contenus (textes, audios, illustrations, questions), de tenter d’accéder aux réponses par des moyens détournés ou de perturber le fonctionnement du service.',
      ],
    ),
    (
      '6. Données personnelles',
      [
        'Nous collectons uniquement ce qui est utile au service : votre numéro de téléphone, votre nom, vos préférences (langue d’écoute, mode d’apprentissage) et votre progression (leçons terminées, réponses, résultats d’examens).',
        'Ces données servent à faire fonctionner votre compte, à suivre votre progression et à améliorer les contenus. Elles ne sont ni vendues ni cédées à des tiers.',
        'Conformément à la loi sénégalaise sur la protection des données à caractère personnel, vous pouvez demander l’accès, la rectification ou la suppression de vos données en écrivant à ${AppConfig.contactEmail}.',
      ],
    ),
    (
      '7. Disponibilité du service',
      [
        'Nous faisons notre possible pour que l’application reste disponible, mais des interruptions peuvent survenir (maintenance, problème de réseau). Les cours téléchargés restent consultables sans connexion ; les examens blancs nécessitent une connexion.',
      ],
    ),
    (
      '8. Propriété intellectuelle',
      [
        'L’application, son nom, son logo et ses contenus appartiennent à ${_nb(AppConfig.publisher)} ou à ses partenaires et sont protégés par le droit de la propriété intellectuelle.',
      ],
    ),
    (
      '9. Modification des conditions',
      [
        'Ces conditions peuvent évoluer. En cas de changement important, vous en serez informé dans l’application.',
      ],
    ),
    (
      '10. Droit applicable',
      [
        'Ces conditions sont soumises au droit sénégalais. En cas de litige, une solution amiable sera recherchée avant toute action devant les tribunaux compétents du Sénégal.',
      ],
    ),
  ];

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: GradientBackground(
        child: SafeArea(
          child: ListView(
            padding: const EdgeInsets.fromLTRB(20, 4, 20, 32),
            children: [
              Align(
                alignment: Alignment.centerLeft,
                child: IconButton(
                  iconSize: 30,
                  tooltip: MaterialLocalizations.of(context).backButtonTooltip,
                  icon: const Icon(
                    Icons.arrow_back_rounded,
                    color: Ink2.strong,
                  ),
                  onPressed: () => Navigator.of(context).pop(),
                ),
              ),
              const Padding(
                padding: EdgeInsets.symmetric(horizontal: 4),
                child: Text(
                  'Conditions générales d’utilisation',
                  style: TextStyle(
                    fontSize: 26,
                    fontWeight: FontWeight.w800,
                    color: Ink2.strong,
                    height: 1.2,
                  ),
                ),
              ),
              const Padding(
                padding: EdgeInsets.fromLTRB(4, 6, 4, 16),
                child: Text(
                  'Mise à jour le $updatedOn',
                  style: TextStyle(fontSize: 15, color: Ink2.muted),
                ),
              ),
              SoftCard(
                padding: const EdgeInsets.fromLTRB(20, 8, 20, 20),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    for (final (title, paragraphs) in _sections) ...[
                      const SizedBox(height: 14),
                      Text(
                        title,
                        style: const TextStyle(
                          fontSize: 18,
                          fontWeight: FontWeight.w800,
                          color: Ink2.strong,
                        ),
                      ),
                      for (final p in paragraphs) ...[
                        const SizedBox(height: 6),
                        Text(
                          p,
                          style: const TextStyle(
                            fontSize: 16,
                            height: 1.45,
                            color: Ink2.strong,
                          ),
                        ),
                      ],
                    ],
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
