import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';

import '../../../core/config.dart';
import '../../../core/localization/app_strings.dart';
import '../../../core/session_viewmodel.dart';
import '../../../core/theme/app_theme.dart';
import '../../../data/models/exam.dart';
import '../../../data/repositories/practice_repository.dart';
import '../../../shared/widgets/answer_option.dart';
import '../../../shared/widgets/auth_scaffold.dart';
import '../../../shared/widgets/feedback.dart';
import '../../../shared/widgets/listen_button.dart';
import '../../../shared/widgets/loading_overlay.dart';
import '../../../shared/widgets/surfaces.dart';
import '../viewmodels/practice_viewmodel.dart';

/// Entraînement : question → choix → « Valider » → correction et explication → suivante.
class PracticeView extends StatelessWidget {
  const PracticeView({
    super.key,
    required this.items,
    this.startIndex = 0,
    this.onRetry,
  });

  final List<PracticeItem> items;
  final int startIndex;
  final VoidCallback? onRetry;

  @override
  Widget build(BuildContext context) {
    return ChangeNotifierProvider(
      create: (c) => PracticeViewModel(
        c.read<PracticeRepository>(),
        items,
        startIndex: startIndex,
      ),
      child: _PracticeBody(onRetry: onRetry),
    );
  }
}

class _PracticeBody extends StatelessWidget {
  const _PracticeBody({this.onRetry});

  final VoidCallback? onRetry;

