import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../../core/theme/app_theme.dart';
import 'surfaces.dart';

/// Pavé numérique géant pour les codes (PIN à 4 chiffres, code SMS à 6 chiffres).
/// Pas de clavier système : utilisable sans savoir lire, chiffres bien visibles.
/// Pour le vider (après une erreur), reconstruire avec une nouvelle `key`.
class PinPad extends StatefulWidget {
  const PinPad({
    super.key,
    required this.length,
    required this.onCompleted,
    this.enabled = true,
    this.obscure = true,
  });

  final int length;
  final ValueChanged<String> onCompleted;
  final bool enabled;
  final bool obscure;

  @override
  State<PinPad> createState() => _PinPadState();
}

class _PinPadState extends State<PinPad> {
  String _value = '';

  void _press(String digit) {
    if (!widget.enabled || _value.length >= widget.length) return;
    HapticFeedback.lightImpact();
    setState(() => _value += digit);
    if (_value.length == widget.length) widget.onCompleted(_value);
  }

  void _erase() {
    if (!widget.enabled || _value.isEmpty) return;
    HapticFeedback.selectionClick();
    setState(() => _value = _value.substring(0, _value.length - 1));
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        Semantics(
          label: '${_value.length} / ${widget.length}',
          child: Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: List.generate(widget.length, _slot),
          ),
        ),
        const SizedBox(height: 28),
        for (final row in const [
          ['1', '2', '3'],
          ['4', '5', '6'],
          ['7', '8', '9'],
        ])
          Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: row.map(_digitKey).toList(),
          ),
        Row(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const SizedBox(width: 96, height: 80),
            _digitKey('0'),
            _key(
              onTap: _erase,
              semantics: 'Effacer',
              plain: true,
              child: const Icon(
                Icons.backspace_outlined,
                size: 30,
                color: Ink2.strong,
              ),
            ),
          ],
        ),
      ],
    );
  }

  Widget _slot(int index) {
    final filled = index < _value.length;
    final current = index == _value.length && widget.enabled;
    final compact = widget.length > 4;
    return AnimatedContainer(
      duration: const Duration(milliseconds: 150),
      margin: EdgeInsets.symmetric(horizontal: compact ? 4 : 8),
      width: compact ? 46 : 58,
      height: compact ? 58 : 66,
      alignment: Alignment.center,
      decoration: BoxDecoration(
        color: filled ? const Color(0xFFE3F2E9) : Colors.white,
        borderRadius: BorderRadius.circular(16),
        boxShadow: filled
            ? const []
            : const [
                BoxShadow(
                  color: Color(0x0D000000),
                  blurRadius: 12,
                  offset: Offset(0, 4),
                ),
              ],
      ),
      child: filled
          ? (widget.obscure
                ? const Icon(Icons.circle, size: 16, color: Ink2.strong)
                : Text(
                    _value[index],
                    style: const TextStyle(
                      fontSize: 26,
                      fontWeight: FontWeight.w800,
                    ),
                  ))
          : (current
                ? Container(width: 18, height: 3, color: AppColors.green)
                : null),
    );
  }

  Widget _digitKey(String digit) => _key(
    onTap: () => _press(digit),
    semantics: digit,
    child: Text(
      digit,
      style: const TextStyle(
        fontSize: 30,
        fontWeight: FontWeight.w700,
        color: Ink2.strong,
      ),
    ),
  );

  Widget _key({
    required VoidCallback onTap,
    required String semantics,
    required Widget child,
    bool plain = false,
  }) {
    return Padding(
      padding: const EdgeInsets.all(6),
      child: Semantics(
        button: true,
        label: semantics,
        excludeSemantics: true,
        child: Material(
          color: plain ? Colors.transparent : Colors.white,
          borderRadius: BorderRadius.circular(24),
          child: InkWell(
            borderRadius: BorderRadius.circular(24),
            onTap: widget.enabled ? onTap : null,
            child: SizedBox(width: 84, height: 68, child: Center(child: child)),
          ),
        ),
      ),
    );
  }
}
