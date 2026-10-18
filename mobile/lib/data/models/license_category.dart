class LicenseCategory {
  const LicenseCategory({
    required this.id,
    required this.code,
    required this.name,
    required this.seriesCount,
    required this.questionsPerSeries,
    required this.maxFaults,
    required this.examDurationMinutes,
  });

  factory LicenseCategory.fromJson(Map<String, dynamic> json) =>
      LicenseCategory(
        id: json['id'] as String,
        code: json['code'] as String,
        name: json['name'] as String,
        seriesCount: json['series_count'] as int,
        questionsPerSeries: json['questions_per_series'] as int,
        maxFaults: json['max_faults'] as int,
        examDurationMinutes: json['exam_duration_minutes'] as int,
      );

  final String id;
  final String code;
  final String name;
  final int seriesCount;
  final int questionsPerSeries;
  final int maxFaults;
  final int examDurationMinutes;
}
