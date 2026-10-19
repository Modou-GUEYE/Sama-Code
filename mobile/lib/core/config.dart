/// Configuration injectée à la compilation :
/// `flutter run --dart-define=API_BASE_URL=https://…/api/v1`.
class AppConfig {
  /// Par défaut : l'API locale vue depuis l'émulateur Android (10.0.2.2 = la machine hôte).
  static const apiBaseUrl = String.fromEnvironment(
    'API_BASE_URL',
    defaultValue: 'http://10.0.2.2:8080/api/v1',
  );

  /// Éditeur et contacts affichés dans « À propos ».
  static const appName = 'Sama Code';

  /// Version des conditions d'utilisation affichées (TermsView). Doit être identique
  /// à users.TermsVersion côté API : sinon l'inscription est refusée (terms_outdated).
  static const termsVersion = '2026-10-19';
  static const publisher = 'CFES SOLUTION';
  static const contactEmail = 'info@samacode.com';

  /// Numéro de contact (appel et WhatsApp), au format +221XXXXXXXXX ; vide masque
  /// les deux boutons.
  static const contactPhone = '+221773531377';

  /// +221771234567 → 77 123 45 67
  static String get contactPhoneDisplay {
    final m = RegExp(r'^\+221(\d{2})(\d{3})(\d{2})(\d{2})$')
        .firstMatch(contactPhone);
    return m == null ? contactPhone : '${m[1]} ${m[2]} ${m[3]} ${m[4]}';
  }

  /// Origine de l'API, pour les médias servis en chemins relatifs (/api/v1/media/…).
  static String mediaUrl(String path) => path.startsWith('http')
      ? path
      : apiBaseUrl.replaceFirst(RegExp(r'/api/v1$'), '') + path;
}
