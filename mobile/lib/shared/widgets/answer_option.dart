import 'package:flutter/material.dart';

import '../../core/config.dart';
import '../../core/theme/app_theme.dart';
import '../../data/models/exam.dart';
import 'surfaces.dart';

/// État d'une proposition après correction (entraînement). null = pas encore corrigée.
enum AnswerFeedback { correct, wrong, neutral }

/// Grande proposition de réponse : lettre, texte (et image éventuelle), coche.
/// Après correction, la bonne réponse est verte avec ✓, une erreur rouge avec ✗ :
/// l'information passe par l'icône et le libellé, jamais par la couleur seule.
class AnswerOption extends StatelessWidget {
  const AnswerOption({
    super.key,
    required this.code,
    required this.content,
    required this.selected,
    required this.multiple,
    this.onTap,
    this.feedback,
    this.feedbackLabel,
  });

  final String code;
  final LocalizedContent content;
  final bool selected;
  final bool multiple;
  final VoidCallback? onTap;
  final AnswerFeedback? feedback;

  /// Libellé affiché sous le texte après correction (« Bonne réponse », « Votre réponse »).
  final String? feedbackLabel;

  @override
  Widget build(BuildContext context) {
    final (bg, badge, accent) = switch (feedback) {
      AnswerFeedback.correct => (
        const Color(0xFFE3F2E9),
        AppColors.green,
        AppColors.greenDark,
      ),
      AnswerFeedback.wrong => (
        AppColors.red.withValues(alpha: 0.09),
        AppColors.red,
        AppColors.red,
      ),
      AnswerFeedback.neutral => (
        Colors.white,
        AppColors.background,
        Ink2.muted,
      ),
      null =>
        selected
            ? (const Color(0xFFE3F2E9), AppColors.green, AppColors.green)
            : (Colors.white, AppColors.background, const Color(0xFFC9CEC6)),
    };
    final highlighted =
        feedback == AnswerFeedback.correct ||
        feedback == AnswerFeedback.wrong ||
        (feedback == null && selected);
    final mark = switch (feedback) {
      AnswerFeedback.correct => Icons.check_circle_rounded,
      AnswerFeedback.wrong => Icons.cancel_rounded,
      _ =>
        multiple
            ? (selected
                  ? Icons.check_box_rounded
                  : Icons.check_box_outline_blank_rounded)
            : (selected ? Icons.check_circle_rounded : Icons.circle_outlined),
    };

    return Semantics(
      button: onTap != null,
      selected: selected,
      label: ['$code. ${content.text}', ?feedbackLabel].join('. '),
      excludeSemantics: true,
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 150),
        decoration: BoxDecoration(
          color: bg,
          borderRadius: BorderRadius.circular(20),
          boxShadow: highlighted
              ? const []
              : const [
                  BoxShadow(
                    color: Color(0x0D000000),
                    blurRadius: 16,
                    offset: Offset(0, 6),
                  ),
                ],
        ),
        child: Material(
          type: MaterialType.transparency,
          child: InkWell(
            borderRadius: BorderRadius.circular(20),
            onTap: onTap,
            child: Padding(
              padding: const EdgeInsets.all(14),
              child: Row(
                children: [
                  AnimatedContainer(
                    duration: const Duration(milliseconds: 150),
                    width: 46,
                    height: 46,
                    alignment: Alignment.center,
                    decoration: BoxDecoration(
                      color: highlighted ? badge : AppColors.background,
                      borderRadius: BorderRadius.circular(14),
                    ),
                    child: Text(
                      code,
                      style: TextStyle(
                        fontSize: 20,
                        fontWeight: FontWeight.w800,
                        color: highlighted ? Colors.white : Ink2.strong,
                      ),
                    ),
                  ),
                  const SizedBox(width: 14),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        if (content.imageUrl != null) ...[
                          ClipRRect(
                            borderRadius: BorderRadius.circular(12),
                            child: Image.network(
                              AppConfig.mediaUrl(content.imageUrl!),
                              height: 90,
                              fit: BoxFit.contain,
                            ),
                          ),
                          const SizedBox(height: 6),
                        ],
                        Text(
                          content.text,
                          style: TextStyle(
                            fontSize: 19,
                            fontWeight: highlighted
                                ? FontWeight.w700
                                : FontWeight.w500,
                            color: Ink2.strong,
                          ),
                        ),
                        if (feedbackLabel != null)
                          Text(
                            feedbackLabel!,
                            style: TextStyle(
                              fontSize: 14,
                              fontWeight: FontWeight.w700,
                              color: accent,
                            ),
                          ),
                      ],
                    ),
                  ),
                  const SizedBox(width: 8),
                  Icon(mark, size: 30, color: accent),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}
