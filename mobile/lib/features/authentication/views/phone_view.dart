import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';

import '../../../core/session_viewmodel.dart';
import '../../../core/theme/app_theme.dart';
import '../../../shared/widgets/auth_scaffold.dart';
import '../../../shared/widgets/feedback.dart';
import '../../../shared/widgets/surfaces.dart';
import '../viewmodels/auth_flow_viewmodel.dart';
import 'auth_navigation.dart';
import 'otp_view.dart';
import 'pin_views.dart';

class PhoneView extends StatefulWidget {
  const PhoneView({super.key});

  @override
  State<PhoneView> createState() => _PhoneViewState();
}

class _PhoneViewState extends State<PhoneView> {
  late final _controller = TextEditingController(
    text: SenegalPhoneFormatter.format(context.read<AuthFlowViewModel>().phone),
  );

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

  bool get _complete =>
      SenegalPhoneFormatter.digits(_controller.text).length == 9;

  Future<void> _submit() async {
    if (!_complete) return;
    final vm = context.read<AuthFlowViewModel>();
    final phone = SenegalPhoneFormatter.digits(_controller.text);
    if (vm.flow == AuthFlow.pinLogin) {
      if (!vm.acceptPhone(phone)) return;
      await pushAuthStep(context, vm, const PinLoginView());
      return;
    }
    if (await vm.requestCode(phone) && mounted) {
      await pushAuthStep(context, vm, const OtpView());
    }
  }

  @override
  Widget build(BuildContext context) {
    final vm = context.watch<AuthFlowViewModel>();
    final s = context.watch<SessionViewModel>().strings;
    final pinLogin = vm.flow == AuthFlow.pinLogin;
    final register = vm.flow == AuthFlow.register;

    return AuthScaffold(
      step: register ? 1 : null,
      totalSteps: register ? registrationSteps : null,
      icon: Icons.phone_android_rounded,
      title: s.t('phone_title'),
      subtitle: s.t(pinLogin ? 'phone_login_hint' : 'phone_sms_hint'),
      bottom: PrimaryButton(
        label: pinLogin ? s.t('next') : s.t('get_code'),
        icon: pinLogin ? Icons.arrow_forward_rounded : Icons.sms_outlined,
        busy: vm.busy,
        onPressed: _complete ? _submit : null,
      ),
      children: [
        SoftCard(
          padding: const EdgeInsets.fromLTRB(10, 10, 16, 10),
          child: Row(
            children: [
              Container(
                padding: const EdgeInsets.symmetric(
                  horizontal: 14,
                  vertical: 14,
                ),
                decoration: BoxDecoration(
                  color: const Color(0xFFE3F2E9),
                  borderRadius: BorderRadius.circular(16),
                ),
                child: const Text(
                  '🇸🇳 +221',
                  style: TextStyle(
                    fontSize: 19,
                    fontWeight: FontWeight.w700,
                    color: AppColors.greenDark,
                  ),
                ),
              ),
              const SizedBox(width: 14),
              Expanded(
                child: TextField(
                  controller: _controller,
                  autofocus: true,
                  keyboardType: TextInputType.phone,
                  autofillHints: const [AutofillHints.telephoneNumber],
                  inputFormatters: [SenegalPhoneFormatter()],
                  style: const TextStyle(
                    fontSize: 28,
                    fontWeight: FontWeight.w700,
                    letterSpacing: 1,
                    color: Ink2.strong,
                  ),
                  decoration: const InputDecoration(
                    hintText: '77 123 45 67',
                    hintStyle: TextStyle(color: Color(0xFFB5BBB1)),
                    filled: false,
                    border: InputBorder.none,
                    enabledBorder: InputBorder.none,
                    focusedBorder: InputBorder.none,
                    contentPadding: EdgeInsets.zero,
                    isDense: true,
                  ),
                  onChanged: (_) => vm.clearError(),
                  onSubmitted: (_) => _submit(),
                ),
              ),
            ],
          ),
        ),
        if (vm.error != null) ...[
          const SizedBox(height: 16),
          ErrorMessage(describeError(s, vm.error!)),
        ],
      ],
    );
  }
}

/// Met en forme un numéro mobile sénégalais pendant la saisie : 77 123 45 67.
/// Ignore l'indicatif 221 s'il est collé, et limite à 9 chiffres.
class SenegalPhoneFormatter extends TextInputFormatter {
  static String digits(String text) {
    var d = text.replaceAll(RegExp(r'\D'), '');
    if (d.startsWith('00221')) d = d.substring(5);
    if (d.startsWith('221') && d.length > 9) d = d.substring(3);
    return d.length > 9 ? d.substring(0, 9) : d;
  }

  static String format(String text) {
    final d = digits(text);
    final b = StringBuffer();
    for (var i = 0; i < d.length; i++) {
      if (i == 2 || i == 5 || i == 7) b.write(' ');
      b.write(d[i]);
    }
    return b.toString();
  }

  @override
  TextEditingValue formatEditUpdate(
    TextEditingValue oldValue,
    TextEditingValue newValue,
  ) {
    final text = format(newValue.text);
    return TextEditingValue(
      text: text,
      selection: TextSelection.collapsed(offset: text.length),
    );
  }
}
