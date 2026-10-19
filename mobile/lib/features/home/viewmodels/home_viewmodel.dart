import 'package:flutter/foundation.dart';

import '../../../core/api/api_exception.dart';
import '../../../data/models/exam.dart';
import '../../../data/models/license_category.dart';
import '../../../data/repositories/exam_repository.dart';
import '../../../data/repositories/license_repository.dart';

class HomeViewModel extends ChangeNotifier {
  HomeViewModel(this._licenses, this._exams);

  final LicenseRepository _licenses;
  final ExamRepository _exams;

  /// Examen commencé et non terminé, à reprendre.
  ExamSession? currentExam;

  List<LicenseCategory> categories = const [];
  bool loading = false;
  ApiException? error;

  Future<void> load() async {
    loading = true;
    error = null;
    notifyListeners();
    try {
      categories = await _licenses.list();
      currentExam = await _exams.current();
    } on ApiException catch (e) {
      error = e;
    } finally {
      loading = false;
      notifyListeners();
    }
  }
}
