import 'package:flutter/foundation.dart';

import '../../../core/api/api_exception.dart';
import '../../../data/models/license_category.dart';
import '../../../data/repositories/license_repository.dart';

class HomeViewModel extends ChangeNotifier {
  HomeViewModel(this._licenses);

  final LicenseRepository _licenses;

  List<LicenseCategory> categories = const [];
  bool loading = false;
  ApiException? error;

  Future<void> load() async {
    loading = true;
    error = null;
    notifyListeners();
    try {
      categories = await _licenses.list();
    } on ApiException catch (e) {
      error = e;
    } finally {
      loading = false;
      notifyListeners();
    }
  }
}
