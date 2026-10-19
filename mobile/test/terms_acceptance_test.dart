import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';
import 'package:sama_code/core/api/api_client.dart';
import 'package:sama_code/core/config.dart';
import 'package:sama_code/core/session_viewmodel.dart';
import 'package:sama_code/core/storage/local_preferences.dart';
import 'package:sama_code/core/storage/token_store.dart';
import 'package:sama_code/core/theme/app_theme.dart';
import 'package:sama_code/data/models/user.dart';
import 'package:sama_code/data/repositories/auth_repository.dart';
import 'package:sama_code/data/repositories/profile_repository.dart';
import 'package:sama_code/features/authentication/viewmodels/auth_flow_viewmodel.dart';
import 'package:sama_code/features/authentication/views/phone_view.dart';
import 'package:sama_code/features/settings/views/terms_view.dart';
import 'package:shared_preferences/shared_preferences.dart';

User _user({required bool accepted}) => User(
  id: 'u',
  displayName: 'Awa Diop',
  phoneNumber: '+221771234567',
  preferredLanguage: 'fr',
  learningMode: 'text',
  roles: const ['candidate'],
  termsAccepted: accepted,
);

class FakeProfile extends ProfileRepository {
  FakeProfile(super.api);
  final accepted = <String>[];

  @override
  Future<User> acceptTerms(String version) async {
    accepted.add(version);
    return _user(accepted: true);
  }
}

Future<(SessionViewModel, FakeProfile, AuthRepository)> _setup(
  WidgetTester tester,
) async {
  SharedPreferences.setMockInitialValues({});
  final prefs = await LocalPreferences.load();
  final tokens = TokenStore();
  final api = ApiClient(baseUrl: 'http://x', tokens: tokens);
  final profile = FakeProfile(api);
  final auth = AuthRepository(api, tokens);
  tester.view.physicalSize = const Size(1080, 2400);
  tester.view.devicePixelRatio = 2.625;
  addTearDown(tester.view.reset);
  return (SessionViewModel(prefs, auth, profile), profile, auth);
}

void main() {
  testWidgets(
    'inscription : « Recevoir le code » attend l’acceptation des CGU',
    (tester) async {
      final (session, _, auth) = await _setup(tester);
      final vm = AuthFlowViewModel(auth, AuthFlow.register);
      await tester.pumpWidget(
        MultiProvider(
          providers: [
            ChangeNotifierProvider.value(value: session),
            ChangeNotifierProvider.value(value: vm),
          ],
          child: MaterialApp(theme: buildAppTheme(), home: const PhoneView()),
        ),
      );
      await tester.enterText(find.byType(TextField), '771234567');
      await tester.pump();

      FilledButton button() => tester.widget<FilledButton>(
        find.ancestor(
          of: find.text('Recevoir le code'),
          matching: find.byType(FilledButton),
        ),
      );
      expect(button().onPressed, isNull);

      await tester.tap(find.byType(Checkbox));
      await tester.pump();
      expect(vm.termsAccepted, isTrue);
      expect(button().onPressed, isNotNull);
    },
  );

  testWidgets(
    'compte sans CGU : « J’accepte » enregistre la version en vigueur',
    (tester) async {
      final (session, profile, _) = await _setup(tester);
      session
        ..status = SessionStatus.signedIn
        ..user = _user(accepted: false);
      expect(session.needsTerms, isTrue);

      await tester.pumpWidget(
        ChangeNotifierProvider.value(
          value: session,
          child: MaterialApp(
            theme: buildAppTheme(),
            home: const TermsView(acceptance: true),
          ),
        ),
      );
      expect(find.byIcon(Icons.arrow_back_rounded), findsNothing);
      await tester.tap(find.text('J’accepte'));
      await tester.pumpAndSettle();

      expect(profile.accepted, [AppConfig.termsVersion]);
      expect(session.needsTerms, isFalse);
    },
  );
}
