import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';
import 'package:sama_code/core/api/api_client.dart';
import 'package:sama_code/core/session_viewmodel.dart';
import 'package:sama_code/core/storage/local_preferences.dart';
import 'package:sama_code/core/storage/token_store.dart';
import 'package:sama_code/core/theme/app_theme.dart';
import 'package:sama_code/data/models/exam.dart';
import 'package:sama_code/data/repositories/auth_repository.dart';
import 'package:sama_code/data/repositories/practice_repository.dart';
import 'package:sama_code/data/repositories/profile_repository.dart';
import 'package:sama_code/features/practice/views/practice_home_view.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// Série de 3 questions dont les 2 premières sont déjà faites.
class FakeRepo extends PracticeRepository {
  FakeRepo(super.api);

  @override
  Future<List<PracticeLicense>> summary() async => [
        PracticeLicense.fromJson({
          'id': 'lic', 'code': 'B', 'name': 'Permis B', 'mistakes': 0,
          'series': [{'id': 's1', 'series_number': 1, 'question_count': 3, 'practiced': 2, 'mastered': 2}],
        }),
      ];

  @override
  Future<List<PracticeItem>> seriesQuestions(String seriesId) async => [
        for (final (i, last) in [(1, 'correct'), (2, 'correct'), (3, null)])
          PracticeItem.fromJson({
            'series_id': 's1',
            'last_outcome': last,
            'question': {
              'position': i, 'question_id': 'q$i', 'question_type': 'single',
              'contents': {'fr': {'text': 'Énoncé $i'}},
              'options': [{'id': 'q$i-A', 'code': 'A', 'contents': {'fr': {'text': 'A'}}}],
            },
          }),
      ];
}

Future<void> openSeries(WidgetTester tester) async {
  SharedPreferences.setMockInitialValues({});
  final prefs = await LocalPreferences.load();
  final tokens = TokenStore();
  final api = ApiClient(baseUrl: 'http://x', tokens: tokens);
  await tester.pumpWidget(MultiProvider(
    providers: [
      Provider<PracticeRepository>.value(value: FakeRepo(api)),
      ChangeNotifierProvider.value(value: SessionViewModel(prefs, AuthRepository(api, tokens), ProfileRepository(api))),
    ],
    child: MaterialApp(theme: buildAppTheme(), home: const PracticeHomeView()),
  ));
  await tester.pumpAndSettle();
  await tester.tap(find.text('Série 1'));
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('série commencée : « Reprendre à la question 3 » ouvre la question 3', (tester) async {
    await openSeries(tester);
    expect(find.text('Reprendre à la question 3'), findsOneWidget);
    await tester.tap(find.text('Reprendre à la question 3'));
    await tester.pumpAndSettle();
    expect(find.text('Énoncé 3'), findsOneWidget);
    expect(find.text('Question 3 / 3'), findsOneWidget);
  });

  testWidgets('« Recommencer depuis le début » ouvre la question 1', (tester) async {
    await openSeries(tester);
    await tester.tap(find.text('Recommencer depuis le début'));
    await tester.pumpAndSettle();
    expect(find.text('Énoncé 1'), findsOneWidget);
  });
}
