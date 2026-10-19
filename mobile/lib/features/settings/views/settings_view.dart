import 'package:flutter/material.dart';
import 'package:package_info_plus/package_info_plus.dart';
import 'package:provider/provider.dart';

import '../../../core/localization/app_strings.dart';
import '../../../core/session_viewmodel.dart';
import '../../../core/theme/app_theme.dart';
import '../../../data/repositories/auth_repository.dart';
import '../../authentication/viewmodels/auth_flow_viewmodel.dart';
import '../../authentication/views/auth_navigation.dart';
import '../../authentication/views/otp_view.dart';
import '../../../shared/widgets/surfaces.dart';
import 'about_view.dart';

const _ink = Ink2.strong;
const _muted = Ink2.muted;
const _logoutTint = Color(0xFFFFEFEA);

/// Profil, langue, mode d'apprentissage (modifiable à tout moment, CDC §3.3), compte.
class SettingsView extends StatelessWidget {
  const SettingsView({super.key});

  @override
  Widget build(BuildContext context) {
    final session = context.watch<SessionViewModel>();
    final s = session.strings;

    return Scaffold(
      body: GradientBackground(
        child: SafeArea(
          child: ListView(
            padding: const EdgeInsets.fromLTRB(20, 4, 20, 24),
            children: [
              Align(
                alignment: Alignment.centerLeft,
                child: IconButton(
                  iconSize: 30,
                  tooltip: MaterialLocalizations.of(context).backButtonTooltip,
                  icon: const Icon(Icons.arrow_back_rounded, color: _ink),
                  onPressed: () => Navigator.of(context).pop(),
                ),
              ),
              const SizedBox(height: 8),
              const _ProfileHeader(),
              const SizedBox(height: 28),
              _MenuCard(
                children: [
                  _MenuRow(
                    icon: Icons.translate_rounded,
                    label: s.t('language'),
                    value: s.t(
                      session.language == 'wo' ? 'lang_wo' : 'lang_fr',
                    ),
                    onTap: () => _chooseLanguage(context),
                  ),
                  _MenuRow(
                    icon: session.learningMode == 'audio'
                        ? Icons.headphones_rounded
                        : Icons.menu_book_rounded,
                    label: s.t('learning_mode'),
                    value: s.t(
                      session.learningMode == 'audio'
                          ? 'mode_audio'
                          : 'mode_text',
                    ),
                    onTap: () => _chooseLearningMode(context),
                  ),
                  _MenuRow(
                    icon: Icons.person_outline_rounded,
                    label: s.t('edit_name'),
                    onTap: () => _editName(context),
                  ),
                  _MenuRow(
                    icon: Icons.shield_outlined,
                    label: s.t('change_pin'),
                    onTap: () => _changePin(context),
                  ),
                  _MenuRow(
                    icon: Icons.info_outline_rounded,
                    label: s.t('about'),
                    onTap: () => Navigator.of(context).push(
                      MaterialPageRoute(builder: (_) => const AboutView()),
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 20),
              _LogoutTile(
                strings: s,
                onConfirmed: () {
                  Navigator.of(context).popUntil((route) => route.isFirst);
                  session.logout();
                },
              ),
              const SizedBox(height: 40),
              Text(
                'Sama Code',
                textAlign: TextAlign.center,
                style: const TextStyle(
                  fontSize: 15,
                  fontWeight: FontWeight.w700,
                  color: _ink,
                ),
              ),
              Text(
                s.t('slogan'),
                textAlign: TextAlign.center,
                style: const TextStyle(fontSize: 13, color: _muted),
              ),
              const SizedBox(height: 4),
              _AppVersion(label: s.t('version')),
            ],
          ),
        ),
      ),
    );
  }

  Future<void> _chooseLanguage(BuildContext context) async {
    final session = context.read<SessionViewModel>();
    final s = session.strings;
    final value = await _showChoices(
      context,
      title: s.t('language'),
      current: session.language,
      options: [
        _Choice('fr', s.t('lang_fr'), const _Initials('FR', AppColors.green)),
        _Choice('wo', s.t('lang_wo'), const _Initials('WO', AppColors.orange)),
      ],
    );
    if (value != null) await session.setLanguage(value);
  }

  Future<void> _chooseLearningMode(BuildContext context) async {
    final session = context.read<SessionViewModel>();
    final s = session.strings;
    final value = await _showChoices(
      context,
      title: s.t('learning_mode'),
      current: session.learningMode,
      options: [
        _Choice(
          'text',
          s.t('mode_text'),
          const Icon(Icons.menu_book_rounded, size: 32, color: AppColors.green),
          subtitle: s.t('mode_text_desc'),
        ),
        _Choice(
          'audio',
          s.t('mode_audio'),
          const Icon(
            Icons.headphones_rounded,
            size: 32,
            color: AppColors.orange,
          ),
          subtitle: s.t('mode_audio_desc'),
        ),
      ],
    );
    if (value != null) await session.setLearningMode(value);
  }

  Future<void> _editName(BuildContext context) async {
    final session = context.read<SessionViewModel>();
    final s = session.strings;
    final name = await showModalBottomSheet<String>(
      context: context,
      isScrollControlled: true,
      showDragHandle: true,
      backgroundColor: Colors.white,
      builder: (_) =>
          _EditNameSheet(initial: session.user?.displayName ?? '', strings: s),
    );
    if (name == null || name.trim().isEmpty) return;
    final error = await session.updateDisplayName(name.trim());
    if (error != null && context.mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(s.error(error.code, error.message))),
      );
    }
  }

  /// Changement de PIN : code SMS envoyé au numéro du compte, puis nouveau PIN ×2.
  Future<void> _changePin(BuildContext context) async {
    final session = context.read<SessionViewModel>();
    final phone = session.user?.phoneNumber;
    if (phone == null) return;
    final vm = AuthFlowViewModel(
      context.read<AuthRepository>(),
      AuthFlow.resetPin,
    );
    final sent = await vm.requestCode(phone);
    if (!context.mounted) return;
    if (sent) {
      await pushAuthStep(context, vm, const OtpView());
    } else if (vm.error != null) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(describeError(session.strings, vm.error!))),
      );
    }
  }
}

