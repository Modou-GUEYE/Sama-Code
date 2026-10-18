import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../../core/localization/app_strings.dart';
import '../../../core/session_viewmodel.dart';
import '../../../core/theme/app_theme.dart';
import '../../../data/repositories/auth_repository.dart';
import '../../../shared/widgets/auth_scaffold.dart';
import '../../../shared/widgets/feedback.dart';
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
    with TickerProviderStateMixin {
  late final _road = AnimationController(
    vsync: this,
    duration: const Duration(seconds: 9),
  );
  late final _intro = AnimationController(
    vsync: this,
    duration: const Duration(milliseconds: 900),
  );

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (MediaQuery.disableAnimationsOf(context)) {
      _road.value = 0.35;
      _intro.value = 1;
    } else {
      if (!_road.isAnimating) _road.repeat();
      if (_intro.value == 0) _intro.forward();
    }
  }

  @override
  void dispose() {
    _road.dispose();
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
            Positioned.fill(child: CustomPaint(painter: _RoadPainter(_road))),
            SafeArea(
              child: Padding(
                padding: const EdgeInsets.fromLTRB(24, 12, 24, 24),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    Align(
                      alignment: Alignment.centerRight,
                      child: _LanguagePill(
                        current: session.language,
                        onChanged: session.setLanguage,
                      ),
                    ),
                    const Spacer(),
                    FadeTransition(
                      opacity: fade,
                      child: SlideTransition(
                        position: Tween(
                          begin: const Offset(0, 0.06),
                          end: Offset.zero,
                        ).animate(fade),
                        child: Column(
                          children: [
                            const AppLogo(size: 240),
                            const SizedBox(height: 4),
                            FittedBox(
                              fit: BoxFit.scaleDown,
                              child: Text(
                                s.t('welcome'),
                                maxLines: 1,
                                softWrap: false,
                                style: const TextStyle(
                                  fontSize: 26,
                                  fontWeight: FontWeight.w800,
                                  color: Ink2.strong,
                                ),
                              ),
                            ),
                          ],
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

/// Route courbe en vert pâle, ligne médiane blanche qui défile, petite voiture.
class _RoadPainter extends CustomPainter {
  _RoadPainter(this.progress) : super(repaint: progress);

  final Animation<double> progress;

  static const _dash = 22.0;
  static const _gap = 20.0;
  static const _roadWidth = 78.0;

  Path _path(Size size) => Path()
    ..moveTo(-60, size.height * 0.80)
    ..cubicTo(
      size.width * 0.30,
      size.height * 0.62,
      size.width * 0.70,
      size.height * 1.02,
      size.width + 60,
      size.height * 0.84,
    );

  @override
  void paint(Canvas canvas, Size size) {
    final path = _path(size);

    canvas.drawPath(
      path,
      Paint()
        ..color = AppColors.green.withValues(alpha: 0.10)
        ..style = PaintingStyle.stroke
        ..strokeWidth = _roadWidth,
    );

    final metric = path.computeMetrics().first;
    final dashPaint = Paint()
      ..color = Colors.white
      ..style = PaintingStyle.stroke
      ..strokeWidth = 5
      ..strokeCap = StrokeCap.round;
    // Les pointillés reculent pendant que la voiture avance : effet de mouvement.
    final period = _dash + _gap;
    var start = -(progress.value * metric.length * 2) % period - period;
    for (; start < metric.length; start += period) {
      final from = start.clamp(0.0, metric.length);
      final to = (start + _dash).clamp(0.0, metric.length);
      if (to > from) canvas.drawPath(metric.extractPath(from, to), dashPaint);
    }

    // Petite voiture vue du dessus, sur la voie de droite, orientée selon la route.
    final tangent = metric.getTangentForOffset(progress.value * metric.length);
    if (tangent == null) return;
    final lane = Offset(-tangent.vector.dy, tangent.vector.dx) * 17;
    canvas
      ..save()
      ..translate(tangent.position.dx + lane.dx, tangent.position.dy + lane.dy)
      ..rotate(-tangent.angle);
    _paintCar(canvas);
    canvas.restore();
  }

  /// Voiture de 36 × 18, avant tourné vers +x, centrée sur l'origine.
  static void _paintCar(Canvas canvas) {
    canvas.drawRRect(
      RRect.fromRectAndRadius(
        const Rect.fromLTWH(-17, -8, 36, 18),
        const Radius.circular(6),
      ),
      Paint()..color = const Color(0x22000000),
    );
    canvas.drawRRect(
      RRect.fromRectAndRadius(
        const Rect.fromLTWH(-18, -9, 36, 18),
        const Radius.circular(6),
      ),
      Paint()..color = AppColors.orange,
    );
    final glass = Paint()..color = const Color(0xFF3B4A54);
    // Pare-brise (avant) et lunette arrière.
    canvas.drawRRect(
      RRect.fromRectAndRadius(
        const Rect.fromLTWH(3, -6.5, 7, 13),
        const Radius.circular(2.5),
      ),
      glass,
    );
    canvas.drawRRect(
      RRect.fromRectAndRadius(
        const Rect.fromLTWH(-13, -6, 5, 12),
        const Radius.circular(2),
      ),
      glass,
    );
  }

  @override
  bool shouldRepaint(_RoadPainter oldDelegate) => false;
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

/// Sélecteur FR / WO compact : pastille blanche, langue active en vert.
class _LanguagePill extends StatelessWidget {
  const _LanguagePill({required this.current, required this.onChanged});

  final String current;
  final ValueChanged<String> onChanged;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(4),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(28),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          for (final code in const ['fr', 'wo'])
            Semantics(
              button: true,
              selected: code == current,
              label: code == 'fr' ? 'Français' : 'Wolof',
              excludeSemantics: true,
              child: GestureDetector(
                onTap: () => onChanged(code),
                child: AnimatedContainer(
                  duration: const Duration(milliseconds: 180),
                  width: 64,
                  height: 44,
                  alignment: Alignment.center,
                  decoration: BoxDecoration(
                    color: code == current
                        ? AppColors.green
                        : Colors.transparent,
                    borderRadius: BorderRadius.circular(24),
                  ),
                  child: Text(
                    code.toUpperCase(),
                    style: TextStyle(
                      fontSize: 17,
                      fontWeight: FontWeight.w800,
                      color: code == current ? Colors.white : Ink2.muted,
                    ),
                  ),
                ),
              ),
            ),
        ],
      ),
    );
  }
}
