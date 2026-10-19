import '../../core/api/api_client.dart';
import '../models/exam.dart';

class PracticeRepository {
  PracticeRepository(this._api);

  final ApiClient _api;

  Future<List<PracticeLicense>> summary() async {
    final json = await _api.get('/practice');
    return (json['items'] as List)
        .map((l) => PracticeLicense.fromJson(l as Map<String, dynamic>))
        .toList();
  }

  Future<List<PracticeItem>> seriesQuestions(String seriesId) async =>
      _items(await _api.get('/practice/series/$seriesId/questions'));

  Future<List<PracticeItem>> mistakes(String licenseCategoryId) async => _items(
    await _api.get('/practice/license-categories/$licenseCategoryId/mistakes'),
  );

  Future<PracticeCheck> check(
    String seriesId,
    String questionId,
    Set<String> optionIds,
  ) async => PracticeCheck.fromJson(
    await _api.post(
      '/practice/series/$seriesId/questions/$questionId/check',
      body: {'option_ids': optionIds.toList()},
    ),
  );

  List<PracticeItem> _items(Map<String, dynamic> json) =>
      (json['items'] as List)
          .map((i) => PracticeItem.fromJson(i as Map<String, dynamic>))
          .toList();
}
