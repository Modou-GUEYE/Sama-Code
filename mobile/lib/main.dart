import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import 'app.dart';
import 'core/api/api_client.dart';
import 'core/config.dart';
import 'core/session_viewmodel.dart';
import 'core/storage/local_preferences.dart';
import 'core/storage/token_store.dart';
import 'data/repositories/auth_repository.dart';
import 'data/repositories/license_repository.dart';
import 'data/repositories/profile_repository.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();

  final prefs = await LocalPreferences.load();
  final tokens = TokenStore();
  final api = ApiClient(baseUrl: AppConfig.apiBaseUrl, tokens: tokens);

  final auth = AuthRepository(api, tokens);
  final profile = ProfileRepository(api);
  final licenses = LicenseRepository(api);
  final session = SessionViewModel(prefs, auth, profile);
  api.onSessionExpired = session.onSessionExpired;

  runApp(
    MultiProvider(
      providers: [
        Provider.value(value: auth),
        Provider.value(value: profile),
        Provider.value(value: licenses),
        ChangeNotifierProvider.value(value: session),
      ],
      child: const SamaCodeApp(),
    ),
  );

  await session.init();
}
