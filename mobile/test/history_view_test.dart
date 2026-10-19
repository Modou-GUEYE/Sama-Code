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
import 'package:sama_code/data/repositories/exam_repository.dart';
import 'package:sama_code/data/repositories/profile_repository.dart';
import 'package:sama_code/features/exam/views/history_view.dart';
import 'package:sama_code/features/exam/views/result_view.dart';
import 'package:shared_preferences/shared_preferences.dart';

const _passed = ExamResult(correct: 23, faults: 2, unanswered: 0, maxFaults: 4, passed: true);
const _failed = ExamResult(correct: 18, faults: 7, unanswered: 1, maxFaults: 4, passed: false);

ExamHistoryItem _item(int i, {ExamResult? result}) => ExamHistoryItem(
      id: 's$i',
      licenseCode: 'B',
      seriesNumber: i,
      submitted: result != null,
      startedAt: DateTime(2026, 10, 1, 9, 5),
      result: result,
    );

class FakeExams extends ExamRepository {
  FakeExams(super.api, this.items);
  final List<ExamHistoryItem> items;
  final pages = <int>[];

  @override
  Future<List<ExamHistoryItem>> history({int page = 1, int pageSize = 20}) async {
    pages.add(page);
    return items.skip((page - 1) * pageSize).take(pageSize).toList();
  }

  @override
  Future<ExamSession> get(String id) async => ExamSession(
        id: id,
        seriesId: 'series',
        licenseCode: 'B',
        seriesNumber: 1,
        submitted: true,
        durationMinutes: 20,
        maxFaults: 4,
        expiresAt: DateTime(2026),
        serverTime: DateTime(2026),
        questions: const [],
        answers: const {},
        result: _passed,
      );
}

Future<FakeExams> pumpHistory(WidgetTester tester, List<ExamHistoryItem> items) async {
  SharedPreferences.setMockInitialValues({});
  final prefs = await LocalPreferences.load();
  final tokens = TokenStore();
  final api = ApiClient(baseUrl: 'http://x', tokens: tokens);
  final exams = FakeExams(api, items);
  tester.view.physicalSize = const Size(1080, 2400);
  tester.view.devicePixelRatio = 2.625;
  addTearDown(tester.view.reset);
  await tester.pumpWidget(MultiProvider(
    providers: [
      Provider<ExamRepository>.value(value: exams),
      ChangeNotifierProvider.value(value: SessionViewModel(prefs, AuthRepository(api, tokens), ProfileRepository(api))),
    ],
    child: MaterialApp(theme: buildAppTheme(), home: const HistoryView()),
  ));
  await tester.pumpAndSettle();
  return exams;
}

void main() {
  testWidgets('historique : statut, fautes et date de chaque tentative', (tester) async {
    await pumpHistory(tester, [_item(3), _item(2, result: _passed), _item(1, result: _failed)]);
    expect(find.text('En cours'), findsOneWidget);
    expect(find.text('Réussi · 2 fautes'), findsOneWidget);
    expect(find.text('Ajourné · 7 fautes'), findsOneWidget);
    expect(find.text('Permis B · Série 2'), findsOneWidget);
    expect(find.text('01/10/2026 09:05'), findsNWidgets(3));
    expect(find.text('Voir plus'), findsNothing);
  });

  testWidgets('historique : une tentative soumise ouvre son résultat', (tester) async {
    await pumpHistory(tester, [_item(2, result: _passed)]);
    await tester.tap(find.text('Permis B · Série 2'));
    await tester.pumpAndSettle();
    expect(find.byType(ResultView), findsOneWidget);
  });

  testWidgets('historique : « Voir plus » charge la page suivante', (tester) async {
    final exams = await pumpHistory(
      tester,
      [for (var i = 1; i <= 25; i++) _item(i, result: _failed)],
    );
    await tester.scrollUntilVisible(find.text('Voir plus'), 300);
    await tester.tap(find.text('Voir plus'));
    await tester.pumpAndSettle();
    expect(exams.pages, [1, 2]);
    await tester.scrollUntilVisible(find.text('Permis B · Série 25'), 300);
    expect(find.text('Voir plus'), findsNothing);
  });
}
