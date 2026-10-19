import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../../core/api/api_exception.dart';
import '../../../core/localization/app_strings.dart';
import '../../../core/session_viewmodel.dart';
import '../../../core/theme/app_theme.dart';
import '../../../data/models/course.dart';
import '../../../data/repositories/course_repository.dart';
import '../../../shared/widgets/auth_scaffold.dart';
import '../../../shared/widgets/feedback.dart';
import '../../../shared/widgets/surfaces.dart';
import 'course_view.dart';

/// Cours publiés, regroupés par thème, avec la progression du candidat.
class CoursesHomeView extends StatefulWidget {
  const CoursesHomeView({super.key});

  @override
  State<CoursesHomeView> createState() => _CoursesHomeViewState();
}

class _CoursesHomeViewState extends State<CoursesHomeView> {
  List<CourseCategory>? _categories;
  ApiException? _error;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => _error = null);
    try {
      final c = await context.read<CourseRepository>().catalog();
      if (mounted) setState(() => _categories = c);
    } on ApiException catch (e) {
      if (mounted) setState(() => _error = e);
    }
  }

  @override
  Widget build(BuildContext context) {
    final session = context.watch<SessionViewModel>();
    final s = session.strings;
    final categories = _categories;

    return AuthScaffold(
      icon: Icons.menu_book_rounded,
      title: s.t('courses_title'),
      subtitle: s.t('courses_subtitle'),
      children: [
        if (_error != null) ...[
          ErrorMessage(s.error(_error!.code, _error!.message)),
          const SizedBox(height: 12),
          SecondaryButton(
            label: s.t('retry'),
            icon: Icons.refresh_rounded,
            onPressed: _load,
          ),
        ] else if (categories == null)
          const Padding(
            padding: EdgeInsets.all(32),
            child: Center(child: CircularProgressIndicator()),
          )
        else if (categories.isEmpty)
          InfoMessage(s.t('no_courses'), icon: Icons.hourglass_empty_rounded)
        else
          for (final category in categories) ...[
            Padding(
              padding: const EdgeInsets.only(left: 4, bottom: 10),
              child: Text(
                pickLabel(category.labels, session.language).title,
                style: const TextStyle(
                  fontSize: 20,
                  fontWeight: FontWeight.w800,
                  color: Ink2.strong,
                ),
              ),
            ),
            for (final course in category.courses) ...[
              _CourseCard(
                course: course,
                language: session.language,
                strings: s,
                onTap: () async {
                  await Navigator.of(context).push(
                    MaterialPageRoute(
                      builder: (_) => CourseView(courseId: course.id),
                    ),
                  );
                  await _load();
                },
              ),
              const SizedBox(height: 12),
            ],
            const SizedBox(height: 16),
          ],
      ],
    );
  }
}

class _CourseCard extends StatelessWidget {
  const _CourseCard({
    required this.course,
    required this.language,
    required this.strings,
    required this.onTap,
  });

  final CourseSummary course;
  final String language;
  final AppStrings strings;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final s = strings;
    final title = pickLabel(course.labels, language).title;
    final done =
        course.lessonCount > 0 && course.completedLessons == course.lessonCount;
    return Semantics(
      button: true,
      label:
          '$title. ${course.completedLessons} / ${course.lessonCount} ${s.t('lessons')} ${s.t('completed')}',
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
                color: (done ? AppColors.green : AppColors.orange).withValues(
                  alpha: 0.14,
                ),
                borderRadius: BorderRadius.circular(18),
              ),
              child: Icon(
                done ? Icons.check_circle_rounded : Icons.menu_book_rounded,
                size: 30,
                color: done ? AppColors.green : AppColors.orange,
              ),
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
                  const SizedBox(height: 4),
                  Text(
                    '${course.completedLessons} / ${course.lessonCount} ${s.t('lessons')} ${s.t('completed')}',
                    style: const TextStyle(fontSize: 14, color: Ink2.muted),
                  ),
                  const SizedBox(height: 6),
                  ClipRRect(
                    borderRadius: BorderRadius.circular(4),
                    child: LinearProgressIndicator(
                      value: course.progress,
                      minHeight: 6,
                      backgroundColor: AppColors.background,
                      color: AppColors.green,
                    ),
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
