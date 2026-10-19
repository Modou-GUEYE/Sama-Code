import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../../core/api/api_exception.dart';
import '../../../core/config.dart';
import '../../../core/localization/app_strings.dart';
import '../../../core/session_viewmodel.dart';
import '../../../core/theme/app_theme.dart';
import '../../../data/models/exam.dart';
import '../../../data/repositories/exam_repository.dart';
import '../../../shared/widgets/auth_scaffold.dart';
import '../../../shared/widgets/feedback.dart';
import '../../../shared/widgets/listen_button.dart';
import '../../../shared/widgets/surfaces.dart';

/// Correction après soumission : bonne réponse, réponse du candidat, explication.
class ReviewView extends StatefulWidget {
  const ReviewView({super.key, required this.sessionId});

  final String sessionId;

  @override
  State<ReviewView> createState() => _ReviewViewState();
}

class _ReviewViewState extends State<ReviewView> {
  List<ReviewQuestion>? _questions;
  ApiException? _error;
  bool _mistakesOnly = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => _error = null);
    try {
      final q = await context.read<ExamRepository>().review(widget.sessionId);
      if (mounted) setState(() => _questions = q);
    } on ApiException catch (e) {
      if (mounted) setState(() => _error = e);
    }
  }

  @override
  Widget build(BuildContext context) {
    final session = context.watch<SessionViewModel>();
    final s = session.strings;
    final all = _questions;
    final shown = all == null
        ? null
        : (_mistakesOnly
              ? all.where((q) => q.outcome != Outcome.correct).toList()
              : all);

    return AuthScaffold(
      icon: Icons.fact_check_rounded,
      title: s.t('review_title'),
      children: [
        if (_error != null) ...[
          ErrorMessage(s.error(_error!.code, _error!.message)),
          const SizedBox(height: 12),
          SecondaryButton(
            label: s.t('retry'),
            icon: Icons.refresh_rounded,
            onPressed: _load,
          ),
        ] else if (shown == null)
          const Padding(
            padding: EdgeInsets.all(32),
            child: Center(child: CircularProgressIndicator()),
          )
        else ...[
          _Filter(
            mistakesOnly: _mistakesOnly,
            strings: s,
            mistakes: all!.where((q) => q.outcome != Outcome.correct).length,
            onChanged: (v) => setState(() => _mistakesOnly = v),
          ),
          const SizedBox(height: 14),
          if (shown.isEmpty)
            InfoMessage(s.t('no_mistakes'), icon: Icons.emoji_events_rounded),
          for (final q in shown) ...[
            _ReviewCard(review: q, language: session.language, strings: s),
            const SizedBox(height: 14),
          ],
        ],
      ],
    );
  }
}

class _Filter extends StatelessWidget {
  const _Filter({
    required this.mistakesOnly,
    required this.strings,
    required this.mistakes,
    required this.onChanged,
  });

  final bool mistakesOnly;
  final AppStrings strings;
  final int mistakes;
  final ValueChanged<bool> onChanged;

  @override
  Widget build(BuildContext context) {
    Widget tab(String label, bool value) => Expanded(
      child: GestureDetector(
        onTap: () => onChanged(value),
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 180),
          height: 46,
          alignment: Alignment.center,
          decoration: BoxDecoration(
            color: mistakesOnly == value ? AppColors.green : Colors.transparent,
            borderRadius: BorderRadius.circular(999),
          ),
          child: Text(
            label,
            style: TextStyle(
              fontSize: 16,
              fontWeight: FontWeight.w700,
              color: mistakesOnly == value ? Colors.white : Ink2.muted,
            ),
          ),
        ),
      ),
    );
    return Container(
      padding: const EdgeInsets.all(4),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(999),
      ),
      child: Row(
        children: [
          tab(strings.t('all_questions'), false),
          tab('${strings.t('my_mistakes')} ($mistakes)', true),
        ],
      ),
    );
  }
}

class _ReviewCard extends StatelessWidget {
  const _ReviewCard({
    required this.review,
    required this.language,
    required this.strings,
  });

  final ReviewQuestion review;
  final String language;
  final AppStrings strings;

