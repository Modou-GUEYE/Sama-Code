/// Libellé d'un thème, d'un cours ou d'une leçon dans une langue.
class Label {
  const Label({required this.title, this.description, this.audioUrl});

  factory Label.fromJson(Map<String, dynamic> json) => Label(
    title: json['title'] as String? ?? '',
    description: json['description'] as String?,
    audioUrl: json['audio_url'] as String?,
  );

  final String title;
  final String? description;
  final String? audioUrl;
}

Map<String, Label> _labels(Object? json) => {
  for (final e in ((json as Map?) ?? const {}).entries)
    e.key as String: Label.fromJson(e.value as Map<String, dynamic>),
};

/// Libellé dans la langue de l'utilisateur, sinon en français.
Label pickLabel(Map<String, Label> labels, String language) =>
    labels[language] ?? labels['fr'] ?? const Label(title: '');

class CourseSummary {
  const CourseSummary({
    required this.id,
    required this.labels,
    required this.lessonCount,
    required this.completedLessons,
    this.licenseCode,
    this.lessons = const [],
  });

  factory CourseSummary.fromJson(Map<String, dynamic> json) => CourseSummary(
    id: json['id'] as String,
    licenseCode: json['license_code'] as String?,
    labels: _labels(json['labels']),
    lessonCount: json['lesson_count'] as int,
    completedLessons: json['completed_lessons'] as int,
    lessons: ((json['lessons'] as List?) ?? const [])
        .map((l) => LessonSummary.fromJson(l as Map<String, dynamic>))
        .toList(),
  );

  final String id;
  final String? licenseCode;
  final Map<String, Label> labels;
  final int lessonCount;
  final int completedLessons;
  final List<LessonSummary> lessons;

  double get progress => lessonCount == 0 ? 0 : completedLessons / lessonCount;
}

class CourseCategory {
  const CourseCategory({
    required this.id,
    required this.labels,
    required this.courses,
  });

  factory CourseCategory.fromJson(Map<String, dynamic> json) => CourseCategory(
    id: json['id'] as String,
    labels: _labels(json['labels']),
    courses: (json['courses'] as List)
        .map((c) => CourseSummary.fromJson(c as Map<String, dynamic>))
        .toList(),
  );

  final String id;
  final Map<String, Label> labels;
  final List<CourseSummary> courses;
}

class LessonSummary {
  const LessonSummary({
    required this.id,
    required this.labels,
    required this.completed,
  });

  factory LessonSummary.fromJson(Map<String, dynamic> json) => LessonSummary(
    id: json['id'] as String,
    labels: _labels(json['labels']),
    completed: json['completed'] as bool,
  );

  final String id;
  final Map<String, Label> labels;
  final bool completed;
}

/// Bloc d'une leçon : texte, image ou audio.
class LessonBlock {
  const LessonBlock({required this.type, this.text, this.mediaUrl});

  factory LessonBlock.fromJson(Map<String, dynamic> json) => LessonBlock(
    type: json['type'] as String,
    text: json['text'] as String?,
    mediaUrl: json['media_url'] as String?,
  );

  final String type; // text | image | audio
  final String? text;
  final String? mediaUrl;
}

class Lesson {
  const Lesson({
    required this.id,
    required this.courseId,
    required this.labels,
    required this.blocks,
    required this.completed,
    this.nextLessonId,
  });

  factory Lesson.fromJson(Map<String, dynamic> json) => Lesson(
    id: json['id'] as String,
    courseId: json['course_id'] as String,
    labels: _labels(json['labels']),
    blocks: {
      for (final e in ((json['blocks'] as Map?) ?? const {}).entries)
        e.key as String: (e.value as List)
            .map((b) => LessonBlock.fromJson(b as Map<String, dynamic>))
            .toList(),
    },
    completed: json['completed'] as bool,
    nextLessonId: json['next_lesson_id'] as String?,
  );

  final String id;
  final String courseId;
  final Map<String, Label> labels;
  final Map<String, List<LessonBlock>> blocks;
  final bool completed;
  final String? nextLessonId;

  /// Blocs dans la langue de l'utilisateur, sinon en français.
  List<LessonBlock> blocksFor(String language) =>
      blocks[language]?.isNotEmpty == true
      ? blocks[language]!
      : (blocks['fr'] ?? const []);
}
