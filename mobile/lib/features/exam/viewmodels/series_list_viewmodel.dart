import 'package:flutter/foundation.dart';

import '../../../core/api/api_exception.dart';
import '../../../data/models/exam.dart';
import '../../../data/models/license_category.dart';
import '../../../data/repositories/exam_repository.dart';

class SeriesListViewModel extends ChangeNotifier {
  SeriesListViewModel(this._repo, this.license);

  final ExamRepository _repo;
  final LicenseCategory license;

  List<SeriesSummary> series = const [];
  bool loading = false;
  bool starting = false;
  ApiException? error;

  Future<void> load() async {
    loading = true;
    error = null;
    notifyListeners();
    try {
      series = await _repo.series(license.id);
    } on ApiException catch (e) {
      error = e;
    } finally {
      loading = false;
      notifyListeners();
    }
  }

  /// Démarre la série. Lève l'ApiException `session_in_progress` pour que la vue
  /// propose de reprendre ou d'abandonner l'examen en cours.
  Future<ExamSession> start(
    SeriesSummary s, {
    bool abandonCurrent = false,
  }) async {
    starting = true;
    notifyListeners();
    try {
      return await _repo.start(
        license.id,
        s.id,
        abandonCurrent: abandonCurrent,
      );
    } finally {
      starting = false;
      notifyListeners();
    }
  }

  Future<ExamSession> resume(String sessionId) => _repo.get(sessionId);
}
