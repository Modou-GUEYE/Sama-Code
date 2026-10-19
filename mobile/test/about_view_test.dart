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
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('à propos : contacts, version et éditeur', (tester) async {
    await pumpAbout(tester);
    expect(find.text('info@samacode.com'), findsOneWidget);
    await tester.scrollUntilVisible(find.text('Une application éditée par'), 300);
    expect(find.text('Appeler'), findsOneWidget);
    expect(find.text('WhatsApp'), findsOneWidget);
    expect(find.text('77 353 13 77'), findsNWidgets(2));
    expect(find.text('1.0.0'), findsOneWidget);
    expect(find.text('Sama Code'), findsOneWidget);
  });
}
