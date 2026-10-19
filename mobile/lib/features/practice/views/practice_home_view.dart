import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../../core/api/api_exception.dart';
import '../../../core/localization/app_strings.dart';
import '../../../core/session_viewmodel.dart';
import '../../../core/theme/app_theme.dart';
import '../../../data/models/exam.dart';
import '../../../data/repositories/practice_repository.dart';
import '../../../shared/widgets/app_sheet.dart';
import '../../../shared/widgets/auth_scaffold.dart';
import '../../../shared/widgets/feedback.dart';
import '../../../shared/widgets/surfaces.dart';
import 'practice_view.dart';

/// Accueil de l'entraînement : séries de chaque permis et erreurs à revoir.
class PracticeHomeView extends StatefulWidget {
  const PracticeHomeView({super.key});

  @override
  State<PracticeHomeView> createState() => _PracticeHomeViewState();
}

class _PracticeHomeViewState extends State<PracticeHomeView> {
  List<PracticeLicense>? _licenses;
  ApiException? _error;
  bool _opening = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => _error = null);
    try {
      final l = await context.read<PracticeRepository>().summary();
      if (mounted) setState(() => _licenses = l);
    } on ApiException catch (e) {
      if (mounted) setState(() => _error = e);
    }
  }

  /// Charge les questions puis ouvre l'entraînement ; rafraîchit la progression au retour.
  /// [resumable] : série commencée mais pas finie → proposer de reprendre.
  Future<void> _open(
    Future<List<PracticeItem>> Function() load, {
    bool resumable = false,
  }) async {
    setState(() => _opening = true);
    List<PracticeItem> items;
    try {
      items = await load();
    } on ApiException catch (e) {
      if (mounted) setState(() => _error = e);
      return;
    } finally {
      if (mounted) setState(() => _opening = false);
    }
    if (!mounted || items.isEmpty) return;

    var start = 0;
    final next = items.indexWhere((i) => i.lastOutcome == null);
    if (resumable && next > 0) {
      final resume = await _askResume(next + 1);
      if (resume == null || !mounted) return;
      if (resume) start = next;
    }
    await Navigator.of(context).push(
      MaterialPageRoute(
        builder: (_) => PracticeView(
          items: items,
          startIndex: start,
          onRetry: () => _open(load),
        ),
      ),
    );
    await _load();
  }

  /// true = reprendre à la question [questionNumber] ; false = recommencer.
  Future<bool?> _askResume(int questionNumber) {
    final s = context.read<SessionViewModel>().strings;
    return showAppSheet<bool>(
      context: context,
      builder: (sheet) => SafeArea(
        child: Padding(
          padding: const EdgeInsets.fromLTRB(24, 0, 24, 24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const Icon(
                Icons.bookmark_rounded,
                size: 44,
                color: AppColors.orange,
              ),
              const SizedBox(height: 12),
              Text(
                s.t('practice_started'),
                textAlign: TextAlign.center,
                style: const TextStyle(
                  fontSize: 21,
                  fontWeight: FontWeight.w800,
                  color: Ink2.strong,
                ),
              ),
              const SizedBox(height: 24),
              PrimaryButton(
                label: '${s.t('resume_at')} $questionNumber',
                icon: Icons.play_arrow_rounded,
                onPressed: () => Navigator.of(sheet).pop(true),
              ),
              const SizedBox(height: 10),
              SecondaryButton(
                label: s.t('restart_from_start'),
                icon: Icons.replay_rounded,
                tinted: true,
                onPressed: () => Navigator.of(sheet).pop(false),
              ),
            ],
          ),
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final s = context.watch<SessionViewModel>().strings;
    final repo = context.read<PracticeRepository>();
    final licenses = _licenses?.where((l) => l.series.isNotEmpty).toList();

    return AuthScaffold(
      icon: Icons.quiz_rounded,
      iconColor: AppColors.orange,
      title: s.t('practice_title'),
      subtitle: s.t('practice_subtitle'),
      busy: _opening,
      busyMessage: s.t('please_wait'),
      children: [
        if (_error != null) ...[
          ErrorMessage(s.error(_error!.code, _error!.message)),
          const SizedBox(height: 12),
          SecondaryButton(
            label: s.t('retry'),
            icon: Icons.refresh_rounded,
            onPressed: _load,
          ),
        ] else if (licenses == null)
          const Padding(
            padding: EdgeInsets.all(32),
            child: Center(child: CircularProgressIndicator()),
          )
        else if (licenses.isEmpty)
          InfoMessage(
            s.t('nothing_to_practice'),
            icon: Icons.hourglass_empty_rounded,
          )
        else
          for (final l in licenses) ...[
            _LicenseSection(
              license: l,
              strings: s,
              onSeries: (series) =>
                  _open(() => repo.seriesQuestions(series.id), resumable: true),
              onMistakes: () => _open(() => repo.mistakes(l.id)),
            ),
            const SizedBox(height: 24),
          ],
      ],
    );
  }
}

class _LicenseSection extends StatelessWidget {
  const _LicenseSection({
    required this.license,
    required this.strings,
    required this.onSeries,
    required this.onMistakes,
  });

  final PracticeLicense license;
  final AppStrings strings;
  final ValueChanged<PracticeSeries> onSeries;
  final VoidCallback onMistakes;

  @override
  Widget build(BuildContext context) {
    final s = strings;
    final isTruck = license.code == 'C';
    final subtitleKey = 'license_${license.code}_subtitle';
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Row(
          children: [
            Icon(
              isTruck
                  ? Icons.local_shipping_rounded
                  : Icons.directions_car_rounded,
              color: isTruck ? AppColors.orange : AppColors.green,
              size: 28,
            ),
            const SizedBox(width: 10),
            Expanded(
              child: Text(
                s.t(subtitleKey) == subtitleKey
                    ? license.name
                    : '${license.name}, ${s.t(subtitleKey).toLowerCase()}',
                style: const TextStyle(
                  fontSize: 19,
                  fontWeight: FontWeight.w800,
                  color: Ink2.strong,
                ),
              ),
            ),
          ],
        ),
        const SizedBox(height: 12),
        if (license.mistakes > 0) ...[
          Material(
            color: AppColors.red.withValues(alpha: 0.09),
            borderRadius: BorderRadius.circular(20),
            child: InkWell(
              borderRadius: BorderRadius.circular(20),
              onTap: onMistakes,
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Row(
                  children: [
                    const Icon(
                      Icons.replay_circle_filled_rounded,
                      color: AppColors.red,
                      size: 32,
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: Text(
                        '${s.t('review_mistakes')} (${license.mistakes})',
                        style: const TextStyle(
                          fontSize: 18,
                          fontWeight: FontWeight.w700,
                          color: Ink2.strong,
                        ),
                      ),
                    ),
                    const Icon(
                      Icons.chevron_right_rounded,
                      color: AppColors.red,
                    ),
                  ],
                ),
              ),
            ),
          ),
          const SizedBox(height: 12),
        ],
        GridView.count(
          crossAxisCount: 2,
          shrinkWrap: true,
          physics: const NeverScrollableScrollPhysics(),
          mainAxisSpacing: 12,
          crossAxisSpacing: 12,
          childAspectRatio: 1.5,
          children: [
            for (final series in license.series)
              _SeriesTile(
                series: series,
                strings: s,
                onTap: () => onSeries(series),
              ),
          ],
        ),
      ],
    );
  }
}