  @override
  Widget build(BuildContext context) {
    final s = strings;
    final q = review.question;
    final content = pick(q.contents, language);
    final (icon, color, label) = switch (review.outcome) {
      Outcome.correct => (
        Icons.check_circle_rounded,
        AppColors.green,
        s.t('good_answer'),
      ),
      Outcome.incorrect => (Icons.cancel_rounded, AppColors.red, s.t('failed')),
      Outcome.unanswered => (
        Icons.remove_circle_rounded,
        Ink2.muted,
        s.t('unanswered'),
      ),
    };
    final explanation =
        review.explanations[language] ?? review.explanations['fr'];

    return SoftCard(
      padding: const EdgeInsets.all(18),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(icon, color: color, size: 26),
              const SizedBox(width: 8),
              Text(
                '${s.t('question')} ${q.position}',
                style: const TextStyle(
                  fontSize: 16,
                  fontWeight: FontWeight.w800,
                  color: Ink2.strong,
                ),
              ),
              const Spacer(),
              Text(
                label,
                style: TextStyle(
                  fontSize: 14,
                  fontWeight: FontWeight.w700,
                  color: color,
                ),
              ),
            ],
          ),
          const SizedBox(height: 10),
          if (content.imageUrl != null) ...[
            ClipRRect(
              borderRadius: BorderRadius.circular(14),
              child: Image.network(
                AppConfig.mediaUrl(content.imageUrl!),
                height: 150,
                width: double.infinity,
                fit: BoxFit.contain,
              ),
            ),
            const SizedBox(height: 10),
          ],
          Text(
            content.text,
            style: const TextStyle(
              fontSize: 19,
              fontWeight: FontWeight.w700,
              color: Ink2.strong,
            ),
          ),
          const SizedBox(height: 12),
          for (final o in q.options)
            _ReviewOption(
              code: o.code,
              text: pick(o.contents, language).text,
              correct: review.correct.contains(o.id),
              selected: review.selected.contains(o.id),
              strings: s,
            ),
          if (explanation != null &&
              (explanation.text.isNotEmpty ||
                  explanation.audioUrl != null)) ...[
            const SizedBox(height: 6),
            Container(
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
                        s.t('explanation'),
                        style: const TextStyle(
                          fontSize: 15,
                          fontWeight: FontWeight.w800,
                          color: Color(0xFF8A6D00),
                        ),
                      ),
                    ],
                  ),
                  if (explanation.text.isNotEmpty) ...[
                    const SizedBox(height: 6),
                    Text(
                      explanation.text,
                      style: const TextStyle(fontSize: 17, color: Ink2.strong),
                    ),
                  ],
                  if (explanation.audioUrl != null) ...[
                    const SizedBox(height: 10),
                    ListenButton(
                      url: explanation.audioUrl!,
                      label: s.t('listen_explanation'),
                    ),
                  ],
                ],
              ),
            ),
          ],
        ],
      ),
    );
  }
}

class _ReviewOption extends StatelessWidget {
  const _ReviewOption({
    required this.code,
    required this.text,
    required this.correct,
    required this.selected,
    required this.strings,
  });

  final String code;
  final String text;
  final bool correct;
  final bool selected;
  final AppStrings strings;

  @override
  Widget build(BuildContext context) {
    final wrong = selected && !correct;
    final bg = correct
        ? const Color(0xFFE3F2E9)
        : wrong
        ? AppColors.red.withValues(alpha: 0.08)
        : AppColors.background;
    final tags = [
      if (correct) strings.t('good_answer'),
      if (selected) strings.t('your_answer'),
    ];
    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: bg,
        borderRadius: BorderRadius.circular(14),
      ),
      child: Row(
        children: [
          Text(
            code,
            style: const TextStyle(
              fontSize: 18,
              fontWeight: FontWeight.w800,
              color: Ink2.strong,
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  text,
                  style: const TextStyle(fontSize: 17, color: Ink2.strong),
                ),
                if (tags.isNotEmpty)
                  Text(
                    tags.join(' · '),
                    style: TextStyle(
                      fontSize: 13,
                      fontWeight: FontWeight.w700,
                      color: correct ? AppColors.greenDark : AppColors.red,
                    ),
                  ),
              ],
            ),
          ),
          if (correct)
            const Icon(Icons.check_circle_rounded, color: AppColors.green)
          else if (wrong)
            const Icon(Icons.cancel_rounded, color: AppColors.red),
        ],
      ),
    );
  }
}
