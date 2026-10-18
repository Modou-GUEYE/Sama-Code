import 'package:flutter/material.dart';

import '../../core/theme/app_theme.dart';
import 'surfaces.dart';

/// Gabarit des écrans d'accueil, d'inscription et de connexion : même fond en
/// dégradé, même flèche de retour et même titre aligné à gauche que l'accueil.
class AuthScaffold extends StatelessWidget {
  const AuthScaffold({
    super.key,
    required this.title,
    required this.children,
    this.subtitle,
    this.icon,
    this.iconColor = AppColors.green,
    this.leading,
    this.step,
    this.totalSteps,
    this.bottom,
  });

  final String title;
  final String? subtitle;
  final IconData? icon;
  final Color iconColor;

  /// Remplace l'icône (ex. le logo sur le premier écran).
  final Widget? leading;
  final List<Widget> children;

  /// Étape courante (à partir de 1) d'un parcours, affichée en haut.
  final int? step;
  final int? totalSteps;

  /// Action principale fixée en bas, qui remonte au-dessus du clavier.
  final Widget? bottom;

  @override
  Widget build(BuildContext context) {
    final canPop = Navigator.of(context).canPop();
    return Scaffold(
      body: GradientBackground(
        child: SafeArea(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Padding(
                padding: const EdgeInsets.fromLTRB(20, 4, 20, 0),
                child: SizedBox(
                  height: 48,
                  child: Row(
                    children: [
                      if (canPop)
                        IconButton(
                          iconSize: 30,
                          tooltip: MaterialLocalizations.of(context)
                              .backButtonTooltip,
                          icon: const Icon(
                            Icons.arrow_back_rounded,
                            color: Ink2.strong,
                          ),
                          onPressed: () => Navigator.of(context).pop(),
                        ),
                      if (step != null && totalSteps != null) ...[
                        const SizedBox(width: 8),
                        Expanded(
                          child: _StepBar(step: step!, total: totalSteps!),
                        ),
                        const SizedBox(width: 56),
                      ],
                    ],
                  ),
                ),
              ),
              Expanded(
                child: ListView(
                  padding: const EdgeInsets.fromLTRB(20, 12, 20, 24),
                  children: [
                    Padding(
                      padding: const EdgeInsets.symmetric(horizontal: 4),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          // Le logo (premier écran) reste au-dessus du titre ;
                          // une icône se place sur la même ligne que le titre.
                          if (leading != null) ...[
                            leading!,
                            const SizedBox(height: 20),
                          ],
                          Row(
                            children: [
                              if (leading == null && icon != null) ...[
                                Container(
                                  width: 56,
                                  height: 56,
                                  decoration: BoxDecoration(
                                    color: iconColor.withValues(alpha: 0.14),
                                    borderRadius: BorderRadius.circular(18),
                                  ),
                                  child: Icon(icon, size: 30, color: iconColor),
                                ),
                                const SizedBox(width: 14),
                              ],
                              Expanded(
                                child: Text(
                                  title,
                                  style: const TextStyle(
                                    fontSize: 26,
                                    fontWeight: FontWeight.w800,
                                    color: Ink2.strong,
                                    height: 1.2,
                                  ),
                                ),
                              ),
                            ],
                          ),
                          if (subtitle != null) ...[
                            const SizedBox(height: 12),
                            Text(
                              subtitle!,
                              style: const TextStyle(
                                fontSize: 17,
                                color: Ink2.muted,
                              ),
                            ),
                          ],
                        ],
                      ),
                    ),
                    const SizedBox(height: 28),
                    ...children,
                  ],
                ),
              ),
              if (bottom != null)
                Padding(
                  padding: const EdgeInsets.fromLTRB(20, 8, 20, 20),
                  child: bottom,
                ),
            ],
          ),
        ),
      ),
    );
  }
}

/// Barre d'étapes segmentée : étapes faites et en cours en vert, à venir en gris.
class _StepBar extends StatelessWidget {
  const _StepBar({required this.step, required this.total});

  final int step;
  final int total;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      label: 'Étape $step sur $total',
      child: Row(
        children: [
          for (var i = 1; i <= total; i++) ...[
            if (i > 1) const SizedBox(width: 6),
            Expanded(
              child: AnimatedContainer(
                duration: const Duration(milliseconds: 250),
                height: 6,
                decoration: BoxDecoration(
                  color: i <= step ? AppColors.green : const Color(0xFFDCE1D9),
                  borderRadius: BorderRadius.circular(3),
                ),
              ),
            ),
          ],
        ],
      ),
    );
  }
}

/// Bouton secondaire : fond blanc, texte vert, sans bordure.
class SecondaryButton extends StatelessWidget {
  const SecondaryButton({
    super.key,
    required this.label,
    this.icon,
    this.onPressed,
    this.tinted = false,
  });

  final String label;
  final IconData? icon;
  final VoidCallback? onPressed;

  /// Fond vert très pâle au lieu de blanc (bouton posé sur une surface blanche).
  final bool tinted;

  @override
  Widget build(BuildContext context) {
    return FilledButton(
      style: FilledButton.styleFrom(
        backgroundColor: tinted ? const Color(0xFFE3F2E9) : Colors.white,
        foregroundColor: AppColors.green,
        elevation: 0,
      ),
      onPressed: onPressed,
      child: Row(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          if (icon != null) ...[
            Icon(icon, size: 26),
            const SizedBox(width: 12),
          ],
          Flexible(
            child: FittedBox(
              fit: BoxFit.scaleDown,
              child: Text(label, maxLines: 1, softWrap: false),
            ),
          ),
        ],
      ),
    );
  }
}
