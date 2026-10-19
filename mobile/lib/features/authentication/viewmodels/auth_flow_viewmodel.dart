import 'package:flutter/foundation.dart';

import '../../../core/api/api_exception.dart';
import '../../../core/config.dart';
import '../../../data/repositories/auth_repository.dart';

/// Parcours d'authentification (D11) :
/// - register  : téléphone → code SMS → nom → PIN ×2
/// - pinLogin  : téléphone → PIN
/// - otpLogin  : téléphone → code SMS (débloque un compte bloqué)
/// - resetPin  : téléphone → code SMS → nouveau PIN ×2
enum AuthFlow { register, pinLogin, otpLogin, resetPin }

/// Nombre d'étapes de l'inscription : numéro, code SMS, nom, code secret.
const registrationSteps = 4;

class AuthFlowViewModel extends ChangeNotifier {
  AuthFlowViewModel(this._auth, this.flow);

  final AuthRepository _auth;
  AuthFlow flow;

  String phone = '';
  String displayName = '';
  String? devCode;
  String? _verificationToken;

  bool busy = false;
  ApiException? error;

  /// Incrémenté à chaque erreur pour vider le pavé numérique.
  int attempt = 0;

  OtpPurpose get otpPurpose => switch (flow) {
    AuthFlow.register => OtpPurpose.register,
    AuthFlow.resetPin => OtpPurpose.resetPin,
    AuthFlow.pinLogin || AuthFlow.otpLogin => OtpPurpose.login,
  };

  bool get choosesNewPin =>
      flow == AuthFlow.register || flow == AuthFlow.resetPin;

  void clearError() {
    if (error == null) return;
    error = null;
    notifyListeners();
  }

  /// Passe à un autre parcours en gardant le numéro (PIN oublié, compte bloqué).
  void switchTo(AuthFlow next) {
    flow = next;
    error = null;
    devCode = null;
    _verificationToken = null;
    notifyListeners();
  }

  /// Connexion par PIN : pas d'appel réseau à cette étape, simple contrôle de longueur.
  /// La validation complète du numéro est faite par le serveur.
  bool acceptPhone(String rawPhone) {
    final digits = rawPhone.replaceAll(RegExp(r'\D'), '');
    if (digits.length < 9) {
      error = ApiException(400, 'invalid_phone', 'Numéro invalide.');
      notifyListeners();
      return false;
    }
    phone = rawPhone.trim();
    error = null;
    notifyListeners();
    return true;
  }

  /// Inscription : case « J'accepte les conditions d'utilisation » (Q13).
  bool termsAccepted = false;

  void setTermsAccepted(bool value) {
    termsAccepted = value;
    notifyListeners();
  }

  Future<bool> requestCode([String? rawPhone]) => _run(() async {
    if (rawPhone != null) phone = rawPhone.trim();
    final result = await _auth.requestOtp(phone, otpPurpose);
    devCode = result.devCode;
  });

  Future<bool> verifyCode(String code) => _run(() async {
    _verificationToken = await _auth.verifyOtp(phone, otpPurpose, code);
  });

  Future<bool> loginWithPin(String pin) => _run(() => _auth.login(phone, pin));

  Future<bool> submitNewPin(
    String pin, {
    required String language,
    required String learningMode,
  }) => _run(() async {
    final token = _verificationToken;
    if (token == null) {
      throw ApiException(
        401,
        'invalid_verification_token',
        'Vérification expirée.',
      );
    }
    if (flow == AuthFlow.register) {
      await _auth.register(
        verificationToken: token,
        displayName: displayName,
        language: language,
        learningMode: learningMode,
        pin: pin,
        termsVersion: AppConfig.termsVersion,
      );
    } else {
      await _auth.resetPin(token, pin);
    }
  });

  Future<bool> _run(Future<void> Function() action) async {
    busy = true;
    error = null;
    notifyListeners();
    try {
      await action();
      return true;
    } on ApiException catch (e) {
      error = e;
      attempt++;
      return false;
    } finally {
      busy = false;
      notifyListeners();
    }
  }
}

/// Mêmes règles que le serveur : 4 chiffres, pas de suite évidente (0000, 1234, 9876…).
bool isWeakPin(String pin) {
  var same = true, asc = true, desc = true;
  for (var i = 1; i < pin.length; i++) {
    final d = pin.codeUnitAt(i) - pin.codeUnitAt(i - 1);
    same &= d == 0;
    asc &= d == 1;
    desc &= d == -1;
  }
  return same || asc || desc;
}
