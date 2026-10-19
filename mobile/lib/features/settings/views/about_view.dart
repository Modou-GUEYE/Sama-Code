import 'package:flutter/material.dart';
import 'package:package_info_plus/package_info_plus.dart';
import 'package:provider/provider.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../core/config.dart';
import '../../../core/localization/app_strings.dart';
import '../../../core/session_viewmodel.dart';
import '../../../core/theme/app_theme.dart';
import '../../../shared/widgets/feedback.dart';
import '../../../shared/widgets/surfaces.dart';

/// À propos : à quoi sert l'application, comment elle se lit et s'écoute, contacts,
/// version, licences des logiciels utilisés et éditeur.
class AboutView extends StatelessWidget {
  const AboutView({super.key});

  @override
  Widget build(BuildContext context) {
    final s = context.watch<SessionViewModel>().strings;

    return Scaffold(
      body: GradientBackground(
        child: SafeArea(
          child: ListView(
            padding: const EdgeInsets.fromLTRB(20, 4, 20, 16),
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
              const Center(child: AppLogo(size: 120)),
              const SizedBox(height: 10),
              Text(
                s.t('about_mission'),
                textAlign: TextAlign.center,
                style: const TextStyle(
                  fontSize: 16,
                  height: 1.4,
                  color: Ink2.strong,
                ),
              ),
              const SizedBox(height: 12),
              SoftCard(
                padding: const EdgeInsets.symmetric(vertical: 6),
                child: Column(
                  children: [
                    _Point(
                      icon: Icons.menu_book_rounded,
                      color: AppColors.green,
                      text: s.t('about_read'),
                    ),
                    _Point(
                      icon: Icons.headphones_rounded,
                      color: AppColors.orange,
                      text: s.t('about_listen'),
                    ),
                    _Point(
                      icon: Icons.image_rounded,
                      color: AppColors.greenDark,
                      text: s.t('about_images'),
                    ),
                  ],
                ),
              ),
              const SizedBox(height: 12),
              SoftCard(
                padding: const EdgeInsets.fromLTRB(12, 12, 12, 12),
                child: Column(
                  children: [
                    Text(
                      s.t('contact_us'),
                      style: const TextStyle(
                        fontSize: 18,
                        fontWeight: FontWeight.w800,
                        color: Ink2.strong,
                      ),
                    ),
                    const SizedBox(height: 10),
                    Row(
                      children: [
                        _ContactButton(
                          icon: Icons.mail_outline_rounded,
                          label: s.t('email'),
                          semanticLabel:
                              '${s.t('email')} : ${AppConfig.contactEmail}',
                          color: AppColors.greenDark,
                          onTap: () => _open(
                            context,
                            Uri(scheme: 'mailto', path: AppConfig.contactEmail),
                          ),
                        ),
                        if (AppConfig.contactPhone.isNotEmpty) ...[
                          _ContactButton(
                            icon: Icons.call_rounded,
                            label: s.t('call'),
                            semanticLabel:
                                '${s.t('call')} : ${_formatPhone(AppConfig.contactPhone)}',
                            color: AppColors.green,
                            onTap: () => _open(
                              context,
                              Uri(scheme: 'tel', path: AppConfig.contactPhone),
                            ),
                          ),
                          _ContactButton(
                            icon: Icons.chat_rounded,
                            label: s.t('whatsapp'),
                            semanticLabel:
                                '${s.t('whatsapp')} : ${_formatPhone(AppConfig.contactPhone)}',
                            color: const Color(0xFF25A244),
                            onTap: () => _open(
                              context,
                              Uri.https(
                                'wa.me',
                                AppConfig.contactPhone.replaceAll('+', ''),
                              ),
                            ),
                          ),
                        ],
                      ],
                    ),
                    const SizedBox(height: 8),
                    Text(
                      [
                        AppConfig.contactEmail,
                        if (AppConfig.contactPhone.isNotEmpty)
                          _formatPhone(AppConfig.contactPhone),
                      ].join(', '),
                      textAlign: TextAlign.center,
                      style: const TextStyle(
                        fontSize: 15,
                        height: 1.4,
                        color: Ink2.muted,
                      ),
                    ),
                  ],
                ),
              ),
              const SizedBox(height: 12),
              _Footer(strings: s),
            ],
          ),
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

/// Une ligne « ce que propose l'application » : icône dans une pastille + texte.
class _Point extends StatelessWidget {
  const _Point({required this.icon, required this.color, required this.text});

  final IconData icon;
  final Color color;
  final String text;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 5),
      child: Row(
        children: [
          Container(
            width: 42,
            height: 42,
            decoration: BoxDecoration(
              color: color.withValues(alpha: 0.12),
              borderRadius: BorderRadius.circular(14),
            ),
            child: Icon(icon, size: 24, color: color),
          ),
          const SizedBox(width: 14),
          Expanded(
            child: Text(
              text,
              style: const TextStyle(fontSize: 15, color: Ink2.strong),
            ),
          ),
        ],
      ),
    );
  }
}

/// Gros bouton rond de contact, icône + libellé court dessous.
class _ContactButton extends StatelessWidget {
  const _ContactButton({
    required this.icon,
    required this.label,
    required this.semanticLabel,
    required this.color,
    required this.onTap,
  });

  final IconData icon;
  final String label;
  final String semanticLabel;
  final Color color;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Expanded(
      child: Semantics(
        button: true,
        label: semanticLabel,
        excludeSemantics: true,
        child: InkWell(
          onTap: onTap,
          borderRadius: BorderRadius.circular(16),
          child: Padding(
            padding: const EdgeInsets.symmetric(vertical: 4),
            child: Column(
              children: [
                Container(
                  width: 60,
                  height: 60,
                  decoration: BoxDecoration(
                    color: color.withValues(alpha: 0.12),
                    shape: BoxShape.circle,
                  ),
                  child: Icon(icon, size: 30, color: color),
                ),
                const SizedBox(height: 6),
                Text(
                  label,
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
