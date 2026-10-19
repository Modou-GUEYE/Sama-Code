import 'package:flutter/material.dart';
import 'package:package_info_plus/package_info_plus.dart';
import 'package:provider/provider.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../core/config.dart';
import '../../../core/localization/app_strings.dart';
import '../../../core/session_viewmodel.dart';
import '../../../core/theme/app_theme.dart';
import '../../../shared/widgets/animated_road.dart';
import '../../../shared/widgets/feedback.dart';
import '../../../shared/widgets/surfaces.dart';

/// À propos, dans le style de l'écran de bienvenue : logo, trois icônes (lire,
/// écouter, voir), route animée, contacts en gros boutons, puis éditeur et version.
/// Tient dans l'écran ; défile seulement sur un petit écran ou en grande police.
class AboutView extends StatelessWidget {
  const AboutView({super.key});

  @override
  Widget build(BuildContext context) {
    final s = context.watch<SessionViewModel>().strings;
    final phone = AppConfig.contactPhone;

    return Scaffold(
      body: GradientBackground(
        child: Stack(
          children: [
            SafeArea(
              child: LayoutBuilder(
                builder: (context, box) => SingleChildScrollView(
                  child: ConstrainedBox(
                    constraints: BoxConstraints(minHeight: box.maxHeight),
                    child: IntrinsicHeight(
                      child: Padding(
                        padding: const EdgeInsets.fromLTRB(20, 4, 20, 12),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.stretch,
                          children: [
                            Align(
                              alignment: Alignment.centerLeft,
                              child: IconButton(
                                iconSize: 30,
                                tooltip: MaterialLocalizations.of(context)
                                    .backButtonTooltip,
                                icon: const Icon(
                                  Icons.arrow_back_rounded,
                                  color: Ink2.strong,
                                ),
                                onPressed: () => Navigator.of(context).pop(),
                              ),
                            ),
                            const Spacer(),
                            Semantics(
                              label: s.t('about_mission'),
                              excludeSemantics: true,
                              child: const Center(child: AppLogo(size: 200)),
                            ),
                            const SizedBox(height: 28),
                            Row(
                              children: [
                                _Feature(
                                  icon: Icons.menu_book_rounded,
                                  color: AppColors.green,
                                  label: s.t('about_read_short'),
                                  semanticLabel: s.t('about_read'),
                                ),
                                _Feature(
                                  icon: Icons.headphones_rounded,
                                  color: AppColors.orange,
                                  label: s.t('about_listen_short'),
                                  semanticLabel: s.t('about_listen'),
                                ),
                                _Feature(
                                  icon: Icons.image_rounded,
                                  color: AppColors.greenDark,
                                  label: s.t('about_images_short'),
                                  semanticLabel: s.t('about_images'),
                                ),
                              ],
                            ),
                            const Spacer(),
                            // La route passe entre les icônes et les contacts.
                            const SizedBox(height: 170, child: AnimatedRoad()),
                            const SizedBox(height: 24),
                            Row(
                              children: [
                                _ContactButton(
                                  icon: Icons.mail_rounded,
                                  label: s.t('email'),
                                  semanticLabel:
                                      '${s.t('email')} : ${AppConfig.contactEmail}',
                                  onTap: () => _open(
                                    context,
                                    Uri(
                                      scheme: 'mailto',
                                      path: AppConfig.contactEmail,
                                    ),
                                  ),
                                ),
                                if (phone.isNotEmpty) ...[
                                  const SizedBox(width: 10),
                                  _ContactButton(
                                    icon: Icons.call_rounded,
                                    label: s.t('call'),
                                    semanticLabel:
                                        '${s.t('call')} : ${_formatPhone(phone)}',
                                    onTap: () => _open(
                                      context,
                                      Uri(scheme: 'tel', path: phone),
                                    ),
                                  ),
                                  const SizedBox(width: 10),
                                  _ContactButton(
                                    icon: Icons.chat_rounded,
                                    label: s.t('whatsapp'),
                                    semanticLabel:
                                        '${s.t('whatsapp')} : ${_formatPhone(phone)}',
                                    onTap: () => _open(
                                      context,
                                      Uri.https(
                                        'wa.me',
                                        phone.replaceAll('+', ''),
                                      ),
                                    ),
                                  ),
                                ],
                              ],
                            ),
                            const SizedBox(height: 14),
                            _Footer(strings: s),
                          ],
                        ),
                      ),
                    ),
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  static Future<void> _open(BuildContext context, Uri uri) async {
    final messenger = ScaffoldMessenger.of(context);
    final message = context.read<SessionViewModel>().strings.t('cannot_open');
    final opened = await launchUrl(
      uri,
      mode: LaunchMode.externalApplication,
    ).catchError((_) => false);
    if (!opened) messenger.showSnackBar(SnackBar(content: Text(message)));
  }

  /// +221771234567 → 77 123 45 67
  static String _formatPhone(String e164) {
    final m = RegExp(r'^\+221(\d{2})(\d{3})(\d{2})(\d{2})$').firstMatch(e164);
    return m == null ? e164 : '${m[1]} ${m[2]} ${m[3]} ${m[4]}';
  }
}

/// Ce que propose l'application : icône ronde et un mot (lire, écouter, voir).
class _Feature extends StatelessWidget {
  const _Feature({
    required this.icon,
    required this.color,
    required this.label,
    required this.semanticLabel,
  });

  final IconData icon;
  final Color color;
  final String label;
  final String semanticLabel;

  @override
  Widget build(BuildContext context) {
    return Expanded(
      child: Semantics(
        label: semanticLabel,
        excludeSemantics: true,
        child: Column(
          children: [
            Container(
              width: 64,
              height: 64,
              decoration: BoxDecoration(
                color: Colors.white,
                shape: BoxShape.circle,
                boxShadow: [
                  BoxShadow(
                    color: color.withValues(alpha: 0.18),
                    blurRadius: 16,
                    offset: const Offset(0, 6),
                  ),
                ],
              ),
              child: Icon(icon, size: 32, color: color),
            ),
            const SizedBox(height: 8),
            Text(
              label,
              style: const TextStyle(
                fontSize: 16,
                fontWeight: FontWeight.w700,
                color: Ink2.strong,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// Bouton de contact : pastille blanche, icône verte au-dessus d'un mot court.
class _ContactButton extends StatelessWidget {
  const _ContactButton({
    required this.icon,
    required this.label,
    required this.semanticLabel,
    required this.onTap,
  });

  final IconData icon;
  final String label;
  final String semanticLabel;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Expanded(
      child: Semantics(
        button: true,
        label: semanticLabel,
        excludeSemantics: true,
        child: Material(
          color: Colors.white,
          borderRadius: BorderRadius.circular(22),
          child: InkWell(
            onTap: onTap,
            borderRadius: BorderRadius.circular(22),
            child: Padding(
              padding: const EdgeInsets.symmetric(vertical: 14),
              child: Column(
                children: [
                  Icon(icon, size: 30, color: AppColors.green),
                  const SizedBox(height: 6),
                  Text(
                    label,
                    maxLines: 1,
                    style: const TextStyle(
                      fontSize: 15,
                      fontWeight: FontWeight.w700,
                      color: Ink2.strong,
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

/// Bas de page discret : version, licences des logiciels et éditeur.
class _Footer extends StatelessWidget {
  const _Footer({required this.strings});

  final AppStrings strings;

  @override
  Widget build(BuildContext context) {
    final s = strings;
    const muted = TextStyle(fontSize: 13, color: Ink2.muted);
    return Column(
      children: [
        FutureBuilder<PackageInfo>(
          future: PackageInfo.fromPlatform(),
          builder: (context, snap) => Text(
            '${s.t('published_by')} ${AppConfig.publisher}. ${s.t('version')} ${snap.data?.version ?? ''}',
            textAlign: TextAlign.center,
            style: muted,
          ),
        ),
        TextButton(
          onPressed: () => showLicensePage(
            context: context,
            applicationName: AppConfig.publisher,
            applicationIcon: const Padding(
              padding: EdgeInsets.all(8),
              child: AppLogo(size: 96),
            ),
          ),
          child: Text(
            s.t('licenses'),
            style: const TextStyle(
              fontSize: 14,
              fontWeight: FontWeight.w700,
              color: AppColors.green,
            ),
          ),
        ),
      ],
    );
  }
}
