/// Configuration injectée à la compilation :
/// `flutter run --dart-define=API_BASE_URL=https://…/api/v1`.
class AppConfig {
  /// Par défaut : l'API locale vue depuis l'émulateur Android (10.0.2.2 = la machine hôte).
  static const apiBaseUrl = String.fromEnvironment(
    'API_BASE_URL',
    defaultValue: 'http://10.0.2.2:8080/api/v1',
  );

  /// Origine de l'API, pour les médias servis en chemins relatifs (/api/v1/media/…).
  static String mediaUrl(String path) => path.startsWith('http')
      ? path
      : apiBaseUrl.replaceFirst(RegExp(r'/api/v1$'), '') + path;
}
