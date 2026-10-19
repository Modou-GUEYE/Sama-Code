import '../../core/api/api_client.dart';
import '../../core/storage/token_store.dart';
import '../models/token_pair.dart';

/// Objectif d'un code SMS, tel qu'attendu par l'API.
enum OtpPurpose {
  register('register'),
  login('login'),
  resetPin('reset_pin');

  const OtpPurpose(this.value);
  final String value;
}

class OtpRequestResult {
  const OtpRequestResult({required this.expiresInSeconds, this.devCode});

  final int expiresInSeconds;

  /// Renseigné uniquement par l'API de développement (pas de vrai SMS).
  final String? devCode;
}

class AuthRepository {
  AuthRepository(this._api, this._tokens);

  final ApiClient _api;
  final TokenStore _tokens;

  Future<bool> hasSession() async => await _tokens.refreshToken != null;

  Future<OtpRequestResult> requestOtp(String phone, OtpPurpose purpose) async {
    final json = await _api.post(
      '/auth/otp/request',
      auth: false,
      body: {'phone_number': phone, 'purpose': purpose.value},
    );
    return OtpRequestResult(
      expiresInSeconds: json['expires_in_seconds'] as int? ?? 300,
      devCode: json['dev_code'] as String?,
    );
  }

  /// Renvoie le jeton de vérification (inscription, PIN oublié) ou ouvre la session (connexion).
  Future<String?> verifyOtp(
    String phone,
    OtpPurpose purpose,
    String code,
  ) async {
    final json = await _api.post(
      '/auth/otp/verify',
      auth: false,
      body: {'phone_number': phone, 'purpose': purpose.value, 'code': code},
    );
    final tokens = json['tokens'];
    if (tokens is Map<String, dynamic>) {
      await _tokens.save(TokenPair.fromJson(tokens));
    }
    return json['verification_token'] as String?;
  }

  Future<void> register({
    required String verificationToken,
    required String displayName,
    required String language,
    required String learningMode,
    required String pin,
    required String termsVersion,
  }) async {
    final json = await _api.post(
      '/auth/register',
      auth: false,
      body: {
        'verification_token': verificationToken,
        'display_name': displayName,
        'preferred_language': language,
        'learning_mode': learningMode,
        'pin': pin,
        'terms_version': termsVersion,
      },
    );
    await _tokens.save(TokenPair.fromJson(json));
  }

  Future<void> login(String phone, String pin) async {
    final json = await _api.post(
      '/auth/login',
      auth: false,
      body: {'phone_number': phone, 'pin': pin},
    );
    await _tokens.save(TokenPair.fromJson(json));
  }

  Future<void> resetPin(String verificationToken, String pin) async {
    final json = await _api.post(
      '/auth/pin/reset',
      auth: false,
      body: {'verification_token': verificationToken, 'pin': pin},
    );
    await _tokens.save(TokenPair.fromJson(json));
  }

  Future<void> logout() async {
    final refreshToken = await _tokens.refreshToken;
    await _tokens.clear();
    if (refreshToken == null) return;
    try {
      await _api.post(
        '/auth/logout',
        auth: false,
        body: {'refresh_token': refreshToken},
      );
    } on Exception {
      // Hors connexion : la session locale est déjà fermée, le jeton expirera côté serveur.
    }
  }
}