// ---------------------------------------------------------------------------
// En-tête : nom en grand, numéro, avatar à droite
// ---------------------------------------------------------------------------

class _ProfileHeader extends StatelessWidget {
  const _ProfileHeader();

  @override
  Widget build(BuildContext context) {
    final user = context.watch<SessionViewModel>().user;
    final name = user?.displayName ?? '';
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 4),
      child: Row(
        children: [
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  name,
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(
                    fontSize: 28,
                    fontWeight: FontWeight.w800,
                    color: _ink,
                    height: 1.15,
                  ),
                ),
                const SizedBox(height: 6),
                Text(
                  _formatPhone(user?.phoneNumber ?? ''),
                  style: const TextStyle(
                    fontSize: 16,
                    color: _muted,
                    letterSpacing: 0.3,
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(width: 16),
          InitialsAvatar(name: name),
        ],
      ),
    );
  }

  /// +221771234567 → +221 77 123 45 67
  static String _formatPhone(String e164) {
    final m = RegExp(r'^\+221(\d{2})(\d{3})(\d{2})(\d{2})$').firstMatch(e164);
    return m == null ? e164 : '+221 ${m[1]} ${m[2]} ${m[3]} ${m[4]}';
  }
}

// ---------------------------------------------------------------------------
// Carte de menu : fond blanc, coins arrondis, ombre très légère, sans bordure
// ---------------------------------------------------------------------------

class _MenuCard extends StatelessWidget {
  const _MenuCard({required this.children});

  final List<Widget> children;

  @override
  Widget build(BuildContext context) {
    return SoftCard(
      padding: const EdgeInsets.symmetric(vertical: 10),
      child: Column(children: children),
    );
  }
}

class _MenuRow extends StatelessWidget {
  const _MenuRow({
    required this.icon,
    required this.label,
    required this.onTap,
    this.value,
  });

