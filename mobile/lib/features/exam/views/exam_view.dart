import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';

import '../../../core/config.dart';
import '../../../core/localization/app_strings.dart';
import '../../../core/session_viewmodel.dart';
import '../../../core/theme/app_theme.dart';
import '../../../data/models/exam.dart';
import '../../../data/repositories/exam_repository.dart';
import '../../../shared/widgets/app_sheet.dart';
import '../../../shared/widgets/auth_scaffold.dart';
import '../../../shared/widgets/feedback.dart';
import '../../../shared/widgets/listen_button.dart';
import '../../../shared/widgets/loading_overlay.dart';
import '../../../shared/widgets/surfaces.dart';
import '../viewmodels/exam_viewmodel.dart';
import 'result_view.dart';

/// Examen blanc : une question à la fois, chronomètre, enregistrement de chaque réponse.
class ExamView extends StatelessWidget {
  const ExamView({super.key, required this.session, this.onRetry});

  final ExamSession session;
  final VoidCallback? onRetry;

  @override
  Widget build(BuildContext context) {
    return ChangeNotifierProvider(
      create: (c) => ExamViewModel(c.read<ExamRepository>(), session),
      child: _ExamBody(onRetry: onRetry),
    );
  }
}

class _ExamBody extends StatefulWidget {
  const _ExamBody({this.onRetry});

  final VoidCallback? onRetry;

  @override
  State<_ExamBody> createState() => _ExamBodyState();
}

class _ExamBodyState extends State<_ExamBody> {
  bool _navigated = false;

