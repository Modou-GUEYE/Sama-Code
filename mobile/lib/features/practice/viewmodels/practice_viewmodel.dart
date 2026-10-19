import 'package:flutter/foundation.dart';

import '../../../core/api/api_exception.dart';
import '../../../data/models/exam.dart';
import '../../../data/repositories/practice_repository.dart';

/// Entraînement : une question à la fois ; correction et explication après « Valider ».
class PracticeViewModel extends ChangeNotifier {
  PracticeViewModel(this._repo, this.items);

  final PracticeRepository _repo;
  final List<PracticeItem> items;

  int index = 0;
  Set<String> selected = {};
  PracticeCheck? check;
  bool busy = false;
  ApiException? error;
  int correctCount = 0;
  bool finished = false;

  PracticeItem get current => items[index];
  ExamQuestion get question => current.question;
  bool get isLast => index == items.length - 1;
  bool get checked => check != null;

  void toggle(ExamOption option) {
    if (checked || busy) return;
    if (question.multiple) {
      if (!selected.remove(option.id)) selected.add(option.id);
    } else {
      selected = selected.contains(option.id) ? {} : {option.id};
    }
    notifyListeners();
  }

  /// Corrige la réponse : le serveur renvoie la bonne réponse et l'explication.
  Future<void> validate() async {
    if (selected.isEmpty || checked || busy) return;
    busy = true;
    error = null;
    notifyListeners();
    try {
      check = await _repo.check(current.seriesId, question.id, selected);
      if (check!.outcome == Outcome.correct) correctCount++;
    } on ApiException catch (e) {
      error = e;
    } finally {
      busy = false;
      notifyListeners();
    }
  }

  void next() {
    if (!checked) return;
    if (isLast) {
      finished = true;
    } else {
      index++;
      selected = {};
      check = null;
    }
    notifyListeners();
  }

  // Une correction peut arriver après la fermeture de l'écran.
  bool _disposed = false;

  @override
  void notifyListeners() {
    if (!_disposed) super.notifyListeners();
  }

  @override
  void dispose() {
    _disposed = true;
    super.dispose();
  }
}