  final IconData icon;
  final String label;
  final String? value;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      button: true,
      label: value == null ? label : '$label : $value',
      excludeSemantics: true,
      child: InkWell(
        onTap: onTap,
        child: ConstrainedBox(
          constraints: const BoxConstraints(minHeight: 64),
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 14),
            child: Row(
              children: [
                Icon(icon, size: 26, color: _ink),
                const SizedBox(width: 18),
                Expanded(
                  child: Text(
                    label,
                    style: const TextStyle(
                      fontSize: 18,
                      color: _ink,
                      fontWeight: FontWeight.w500,
                    ),
                  ),
                ),
                if (value != null)
                  Text(
                    value!,
                    style: const TextStyle(
                      fontSize: 16,
                      color: AppColors.green,
                      fontWeight: FontWeight.w700,
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

class _LogoutTile extends StatelessWidget {
  const _LogoutTile({required this.strings, required this.onConfirmed});

  final AppStrings strings;
  final VoidCallback onConfirmed;

  @override
  Widget build(BuildContext context) {
    final s = strings;
    return Material(
      color: _logoutTint,
      borderRadius: BorderRadius.circular(20),
      child: InkWell(
        borderRadius: BorderRadius.circular(20),
        onTap: () async {
          final confirmed = await showModalBottomSheet<bool>(
            context: context,
            showDragHandle: true,
            backgroundColor: Colors.white,
            builder: (context) => Padding(
              padding: const EdgeInsets.fromLTRB(24, 0, 24, 24),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  const Icon(
                    Icons.logout_rounded,
                    size: 44,
                    color: AppColors.red,
                  ),
                  const SizedBox(height: 12),
                  Text(
                    s.t('logout_confirm'),
                    textAlign: TextAlign.center,
                    style: const TextStyle(
                      fontSize: 22,
                      fontWeight: FontWeight.w800,
                      color: _ink,
                    ),
                  ),
                  const SizedBox(height: 24),
                  Row(
                    children: [
                      Expanded(
                        child: FilledButton(
                          style: FilledButton.styleFrom(
                            backgroundColor: AppColors.background,
                            foregroundColor: _ink,
                            padding: const EdgeInsets.symmetric(horizontal: 12),
                            textStyle: const TextStyle(
                              fontSize: 18,
                              fontWeight: FontWeight.w700,
                            ),
                          ),
                          onPressed: () => Navigator.of(context).pop(false),
                          child: _OneLine(s.t('cancel')),
                        ),
                      ),
                      const SizedBox(width: 12),
                      Expanded(
                        child: FilledButton(
                          style: FilledButton.styleFrom(
                            backgroundColor: AppColors.red,
                            padding: const EdgeInsets.symmetric(horizontal: 12),
                            textStyle: const TextStyle(
                              fontSize: 18,
                              fontWeight: FontWeight.w700,
                            ),
                          ),
                          onPressed: () => Navigator.of(context).pop(true),
                          child: _OneLine(s.t('logout')),
                        ),
                      ),
                    ],
                  ),
                ],
              ),
            ),
          );
          if (confirmed == true) onConfirmed();
        },
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 20),
          child: Row(
            children: [
              const Icon(Icons.logout_rounded, size: 26, color: AppColors.red),
              const SizedBox(width: 18),
              Text(
                s.t('logout'),
                style: const TextStyle(
                  fontSize: 18,
                  color: AppColors.red,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Panneau de choix (langue, mode)
// ---------------------------------------------------------------------------

class _Choice {
  const _Choice(this.value, this.label, this.leading, {this.subtitle});

  final String value;
  final String label;
  final Widget leading;
  final String? subtitle;
}

Future<String?> _showChoices(
  BuildContext context, {
  required String title,
  required String current,
  required List<_Choice> options,
}) {
  return showModalBottomSheet<String>(
    context: context,
    showDragHandle: true,
    backgroundColor: Colors.white,
    builder: (context) => SafeArea(
      child: Padding(
        padding: const EdgeInsets.fromLTRB(12, 0, 12, 16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(12, 0, 12, 12),
              child: Text(
                title,
                style: const TextStyle(
                  fontSize: 22,
                  fontWeight: FontWeight.w800,
                  color: _ink,
                ),
              ),
            ),
            for (final option in options)
              Semantics(
                button: true,
                selected: option.value == current,
                label: option.label,
                excludeSemantics: true,
                child: InkWell(
                  borderRadius: BorderRadius.circular(16),
                  onTap: () => Navigator.of(context).pop(option.value),
                  child: Container(
                    margin: const EdgeInsets.symmetric(vertical: 4),
                    padding: const EdgeInsets.symmetric(
                      horizontal: 12,
                      vertical: 14,
                    ),
                    decoration: BoxDecoration(
                      color: option.value == current
                          ? AppColors.green.withValues(alpha: 0.08)
                          : null,
                      borderRadius: BorderRadius.circular(16),
                    ),
                    child: Row(
                      children: [
                        SizedBox(
                          width: 48,
                          child: Center(child: option.leading),
                        ),
                        const SizedBox(width: 14),
                        Expanded(
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Text(
                                option.label,
                                style: const TextStyle(
                                  fontSize: 19,
                                  fontWeight: FontWeight.w700,
                                  color: _ink,
                                ),
                              ),
                              if (option.subtitle != null)
                                Text(
                                  option.subtitle!,
                                  style: const TextStyle(
                                    fontSize: 15,
                                    color: _muted,
                                  ),
                                ),
                            ],
                          ),
                        ),
                        if (option.value == current)
                          const Icon(
                            Icons.check_circle_rounded,
                            size: 30,
                            color: AppColors.green,
                          ),
                      ],
                    ),
                  ),
                ),
              ),
          ],
        ),
      ),
    ),
  );
}

class _Initials extends StatelessWidget {
  const _Initials(this.text, this.color);

  final String text;
  final Color color;

  @override
  Widget build(BuildContext context) => CircleAvatar(
    radius: 22,
    backgroundColor: color,
    child: Text(
      text,
      style: const TextStyle(
        color: Colors.white,
        fontSize: 15,
        fontWeight: FontWeight.w800,
      ),
    ),
  );
}

/// « Version 1.0.0 (1) », lu depuis le pubspec via package_info_plus.
class _AppVersion extends StatelessWidget {
  const _AppVersion({required this.label});

  final String label;

  static final _info = PackageInfo.fromPlatform();

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<PackageInfo>(
      future: _info,
      builder: (context, snapshot) {
        final info = snapshot.data;
        return Text(
          info == null ? '' : '$label ${info.version} (${info.buildNumber})',
          textAlign: TextAlign.center,
          style: const TextStyle(fontSize: 13, color: _muted),
        );
      },
    );
  }
}

/// Libellé de bouton sur une seule ligne, réduit si la place manque.
class _OneLine extends StatelessWidget {
  const _OneLine(this.text);

  final String text;

  @override
  Widget build(BuildContext context) => FittedBox(
    fit: BoxFit.scaleDown,
    child: Text(text, maxLines: 1, softWrap: false),
  );
}

/// Panneau « Modifier mon nom ». Il possède son contrôleur et ne le libère qu'à sa
/// destruction, après l'animation de fermeture.
class _EditNameSheet extends StatefulWidget {
  const _EditNameSheet({required this.initial, required this.strings});

  final String initial;
  final AppStrings strings;

  @override
  State<_EditNameSheet> createState() => _EditNameSheetState();
}

class _EditNameSheetState extends State<_EditNameSheet> {
  late final _controller = TextEditingController(text: widget.initial);

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  void _submit() => Navigator.of(context).pop(_controller.text);

  @override
  Widget build(BuildContext context) {
    final s = widget.strings;
    return Padding(
      padding: EdgeInsets.fromLTRB(
        24,
        0,
        24,
        24 + MediaQuery.viewInsetsOf(context).bottom,
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(
            s.t('edit_name'),
            style: const TextStyle(fontSize: 22, fontWeight: FontWeight.w800),
          ),
          const SizedBox(height: 16),
          TextField(
            controller: _controller,
            autofocus: true,
            maxLength: 80,
            textCapitalization: TextCapitalization.words,
            style: const TextStyle(fontSize: 22),
            decoration: InputDecoration(
              hintText: s.t('name_hint'),
              counterText: '',
              fillColor: AppColors.background,
              enabledBorder: OutlineInputBorder(
                borderRadius: BorderRadius.circular(16),
                borderSide: BorderSide.none,
              ),
            ),
            onSubmitted: (_) => _submit(),
          ),
          const SizedBox(height: 16),
          FilledButton(onPressed: _submit, child: Text(s.t('save'))),
        ],
      ),
    );
  }
}
