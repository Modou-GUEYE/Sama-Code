import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../../core/session_viewmodel.dart';
import '../../../core/theme/app_theme.dart';
import '../../../data/models/exam.dart';
import '../../../shared/widgets/auth_scaffold.dart';
import '../../../shared/widgets/feedback.dart';
import '../../../shared/widgets/surfaces.dart';
import 'review_view.dart';

/// Résultat : réussi ou ajourné, signalé par l'icône, la couleur ET le texte.
class ResultView extends StatelessWidget {
  const ResultView({super.key, required this.session, this.onRetry});

  final ExamSession session;
  final VoidCallback? onRetry;

  @override
  Widget build(BuildContext context) {
    final s = context.watch<SessionViewModel>().strings;
    final r = session.result!;
    final color = r.passed ? AppColors.green : AppColors.red;

    return Scaffold(
      body: GradientBackground(
        child: SafeArea(
          child: ListView(
            padding: const EdgeInsets.fromLTRB(24, 32, 24, 24),
            children: [
              Center(
                child: Container(
                  width: 120,
                  height: 120,
                  decoration: BoxDecoration(
                    color: color.withValues(alpha: 0.12),
                    shape: BoxShape.circle,
                  ),
                  child: Icon(
                    r.passed
                        ? Icons.emoji_events_rounded
                        : Icons.replay_rounded,
                    size: 64,
                    color: color,
                  ),
                ),
              ),
              const SizedBox(height: 16),
              Text(
                s.t(r.passed ? 'passed' : 'failed'),
                textAlign: TextAlign.center,
                style: TextStyle(
                  fontSize: 34,
                  fontWeight: FontWeight.w800,
                  color: color,
                ),
              ),
              Text(
                s.series('Permis ${session.licenseCode}', session.seriesNumber),
                textAlign: TextAlign.center,
                style: const TextStyle(fontSize: 17, color: Ink2.muted),
              ),
              const SizedBox(height: 24),
              SoftCard(
                padding: const EdgeInsets.all(20),
                child: Column(
                  children: [
                    Text(
                      '${r.faults} ${r.faults > 1 ? s.t('faults') : s.t('fault')}',
                      style: const TextStyle(
                        fontSize: 40,
                        fontWeight: FontWeight.w800,
                        color: Ink2.strong,
                      ),
                    ),
                    Text(
                      '${s.t('max_allowed')} : ${r.maxFaults}',
                      style: const TextStyle(fontSize: 16, color: Ink2.muted),
                    ),
                    const SizedBox(height: 18),
                    IntrinsicHeight(
                      child: Row(
                        crossAxisAlignment: CrossAxisAlignment.stretch,
                        children: [
                          _Stat(
                            icon: Icons.check_circle_rounded,
                            color: AppColors.green,
                            value: r.correct,
                            label: s.t('correct_answers'),
                          ),
                          const SizedBox(width: 10),
                          _Stat(
                            icon: Icons.cancel_rounded,
                            color: AppColors.red,
                            value: r.faults - r.unanswered,
                            label: s.t('faults'),
                          ),
                          const SizedBox(width: 10),
                          _Stat(
                            icon: Icons.remove_circle_rounded,
                            color: Ink2.muted,
                            value: r.unanswered,
                            label: s.t('unanswered'),
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
              ),
              const SizedBox(height: 28),
              PrimaryButton(
                label: s.t('see_review'),
                icon: Icons.fact_check_rounded,
                onPressed: () => Navigator.of(context).push(
                  MaterialPageRoute(
                    builder: (_) => ReviewView(sessionId: session.id),
                  ),
                ),
              ),
              const SizedBox(height: 10),
              Row(
                children: [
                  if (onRetry != null) ...[
                    Expanded(
                      child: SecondaryButton(
                        label: s.t('retry_exam'),
                        icon: Icons.replay_rounded,
                        onPressed: () {
                          Navigator.of(context).pop();
                          onRetry!();
                        },
                      ),
                    ),
                    const SizedBox(width: 10),
                  ],
                  Expanded(
                    child: SecondaryButton(
                      label: s.t('back_home'),
                      icon: Icons.home_rounded,
                      onPressed: () =>
                          Navigator.of(context)
                              .popUntil((route) => route.isFirst),
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _Stat extends StatelessWidget {
  const _Stat({
    required this.icon,
    required this.color,
    required this.value,
    required this.label,
  });

  final IconData icon;
  final Color color;
  final int value;
  final String label;

  @override
  Widget build(BuildContext context) {
    return Expanded(
      child: Container(
        padding: const EdgeInsets.symmetric(vertical: 12, horizontal: 6),
        decoration: BoxDecoration(
          color: AppColors.background,
          borderRadius: BorderRadius.circular(16),
        ),
        child: Column(
          children: [
            Icon(icon, color: color),
            Text(
              '$value',
              style: const TextStyle(
                fontSize: 24,
                fontWeight: FontWeight.w800,
                color: Ink2.strong,
              ),
            ),
            Text(
              label,
              textAlign: TextAlign.center,
              maxLines: 2,
              style: const TextStyle(fontSize: 13, color: Ink2.muted),
            ),
          ],
        ),
      ),
    );
  }
}