class _SeriesTile extends StatelessWidget {
  const _SeriesTile({
    required this.series,
    required this.strings,
    required this.onTap,
  });

  final PracticeSeries series;
  final AppStrings strings;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final s = strings;
    final progress = series.questionCount == 0
        ? 0.0
        : series.mastered / series.questionCount;
    return Semantics(
      button: true,
      label:
          '${s.t('series_n')} ${series.number}. ${series.mastered} ${s.t('out_of')} ${series.questionCount} ${s.t('mastered')}',
      excludeSemantics: true,
      child: SoftCard(
        onTap: onTap,
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              '${s.t('series_n')} ${series.number}',
              style: const TextStyle(
                fontSize: 20,
                fontWeight: FontWeight.w800,
                color: Ink2.strong,
              ),
            ),
            const Spacer(),
            FittedBox(
              fit: BoxFit.scaleDown,
              alignment: Alignment.centerLeft,
              child: Text(
                '${series.mastered} / ${series.questionCount} ${s.t('mastered')}',
                maxLines: 1,
                softWrap: false,
                style: const TextStyle(fontSize: 14, color: Ink2.muted),
              ),
            ),
            const SizedBox(height: 6),
            ClipRRect(
              borderRadius: BorderRadius.circular(4),
              child: LinearProgressIndicator(
                value: progress,
                minHeight: 6,
                backgroundColor: AppColors.background,
                color: AppColors.green,
              ),
            ),
          ],
        ),
      ),
    );
  }
}
