import 'dart:async';
import 'dart:convert';

import 'package:http/http.dart' as http;

import '../../data/models/token_pair.dart';
import '../storage/token_store.dart';
import 'api_exception.dart';

/// Client HTTP de l'API : JSON, jeton d'accès, renouvellement automatique de la session.
class ApiClient {
  ApiClient({
    required this.baseUrl,
    required this.tokens,
    http.Client? httpClient,
  }) : _http = httpClient ?? http.Client();

  final String baseUrl;
  final TokenStore tokens;
  final http.Client _http;

  /// Appelé quand la session ne peut plus être renouvelée (retour à l'écran de connexion).
  void Function()? onSessionExpired;

  Future<bool>? _refreshing;

  static const _timeout = Duration(seconds: 15);

  Future<Map<String, dynamic>> get(String path) => _send('GET', path);

  Future<Map<String, dynamic>> post(
    String path, {
    Object? body,
    bool auth = true,
  }) => _send('POST', path, body: body, auth: auth);

  Future<Map<String, dynamic>> patch(String path, {Object? body}) =>
      _send('PATCH', path, body: body);

  Future<Map<String, dynamic>> _send(
    String method,
    String path, {
    Object? body,
    bool auth = true,
    bool retried = false,
  }) async {
    final request = http.Request(method, Uri.parse('$baseUrl$path'))
      ..headers['Content-Type'] = 'application/json'
      ..headers['Accept'] = 'application/json';
    if (auth) {
      final token = await tokens.accessToken;
      if (token != null) request.headers['Authorization'] = 'Bearer $token';
    }
    if (body != null) request.body = jsonEncode(body);

    final http.Response response;
    try {
      response = await http.Response.fromStream(
        await _http.send(request).timeout(_timeout),
      );
    } on Exception {
      throw ApiException.network();
    }

    if (response.statusCode == 401 && auth && !retried && await _refresh()) {
      return _send(method, path, body: body, auth: auth, retried: true);
    }

    final json = _decode(response);
    if (response.statusCode >= 400) {
      if (response.statusCode == 401 && auth) {
        await tokens.clear();
        onSessionExpired?.call();
      }
      final error =
          (json['error'] as Map?)?.cast<String, dynamic>() ?? const {};
      throw ApiException(
        response.statusCode,
        error['code'] as String? ?? 'unknown_error',
        error['message'] as String? ?? 'Erreur inattendue.',
        (error['details'] as Map?)?.cast<String, dynamic>() ?? const {},
      );
    }
    return json;
  }

  Map<String, dynamic> _decode(http.Response response) {
    if (response.bodyBytes.isEmpty) return const {};
    try {
      final decoded = jsonDecode(utf8.decode(response.bodyBytes));
      return decoded is Map<String, dynamic> ? decoded : const {};
    } on FormatException {
      return const {};
    }
  }

  /// Un seul renouvellement à la fois : le jeton de rafraîchissement est à usage unique.
  Future<bool> _refresh() =>
      _refreshing ??= _doRefresh().whenComplete(() => _refreshing = null);

  Future<bool> _doRefresh() async {
    final refreshToken = await tokens.refreshToken;
    if (refreshToken == null) return false;
    try {
      final response = await _http
          .post(
            Uri.parse('$baseUrl/auth/refresh'),
            headers: {'Content-Type': 'application/json'},
            body: jsonEncode({'refresh_token': refreshToken}),
          )
          .timeout(_timeout);
      if (response.statusCode != 200) return false;
      await tokens.save(TokenPair.fromJson(_decode(response)));
      return true;
    } on Exception {
      return false;
    }
  }
}
