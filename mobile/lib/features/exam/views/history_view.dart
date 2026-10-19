import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../../core/api/api_exception.dart';
import '../../../core/localization/app_strings.dart';
import '../../../core/session_viewmodel.dart';
import '../../../core/theme/app_theme.dart';
import '../../../data/models/exam.dart';
import '../../../data/repositories/exam_repository.dart';
import '../../../shared/widgets/auth_scaffold.dart';
import '../../../shared/widgets/feedback.dart';
import '../../../shared/widgets/surfaces.dart';
import 'exam_view.dart';
import 'result_view.dart';

/// « Mes examens » : toutes les tentatives, la plus récente d'abord. Une tentative
/// soumise ouvre son résultat (et la correction) ; une tentative en cours la reprend.
class HistoryView extends StatefulWidget {
  const HistoryView({super.key});

  static const pageSize = 20;

  @override
  State<HistoryView> createState() => _HistoryViewState();
}

class _HistoryViewState extends State<HistoryView> {
  final List<ExamHistoryItem> _items = [];
  int _page = 0;
  bool _hasMore = false;
  bool _loading = true;
  bool _opening = false;
  ApiException? _error;

  @override
  void initState() {
    super.initState();
    _reload();
  }

  Future<void> _reload() async {
    _items.clear();
    _page = 0;
    await _loadMore();
  }

  Future<void> _loadMore() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final next = await context.read<ExamRepository>().history(
        page: _page + 1,
        pageSize: HistoryView.pageSize,
      );
      if (!mounted) return;
      setState(() {
        _page++;
        _items.addAll(next);
        _hasMore = next.length == HistoryView.pageSize;
      });
    } on ApiException catch (e) {
      if (mounted) setState(() => _error = e);
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _open(ExamHistoryItem item) async {
    final repo = context.read<ExamRepository>();
    final navigator = Navigator.of(context);
    setState(() => _opening = true);
    ExamSession session;
    try {
      session = await repo.get(item.id);
    } on ApiException catch (e) {
      if (mounted) setState(() => _error = e);
      return;
    } finally {
      if (mounted) setState(() => _opening = false);
    }
    // Un examen en cours a pu expirer entre-temps : le serveur l'a alors soumis.
    await navigator.push(
      MaterialPageRoute(
        builder: (_) => session.submitted
            ? ResultView(session: session)
            : ExamView(session: session),
      ),
    );
    if (mounted) await _reload();
  }

  @override
  Widget build(BuildContext context) {
    final s = context.watch<SessionViewModel>().strings;

    return AuthScaffold(
      icon: Icons.history_rounded,
      title: s.t('my_exams'),
      subtitle: s.t('my_exams_subtitle'),
      busy: _opening,
      busyMessage: s.t('please_wait'),
      children: [
        if (_error != null) ...[
          ErrorMessage(s.error(_error!.code, _error!.message)),
          const SizedBox(height: 12),
          SecondaryButton(
            label: s.t('retry'),
            icon: Icons.refresh_rounded,
            onPressed: _items.isEmpty ? _reload : _loadMore,
          ),
          const SizedBox(height: 16),
        ],
        if (_items.isEmpty && _loading)
          const Padding(
            padding: EdgeInsets.all(32),
            child: Center(child: CircularProgressIndicator()),
          )
        else if (_items.isEmpty && _error == null)
          InfoMessage(s.t('no_exams'), icon: Icons.hourglass_empty_rounded)
        else ...[
          for (final item in _items) ...[
            _HistoryCard(item: item, strings: s, onTap: () => _open(item)),
            const SizedBox(height: 12),
          ],
          if (_hasMore)
            _loading
                ? const Padding(
                    padding: EdgeInsets.all(16),
                    child: Center(child: CircularProgressIndicator()),
                  )
                : SecondaryButton(
                    label: s.t('load_more'),
                    icon: Icons.expand_more_rounded,
                    onPressed: _loadMore,
                  ),
        ],
      ],
    );
  }
}

class _HistoryCard extends StatelessWidget {
  const _HistoryCard({
    required this.item,
    required this.strings,
    required this.onTap,
  });

  final ExamHistoryItem item;
  final AppStrings strings;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final s = strings;
    final r = item.result;
    final (IconData icon, Color color, String status) = r == null
        ? (Icons.timer_outlined, AppColors.orange, s.t('in_progress'))
        : r.passed
        ? (Icons.emoji_events_rounded, AppColors.green, s.t('passed'))
        : (Icons.replay_rounded, AppColors.red, s.t('failed'));
    final faults = r == null
        ? null
        : '${r.faults} ${r.faults > 1 ? s.t('faults') : s.t('fault')}';
    final title =
        'Permis ${item.licenseCode} · ${s.t('series_n')} ${item.seriesNumber}';
    final date = _formatDate(item.startedAt.toLocal(), s);

    return Semantics(
      button: true,
      label: [title, status, ?faults, date].join('. '),
      excludeSemantics: true,
      child: SoftCard(
        onTap: onTap,
        padding: const EdgeInsets.all(16),
        child: Row(
          children: [
            Container(
              width: 56,
              height: 56,
              decoration: BoxDecoration(
                color: color.withValues(alpha: 0.14),
                borderRadius: BorderRadius.circular(18),
              ),
              child: Icon(icon, size: 30, color: color),
            ),
            const SizedBox(width: 14),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    title,
                    style: const TextStyle(
                      fontSize: 18,
                      fontWeight: FontWeight.w800,
                      color: Ink2.strong,
                    ),
                  ),
                  const SizedBox(height: 2),
                  Text(
                    faults == null ? status : '$status · $faults',
                    style: TextStyle(
                      fontSize: 16,
                      fontWeight: FontWeight.w700,
                      color: color,
                    ),
                  ),
                  const SizedBox(height: 2),
                  Text(
                    date,
                    style: const TextStyle(fontSize: 14, color: Ink2.muted),
                  ),
                ],
              ),
            ),
            const SizedBox(width: 8),
            const Icon(
              Icons.chevron_right_rounded,
              size: 30,
              color: Ink2.muted,
            ),
          ],
        ),
      ),
    );
  }
}

/// « Aujourd'hui 14:05 », « Hier 09:30 », sinon « 18/10/2026 14:05 ».
String _formatDate(DateTime d, AppStrings s) {
  String two(int n) => n.toString().padLeft(2, '0');
  final time = '${two(d.hour)}:${two(d.minute)}';
  final now = DateTime.now();
  final days = DateTime(
    now.year,
    now.month,
    now.day,
  ).difference(DateTime(d.year, d.month, d.day)).inDays;
  final day = switch (days) {
    0 => s.t('today'),
    1 => s.t('yesterday'),
    _ => '${two(d.day)}/${two(d.month)}/${d.year}',
  };
  return '$day $time';
}
