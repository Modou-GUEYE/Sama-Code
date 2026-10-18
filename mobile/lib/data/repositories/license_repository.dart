import '../../core/api/api_client.dart';
import '../models/license_category.dart';

class LicenseRepository {
  LicenseRepository(this._api);

  final ApiClient _api;

  Future<List<LicenseCategory>> list() async {
    final json = await _api.get('/license-categories');
    return (json['items'] as List)
        .map((item) => LicenseCategory.fromJson(item as Map<String, dynamic>))
        .toList();
  }
}
