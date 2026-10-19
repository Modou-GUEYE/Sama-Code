import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:package_info_plus/package_info_plus.dart';
import 'package:provider/provider.dart';
import 'package:sama_code/core/api/api_client.dart';
import 'package:sama_code/core/session_viewmodel.dart';
import 'package:sama_code/core/storage/local_preferences.dart';
import 'package:sama_code/core/storage/token_store.dart';
import 'package:sama_code/core/theme/app_theme.dart';
import 'package:sama_code/data/repositories/auth_repository.dart';
import 'package:sama_code/data/repositories/profile_repository.dart';
import 'package:sama_code/features/settings/views/about_view.dart';
import 'package:shared_preferences/shared_preferences.dart';

Future<void> pumpAbout(WidgetTester tester) async {
  PackageInfo.setMockInitialValues(
    appName: 'Sama Code',
    packageName: 'sn.samacode.sama_code',
    version: '1.0.0',
    buildNumber: '1',
    buildSignature: '',
  );
  SharedPreferences.setMockInitialValues({});
  final prefs = await LocalPreferences.load();
  final tokens = TokenStore();
  final api = ApiClient(baseUrl: 'http://x', tokens: tokens);
  tester.view.physicalSize = const Size(1080, 2400);
  tester.view.devicePixelRatio = 2.625;
  addTearDown(tester.view.reset);
  await tester.pumpWidget(
    MultiProvider(
      providers: [
        ChangeNotifierProvider.value(
          value: SessionViewModel(
            prefs,
            AuthRepository(api, tokens),
            ProfileRepository(api),
          ),
        ),
      ],
      child: MaterialApp(theme: buildAppTheme(), home: const AboutView()),
    ),
  );
  // La route est animée en continu : pas de pumpAndSettle.
  await tester.pump();
  await tester.pump(const Duration(milliseconds: 500));
}

void main() {
  testWidgets('à propos : contacts, version et éditeur', (tester) async {
    await pumpAbout(tester);
    expect(find.text('E-mail'), findsOneWidget);
    expect(find.text('Appeler'), findsOneWidget);
    expect(find.text('WhatsApp'), findsOneWidget);
    expect(find.text('Lire'), findsOneWidget);
    expect(find.text('Écouter'), findsOneWidget);
    expect(find.text('Voir'), findsOneWidget);
    expect(
      find.bySemanticsLabel('WhatsApp : 77 353 13 77'),
      findsOneWidget,
    );
    expect(
      find.text('Une application éditée par Sama Code. Version 1.0.0'),
      findsOneWidget,
    );
    expect(find.text('Licences des logiciels'), findsOneWidget);
  });
}
