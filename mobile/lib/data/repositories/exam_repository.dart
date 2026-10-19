import '../../core/api/api_client.dart';
import '../../core/api/api_exception.dart';
import '../models/exam.dart';

class ExamRepository {
  ExamRepository(this._api);

  final ApiClient _api;

  Future<List<SeriesSummary>> series(String licenseCategoryId) async {
    final json = await _api.get(
      '/license-categories/$licenseCategoryId/series',
    );
    return (json['items'] as List)
        .map((s) => SeriesSummary.fromJson(s as Map<String, dynamic>))
        .toList();
  }

  /// Démarre un examen. Lève ApiException `session_in_progress` si un examen est en cours,
  /// sauf si [abandonCurrent] (l'examen en cours est alors soumis en l'état).
  Future<ExamSession> start(
    String licenseCategoryId,
    String seriesId, {
    bool abandonCurrent = false,
  }) async {
    return ExamSession.fromJson(
      await _api.post(
        '/exam-sessions',
        body: {
          'license_category_id': licenseCategoryId,
          'series_id': seriesId,
          'abandon_current': abandonCurrent,
        },
      ),
    );
  }

  /// Examen en cours du candidat, ou null.
  Future<ExamSession?> current() async {
    try {
      return ExamSession.fromJson(await _api.get('/exam-sessions/current'));
    } on ApiException catch (e) {
      if (e.statusCode == 404) return null;
      rethrow;
    }
  }

  /// Historique des tentatives, la plus récente d'abord ([page] commence à 1).
  Future<List<ExamHistoryItem>> history({
    int page = 1,
    int pageSize = 20,
  }) async {
    final json = await _api.get(
      '/exam-sessions?page=$page&page_size=$pageSize',
    );
    return (json['items'] as List)
        .map((e) => ExamHistoryItem.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  Future<ExamSession> get(String id) async =>
      ExamSession.fromJson(await _api.get('/exam-sessions/$id'));

  Future<void> answer(
    String sessionId,
    String questionId,
    Set<String> optionIds,
  ) => _api.put(
    '/exam-sessions/$sessionId/answers/$questionId',
    body: {'option_ids': optionIds.toList()},
  );

  Future<ExamSession> submit(String id) async =>
      ExamSession.fromJson(await _api.post('/exam-sessions/$id/submit'));

  Future<List<ReviewQuestion>> review(String id) async {
    final json = await _api.get('/exam-sessions/$id/review');
    return (json['questions'] as List)
        .map((q) => ReviewQuestion.fromJson(q as Map<String, dynamic>))
        .toList();
  }
}
