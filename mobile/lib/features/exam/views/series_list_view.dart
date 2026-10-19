import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../../core/api/api_exception.dart';
import '../../../core/localization/app_strings.dart';
import '../../../core/session_viewmodel.dart';
import '../../../core/theme/app_theme.dart';
import '../../../data/models/exam.dart';
import '../../../data/models/license_category.dart';
import '../../../data/repositories/exam_repository.dart';
import '../../../shared/widgets/app_sheet.dart';
import '../../../shared/widgets/auth_scaffold.dart';
import '../../../shared/widgets/feedback.dart';
import '../../../shared/widgets/surfaces.dart';
import '../viewmodels/series_list_viewmodel.dart';
import 'exam_view.dart';

/// Choix de la série par le candidat (CDC §5.2) : séries publiées du permis.
class SeriesListView extends StatelessWidget {
  const SeriesListView({super.key, required this.license});

  final LicenseCategory license;

  @override
  Widget build(BuildContext context) {
    return ChangeNotifierProvider(
      create: (c) =>
          SeriesListViewModel(c.read<ExamRepository>(), license)..load(),
      child: const _SeriesListBody(),
    );
  }
}

class _SeriesListBody extends StatelessWidget {
  const _SeriesListBody();

  @override
  Widget build(BuildContext context) {
    final vm = context.watch<SeriesListViewModel>();
    final s = context.watch<SessionViewModel>().strings;
    final isTruck = vm.license.code == 'C';
    final subtitleKey = 'license_${vm.license.code}_subtitle';

    return AuthScaffold(
      busy: vm.starting,
      busyMessage: s.t('please_wait'),
      icon: isTruck
          ? Icons.local_shipping_rounded
          : Icons.directions_car_rounded,
      iconColor: isTruck ? AppColors.orange : AppColors.green,
      title: vm.license.name,
      subtitle: s.t(subtitleKey) == subtitleKey
          ? s.t('exam_choose_series')
          : '${s.t(subtitleKey)}. ${s.t('exam_choose_series')}',
      children: [
        if (vm.loading && vm.series.isEmpty)
          const Padding(
            padding: EdgeInsets.all(32),
            child: Center(child: CircularProgressIndicator()),
          )
        else if (vm.error != null) ...[
          ErrorMessage(s.error(vm.error!.code, vm.error!.message)),
          const SizedBox(height: 12),
          SecondaryButton(
            label: s.t('retry'),
            icon: Icons.refresh_rounded,
            onPressed: vm.load,
          ),
        ] else if (vm.series.isEmpty)
          InfoMessage(s.t('no_series'), icon: Icons.hourglass_empty_rounded)
        else
          GridView.count(
            crossAxisCount: 2,
            shrinkWrap: true,
            physics: const NeverScrollableScrollPhysics(),
            mainAxisSpacing: 14,
            crossAxisSpacing: 14,
            childAspectRatio: 1.05,
            children: [
              for (final series in vm.series)
                _SeriesTile(
                  series: series,
                  strings: s,
                  onTap: () => _openStartSheet(context, vm, series, s),
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

  final SeriesSummary series;
  final AppStrings strings;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final s = strings;
    final (label, icon, color) = switch (series.lastPassed) {
      true => (s.t('passed'), Icons.check_circle_rounded, AppColors.green),
      false => (s.t('failed'), Icons.cancel_rounded, AppColors.red),
      null => (
        s.t('not_taken'),
        Icons.radio_button_unchecked_rounded,
        Ink2.muted,
      ),
    };
    return Semantics(
      button: true,
      label: '${s.t('series_n')} ${series.number}. $label',
      excludeSemantics: true,
      child: SoftCard(
        onTap: onTap,
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              s.t('series_n'),
              style: const TextStyle(fontSize: 15, color: Ink2.muted),
            ),
            Text(
              '${series.number}',
              style: const TextStyle(
                fontSize: 40,
                fontWeight: FontWeight.w800,
                color: Ink2.strong,
                height: 1.1,
              ),
            ),
            const Spacer(),
            Row(
              children: [
                Icon(icon, size: 20, color: color),
                const SizedBox(width: 6),
                Flexible(
                  child: Text(
                    label,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(
                      fontSize: 15,
                      fontWeight: FontWeight.w700,
                      color: color,
                    ),
                  ),
                ),
              ],
            ),
            if (series.bestFaults != null)
              // Une seule ligne : le texte rétrécit s'il manque de place.
              FittedBox(
                fit: BoxFit.scaleDown,
                alignment: Alignment.centerLeft,
                child: Text(
                  '${s.t('best_result')} : ${series.bestFaults} ${series.bestFaults! > 1 ? s.t('faults') : s.t('fault')}',
                  maxLines: 1,
                  softWrap: false,
                  style: const TextStyle(fontSize: 13, color: Ink2.muted),
                ),
              ),
          ],
        ),
      ),
    );
  }
}

Future<void> _openStartSheet(
  BuildContext context,
  SeriesListViewModel vm,
  SeriesSummary series,
  AppStrings s,
) {
  return showAppSheet<void>(
    context: context,
    builder: (sheet) => SafeArea(
      child: Padding(
        padding: const EdgeInsets.fromLTRB(24, 0, 24, 24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(
              s.series(vm.license.name, series.number),
              style: const TextStyle(
                fontSize: 22,
                fontWeight: FontWeight.w800,
                color: Ink2.strong,
              ),
            ),
            const SizedBox(height: 16),
            Row(
              children: [
                _Info(
                  icon: Icons.help_outline_rounded,
                  value: '${series.questionCount}',
                  label: s.t('questions'),
                ),
                const SizedBox(width: 10),
                _Info(
                  icon: Icons.timer_outlined,
                  value: '${series.durationMinutes}',
                  label: s.t('minutes'),
                ),
                const SizedBox(width: 10),
                _Info(
                  icon: Icons.error_outline_rounded,
                  value: '${series.maxFaults}',
                  label: s.t('faults_max'),
                ),
              ],
            ),
            const SizedBox(height: 24),
            PrimaryButton(
              label: s.t('start_exam'),
              icon: Icons.play_arrow_rounded,
              onPressed: () {
                Navigator.of(sheet).pop();
                _start(context, vm, series, s);
              },
            ),
          ],
        ),
      ),
    ),
  );
}

Future<void> _start(
  BuildContext context,
  SeriesListViewModel vm,
  SeriesSummary series,
  AppStrings s, {
  bool abandon = false,
}) async {
  final navigator = Navigator.of(context);
  try {
    final session = await vm.start(series, abandonCurrent: abandon);
    await navigator.push(
      MaterialPageRoute(
        builder: (_) => ExamView(
          session: session,
          onRetry: () => _start(context, vm, series, s),
        ),
      ),
    );
  } on ApiException catch (e) {
    if (!context.mounted) return;
    if (e.code == 'session_in_progress') {
      final choice = await _askResume(context, s);
      if (!context.mounted || choice == null) return;
      if (choice) {
        final current = await vm.resume(e.details['session_id'] as String);
        await navigator.push(
          MaterialPageRoute(builder: (_) => ExamView(session: current)),
        );
      } else {
        await _start(context, vm, series, s, abandon: true);
        return;
      }
    } else {
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(s.error(e.code, e.message))));
    }
  }
  await vm.load();
}

