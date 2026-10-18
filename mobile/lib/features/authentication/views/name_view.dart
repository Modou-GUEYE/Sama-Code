import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../../core/session_viewmodel.dart';
import '../../../shared/widgets/auth_scaffold.dart';
import '../../../shared/widgets/feedback.dart';
import '../../../shared/widgets/surfaces.dart';
import '../viewmodels/auth_flow_viewmodel.dart';
import 'auth_navigation.dart';
import 'pin_views.dart';

class NameView extends StatefulWidget {
  const NameView({super.key});

  @override
  State<NameView> createState() => _NameViewState();
}

class _NameViewState extends State<NameView> {
  final _controller = TextEditingController();

  @override
  void initState() {
    super.initState();
    _controller.addListener(() => setState(() {}));
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  void _submit() {
    final name = _controller.text.trim();
    if (name.isEmpty) return;
    final vm = context.read<AuthFlowViewModel>()..displayName = name;
    pushAuthStep(context, vm, const NewPinView());
  }

  @override
  Widget build(BuildContext context) {
    final s = context.watch<SessionViewModel>().strings;
    return AuthScaffold(
      step: 3,
      totalSteps: registrationSteps,
      icon: Icons.badge_rounded,
      title: s.t('name_title'),
      bottom: PrimaryButton(
        label: s.t('next'),
        icon: Icons.arrow_forward_rounded,
        onPressed: _controller.text.trim().isEmpty ? null : _submit,
      ),
      children: [
        TextField(
          controller: _controller,
          autofocus: true,
          maxLength: 80,
          textCapitalization: TextCapitalization.words,
          autofillHints: const [AutofillHints.name],
          style: const TextStyle(fontSize: 24, color: Ink2.strong),
          decoration: InputDecoration(
            hintText: s.t('name_hint'),
            counterText: '',
          ),
          onSubmitted: (_) => _submit(),
        ),
      ],
    );
  }
}
