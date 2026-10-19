import 'package:flutter/material.dart';

import '../../../core/config.dart';
import '../../../core/theme/app_theme.dart';
import '../../../shared/widgets/surfaces.dart';

/// Conditions générales d'utilisation : introduction, résumé en clair, 12 articles
/// et phrase d'acceptation.
///
/// ⚠️ Projet rédigé pour le MVP : à faire relire et valider par un juriste avant la
/// publication sur les boutiques d'applications (question Q13).
class TermsView extends StatelessWidget {
  const TermsView({super.key});

  static const updatedOn = '19 octobre 2026';

  /// Espaces insécables : le nom de l'éditeur et le numéro ne sont jamais coupés.
  static String _nb(String text) => text.replaceAll(' ', ' ');

  static String get _app => AppConfig.appName;
  static String get _publisher => _nb(AppConfig.publisher);
  static String get _email => AppConfig.contactEmail;
  static String get _phone => _nb(AppConfig.contactPhoneDisplay);

  static String get _intro =>
      'Merci de lire ces conditions avant d’utiliser $_app. Elles expliquent vos droits et vos obligations, ainsi que les nôtres. En créant un compte ou en utilisant l’application, vous les acceptez. Si vous ne les acceptez pas, n’utilisez pas l’application.';

  static List<(IconData, String)> get _summary => [
    (
      Icons.school_rounded,
      '$_app vous aide à réviser. Elle ne remplace ni l’auto-école ni l’examen officiel.',
    ),
    (
      Icons.lock_rounded,
      'Votre compte est personnel. Gardez votre code secret pour vous.',
    ),
    (
      Icons.verified_user_rounded,
      'Vos données servent seulement à faire fonctionner l’application. Elles ne sont jamais vendues.',
    ),
    (
      Icons.mail_rounded,
      'Une question ou une erreur dans un contenu ? Écrivez-nous à $_email.',
    ),
  ];

  static List<(String, List<String>)> get _sections => [
    (
      '1. Objet de l’application',
      [
        '$_app aide à préparer l’examen du Code de la route au Sénégal. Elle propose des cours, des exercices d’entraînement avec correction et des examens blancs.',
        'Les textes sont en français. Les cours et les questions peuvent être écoutés en français ou en wolof, et sont accompagnés d’illustrations, pour que chacun puisse apprendre, qu’il sache lire ou non.',
        '$_app est un complément d’entraînement. Elle ne remplace pas la formation dispensée par une auto-école.',
      ],
    ),
    (
      '2. Éditeur et contact',
      [
        'L’application est éditée par $_publisher.',
        'Pour toute question, réclamation ou demande concernant vos données : $_email ou $_phone (appel et WhatsApp).',
      ],
    ),
    (
      '3. Accès et compte',
      [
        'Un compte est nécessaire pour accéder aux contenus. Il est créé avec votre numéro de téléphone, vérifié par un code reçu par SMS.',
        'Vous vous engagez à fournir des informations exactes et à utiliser votre propre numéro. Une personne peut vous aider à vous inscrire, mais le compte reste le vôtre.',
        'Votre compte est personnel : ne le partagez pas. Votre code secret à 4 chiffres ne doit être communiqué à personne. Après plusieurs codes erronés, l’accès est bloqué quelques minutes pour protéger votre compte.',
        'Si vous pensez qu’une autre personne utilise votre compte, changez votre code secret et contactez-nous.',
        'L’application utilise votre connexion internet. Les données mobiles consommées sont décomptées de votre forfait, selon les conditions de votre opérateur.',
      ],
    ),
    (
      '4. Contenus pédagogiques',
      [
        'Les cours, questions, audios et explications sont préparés puis vérifiés par notre équipe avant d’être publiés, en suivant les règles du Code de la route en vigueur au Sénégal.',
        'Malgré ce soin, une erreur ou un oubli reste possible, notamment après un changement de réglementation. Si vous en trouvez un, signalez-le-nous : il sera corrigé.',
        'Les résultats obtenus dans l’application, y compris aux examens blancs, n’ont aucune valeur officielle et ne garantissent pas la réussite à l’examen du permis de conduire.',
      ],
    ),
    (
      '5. Règles d’utilisation',
      [
        'L’application est réservée à un usage personnel, pour se préparer à l’examen du Code de la route.',
        'Il est interdit de copier, revendre ou diffuser les contenus (textes, audios, illustrations, questions), de chercher à obtenir les réponses par des moyens détournés, ou de perturber le fonctionnement du service.',
      ],
    ),
    (
      '6. Données personnelles',
      [
        'Nous collectons uniquement ce qui est utile au service : votre numéro de téléphone, votre nom, vos préférences (langue d’écoute, mode d’apprentissage) et votre progression (leçons terminées, réponses, résultats d’examens).',
        'Ces données servent à faire fonctionner votre compte, à suivre votre progression et à améliorer les contenus. Elles ne sont ni vendues ni cédées à des tiers, et sont protégées par des mesures de sécurité adaptées.',
        'Elles sont conservées tant que votre compte existe, puis supprimées à votre demande.',
        'Conformément à la loi sénégalaise n° 2008-12 du 25 janvier 2008 sur la protection des données à caractère personnel, vous pouvez demander l’accès, la rectification ou la suppression de vos données en écrivant à $_email. Vous pouvez aussi saisir la Commission de protection des données personnelles (CDP).',
      ],
    ),
    (
      '7. Disponibilité du service',
      [
        'Nous faisons notre possible pour que l’application reste disponible, mais des interruptions peuvent survenir (maintenance, panne, problème de réseau).',
        'Les cours téléchargés restent consultables sans connexion. Les examens blancs nécessitent une connexion ; une coupure courte n’efface pas vos réponses.',
      ],
    ),
    (
      '8. Responsabilité',
      [
        '$_publisher s’engage à fournir un service sérieux et des contenus vérifiés.',
        'Dans les limites prévues par la loi, $_publisher ne peut être tenu responsable du résultat obtenu à l’examen officiel, d’une interruption temporaire du service, ni d’une utilisation de l’application contraire à ces conditions.',
      ],
    ),
    (
      '9. Propriété intellectuelle',
      [
        'L’application, son nom, son logo et ses contenus appartiennent à $_publisher ou à ses partenaires et sont protégés par le droit de la propriété intellectuelle.',
      ],
    ),
    (
      '10. Durée et suppression du compte',
      [
        'Vous pouvez cesser d’utiliser l’application à tout moment et demander la suppression de votre compte en nous contactant.',
        'En cas de non-respect de ces conditions, un compte peut être suspendu, après vous en avoir informé lorsque c’est possible.',
      ],
    ),
    (
      '11. Modification des conditions',
      [
        'Ces conditions peuvent évoluer, par exemple pour suivre la réglementation. En cas de changement important, vous en serez informé dans l’application. La date de mise à jour figure en haut de cette page.',
      ],
    ),
    (
      '12. Droit applicable',
      [
        'Ces conditions sont régies par le droit sénégalais. En cas de litige, une solution amiable sera d’abord recherchée ; à défaut, les tribunaux compétents du Sénégal seront saisis.',
      ],
    ),
  ];

