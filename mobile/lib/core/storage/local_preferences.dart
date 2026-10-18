import 'package:shared_preferences/shared_preferences.dart';

/// Préférences choisies avant même d'avoir un compte (langue, mode, onboarding vu).
class LocalPreferences {
  LocalPreferences._(this._prefs);

  static Future<LocalPreferences> load() async =>
      LocalPreferences._(await SharedPreferences.getInstance());

  final SharedPreferences _prefs;

  String get language => _prefs.getString('language') ?? 'fr';
  Future<void> setLanguage(String value) => _prefs.setString('language', value);

  String get learningMode => _prefs.getString('learning_mode') ?? 'text';
  Future<void> setLearningMode(String value) =>
      _prefs.setString('learning_mode', value);

  bool get onboardingDone => _prefs.getBool('onboarding_done') ?? false;
  Future<void> setOnboardingDone() => _prefs.setBool('onboarding_done', true);
}
