/// Erreur normalisée de l'API : `{"error": {"code", "message", "details"}}`.
class ApiException implements Exception {
  ApiException(
    this.statusCode,
    this.code,
    this.message, [
    this.details = const {},
  ]);

  ApiException.network()
    : statusCode = 0,
      code = 'network_error',
      message = 'Connexion au serveur impossible.',
      details = const {};

  final int statusCode;
  final String code;
  final String message;
  final Map<String, dynamic> details;

  bool get isNetwork => code == 'network_error';

  @override
  String toString() => 'ApiException($statusCode, $code)';
}
