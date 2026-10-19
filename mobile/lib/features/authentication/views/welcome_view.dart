import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../../core/localization/app_strings.dart';
import '../../../core/session_viewmodel.dart';
import '../../../data/repositories/auth_repository.dart';
import '../../../shared/widgets/auth_scaffold.dart';
import '../../../shared/widgets/feedback.dart';
import '../../../shared/widgets/animated_road.dart';
import '../../../shared/widgets/surfaces.dart';
import '../viewmodels/auth_flow_viewmodel.dart';
import 'auth_navigation.dart';
import 'phone_view.dart';

/// Accueil non connecté : créer un compte ou se connecter (compte obligatoire, D19).
///
/// Une route stylisée traverse le bas de l'écran : pointillés qui défilent et
/// petite voiture, désactivés si le système demande de réduire les animations.
class WelcomeView extends StatefulWidget {
  const WelcomeView({super.key});

  @override
  State<WelcomeView> createState() => _WelcomeViewState();
}

class _WelcomeViewState extends State<WelcomeView>
    with SingleTickerProviderStateMixin {
  late final _intro = AnimationController(
    vsync: this,
    duration: const Duration(milliseconds: 900),
  );

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (MediaQuery.disableAnimationsOf(context)) {
      _intro.value = 1;
    } else if (_intro.value == 0) {
      _intro.forward();
    }
  }

  @override
  void dispose() {
    _intro.dispose();
    super.dispose();
  }

  void _start(AuthFlow flow) {
    final vm = AuthFlowViewModel(context.read<AuthRepository>(), flow);
    pushAuthStep(context, vm, const PhoneView());
  }

  @override
  Widget build(BuildContext context) {
    final session = context.watch<SessionViewModel>();
    final s = session.strings;
    final fade = CurvedAnimation(parent: _intro, curve: Curves.easeOutCubic);

    return Scaffold(
      body: GradientBackground(
        child: Stack(
          children: [
            const Positioned.fill(child: AnimatedRoad()),
            SafeArea(
              child: Padding(
                padding: const EdgeInsets.fromLTRB(24, 12, 24, 24),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    const Spacer(),
                    FadeTransition(
                      opacity: fade,
                      child: SlideTransition(
                        position: Tween(
                          begin: const Offset(0, 0.06),
                          end: Offset.zero,
                        ).animate(fade),
                        // Le logo porte le nom et le slogan : pas de titre écrit.
                        // « Bienvenue sur Sama Code » reste lu par le lecteur d'écran.
                        child: Semantics(
                          label: s.t('welcome'),
                          excludeSemantics: true,
                          child: const AppLogo(size: 280),
                        ),
                      ),
                    ),
                    const Spacer(flex: 2),
                    FadeTransition(
                      opacity: fade,
                      child: _Actions(
                        strings: s,
                        onRegister: () => _start(AuthFlow.register),
                        onLogin: () => _start(AuthFlow.pinLogin),
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _Actions extends StatelessWidget {
  const _Actions({
    required this.strings,
    required this.onRegister,
    required this.onLogin,
  });

  final AppStrings strings;
  final VoidCallback onRegister;
  final VoidCallback onLogin;

  @override
  Widget build(BuildContext context) {
    final s = strings;
    return Row(
      children: [
        Expanded(
          child: PrimaryButton(
            label: s.t('register'),
            icon: Icons.person_add_alt_1_rounded,
            onPressed: onRegister,
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: SecondaryButton(
            tinted: true,
            label: s.t('login'),
            icon: Icons.login_rounded,
            onPressed: onLogin,
          ),
        ),
      ],
    );
  }
}