  static const _body = TextStyle(
    fontSize: 16,
    height: 1.45,
    color: Ink2.strong,
  );

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
                padding: EdgeInsets.fromLTRB(4, 6, 4, 14),
                child: Text(
                  'Mise à jour le $updatedOn',
                  style: TextStyle(fontSize: 15, color: Ink2.muted),
                ),
              ),
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 4),
                child: Text(_intro, style: _body),
              ),
              const SizedBox(height: 16),
              _Summary(items: _summary),
              const SizedBox(height: 16),
              SoftCard(
                padding: const EdgeInsets.fromLTRB(20, 8, 20, 20),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    for (final (title, paragraphs) in _sections) ...[
                      const SizedBox(height: 16),
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
                        Text(p, style: _body),
                      ],
                    ],
                  ],
                ),
              ),
              const SizedBox(height: 20),
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 4),
                child: Text(
                  'En utilisant $_app, vous acceptez ces conditions générales d’utilisation.',
                  textAlign: TextAlign.center,
                  style: const TextStyle(
                    fontSize: 16,
                    fontWeight: FontWeight.w700,
                    color: Ink2.strong,
                    height: 1.4,
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// « En résumé » : l'essentiel en quatre phrases, sur fond vert pâle.
class _Summary extends StatelessWidget {
  const _Summary({required this.items});

  final List<(IconData, String)> items;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.fromLTRB(18, 16, 18, 8),
      decoration: BoxDecoration(
        color: AppColors.green.withValues(alpha: 0.09),
        borderRadius: BorderRadius.circular(22),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text(
            'En résumé',
            style: TextStyle(
              fontSize: 18,
              fontWeight: FontWeight.w800,
              color: AppColors.greenDark,
            ),
          ),
          const SizedBox(height: 10),
          for (final (icon, text) in items)
            Padding(
              padding: const EdgeInsets.only(bottom: 12),
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Icon(icon, size: 24, color: AppColors.green),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Text(
                      text,
                      style: const TextStyle(
                        fontSize: 16,
                        height: 1.4,
                        color: Ink2.strong,
                      ),
                    ),
                  ),
                ],
              ),
            ),
        ],
      ),
    );
  }
}
