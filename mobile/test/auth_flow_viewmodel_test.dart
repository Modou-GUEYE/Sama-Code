import 'package:flutter_test/flutter_test.dart';
import 'package:sama_code/features/authentication/viewmodels/auth_flow_viewmodel.dart';

void main() {
  test('isWeakPin applique les mêmes règles que le serveur', () {
    for (final pin in ['0000', '7777', '1234', '6789', '4321', '9876', '0123']) {
      expect(isWeakPin(pin), isTrue, reason: pin);
    }
    for (final pin in ['2580', '1357', '9071', '1212', '4096']) {
      expect(isWeakPin(pin), isFalse, reason: pin);
    }
  });
}
