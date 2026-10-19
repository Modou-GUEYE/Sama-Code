import 'package:fake_async/fake_async.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:sama_code/core/api/api_exception.dart';
import 'package:sama_code/core/api/api_client.dart';
import 'package:sama_code/core/storage/token_store.dart';
import 'package:sama_code/data/models/exam.dart';
import 'package:sama_code/data/repositories/exam_repository.dart';
import 'package:sama_code/features/exam/viewmodels/exam_viewmodel.dart';

class FakeRepo extends ExamRepository {
  FakeRepo() : super(ApiClient(baseUrl: 'http://x', tokens: TokenStore()));

  final saved = <String, Set<String>>{};
  int failuresLeft = 0;
  int submits = 0;
  late ExamSession session;

  @override
  Future<void> answer(String sessionId, String questionId, Set<String> optionIds) async {
    if (failuresLeft > 0) {
      failuresLeft--;
      throw ApiException.network();
    }
    saved[questionId] = {...optionIds};
  }

  @override
  Future<ExamSession> submit(String id) async {
    submits++;
    return session;
  }
}

ExamSession sessionJson({Duration left = const Duration(minutes: 20)}) {
  final now = DateTime.now().toUtc();
  Map<String, dynamic> q(String id, String type) => {
        'position': 1,
        'question_id': id,
        'question_type': type,
        'contents': {'fr': {'text': id}},
        'options': [
          for (final c in ['A', 'B', 'C']) {'id': '$id-$c', 'code': c, 'contents': {'fr': {'text': c}}},
        ],
      };
  return ExamSession.fromJson({
    'id': 's1',
    'series_id': 'x',
    'license_code': 'C',
    'series_number': 1,
    'status': 'in_progress',
    'duration_minutes': 20,
    'max_faults': 4,
    'expires_at': now.add(left).toIso8601String(),
    'server_time': now.toIso8601String(),
    'questions': [q('single', 'single'), q('multi', 'multiple')],
    'answers': {},
  });
}

void main() {
  test('réponse unique : remplace le choix, un second appui le retire', () async {
    final repo = FakeRepo();
    final vm = ExamViewModel(repo, sessionJson());
    final opts = vm.current.options;
    vm.toggle(opts[0]);
    vm.toggle(opts[1]);
    expect(vm.answers['single'], {'single-B'});
    vm.toggle(opts[1]);
    expect(vm.answers['single'], isEmpty);
    await Future<void>.delayed(Duration.zero);
    expect(repo.saved['single'], isEmpty);
    vm.dispose();
  });

  test('réponses multiples : coche et décoche indépendamment', () async {
    final vm = ExamViewModel(FakeRepo(), sessionJson());
    vm.goTo(1);
    final opts = vm.current.options;
    vm.toggle(opts[0]);
    vm.toggle(opts[2]);
    expect(vm.answers['multi'], {'multi-A', 'multi-C'});
    vm.toggle(opts[0]);
    expect(vm.answers['multi'], {'multi-C'});
    expect(vm.unansweredCount, 1);
    vm.dispose();
  });

  test('temps écoulé : soumission automatique une seule fois', () async {
    final repo = FakeRepo();
    repo.session = sessionJson(left: Duration.zero);
    final vm = ExamViewModel(repo, sessionJson(left: Duration.zero));
    await Future<void>.delayed(const Duration(milliseconds: 50));
    expect(repo.submits, 1);
    expect(vm.finished, isNotNull);
    await vm.submit();
    expect(repo.submits, 1);
    vm.dispose();
  });

  test('hors connexion : la réponse est renvoyée automatiquement au retour du réseau', () {
    fakeAsync((async) {
      final repo = FakeRepo()..failuresLeft = 2;
      final vm = ExamViewModel(repo, sessionJson());
      vm.toggle(vm.current.options[0]);
      async.flushMicrotasks();
      expect(vm.hasPendingAnswers, isTrue);
      expect(repo.saved, isEmpty);

      async.elapse(const Duration(seconds: 6)); // 1er renvoi : encore hors connexion
      expect(vm.hasPendingAnswers, isTrue);
      async.elapse(const Duration(seconds: 6)); // 2e renvoi : réseau revenu
      expect(vm.hasPendingAnswers, isFalse);
      expect(repo.saved['single'], {'single-A'});
      vm.dispose();
    });
  });
}
