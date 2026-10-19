import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../../core/session_viewmodel.dart';
import '../../../core/theme/app_theme.dart';
import '../../../shared/widgets/auth_scaffold.dart';
import '../../../shared/widgets/big_choice_card.dart';
import '../../../shared/widgets/feedback.dart';

/// Écran 1 : langue d'écoute (fr / wo). Les textes restent en français (D25).
class LanguageView extends StatelessWidget {
  const LanguageView({super.key});

  @override
  Widget build(BuildContext context) {
    final session = context.watch<SessionViewModel>();
    final s = session.strings;
    return AuthScaffold(
      leading: const AppLogo(size: 120),
      title: s.t('choose_language'),
      subtitle: s.t('choose_language_desc'),
      children: [
        BigChoiceCard(
          leading: const InitialsBadge('FR'),
          title: s.t('lang_fr'),
          selected: session.language == 'fr',
          onTap: () => session.setLanguage('fr'),
        ),
        const SizedBox(height: 14),
        BigChoiceCard(
          leading: const InitialsBadge('WO', color: AppColors.orange),
          title: s.t('lang_wo'),
          selected: session.language == 'wo',
          onTap: () => session.setLanguage('wo'),
        ),
        const SizedBox(height: 32),
        PrimaryButton(
          label: s.t('continue'),
          icon: Icons.arrow_forward_rounded,
          onPressed: () => Navigator.of(
            context,
          ).push(MaterialPageRoute(builder: (_) => const LearningModeView())),
        ),
      ],
    );
  }
}

/// Écran 2 : lire ou écouter.
class LearningModeView extends StatelessWidget {
  const LearningModeView({super.key});

  @override
  Widget build(BuildContext context) {
    final session = context.watch<SessionViewModel>();
    final s = session.strings;
    return AuthScaffold(
      icon: Icons.school_rounded,
      title: s.t('choose_mode'),
      children: [
        BigChoiceCard(
          leading: const Icon(
            Icons.menu_book_rounded,
            size: 46,
            color: AppColors.green,
          ),
          title: s.t('mode_text'),
          subtitle: s.t('mode_text_desc'),
          selected: session.learningMode == 'text',
          onTap: () => session.setLearningMode('text'),
        ),
        const SizedBox(height: 14),
        BigChoiceCard(
          leading: const Icon(
            Icons.headphones_rounded,
            size: 46,
            color: AppColors.orange,
          ),
          title: s.t('mode_audio'),
          subtitle: s.t('mode_audio_desc'),
          selected: session.learningMode == 'audio',
          onTap: () => session.setLearningMode('audio'),
        ),
        const SizedBox(height: 32),
        PrimaryButton(
          label: s.t('continue'),
          icon: Icons.arrow_forward_rounded,
          onPressed: () {
            Navigator.of(context).popUntil((route) => route.isFirst);
            session.completeOnboarding();
          },
        ),
      ],
    );
  }
}
