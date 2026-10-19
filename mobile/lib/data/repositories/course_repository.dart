import '../../core/api/api_client.dart';
import '../models/course.dart';

class CourseRepository {
  CourseRepository(this._api);

  final ApiClient _api;

  Future<List<CourseCategory>> catalog() async {
    final json = await _api.get('/course-categories');
    return (json['items'] as List)
        .map((c) => CourseCategory.fromJson(c as Map<String, dynamic>))
        .toList();
  }

  Future<CourseSummary> course(String id) async =>
      CourseSummary.fromJson(await _api.get('/courses/$id'));

  Future<Lesson> lesson(String id) async =>
      Lesson.fromJson(await _api.get('/lessons/$id'));

  Future<void> complete(String lessonId) =>
      _api.post('/lessons/$lessonId/completion');
}
