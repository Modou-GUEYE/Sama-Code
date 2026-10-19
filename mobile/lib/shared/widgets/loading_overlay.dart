import 'package:flutter/material.dart';

import '../../core/theme/app_theme.dart';
import 'surfaces.dart';

/// Chargement par-dessus l'écran (et non dans le bouton) : voile clair qui bloque
/// les appuis, avec un indicateur et un message facultatif.
class LoadingOverlay extends StatelessWidget {
  const LoadingOverlay({
    super.key,
    required this.loading,
    required this.child,
    this.message,
  });

  final bool loading;
  final Widget child;
  final String? message;

  @override
  Widget build(BuildContext context) {
    return Stack(
      children: [
        child,
        if (loading)
          Positioned.fill(
            child: Semantics(
              liveRegion: true,
              label: message,
              child: ColoredBox(
                color: Colors.white.withValues(alpha: 0.7),
                child: Center(
                  child: SoftCard(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 28,
                      vertical: 24,
                    ),
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        const SizedBox(
                          width: 44,
                          height: 44,
                          child: CircularProgressIndicator(
                            strokeWidth: 4,
                            color: AppColors.green,
                          ),
                        ),
                        if (message != null) ...[
                          const SizedBox(height: 16),
                          Text(
                            message!,
                            textAlign: TextAlign.center,
                            style: const TextStyle(
                              fontSize: 17,
                              fontWeight: FontWeight.w600,
                              color: Ink2.strong,
                            ),
                          ),
                        ],
                      ],
                    ),
                  ),
                ),
              ),
            ),
          ),
      ],
    );
  }
}
