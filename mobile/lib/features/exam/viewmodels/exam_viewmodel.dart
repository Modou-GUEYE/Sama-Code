import 'dart:async';

import 'package:flutter/foundation.dart';

import '../../../core/api/api_exception.dart';
import '../../../data/models/exam.dart';
import '../../../data/repositories/exam_repository.dart';

/// Examen en cours. Le chronomètre suit `expires_at` du serveur (corrigé du décalage
/// entre l'horloge du téléphone et celle du serveur), jamais une durée locale.
class ExamViewModel extends ChangeNotifier {
  ExamViewModel(this._repo, ExamSession session)
    : _session = session,
      _clockOffset = session.serverTime.difference(DateTime.now()),
      answers = {
        for (final e in session.answers.entries) e.key: {...e.value},
      } {
    _tick();
    _ticker = Timer.periodic(const Duration(seconds: 1), (_) => _tick());
  }

  final ExamRepository _repo;
  ExamSession _session;
  final Duration _clockOffset;
  Timer? _ticker;

  final Map<String, Set<String>> answers;
  final Set<String> _unsaved = {};
  int index = 0;
  Duration remaining = Duration.zero;
  bool submitting = false;
  ApiException? error;

  /// Rempli quand l'examen est terminé (soumis par le candidat ou à la fin du temps).
  ExamSession? finished;

  /// Alertes de temps : 5 min puis 1 min (déclenchées une seule fois).
  final _alerts = <int>{};
  void Function(int minutesLeft)? onTimeAlert;

  ExamSession get session => _session;
  List<ExamQuestion> get questions => _session.questions;
  ExamQuestion get current => questions[index];
  bool get isLast => index == questions.length - 1;
  int get answeredCount =>
      questions.where((q) => (answers[q.id] ?? const {}).isNotEmpty).length;
  int get unansweredCount => questions.length - answeredCount;

  /// Réponses pas encore reçues par le serveur (réseau coupé) : renvoyées automatiquement.
  bool get hasPendingAnswers => _unsaved.isNotEmpty;
  int _secondsSinceRetry = 0;
  bool _retrying = false;

  bool isAnswered(ExamQuestion q) => (answers[q.id] ?? const {}).isNotEmpty;
  bool isSelected(ExamOption o) => answers[current.id]?.contains(o.id) ?? false;

  void _tick() {
    final now = DateTime.now().add(_clockOffset);
    remaining = _session.expiresAt.difference(now);
    if (remaining.isNegative) remaining = Duration.zero;
    for (final m in const [5, 1]) {
      if (remaining.inSeconds <= m * 60 &&
          remaining.inSeconds > 0 &&
          _alerts.add(m)) {
        onTimeAlert?.call(m);
      }
    }
    if (_unsaved.isNotEmpty && !_retrying && ++_secondsSinceRetry >= 5) {
      _secondsSinceRetry = 0;
      unawaited(_retryPending());
    }
    if (remaining == Duration.zero && finished == null && !submitting) {
      unawaited(submit());
    }
    notifyListeners();
  }

  void goTo(int i) {
    if (i < 0 || i >= questions.length) return;
    index = i;
    notifyListeners();
  }

  /// Réponse unique : remplace le choix (un second appui le retire).
  /// Réponses multiples : coche ou décoche.
  void toggle(ExamOption option) {
    if (finished != null || submitting) return;
    final q = current;
    final set = answers.putIfAbsent(q.id, () => {});
    if (q.multiple) {
      if (!set.remove(option.id)) set.add(option.id);
    } else {
      final wasSelected = set.contains(option.id);
      set.clear();
      if (!wasSelected) set.add(option.id);
    }
    _unsaved.add(q.id);
    notifyListeners();
    unawaited(_save(q.id));
  }

  Future<void> _retryPending() async {
    _retrying = true;
    for (final id in _unsaved.toList()) {
      await _save(id);
      if (_unsaved.contains(id)) {
        break; // toujours hors connexion : on réessaiera
      }
    }
    _retrying = false;
  }

  Future<void> _save(String questionId) async {
    try {
      await _repo.answer(
        _session.id,
        questionId,
        answers[questionId] ?? const {},
      );
      _unsaved.remove(questionId);
      error = null;
    } on ApiException catch (e) {
      if (e.code == 'session_expired' || e.code == 'session_submitted') {
        await _loadFinished();
      } else {
        error = e; // réessayé au prochain enregistrement ou à la soumission
      }
    }
    notifyListeners();
  }

  /// Soumet l'examen (idempotent côté serveur). Les réponses non encore enregistrées
  /// (réseau coupé) sont renvoyées d'abord.
  Future<void> submit() async {
    if (submitting || finished != null) return;
    submitting = true;
    notifyListeners();
    try {
      for (final id in _unsaved.toList()) {
        await _save(id);
        if (finished != null) return;
      }
      finished = await _repo.submit(_session.id);
      _ticker?.cancel();
      error = null;
    } on ApiException catch (e) {
      error = e;
    } finally {
      submitting = false;
      notifyListeners();
    }
  }

  Future<void> _loadFinished() async {
    try {
      final s = await _repo.get(_session.id);
      _session = s;
      if (s.submitted) {
        finished = s;
        _ticker?.cancel();
      }
    } on ApiException catch (e) {
      error = e;
    }
  }

  // Un enregistrement peut se terminer après la fermeture de l'écran : on ignore
  // alors la notification au lieu de planter.
  bool _disposed = false;

  @override
  void notifyListeners() {
    if (!_disposed) super.notifyListeners();
  }

  @override
  void dispose() {
    _disposed = true;
    _ticker?.cancel();
    super.dispose();
  }
}
