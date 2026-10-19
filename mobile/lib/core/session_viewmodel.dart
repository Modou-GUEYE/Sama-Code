import 'package:flutter/foundation.dart';

import '../data/models/user.dart';
import '../data/repositories/auth_repository.dart';
import '../data/repositories/profile_repository.dart';
import 'api/api_exception.dart';
import 'localization/app_strings.dart';
import 'storage/local_preferences.dart';

enum SessionStatus { loading, onboarding, signedOut, signedIn }

/// État global : langue, mode d'apprentissage et session de l'utilisateur.
class SessionViewModel extends ChangeNotifier {
  SessionViewModel(this._prefs, this._auth, this._profile)
    : language = _prefs.language,
      learningMode = _prefs.learningMode;

  final LocalPreferences _prefs;
  final AuthRepository _auth;
  final ProfileRepository _profile;

  SessionStatus status = SessionStatus.loading;
  String language;
  String learningMode;
  User? user;

  AppStrings get strings => const AppStrings();

  Future<void> init() async {
    if (await _auth.hasSession()) {
      await _loadUser();
      // Hors connexion, on garde la session ouverte ; sinon le jeton a expiré.
      status = user != null || _lastLoadWasNetworkError
          ? SessionStatus.signedIn
          : SessionStatus.signedOut;
    } else {
      status = _prefs.onboardingDone
          ? SessionStatus.signedOut
          : SessionStatus.onboarding;
    }
    notifyListeners();
  }

  bool _lastLoadWasNetworkError = false;

  Future<void> _loadUser() async {
    try {
      user = await _profile.me();
      _lastLoadWasNetworkError = false;
      // Les préférences du compte priment sur celles de l'appareil.
      await _applyLocal(user!.preferredLanguage, user!.learningMode);
    } on ApiException catch (e) {
      _lastLoadWasNetworkError = e.isNetwork;
    }
  }

  Future<void> _applyLocal(String lang, String mode) async {
    language = lang;
    learningMode = mode;
    await _prefs.setLanguage(lang);
    await _prefs.setLearningMode(mode);
  }

  Future<void> setLanguage(String value) => _updatePreferences(lang: value);

  Future<void> setLearningMode(String value) => _updatePreferences(mode: value);

  Future<void> _updatePreferences({String? lang, String? mode}) async {
    await _applyLocal(lang ?? language, mode ?? learningMode);
    notifyListeners();
    if (status == SessionStatus.signedIn) {
      try {
        user = await _profile.update(language: lang, learningMode: mode);
      } on ApiException {
        // Préférence gardée localement ; elle sera renvoyée au prochain changement.
      }
    }
  }

  /// Renvoie null si le nom a été enregistré, sinon l'erreur de l'API.
  Future<ApiException?> updateDisplayName(String name) async {
    try {
      user = await _profile.update(displayName: name);
      notifyListeners();
      return null;
    } on ApiException catch (e) {
      return e;
    }
  }

  Future<void> completeOnboarding() async {
    await _prefs.setOnboardingDone();
    status = SessionStatus.signedOut;
    notifyListeners();
  }

  /// Après une inscription ou une connexion réussie (jetons déjà enregistrés).
  Future<void> onSignedIn() async {
    await _prefs.setOnboardingDone();
    await _loadUser();
    status = SessionStatus.signedIn;
    notifyListeners();
  }

  Future<void> reloadUser() async {
    await _loadUser();
    notifyListeners();
  }

  Future<void> logout() async {
    await _auth.logout();
    onSessionExpired();
  }

  void onSessionExpired() {
    user = null;
    status = SessionStatus.signedOut;
    notifyListeners();
  }
}
