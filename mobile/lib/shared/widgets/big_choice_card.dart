import 'package:flutter/material.dart';

import '../../core/theme/app_theme.dart';
import 'surfaces.dart';

/// Grande carte de choix sans bordure. La sélection est signalée par la coche,
/// le fond teinté et le texte en gras, jamais par la couleur seule.
class BigChoiceCard extends StatelessWidget {
  const BigChoiceCard({
    super.key,
    required this.leading,
    required this.title,
    this.subtitle,
    required this.selected,
    required this.onTap,
  });

  final Widget leading;
  final String title;
  final String? subtitle;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      button: true,
      selected: selected,
      label: subtitle == null ? title : '$title. $subtitle',
      excludeSemantics: true,
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 180),
        decoration: BoxDecoration(
          color: selected ? const Color(0xFFE3F2E9) : Colors.white,
          borderRadius: BorderRadius.circular(24),
          boxShadow: selected
              ? const []
              : const [
                  BoxShadow(
                    color: Color(0x0F000000),
                    blurRadius: 24,
                    offset: Offset(0, 8),
                  ),
                ],
        ),
        child: Material(
          type: MaterialType.transparency,
          child: InkWell(
            borderRadius: BorderRadius.circular(24),
            onTap: onTap,
            child: Padding(
              padding: const EdgeInsets.all(18),
              child: Row(
                children: [
                  SizedBox(
                    width: 60,
                    height: 60,
                    child: Center(child: leading),
                  ),
                  const SizedBox(width: 16),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          title,
                          style: const TextStyle(
                            fontSize: 21,
                            fontWeight: FontWeight.w800,
                            color: Ink2.strong,
                          ),
                        ),
                        if (subtitle != null) ...[
                          const SizedBox(height: 2),
                          Text(
                            subtitle!,
                            style: const TextStyle(
                              fontSize: 15,
                              color: Ink2.muted,
                            ),
                          ),
                        ],
                      ],
                    ),
                  ),
                  AnimatedSwitcher(
                    duration: const Duration(milliseconds: 180),
                    child: selected
                        ? const Icon(
                            Icons.check_circle_rounded,
                            key: ValueKey(true),
                            size: 32,
                            color: AppColors.green,
                          )
                        : const Icon(
                            Icons.circle_outlined,
                            key: ValueKey(false),
                            size: 32,
                            color: Color(0xFFC9CEC6),
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

/// Pastille ronde avec un sigle (ex. « FR », « WO »).
class InitialsBadge extends StatelessWidget {
  const InitialsBadge(this.text, {super.key, this.color = AppColors.green});

  final String text;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return CircleAvatar(
      radius: 28,
      backgroundColor: color,
      child: Text(
        text,
        style: const TextStyle(
          color: Colors.white,
          fontSize: 19,
          fontWeight: FontWeight.w800,
        ),
      ),
    );
  }
}
