import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../../core/session_viewmodel.dart';
import '../../../shared/widgets/auth_scaffold.dart';
import '../../../shared/widgets/feedback.dart';
import '../../../shared/widgets/pin_pad.dart';
import '../viewmodels/auth_flow_viewmodel.dart';
import 'auth_navigation.dart';
import 'name_view.dart';
import 'pin_views.dart';

/// Saisie du code SMS à 6 chiffres.
class OtpView extends StatelessWidget {
  const OtpView({super.key});

  Future<void> _verify(BuildContext context, String code) async {
    final vm = context.read<AuthFlowViewModel>();
    if (!await vm.verifyCode(code) || !context.mounted) return;
    switch (vm.flow) {
      case AuthFlow.register:
        await pushAuthStep(context, vm, const NameView());
      case AuthFlow.resetPin:
        await pushAuthStep(context, vm, const NewPinView());
      case AuthFlow.otpLogin:
      case AuthFlow.pinLogin:
        await finishAuth(context);
    }
  }

  @override
  Widget build(BuildContext context) {
    final vm = context.watch<AuthFlowViewModel>();
    final s = context.watch<SessionViewModel>().strings;
    return AuthScaffold(
      step: vm.flow == AuthFlow.register ? 2 : null,
      totalSteps: registrationSteps,
      icon: Icons.sms_rounded,
      title: s.t('otp_title'),
      subtitle: '${s.t('otp_sent_to')} ${vm.phone}',
      children: [
        if (vm.devCode != null) ...[
          InfoMessage(
            '${s.t('dev_code')} : ${vm.devCode}',
            icon: Icons.bug_report_outlined,
          ),
          const SizedBox(height: 12),
        ],
        if (vm.error != null) ...[
          ErrorMessage(describeError(s, vm.error!)),
          const SizedBox(height: 12),
        ],
        const SizedBox(height: 4),
        PinPad(
          key: ValueKey('otp-${vm.attempt}'),
          length: 6,
          obscure: false,
          enabled: !vm.busy,
          onCompleted: (code) => _verify(context, code),
        ),
        const SizedBox(height: 12),
        TextButton.icon(
          icon: const Icon(Icons.refresh_rounded, size: 26),
          label: Text(s.t('resend_code')),
          onPressed: vm.busy ? null : () => vm.requestCode(),
        ),
      ],
    );
  }
}
