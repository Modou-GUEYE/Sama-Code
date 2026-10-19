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
              const Center(child: AppLogo(size: 200)),
              const SizedBox(height: 20),
              SoftCard(
                padding: const EdgeInsets.all(20),
                child: Text(
                  s.t('about_mission'),
                  style: const TextStyle(
                    fontSize: 17,
                    height: 1.45,
                    color: Ink2.strong,
                  ),
                ),
              ),
              const SizedBox(height: 16),
              SoftCard(
                padding: const EdgeInsets.symmetric(vertical: 8),
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
              const SizedBox(height: 24),
              Padding(
                padding: const EdgeInsets.only(left: 4, bottom: 10),
                child: Text(
                  s.t('contact_us'),
                  style: const TextStyle(
                    fontSize: 20,
                    fontWeight: FontWeight.w800,
                    color: Ink2.strong,
                  ),
                ),
              ),
              SoftCard(
                padding: const EdgeInsets.symmetric(vertical: 8),
                child: Column(
                  children: [
                    _LinkRow(
                      icon: Icons.mail_outline_rounded,
                      label: AppConfig.contactEmail,
                      semanticLabel:
                          '${s.t('email')} : ${AppConfig.contactEmail}',
                      onTap: () => _open(
                        context,
                        Uri(scheme: 'mailto', path: AppConfig.contactEmail),
                      ),
                    ),
                    if (AppConfig.contactPhone.isNotEmpty) ...[
                      _LinkRow(
                        icon: Icons.call_rounded,
                        label: s.t('call'),
                        value: _formatPhone(AppConfig.contactPhone),
                        onTap: () => _open(
                          context,
                          Uri(scheme: 'tel', path: AppConfig.contactPhone),
                        ),
                      ),
                      _LinkRow(
                        icon: Icons.chat_rounded,
                        label: s.t('whatsapp'),
                        value: _formatPhone(AppConfig.contactPhone),
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
              ),
              const SizedBox(height: 16),
              SoftCard(
                padding: const EdgeInsets.symmetric(vertical: 8),
                child: Column(
                  children: [
                    _VersionRow(strings: s),
                    _LinkRow(
                      icon: Icons.description_outlined,
                      label: s.t('licenses'),
                      onTap: () => showLicensePage(
                        context: context,
                        applicationName: 'Sama Code',
                        applicationIcon: const Padding(
                          padding: EdgeInsets.all(8),
                          child: AppLogo(size: 96),
                        ),
                      ),
                    ),
                  ],
                ),
              ),
              const SizedBox(height: 28),
              Text(
                s.t('published_by'),
                textAlign: TextAlign.center,
                style: const TextStyle(fontSize: 14, color: Ink2.muted),
              ),
              const Text(
                AppConfig.publisher,
                textAlign: TextAlign.center,
                style: TextStyle(
                  fontSize: 16,
                  fontWeight: FontWeight.w800,
                  color: Ink2.strong,
                ),
              ),
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
      padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 10),
      child: Row(
        children: [
          Container(
            width: 48,
            height: 48,
            decoration: BoxDecoration(
              color: color.withValues(alpha: 0.12),
              borderRadius: BorderRadius.circular(14),
            ),
            child: Icon(icon, size: 26, color: color),
          ),
          const SizedBox(width: 16),
          Expanded(
            child: Text(
              text,
              style: const TextStyle(fontSize: 16, color: Ink2.strong),
            ),
          ),
        ],
      ),
    );
  }
}

class _VersionRow extends StatelessWidget {
  const _VersionRow({required this.strings});

  final AppStrings strings;

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<PackageInfo>(
      future: PackageInfo.fromPlatform(),
      builder: (context, snap) {
        final version = snap.data?.version ?? '';
        return _Row(
          icon: Icons.info_outline_rounded,
          label: strings.t('version'),
          trailing: Text(
            version,
            style: const TextStyle(
              fontSize: 16,
              color: AppColors.green,
              fontWeight: FontWeight.w700,
            ),
          ),
        );
      },
    );
  }
}

class _LinkRow extends StatelessWidget {
  const _LinkRow({
    required this.icon,
    required this.label,
    required this.onTap,
    this.value,
    this.semanticLabel,
  });

  final IconData icon;
  final String label;
  final String? value;
  final String? semanticLabel;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      button: true,
      label: semanticLabel ?? (value == null ? label : '$label : $value'),
      excludeSemantics: true,
      child: InkWell(
        onTap: onTap,
        child: _Row(
          icon: icon,
          label: label,
          trailing: value == null
              ? const Icon(
                  Icons.chevron_right_rounded,
                  size: 28,
                  color: Ink2.muted,
                )
              : Text(
                  value!,
                  style: const TextStyle(
                    fontSize: 16,
                    color: AppColors.green,
                    fontWeight: FontWeight.w700,
                  ),
                ),
        ),
      ),
    );
  }
}

class _Row extends StatelessWidget {
  const _Row({required this.icon, required this.label, required this.trailing});

  final IconData icon;
  final String label;
  final Widget trailing;

  @override
  Widget build(BuildContext context) {
    return ConstrainedBox(
      constraints: const BoxConstraints(minHeight: 60),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 12),
        child: Row(
          children: [
            Icon(icon, size: 26, color: Ink2.strong),
            const SizedBox(width: 18),
            Expanded(
              child: Text(
                label,
                style: const TextStyle(
                  fontSize: 18,
                  color: Ink2.strong,
                  fontWeight: FontWeight.w500,
                ),
              ),
            ),
            trailing,
          ],
        ),
      ),
    );
  }
}
