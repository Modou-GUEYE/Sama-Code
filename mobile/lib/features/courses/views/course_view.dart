import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../../core/api/api_exception.dart';
import '../../../core/session_viewmodel.dart';
import '../../../core/theme/app_theme.dart';
import '../../../data/models/course.dart';
import '../../../data/repositories/course_repository.dart';
import '../../../shared/widgets/auth_scaffold.dart';
import '../../../shared/widgets/feedback.dart';
import '../../../shared/widgets/listen_button.dart';
import '../../../shared/widgets/surfaces.dart';
import 'lesson_view.dart';

/// Un cours : ses leçons dans l'ordre, celles terminées cochées.
class CourseView extends StatefulWidget {
  const CourseView({super.key, required this.courseId});

  final String courseId;

  @override
  State<CourseView> createState() => _CourseViewState();
}

class _CourseViewState extends State<CourseView> {
  CourseSummary? _course;
  ApiException? _error;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => _error = null);
    try {
      final c = await context.read<CourseRepository>().course(widget.courseId);
      if (mounted) setState(() => _course = c);
    } on ApiException catch (e) {
      if (mounted) setState(() => _error = e);
    }
  }

  @override
  Widget build(BuildContext context) {
    final session = context.watch<SessionViewModel>();
    final s = session.strings;
    final course = _course;
    final label = course == null
        ? null
        : pickLabel(course.labels, session.language);

    return AuthScaffold(
      icon: Icons.menu_book_rounded,
      title: label?.title ?? s.t('courses_title'),
      subtitle: course == null
          ? null
          : '${course.completedLessons} / ${course.lessonCount} ${s.t('lessons')} ${s.t('completed')}',
      children: [
        if (label?.audioUrl != null) ...[
          Align(
            alignment: Alignment.centerLeft,
            child: ListenButton(
              url: label!.audioUrl!,
              label: s.t('listen_title'),
            ),
          ),
          const SizedBox(height: 16),
        ],
        if (_error != null) ...[
          ErrorMessage(s.error(_error!.code, _error!.message)),
          const SizedBox(height: 12),
          SecondaryButton(
            label: s.t('retry'),
            icon: Icons.refresh_rounded,
            onPressed: _load,
          ),
        ] else if (course == null)
          const Padding(
            padding: EdgeInsets.all(32),
            child: Center(child: CircularProgressIndicator()),
          )
        else
          for (final (i, lesson) in course.lessons.indexed) ...[
            _LessonTile(
              number: i + 1,
              title: pickLabel(lesson.labels, session.language).title,
              completed: lesson.completed,
              onTap: () async {
                await Navigator.of(context).push(
                  MaterialPageRoute(
                    builder: (_) => LessonView(lessonId: lesson.id),
                  ),
                );
                await _load();
              },
            ),
            const SizedBox(height: 10),
          ],
      ],
    );
  }
}

class _LessonTile extends StatelessWidget {
  const _LessonTile({
    required this.number,
    required this.title,
    required this.completed,
    required this.onTap,
  });

  final int number;
  final String title;
  final bool completed;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      button: true,
      label: '$number. $title${completed ? ' ✓' : ''}',
      excludeSemantics: true,
      child: SoftCard(
        onTap: onTap,
        padding: const EdgeInsets.all(14),
        child: Row(
          children: [
            Container(
              width: 46,
              height: 46,
              alignment: Alignment.center,
              decoration: BoxDecoration(
                color: completed ? AppColors.green : AppColors.background,
                borderRadius: BorderRadius.circular(14),
              ),
              child: completed
                  ? const Icon(
                      Icons.check_rounded,
                      color: Colors.white,
                      size: 26,
                    )
                  : Text(
                      '$number',
                      style: const TextStyle(
                        fontSize: 19,
                        fontWeight: FontWeight.w800,
                        color: Ink2.strong,
                      ),
                    ),
            ),
            const SizedBox(width: 14),
            Expanded(
              child: Text(
                title,
                style: const TextStyle(
                  fontSize: 18,
                  fontWeight: FontWeight.w700,
                  color: Ink2.strong,
                ),
              ),
            ),
            const Icon(
              Icons.chevron_right_rounded,
              size: 28,
              color: Ink2.muted,
            ),
          ],
        ),
      ),
    );
  }
}
