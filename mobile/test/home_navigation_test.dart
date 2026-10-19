import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';
import 'package:sama_code/core/api/api_client.dart';
import 'package:sama_code/core/session_viewmodel.dart';
import 'package:sama_code/core/storage/local_preferences.dart';
import 'package:sama_code/core/storage/token_store.dart';
import 'package:sama_code/core/theme/app_theme.dart';
import 'package:sama_code/data/models/course.dart';
import 'package:sama_code/data/models/exam.dart';
import 'package:sama_code/data/models/license_category.dart';
import 'package:sama_code/data/repositories/auth_repository.dart';
import 'package:sama_code/data/repositories/course_repository.dart';
import 'package:sama_code/data/repositories/exam_repository.dart';
import 'package:sama_code/data/repositories/license_repository.dart';
import 'package:sama_code/data/repositories/practice_repository.dart';
import 'package:sama_code/data/repositories/profile_repository.dart';
import 'package:sama_code/features/courses/views/courses_home_view.dart';
import 'package:sama_code/features/exam/views/history_view.dart';
import 'package:sama_code/features/home/views/home_view.dart';
import 'package:sama_code/features/practice/views/practice_home_view.dart';
import 'package:shared_preferences/shared_preferences.dart';

class FakeLicenses extends LicenseRepository {
  FakeLicenses(super.api);
  @override
  Future<List<LicenseCategory>> list() async => const [];
}

class FakeExams extends ExamRepository {
  FakeExams(super.api);
  @override
  Future<ExamSession?> current() async => null;
  @override
  Future<List<ExamHistoryItem>> history({int page = 1, int pageSize = 20}) async => const [];
}

class FakeCourses extends CourseRepository {
  FakeCourses(super.api);
  @override
  Future<List<CourseCategory>> catalog() async => const [];
}

class FakePractice extends PracticeRepository {
  FakePractice(super.api);
  @override
  Future<List<PracticeLicense>> summary() async => const [];
}

Future<void> pumpHome(WidgetTester tester) async {
  SharedPreferences.setMockInitialValues({});
  final prefs = await LocalPreferences.load();
  final tokens = TokenStore();
  final api = ApiClient(baseUrl: 'http://x', tokens: tokens);
  tester.view.physicalSize = const Size(1080, 2400);
  tester.view.devicePixelRatio = 2.625;
  addTearDown(tester.view.reset);
  await tester.pumpWidget(MultiProvider(
    providers: [
      Provider<LicenseRepository>.value(value: FakeLicenses(api)),
      Provider<ExamRepository>.value(value: FakeExams(api)),
      Provider<CourseRepository>.value(value: FakeCourses(api)),
      Provider<PracticeRepository>.value(value: FakePractice(api)),
      ChangeNotifierProvider.value(value: SessionViewModel(prefs, AuthRepository(api, tokens), ProfileRepository(api))),
    ],
    child: MaterialApp(theme: buildAppTheme(), home: const HomeView()),
  ));
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('accueil : les tuiles ont le bon sous-titre', (tester) async {
    await pumpHome(tester);
    expect(find.text('À lire ou à écouter'), findsOneWidget);
    expect(find.text('Avec correction'), findsOneWidget);
    expect(find.text('Bientôt disponible'), findsNothing);
  });

  testWidgets('accueil : la tuile Cours ouvre les cours', (tester) async {
    await pumpHome(tester);
    await tester.tap(find.text('Cours'));
    await tester.pumpAndSettle();
    expect(find.byType(CoursesHomeView), findsOneWidget);
  });

  testWidgets('accueil : la tuile Entraînement ouvre l’entraînement', (tester) async {
    await pumpHome(tester);
    await tester.tap(find.text('Entraînement'));
    await tester.pumpAndSettle();
    expect(find.byType(PracticeHomeView), findsOneWidget);
  });

  testWidgets('accueil : « Mes examens » ouvre l’historique', (tester) async {
    await pumpHome(tester);
    await tester.scrollUntilVisible(find.text('Mes examens'), 200);
    await tester.tap(find.text('Mes examens'));
    await tester.pumpAndSettle();
    expect(find.byType(HistoryView), findsOneWidget);
    expect(find.text('Vous n’avez pas encore passé d’examen.'), findsOneWidget);
  });
}
