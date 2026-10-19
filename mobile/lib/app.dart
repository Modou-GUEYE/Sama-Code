import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:provider/provider.dart';

import 'core/session_viewmodel.dart';
import 'core/theme/app_theme.dart';
import 'features/authentication/views/welcome_view.dart';
import 'features/home/views/home_view.dart';
import 'features/onboarding/views/onboarding_views.dart';
import 'features/settings/views/terms_view.dart';
import 'shared/widgets/feedback.dart';

class SamaCodeApp extends StatelessWidget {
  const SamaCodeApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Sama Code',
      debugShowCheckedModeBanner: false,
      theme: buildAppTheme(),
      // Le wolof n'a pas de traduction Material : les libellés système restent en français.
      locale: const Locale('fr'),
      supportedLocales: const [Locale('fr')],
      localizationsDelegates: GlobalMaterialLocalizations.delegates,
      home: const _RootGate(),
    );
  }
}

/// Affiche l'écran racine selon l'état de la session.
class _RootGate extends StatelessWidget {
  const _RootGate();

  @override
  Widget build(BuildContext context) {
    final status = context.select<SessionViewModel, SessionStatus>(
      (s) => s.status,
    );
    final needsTerms = context.select<SessionViewModel, bool>(
      (s) => s.needsTerms,
    );
    return KeyedSubtree(
      key: ValueKey((status, needsTerms)),
      child: switch (status) {
        SessionStatus.loading => const _SplashView(),
        SessionStatus.onboarding => const LanguageView(),
        SessionStatus.signedOut => const WelcomeView(),
        // Compte créé avant les CGU, ou CGU mises à jour : acceptation d'abord.
        SessionStatus.signedIn when needsTerms => const TermsView(
          acceptance: true,
        ),
        SessionStatus.signedIn => const HomeView(),
      },
    );
  }
}

class _SplashView extends StatelessWidget {
  const _SplashView();

  @override
  Widget build(BuildContext context) {
    return const Scaffold(
      body: Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            AppLogo(size: 260),
            SizedBox(height: 24),
            CircularProgressIndicator(color: AppColors.green),
          ],
        ),
      ),
    );
  }
}