/// true = reprendre l'examen en cours ; false = l'abandonner et commencer celui-ci.
Future<bool?> _askResume(BuildContext context, AppStrings s) {
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
              Icons.pending_actions_rounded,
              size: 44,
              color: AppColors.orange,
            ),
            const SizedBox(height: 12),
            Text(
              s.t('exam_in_progress'),
              textAlign: TextAlign.center,
              style: const TextStyle(
                fontSize: 21,
                fontWeight: FontWeight.w800,
                color: Ink2.strong,
              ),
            ),
            const SizedBox(height: 24),
            PrimaryButton(
              label: s.t('resume'),
              icon: Icons.play_arrow_rounded,
              onPressed: () => Navigator.of(sheet).pop(true),
            ),
            const SizedBox(height: 10),
            SecondaryButton(
              label: s.t('abandon_and_start'),
              tinted: true,
              onPressed: () => Navigator.of(sheet).pop(false),
            ),
          ],
        ),
      ),
    ),
  );
}

class _Info extends StatelessWidget {
  const _Info({required this.icon, required this.value, required this.label});

  final IconData icon;
  final String value;
  final String label;

  @override
  Widget build(BuildContext context) {
    return Expanded(
      child: Container(
        padding: const EdgeInsets.symmetric(vertical: 12, horizontal: 8),
        decoration: BoxDecoration(
          color: AppColors.background,
          borderRadius: BorderRadius.circular(16),
        ),
        child: Column(
          children: [
            Icon(icon, color: AppColors.green),
            Text(
              value,
              style: const TextStyle(
                fontSize: 22,
                fontWeight: FontWeight.w800,
                color: Ink2.strong,
              ),
            ),
            Text(
              label,
              textAlign: TextAlign.center,
              style: const TextStyle(fontSize: 13, color: Ink2.muted),
            ),
          ],
        ),
      ),
    );
  }
}
