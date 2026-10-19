import '../../core/api/api_client.dart';
import '../models/user.dart';

class ProfileRepository {
  ProfileRepository(this._api);

  final ApiClient _api;

  Future<User> me() async => User.fromJson(await _api.get('/me'));

  Future<User> update({
    String? displayName,
    String? language,
    String? learningMode,
  }) async {
    final json = await _api.patch(
      '/me',
      body: {
        'display_name': ?displayName,
        'preferred_language': ?language,
        'learning_mode': ?learningMode,
      },
    );
    return User.fromJson(json);
  }

  /// Accepte la version [version] des conditions d'utilisation.
  Future<User> acceptTerms(String version) async => User.fromJson(
    await _api.post('/me/terms-acceptance', body: {'version': version}),
  );
}
