import 'package:flutter/material.dart';

/// Panneau du bas de l'application : fond blanc, poignée, hauteur adaptée au contenu
/// jusqu'à 85 % de l'écran, puis défilement. Évite les débordements quand le contenu
/// est long (25 questions, petit écran, grande taille de police).
Future<T?> showAppSheet<T>({
  required BuildContext context,
  required WidgetBuilder builder,
}) {
  return showModalBottomSheet<T>(
    context: context,
    isScrollControlled: true,
    showDragHandle: true,
    backgroundColor: Colors.white,
    builder: (sheet) => ConstrainedBox(
      constraints: BoxConstraints(
        maxHeight: MediaQuery.sizeOf(sheet).height * 0.85,
      ),
      child: SingleChildScrollView(child: builder(sheet)),
    ),
  );
}