  @override
  void initState() {
    super.initState();
    final vm = context.read<ExamViewModel>();
    vm.onTimeAlert = (minutes) {
      HapticFeedback.heavyImpact();
      final s = context.read<SessionViewModel>().strings;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          backgroundColor: minutes <= 1 ? AppColors.red : AppColors.orange,
          content: Row(
            children: [
              const Icon(Icons.timer_outlined, color: Colors.white),
              const SizedBox(width: 12),
              Text(
                '$minutes ${s.t('minutes_left')}',
                style: const TextStyle(
                  fontSize: 18,
                  fontWeight: FontWeight.w700,
                ),
              ),
            ],
          ),
        ),
      );
    };
    vm.addListener(_onChange);
  }

  void _onChange() {
    final finished = context.read<ExamViewModel>().finished;
    if (finished == null || _navigated) return;
    _navigated = true;
    Navigator.of(context).pushReplacement(
      MaterialPageRoute(
        builder: (_) => ResultView(session: finished, onRetry: widget.onRetry),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final vm = context.watch<ExamViewModel>();
    final session = context.watch<SessionViewModel>();
    final s = session.strings;
    final q = vm.current;
    final content = pick(q.contents, session.language);

    return PopScope(
      canPop: false,
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop) _confirmQuit(context, s);
      },
      child: Scaffold(
        body: LoadingOverlay(
          loading: vm.submitting,
          message: s.t('sending_answers'),
          child: GradientBackground(
            child: SafeArea(
              child: Column(
                children: [
                  _TopBar(
                    vm: vm,
                    strings: s,
                    onClose: () => _confirmQuit(context, s),
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
                        ),
                        const SizedBox(height: 10),
                        for (final o in q.options) ...[
                          _OptionTile(
                            option: o,
                            content: pick(o.contents, session.language),
                            selected: vm.isSelected(o),
                            multiple: q.multiple,
                            onTap: () {
                              HapticFeedback.selectionClick();
                              vm.toggle(o);
                            },
                          ),
                          const SizedBox(height: 10),
                        ],
                        if (vm.hasPendingAnswers && vm.finished == null) ...[
                          const SizedBox(height: 4),
                          InfoMessage(
                            s.t('saving_failed'),
                            icon: Icons.wifi_off_rounded,
                          ),
                        ],
                      ],
                    ),
                  ),
                  _BottomBar(vm: vm, strings: s),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

class _TopBar extends StatelessWidget {
  const _TopBar({
    required this.vm,
    required this.strings,
    required this.onClose,
  });

  final ExamViewModel vm;
  final AppStrings strings;
  final VoidCallback onClose;

  @override
  Widget build(BuildContext context) {
    final total = vm.questions.length;
    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 4, 20, 8),
      child: Column(
        children: [
          Row(
            children: [
              IconButton(
                iconSize: 30,
                tooltip: strings.t('quit'),
                icon: const Icon(Icons.close_rounded, color: Ink2.strong),
                onPressed: onClose,
              ),
              Expanded(
                child: Text(
                  '${strings.t('question')} ${vm.index + 1} / $total',
                  style: const TextStyle(
                    fontSize: 18,
                    fontWeight: FontWeight.w800,
                    color: Ink2.strong,
                  ),
                ),
              ),
              _TimerPill(remaining: vm.remaining, strings: strings),
            ],
          ),
          const SizedBox(height: 6),
          Padding(
            padding: const EdgeInsets.only(left: 8),
            child: ClipRRect(
              borderRadius: BorderRadius.circular(4),
              child: LinearProgressIndicator(
                value: (vm.index + 1) / total,
                minHeight: 6,
                backgroundColor: const Color(0xFFDCE1D9),
                color: AppColors.green,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _TimerPill extends StatelessWidget {
  const _TimerPill({required this.remaining, required this.strings});

  final Duration remaining;
  final AppStrings strings;

  @override
  Widget build(BuildContext context) {
    final secs = remaining.inSeconds;
    final color = secs <= 60
        ? AppColors.red
        : secs <= 300
        ? const Color(0xFFB45F00)
        : AppColors.greenDark;
    final text =
        '${(secs ~/ 60).toString().padLeft(2, '0')}:${(secs % 60).toString().padLeft(2, '0')}';
    return Semantics(
      label: '${(secs / 60).ceil()} ${strings.t('minutes_left')}',
      excludeSemantics: true,
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 300),
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
        decoration: BoxDecoration(
          color: color.withValues(alpha: 0.12),
          borderRadius: BorderRadius.circular(999),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.timer_outlined, color: color, size: 22),
            const SizedBox(width: 6),
            Text(
              text,
              style: TextStyle(
                fontSize: 20,
                fontWeight: FontWeight.w800,
                color: color,
                fontFeatures: const [FontFeature.tabularFigures()],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _OptionTile extends StatelessWidget {
  const _OptionTile({
    required this.option,
    required this.content,
    required this.selected,
    required this.multiple,
    required this.onTap,
  });

  final ExamOption option;
  final LocalizedContent content;
  final bool selected;
  final bool multiple;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final mark = multiple
        ? (selected
              ? Icons.check_box_rounded
              : Icons.check_box_outline_blank_rounded)
        : (selected ? Icons.check_circle_rounded : Icons.circle_outlined);
    return Semantics(
      button: true,
      selected: selected,
      label: '${option.code}. ${content.text}',
      excludeSemantics: true,
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 150),
        decoration: BoxDecoration(
          color: selected ? const Color(0xFFE3F2E9) : Colors.white,
          borderRadius: BorderRadius.circular(20),
          boxShadow: selected
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
                      color: selected ? AppColors.green : AppColors.background,
                      borderRadius: BorderRadius.circular(14),
                    ),
                    child: Text(
                      option.code,
                      style: TextStyle(
                        fontSize: 20,
                        fontWeight: FontWeight.w800,
                        color: selected ? Colors.white : Ink2.strong,
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
                            fontWeight: selected
                                ? FontWeight.w700
                                : FontWeight.w500,
                            color: Ink2.strong,
                          ),
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(width: 8),
                  Icon(
                    mark,
                    size: 30,
                    color: selected ? AppColors.green : const Color(0xFFC9CEC6),
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

class _BottomBar extends StatelessWidget {
  const _BottomBar({required this.vm, required this.strings});

  final ExamViewModel vm;
  final AppStrings strings;

  @override
  Widget build(BuildContext context) {
    final s = strings;
    return Padding(
      padding: const EdgeInsets.fromLTRB(20, 8, 20, 16),
      child: Row(
        children: [
          Expanded(
            child: SecondaryButton(
              label: s.t('previous'),
              icon: Icons.arrow_back_rounded,
              onPressed: vm.index == 0 ? null : () => vm.goTo(vm.index - 1),
            ),
          ),
          const SizedBox(width: 10),
          SizedBox(
            width: 60,
            height: 60,
            child: Material(
              color: Colors.white,
              shape: const CircleBorder(),
              child: IconButton(
                tooltip: s.t('questions_grid'),
                icon: const Icon(
                  Icons.grid_view_rounded,
                  color: AppColors.greenDark,
                ),
                onPressed: () => _openGrid(context, vm, s),
              ),
            ),
          ),
          const SizedBox(width: 10),
          Expanded(
            child: vm.isLast
                ? PrimaryButton(
                    label: s.t('finish'),
                    icon: Icons.flag_rounded,
                    busy: vm.submitting,
                    onPressed: () => _confirmFinish(context, vm, s),
                  )
                : PrimaryButton(
                    label: s.t('next'),
                    icon: Icons.arrow_forward_rounded,
                    onPressed: () => vm.goTo(vm.index + 1),
                  ),
          ),
        ],
      ),
    );
  }
}

Future<void> _openGrid(BuildContext context, ExamViewModel vm, AppStrings s) {
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
              s.t('questions_grid'),
              style: const TextStyle(
                fontSize: 21,
                fontWeight: FontWeight.w800,
                color: Ink2.strong,
              ),
            ),
            Text(
              '${vm.answeredCount} / ${vm.questions.length} ${s.t('answered')}',
              style: const TextStyle(fontSize: 15, color: Ink2.muted),
            ),
            const SizedBox(height: 16),
            // Cases de 52 px minimum, élargies pour remplir exactement la largeur.
            LayoutBuilder(
              builder: (context, box) {
                const minCell = 52.0, gap = 10.0;
                final columns = ((box.maxWidth + gap) / (minCell + gap))
                    .floor()
                    .clamp(1, 10);
                final cell = (box.maxWidth - gap * (columns - 1)) / columns;
                return Wrap(
                  spacing: gap,
                  runSpacing: gap,
                  children: [
                    for (var i = 0; i < vm.questions.length; i++)
                      _GridCell(
                        number: i + 1,
                        size: cell,
                        answered: vm.isAnswered(vm.questions[i]),
                        current: i == vm.index,
                        onTap: () {
                          vm.goTo(i);
                          Navigator.of(sheet).pop();
                        },
                      ),
                  ],
                );
              },
            ),
          ],
        ),
      ),
    ),
  );
}

class _GridCell extends StatelessWidget {
  const _GridCell({
    required this.number,
    required this.size,
    required this.answered,
    required this.current,
    required this.onTap,
  });

  final int number;
  final double size;
  final bool answered;
  final bool current;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final bg = current
        ? AppColors.greenDark
        : answered
        ? const Color(0xFFE3F2E9)
        : AppColors.background;
    return Material(
      color: bg,
      borderRadius: BorderRadius.circular(14),
      child: InkWell(
        borderRadius: BorderRadius.circular(14),
        onTap: onTap,
        child: SizedBox(
          width: size,
          height: 52,
          child: Stack(
            alignment: Alignment.center,
            children: [
              Text(
                '$number',
                style: TextStyle(
                  fontSize: 18,
                  fontWeight: FontWeight.w800,
                  color: current ? Colors.white : Ink2.strong,
                ),
              ),
              if (answered && !current)
                const Positioned(
                  right: 4,
                  top: 4,
                  child: Icon(
                    Icons.check_rounded,
                    size: 14,
                    color: AppColors.green,
                  ),
                ),
            ],
          ),
        ),
      ),
    );
  }
}

Future<void> _confirmFinish(
  BuildContext context,
  ExamViewModel vm,
  AppStrings s,
) async {
  final unanswered = vm.unansweredCount;
  final ok = await showAppSheet<bool>(
    context: context,
    builder: (sheet) => SafeArea(
      child: Padding(
        padding: const EdgeInsets.fromLTRB(24, 0, 24, 24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            const Icon(Icons.flag_rounded, size: 44, color: AppColors.green),
            const SizedBox(height: 12),
            Text(
              s.t('finish_confirm'),
              textAlign: TextAlign.center,
              style: const TextStyle(
                fontSize: 22,
                fontWeight: FontWeight.w800,
                color: Ink2.strong,
              ),
            ),
            const SizedBox(height: 8),
            Text(
              unanswered == 0
                  ? s.t('all_answered')
                  : '$unanswered ${s.t('unanswered_left')}',
              textAlign: TextAlign.center,
              style: TextStyle(
                fontSize: 17,
                color: unanswered == 0 ? Ink2.muted : AppColors.red,
              ),
            ),
            const SizedBox(height: 24),
            Row(
              children: [
                Expanded(
                  child: SecondaryButton(
                    label: s.t('cancel'),
                    tinted: true,
                    onPressed: () => Navigator.of(sheet).pop(false),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: PrimaryButton(
                    label: s.t('finish'),
                    icon: Icons.flag_rounded,
                    onPressed: () => Navigator.of(sheet).pop(true),
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    ),
  );
  if (ok == true) await vm.submit();
}

Future<void> _confirmQuit(BuildContext context, AppStrings s) async {
  final quit = await showAppSheet<bool>(
    context: context,
    builder: (sheet) => SafeArea(
      child: Padding(
        padding: const EdgeInsets.fromLTRB(24, 0, 24, 24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            const Icon(Icons.timer_outlined, size: 44, color: AppColors.orange),
            const SizedBox(height: 12),
            Text(
              s.t('quit_exam_title'),
              textAlign: TextAlign.center,
              style: const TextStyle(
                fontSize: 22,
                fontWeight: FontWeight.w800,
                color: Ink2.strong,
              ),
            ),
            const SizedBox(height: 8),
            Text(
              s.t('quit_exam_body'),
              textAlign: TextAlign.center,
              style: const TextStyle(fontSize: 17, color: Ink2.muted),
            ),
            const SizedBox(height: 24),
            Row(
              children: [
                Expanded(
                  child: SecondaryButton(
                    label: s.t('cancel'),
                    tinted: true,
                    onPressed: () => Navigator.of(sheet).pop(false),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: PrimaryButton(
                    label: s.t('quit'),
                    icon: Icons.logout_rounded,
                    onPressed: () => Navigator.of(sheet).pop(true),
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    ),
  );
  if (quit == true && context.mounted) Navigator.of(context).pop();
}
