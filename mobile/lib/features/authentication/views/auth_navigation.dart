import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../../core/api/api_exception.dart';
import '../../../core/localization/app_strings.dart';
import '../../../core/session_viewmodel.dart';
import '../viewmodels/auth_flow_viewmodel.dart';

/// Ouvre une étape du parcours en partageant le même ViewModel.
Future<void> pushAuthStep(
  BuildContext context,
  AuthFlowViewModel vm,
  Widget page,
) {
  return Navigator.of(context).push(
    MaterialPageRoute(
      builder: (_) => ChangeNotifierProvider.value(value: vm, child: page),
    ),
  );
}

/// Fin du parcours : retour à la racine, qui affiche l'accueil connecté.
Future<void> finishAuth(BuildContext context) {
  final session = context.read<SessionViewModel>();
  Navigator.of(context).popUntil((route) => route.isFirst);
  return session.onSignedIn();
}

String describeError(AppStrings s, ApiException e) {
  final message = s.error(e.code, e.message);
  final remaining = e.details['remaining_attempts'];
  return remaining == null
      ? message
      : '$message\n${s.t('remaining_attempts')} : $remaining';
}