  @override
  Widget build(BuildContext context) {
    final vm = context.watch<PracticeViewModel>();
    final session = context.watch<SessionViewModel>();
    final s = session.strings;
    if (vm.finished) return _Summary(vm: vm, strings: s, onRetry: onRetry);

    final q = vm.question;
    final content = pick(q.contents, session.language);
    final check = vm.check;

    return Scaffold(
      body: LoadingOverlay(
        loading: vm.busy,
        child: GradientBackground(
          child: SafeArea(
            child: Column(
              children: [
                Padding(
                  padding: const EdgeInsets.fromLTRB(12, 4, 20, 8),
                  child: Column(
                    children: [
                      Row(
                        children: [
                          IconButton(
                            iconSize: 30,
                            tooltip: s.t('quit'),
                            icon: const Icon(
                              Icons.close_rounded,
                              color: Ink2.strong,
                            ),
                            onPressed: () => Navigator.of(context).pop(),
                          ),
                          Expanded(
                            child: Text(
                              '${s.t('question')} ${vm.index + 1} / ${vm.items.length}',
                              style: const TextStyle(
                                fontSize: 18,
                                fontWeight: FontWeight.w800,
                                color: Ink2.strong,
                              ),
                            ),
                          ),
                          Row(
                            children: [
                              const Icon(
                                Icons.check_circle_rounded,
                                color: AppColors.green,
                                size: 22,
                              ),
                              const SizedBox(width: 4),
                              Text(
                                '${vm.correctCount}',
                                style: const TextStyle(
                                  fontSize: 18,
                                  fontWeight: FontWeight.w800,
                                  color: AppColors.greenDark,
                                ),
                              ),
                            ],
                          ),
                        ],
                      ),
                      const SizedBox(height: 6),
                      Padding(
                        padding: const EdgeInsets.only(left: 8),
                        child: ClipRRect(
                          borderRadius: BorderRadius.circular(4),
                          child: LinearProgressIndicator(
                            value:
                                (vm.index + (vm.checked ? 1 : 0)) /
                                vm.items.length,
                            minHeight: 6,
                            backgroundColor: const Color(0xFFDCE1D9),
                            color: AppColors.orange,
                          ),
                        ),
                      ),
                    ],
                  ),
                ),
                Expanded(
                  child: ListView(
                    key: ValueKey(q.id),
                    padding: const EdgeInsets.fromLTRB(20, 8, 20, 20),
                    children: [
                      SoftCard(
                        padding: const EdgeInsets.all(18),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            if (content.imageUrl != null) ...[
                              ClipRRect(
                                borderRadius: BorderRadius.circular(16),
                                child: Image.network(
                                  AppConfig.mediaUrl(content.imageUrl!),
                                  height: 200,
                                  width: double.infinity,
                                  fit: BoxFit.contain,
                                ),
                              ),
                              const SizedBox(height: 14),
                            ],
                            Text(
                              content.text,
                              style: const TextStyle(
                                fontSize: 22,
                                fontWeight: FontWeight.w700,
                                color: Ink2.strong,
                                height: 1.3,
                              ),
                            ),
                            if (content.audioUrl != null) ...[
                              const SizedBox(height: 14),
                              ListenButton(
                                key: ValueKey('audio-${q.id}'),
                                url: content.audioUrl!,
                                label: s.t('listen'),
                                autoPlay: session.learningMode == 'audio',
                              ),
                            ],
                          ],
                        ),
                      ),
                      const SizedBox(height: 14),
                      if (check == null)
                        Row(
                          children: [
                            Icon(
                              q.multiple
                                  ? Icons.checklist_rounded
                                  : Icons.touch_app_rounded,
                              size: 20,
                              color: Ink2.muted,
                            ),
                            const SizedBox(width: 8),
                            Text(
                              s.t(
                                q.multiple ? 'several_answers' : 'one_answer',
                              ),
                              style: const TextStyle(
                                fontSize: 16,
                                fontWeight: FontWeight.w600,
                                color: Ink2.muted,
                              ),
                            ),
                          ],
                        )
                      else
                        _Verdict(
                          correct: check.outcome == Outcome.correct,
                          strings: s,
                        ),
                      const SizedBox(height: 10),
                      for (final o in q.options) ...[
                        AnswerOption(
                          code: o.code,
                          content: pick(o.contents, session.language),
                          selected: vm.selected.contains(o.id),
                          multiple: q.multiple,
                          feedback: _feedback(check, vm.selected, o.id),
                          feedbackLabel: _label(check, vm.selected, o.id, s),
                          onTap: check == null
                              ? () {
                                  HapticFeedback.selectionClick();
                                  vm.toggle(o);
                                }
                              : null,
                        ),
                        const SizedBox(height: 10),
                      ],
                      if (check != null)
                        _Explanation(
                          check: check,
                          language: session.language,
                          strings: s,
                        ),
                      if (vm.error != null) ...[
                        const SizedBox(height: 10),
                        ErrorMessage(
                          s.error(vm.error!.code, vm.error!.message),
                        ),
                      ],
                    ],
                  ),
                ),
                Padding(
                  padding: const EdgeInsets.fromLTRB(20, 8, 20, 16),
                  child: check == null
                      ? PrimaryButton(
                          label: s.t('validate'),
                          icon: Icons.check_rounded,
                          busy: vm.busy,
                          onPressed: vm.selected.isEmpty ? null : vm.validate,
                        )
                      : PrimaryButton(
                          label: vm.isLast
                              ? s.t('see_summary')
                              : s.t('next_question'),
                          icon: vm.isLast
                              ? Icons.flag_rounded
                              : Icons.arrow_forward_rounded,
                          onPressed: vm.next,
                        ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  static AnswerFeedback? _feedback(
    PracticeCheck? check,
    Set<String> selected,
    String id,
  ) {
    if (check == null) return null;
    if (check.correct.contains(id)) return AnswerFeedback.correct;
    if (selected.contains(id)) return AnswerFeedback.wrong;
    return AnswerFeedback.neutral;
  }

  static String? _label(
    PracticeCheck? check,
    Set<String> selected,
    String id,
    AppStrings s,
  ) {
    if (check == null) return null;
    final labels = [
      if (check.correct.contains(id)) s.t('good_answer'),
      if (selected.contains(id)) s.t('your_answer'),
    ];
    return labels.isEmpty ? null : labels.join('\n');
  }
}

class _Verdict extends StatelessWidget {
  const _Verdict({required this.correct, required this.strings});

  final bool correct;
  final AppStrings strings;

  @override
  Widget build(BuildContext context) {
    final color = correct ? AppColors.green : AppColors.red;
    return Semantics(
      liveRegion: true,
      child: Row(
        children: [
          Icon(
            correct ? Icons.check_circle_rounded : Icons.cancel_rounded,
            color: color,
            size: 28,
          ),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              strings.t(correct ? 'correct_feedback' : 'wrong_feedback'),
              style: TextStyle(
                fontSize: 18,
                fontWeight: FontWeight.w800,
                color: color,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _Explanation extends StatelessWidget {
  const _Explanation({
    required this.check,
    required this.language,
    required this.strings,
  });

  final PracticeCheck check;
  final String language;
  final AppStrings strings;

  @override
  Widget build(BuildContext context) {
    final e = check.explanations[language] ?? check.explanations['fr'];
    if (e == null || (e.text.isEmpty && e.audioUrl == null)) {
      return const SizedBox.shrink();
    }
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: AppColors.yellow.withValues(alpha: 0.3),
        borderRadius: BorderRadius.circular(16),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              const Icon(
                Icons.lightbulb_rounded,
                size: 22,
                color: Color(0xFF8A6D00),
              ),
              const SizedBox(width: 6),
              Text(
                strings.t('explanation'),
                style: const TextStyle(
                  fontSize: 15,
                  fontWeight: FontWeight.w800,
                  color: Color(0xFF8A6D00),
                ),
              ),
            ],
          ),
          if (e.text.isNotEmpty) ...[
            const SizedBox(height: 6),
            Text(
              e.text,
              style: const TextStyle(fontSize: 17, color: Ink2.strong),
            ),
          ],
          if (e.audioUrl != null) ...[
            const SizedBox(height: 10),
            ListenButton(
              url: e.audioUrl!,
              label: strings.t('listen_explanation'),
            ),
          ],
        ],
      ),
    );
  }
}

class _Summary extends StatelessWidget {
  const _Summary({required this.vm, required this.strings, this.onRetry});

  final PracticeViewModel vm;
  final AppStrings strings;
  final VoidCallback? onRetry;

  @override
  Widget build(BuildContext context) {
    final s = strings;
    final total = vm.items.length;
    return Scaffold(
      body: GradientBackground(
        child: SafeArea(
          child: ListView(
            padding: const EdgeInsets.fromLTRB(24, 40, 24, 24),
            children: [
              Center(
                child: Container(
                  width: 120,
                  height: 120,
                  decoration: BoxDecoration(
                    color: AppColors.orange.withValues(alpha: 0.14),
                    shape: BoxShape.circle,
                  ),
                  child: const Icon(
                    Icons.emoji_events_rounded,
                    size: 64,
                    color: AppColors.orange,
                  ),
                ),
              ),
              const SizedBox(height: 16),
              Text(
                s.t('practice_done'),
                textAlign: TextAlign.center,
                style: const TextStyle(
                  fontSize: 28,
                  fontWeight: FontWeight.w800,
                  color: Ink2.strong,
                ),
              ),
              const SizedBox(height: 8),
              Text(
                '${vm.correctCount} ${s.t('out_of')} $total ${s.t('correct_answers')}',
                textAlign: TextAlign.center,
                style: const TextStyle(fontSize: 20, color: Ink2.muted),
              ),
              const SizedBox(height: 32),
              Row(
                children: [
                  if (onRetry != null) ...[
                    Expanded(
                      child: SecondaryButton(
                        label: s.t('practice_again'),
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
                    child: PrimaryButton(
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
