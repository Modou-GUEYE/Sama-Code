import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../../core/session_viewmodel.dart';
import '../../../shared/widgets/auth_scaffold.dart';
import '../../../shared/widgets/feedback.dart';
import '../../../shared/widgets/pin_pad.dart';
import '../viewmodels/auth_flow_viewmodel.dart';
import 'auth_navigation.dart';
import 'otp_view.dart';

/// Choix d'un nouveau PIN (inscription ou PIN oublié) : saisie puis confirmation.
class NewPinView extends StatefulWidget {
  const NewPinView({super.key});

  @override
  State<NewPinView> createState() => _NewPinViewState();
}

class _NewPinViewState extends State<NewPinView> {
  String? _first;
  String? _localErrorKey;
  int _resets = 0;

  Future<void> _onCompleted(String pin) async {
    if (_first == null) {
      setState(() {
        if (isWeakPin(pin)) {
          _localErrorKey = 'error_weak_pin';
          _resets++;
        } else {
          _first = pin;
          _localErrorKey = null;
        }
      });
      return;
    }
    if (pin != _first) {
      setState(() {
        _first = null;
        _localErrorKey = 'pin_mismatch';
        _resets++;
      });
      return;
    }

    final session = context.read<SessionViewModel>();
    final vm = context.read<AuthFlowViewModel>();
    final ok = await vm.submitNewPin(
      pin,
      language: session.language,
      learningMode: session.learningMode,
    );
    if (!mounted) return;
    if (ok) {
      await finishAuth(context);
    } else {
      setState(() => _first = null);
    }
  }

  @override
  Widget build(BuildContext context) {
    final vm = context.watch<AuthFlowViewModel>();
    final s = context.watch<SessionViewModel>().strings;
    final confirming = _first != null;
    final error = _localErrorKey != null
        ? s.t(_localErrorKey!)
        : (vm.error != null ? describeError(s, vm.error!) : null);

    return AuthScaffold(
      step: vm.flow == AuthFlow.register ? 4 : null,
      totalSteps: registrationSteps,
      icon: confirming ? Icons.lock_reset_rounded : Icons.lock_rounded,
      title: s.t(confirming ? 'pin_confirm' : 'pin_create'),
      subtitle: confirming ? null : s.t('pin_create_help'),
      children: [
        if (error != null) ...[ErrorMessage(error), const SizedBox(height: 16)],
        PinPad(
          key: ValueKey('new-pin-$confirming-$_resets-${vm.attempt}'),
          length: 4,
          enabled: !vm.busy,
          onCompleted: _onCompleted,
        ),
        if (vm.busy)
          const Padding(
            padding: EdgeInsets.all(16),
            child: Center(child: CircularProgressIndicator()),
          ),
      ],
    );
  }
}

/// Connexion par PIN, avec accès au code SMS si le compte est bloqué ou le PIN oublié.
class PinLoginView extends StatelessWidget {
  const PinLoginView({super.key});

  Future<void> _login(BuildContext context, String pin) async {
    final vm = context.read<AuthFlowViewModel>();
    if (await vm.loginWithPin(pin) && context.mounted) {
      await finishAuth(context);
    }
  }

  Future<void> _switchToSms(BuildContext context, AuthFlow flow) async {
    final vm = context.read<AuthFlowViewModel>()..switchTo(flow);
    if (await vm.requestCode() && context.mounted) {
      await pushAuthStep(context, vm, const OtpView());
    }
  }

  @override
  Widget build(BuildContext context) {
    final vm = context.watch<AuthFlowViewModel>();
    final s = context.watch<SessionViewModel>().strings;
    final locked = vm.error?.code == 'account_locked';
    return AuthScaffold(
      icon: Icons.lock_open_rounded,
      title: s.t('pin_enter'),
      subtitle: vm.phone,
      children: [
        if (vm.error != null) ...[
          ErrorMessage(describeError(s, vm.error!)),
          const SizedBox(height: 16),
        ],
        if (locked)
          PrimaryButton(
            label: s.t('sms_login'),
            icon: Icons.sms_outlined,
            busy: vm.busy,
            onPressed: () => _switchToSms(context, AuthFlow.otpLogin),
          )
        else
          PinPad(
            key: ValueKey('login-${vm.attempt}'),
            length: 4,
            enabled: !vm.busy,
            onCompleted: (pin) => _login(context, pin),
          ),
        const SizedBox(height: 12),
        TextButton.icon(
          icon: const Icon(Icons.help_outline_rounded, size: 26),
          label: Text(s.t('forgot_pin')),
          onPressed: vm.busy
              ? null
              : () => _switchToSms(context, AuthFlow.resetPin),
        ),
      ],
    );
  }
}
