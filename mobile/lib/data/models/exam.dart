/// Texte et médias d'une question ou d'une proposition dans une langue.
class LocalizedContent {
  const LocalizedContent({required this.text, this.audioUrl, this.imageUrl});

  factory LocalizedContent.fromJson(Map<String, dynamic> json) =>
      LocalizedContent(
        text: json['text'] as String? ?? '',
        audioUrl: json['audio_url'] as String?,
        imageUrl: json['image_url'] as String?,
      );

  final String text;
  final String? audioUrl;
  final String? imageUrl;
}

Map<String, LocalizedContent> _contents(Object? json) => {
  for (final e in ((json as Map?) ?? const {}).entries)
    e.key as String: LocalizedContent.fromJson(e.value as Map<String, dynamic>),
};

/// Choisit la langue de l'utilisateur, sinon le français.
LocalizedContent pick(
  Map<String, LocalizedContent> contents,
  String language,
) => contents[language] ?? contents['fr'] ?? const LocalizedContent(text: '');

class ExamOption {
  const ExamOption({
    required this.id,
    required this.code,
    required this.contents,
  });

  factory ExamOption.fromJson(Map<String, dynamic> json) => ExamOption(
    id: json['id'] as String,
    code: json['code'] as String,
    contents: _contents(json['contents']),
  );

  final String id;
  final String code;
  final Map<String, LocalizedContent> contents;
}

class ExamQuestion {
  const ExamQuestion({
    required this.position,
    required this.id,
    required this.multiple,
    required this.contents,
    required this.options,
  });

  factory ExamQuestion.fromJson(Map<String, dynamic> json) => ExamQuestion(
    position: json['position'] as int,
    id: json['question_id'] as String,
    multiple: json['question_type'] == 'multiple',
    contents: _contents(json['contents']),
    options: (json['options'] as List)
        .map((o) => ExamOption.fromJson(o as Map<String, dynamic>))
        .toList(),
  );

  final int position;
  final String id;
  final bool multiple;
  final Map<String, LocalizedContent> contents;
  final List<ExamOption> options;
}

class ExamResult {
  const ExamResult({
    required this.correct,
    required this.faults,
    required this.unanswered,
    required this.maxFaults,
    required this.passed,
  });

  factory ExamResult.fromJson(Map<String, dynamic> json) => ExamResult(
    correct: json['correct_count'] as int,
    faults: json['fault_count'] as int,
    unanswered: json['unanswered_count'] as int,
    maxFaults: json['max_faults'] as int,
    passed: json['passed'] as bool,
  );

  final int correct;
  final int faults;
  final int unanswered;
  final int maxFaults;
  final bool passed;
}

class ExamSession {
  const ExamSession({
    required this.id,
    required this.seriesId,
    required this.licenseCode,
    required this.seriesNumber,
    required this.submitted,
    required this.durationMinutes,
    required this.maxFaults,
    required this.expiresAt,
    required this.serverTime,
    required this.questions,
    required this.answers,
    this.result,
  });

  factory ExamSession.fromJson(Map<String, dynamic> json) => ExamSession(
    id: json['id'] as String,
    seriesId: json['series_id'] as String,
    licenseCode: json['license_code'] as String,
    seriesNumber: json['series_number'] as int,
    submitted: json['status'] == 'submitted',
    durationMinutes: json['duration_minutes'] as int,
    maxFaults: json['max_faults'] as int,
    expiresAt: DateTime.parse(json['expires_at'] as String),
    serverTime: DateTime.parse(json['server_time'] as String),
    questions: (json['questions'] as List)
        .map((q) => ExamQuestion.fromJson(q as Map<String, dynamic>))
        .toList(),
    answers: {
      for (final e in ((json['answers'] as Map?) ?? const {}).entries)
        e.key as String: (e.value as List).cast<String>().toSet(),
    },
    result: json['result'] == null
        ? null
        : ExamResult.fromJson(json['result'] as Map<String, dynamic>),
  );

  final String id;
  final String seriesId;
  final String licenseCode;
  final int seriesNumber;
  final bool submitted;
  final int durationMinutes;
  final int maxFaults;
  final DateTime expiresAt;
  final DateTime serverTime;
  final List<ExamQuestion> questions;
  final Map<String, Set<String>> answers;
  final ExamResult? result;
}

/// Série publiée proposée au candidat, avec ses propres résultats.
class SeriesSummary {
  const SeriesSummary({
    required this.id,
    required this.number,
    required this.questionCount,
    required this.durationMinutes,
    required this.maxFaults,
    required this.attempts,
    this.bestFaults,
    this.lastPassed,
  });

  factory SeriesSummary.fromJson(Map<String, dynamic> json) => SeriesSummary(
    id: json['id'] as String,
    number: json['series_number'] as int,
    questionCount: json['question_count'] as int,
    durationMinutes: json['duration_minutes'] as int,
    maxFaults: json['max_faults'] as int,
    attempts: json['attempts'] as int,
    bestFaults: json['best_faults'] as int?,
    lastPassed: json['last_passed'] as bool?,
  );

  final String id;
  final int number;
  final int questionCount;
  final int durationMinutes;
  final int maxFaults;
  final int attempts;
  final int? bestFaults;
  final bool? lastPassed;
}

enum Outcome { correct, incorrect, unanswered }

class ReviewQuestion {
  const ReviewQuestion({
    required this.question,
    required this.selected,
    required this.correct,
    required this.outcome,
    required this.explanations,
  });

  factory ReviewQuestion.fromJson(Map<String, dynamic> json) => ReviewQuestion(
    question: ExamQuestion.fromJson(json),
    selected: (json['selected_option_ids'] as List).cast<String>().toSet(),
    correct: (json['correct_option_ids'] as List).cast<String>().toSet(),
    outcome: Outcome.values.byName(json['outcome'] as String),
    explanations: _contents(json['explanations']),
  );

  final ExamQuestion question;
  final Set<String> selected;
  final Set<String> correct;
  final Outcome outcome;
  final Map<String, LocalizedContent> explanations;
}
