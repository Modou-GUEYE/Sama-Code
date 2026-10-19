import 'package:flutter_test/flutter_test.dart';
import 'package:sama_code/core/api/api_client.dart';
import 'package:sama_code/core/storage/token_store.dart';
import 'package:sama_code/data/models/exam.dart';
import 'package:sama_code/data/repositories/practice_repository.dart';
import 'package:sama_code/features/practice/viewmodels/practice_viewmodel.dart';

class FakeRepo extends PracticeRepository {
  FakeRepo() : super(ApiClient(baseUrl: 'http://x', tokens: TokenStore()));

  @override
  Future<PracticeCheck> check(String seriesId, String questionId, Set<String> optionIds) async =>
      PracticeCheck.fromJson({
        'outcome': optionIds.contains('$questionId-A') ? 'correct' : 'incorrect',
        'correct_option_ids': ['$questionId-A'],
        'explanations': {},
      });
}

PracticeItem item(String id, {String? last}) => PracticeItem.fromJson({
      'series_id': 's',
      'last_outcome': last,
      'question': {
        'position': 1,
        'question_id': id,
        'question_type': 'single',
        'contents': {'fr': {'text': id}},
        'options': [
          for (final c in ['A', 'B']) {'id': '$id-$c', 'code': c, 'contents': {'fr': {'text': c}}},
        ],
      },
    });

void main() {
  test('valider, corriger, compter les bonnes réponses, terminer', () async {
    final vm = PracticeViewModel(FakeRepo(), [item('q1'), item('q2')]);
    await vm.validate(); // rien de coché : ignoré
    expect(vm.checked, isFalse);

    vm.toggle(vm.question.options[0]);
    await vm.validate();
    expect(vm.check!.outcome, Outcome.correct);
    vm.toggle(vm.question.options[1]); // verrouillé après correction
    expect(vm.selected, {'q1-A'});

    vm.next();
    vm.toggle(vm.question.options[1]);
    await vm.validate();
    expect(vm.check!.outcome, Outcome.incorrect);
    expect(vm.correctCount, 1);

    vm.next();
    expect(vm.finished, isTrue);
  });

  test('reprise : démarre à la question demandée et garde les bonnes réponses déjà données', () {
    final vm = PracticeViewModel(
      FakeRepo(),
      [item('q1', last: 'correct'), item('q2', last: 'incorrect'), item('q3')],
      startIndex: 2,
    );
    expect(vm.question.id, 'q3');
    expect(vm.correctCount, 1);
  });
}
