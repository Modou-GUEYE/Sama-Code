import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../../core/api/api_exception.dart';
import '../../../core/config.dart';
import '../../../core/session_viewmodel.dart';
import '../../../core/theme/app_theme.dart';
import '../../../data/models/course.dart';
import '../../../data/repositories/course_repository.dart';
import '../../../shared/widgets/auth_scaffold.dart';
import '../../../shared/widgets/feedback.dart';
import '../../../shared/widgets/listen_button.dart';
import '../../../shared/widgets/loading_overlay.dart';
import '../../../shared/widgets/surfaces.dart';

/// Lecture d'une leçon : titre (avec audio), blocs texte / image / audio.
/// Le bouton du bas marque la leçon comme terminée puis enchaîne sur la suivante.
class LessonView extends StatefulWidget {
  const LessonView({super.key, required this.lessonId});

  final String lessonId;

  @override
  State<LessonView> createState() => _LessonViewState();
}

class _LessonViewState extends State<LessonView> {
  Lesson? _lesson;
  ApiException? _error;
  bool _busy = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => _error = null);
    try {
      final l = await context.read<CourseRepository>().lesson(widget.lessonId);
      if (mounted) setState(() => _lesson = l);
    } on ApiException catch (e) {
      if (mounted) setState(() => _error = e);
    }
  }

  Future<void> _finish() async {
    final lesson = _lesson!;
    setState(() => _busy = true);
    try {
      if (!lesson.completed) {
        await context.read<CourseRepository>().complete(lesson.id);
      }
      if (!mounted) return;
      final next = lesson.nextLessonId;
      if (next != null) {
        await Navigator.of(context).pushReplacement(
          MaterialPageRoute(builder: (_) => LessonView(lessonId: next)),
        );
      } else {
        Navigator.of(context).pop();
      }
    } on ApiException catch (e) {
      if (mounted) setState(() => _error = e);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final session = context.watch<SessionViewModel>();
    final s = session.strings;
    final lesson = _lesson;
    final label = lesson == null
        ? null
        : pickLabel(lesson.labels, session.language);
    // En mode audio, le premier audio de la leçon (ou celui du titre) se lance tout seul.
    var autoPlayed = false;
    bool takeAutoPlay() {
      if (autoPlayed || session.learningMode != 'audio') return false;
      autoPlayed = true;
      return true;
    }

    return Scaffold(
      body: LoadingOverlay(
        loading: _busy,
        message: s.t('please_wait'),
        child: GradientBackground(
          child: SafeArea(
            child: Column(
              children: [
                Align(
                  alignment: Alignment.centerLeft,
                  child: Padding(
                    padding: const EdgeInsets.fromLTRB(12, 4, 12, 0),
                    child: IconButton(
                      iconSize: 30,
                      tooltip: MaterialLocalizations.of(context)
                          .backButtonTooltip,
                      icon: const Icon(
                        Icons.arrow_back_rounded,
                        color: Ink2.strong,
                      ),
                      onPressed: () => Navigator.of(context).pop(),
                    ),
                  ),
                ),
                Expanded(
                  child: _error != null
                      ? ListView(
                          padding: const EdgeInsets.all(20),
                          children: [
                            ErrorMessage(
                              s.error(_error!.code, _error!.message),
                            ),
                            const SizedBox(height: 12),
                            SecondaryButton(
                              label: s.t('retry'),
                              icon: Icons.refresh_rounded,
                              onPressed: _load,
                            ),
                          ],
                        )
                      : lesson == null
                      ? const Center(child: CircularProgressIndicator())
                      : ListView(
                          padding: const EdgeInsets.fromLTRB(24, 4, 24, 24),
                          children: [
                            if (lesson.completed)
                              Padding(
                                padding: const EdgeInsets.only(bottom: 8),
                                child: Row(
                                  children: [
                                    const Icon(
                                      Icons.check_circle_rounded,
                                      color: AppColors.green,
                                      size: 22,
                                    ),
                                    const SizedBox(width: 6),
                                    Text(
                                      s.t('lesson_completed'),
                                      style: const TextStyle(
                                        fontSize: 15,
                                        fontWeight: FontWeight.w700,
                                        color: AppColors.greenDark,
                                      ),
                                    ),
                                  ],
                                ),
                              ),
                            Text(
                              label!.title,
                              style: const TextStyle(
                                fontSize: 26,
                                fontWeight: FontWeight.w800,
                                color: Ink2.strong,
                                height: 1.2,
                              ),
                            ),
                            if (label.audioUrl != null) ...[
                              const SizedBox(height: 12),
                              Align(
                                alignment: Alignment.centerLeft,
                                child: ListenButton(
                                  url: label.audioUrl!,
                                  label: s.t('listen_title'),
                                  autoPlay: takeAutoPlay(),
                                ),
                              ),
                            ],
                            const SizedBox(height: 20),
                            for (final block in lesson.blocksFor(
                              session.language,
                            )) ...[
                              switch (block.type) {
                                'image' => ClipRRect(
                                  borderRadius: BorderRadius.circular(20),
                                  child: Image.network(
                                    AppConfig.mediaUrl(block.mediaUrl!),
                                    width: double.infinity,
                                    height: 220,
                                    fit: BoxFit.contain,
                                  ),
                                ),
                                'audio' => Align(
                                  alignment: Alignment.centerLeft,
                                  child: ListenButton(
                                    url: block.mediaUrl!,
                                    label: s.t('listen'),
                                    autoPlay: takeAutoPlay(),
                                  ),
                                ),
                                _ => Text(
                                  block.text ?? '',
                                  style: const TextStyle(
                                    fontSize: 19,
                                    color: Ink2.strong,
                                    height: 1.45,
                                  ),
                                ),
                              },
                              const SizedBox(height: 16),
                            ],
                          ],
                        ),
                ),
                if (lesson != null)
                  Padding(
                    padding: const EdgeInsets.fromLTRB(20, 8, 20, 16),
                    child: PrimaryButton(
                      label: lesson.nextLessonId != null
                          ? (lesson.completed
                                ? s.t('next_lesson')
                                : '${s.t('lesson_done')} →')
                          : (lesson.completed
                                ? s.t('back_to_course')
                                : s.t('lesson_done')),
                      icon: lesson.completed
                          ? Icons.arrow_forward_rounded
                          : Icons.check_rounded,
                      busy: _busy,
                      onPressed: _finish,
                    ),
                  ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
