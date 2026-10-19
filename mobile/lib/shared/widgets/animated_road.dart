import 'package:flutter/material.dart';

import '../../core/theme/app_theme.dart';

/// Route stylisée en fond d'écran (bienvenue, À propos) : pointillés qui défilent et
/// petite voiture. Figée si le système demande de réduire les animations.
class AnimatedRoad extends StatefulWidget {
  const AnimatedRoad({super.key});

  @override
  State<AnimatedRoad> createState() => _AnimatedRoadState();
}

class _AnimatedRoadState extends State<AnimatedRoad>
    with SingleTickerProviderStateMixin {
  late final _road = AnimationController(
    vsync: this,
    duration: const Duration(seconds: 9),
  );

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (MediaQuery.disableAnimationsOf(context)) {
      _road
        ..stop()
        ..value = 0.35;
    } else if (!_road.isAnimating) {
      _road.repeat();
    }
  }

  @override
  void dispose() {
    _road.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) =>
      ExcludeSemantics(child: CustomPaint(painter: _RoadPainter(_road)));
}

/// Route courbe en vert pâle, ligne médiane blanche qui défile, petite voiture.
class _RoadPainter extends CustomPainter {
  _RoadPainter(this.progress) : super(repaint: progress);

  final Animation<double> progress;

  static const _dash = 22.0;
  static const _gap = 20.0;
  static const _roadWidth = 78.0;

  Path _path(Size size) => Path()
    ..moveTo(-60, size.height * 0.80)
    ..cubicTo(
      size.width * 0.30,
      size.height * 0.62,
      size.width * 0.70,
      size.height * 1.02,
      size.width + 60,
      size.height * 0.84,
    );

  @override
  void paint(Canvas canvas, Size size) {
    final path = _path(size);

    canvas.drawPath(
      path,
      Paint()
        ..color = AppColors.green.withValues(alpha: 0.10)
        ..style = PaintingStyle.stroke
        ..strokeWidth = _roadWidth,
    );

    final metric = path.computeMetrics().first;
    final dashPaint = Paint()
      ..color = Colors.white
      ..style = PaintingStyle.stroke
      ..strokeWidth = 5
      ..strokeCap = StrokeCap.round;
    // Les pointillés reculent pendant que la voiture avance : effet de mouvement.
    final period = _dash + _gap;
    var start = -(progress.value * metric.length * 2) % period - period;
    for (; start < metric.length; start += period) {
      final from = start.clamp(0.0, metric.length);
      final to = (start + _dash).clamp(0.0, metric.length);
      if (to > from) canvas.drawPath(metric.extractPath(from, to), dashPaint);
    }

    // Petite voiture vue du dessus, sur la voie de droite, orientée selon la route.
    final tangent = metric.getTangentForOffset(progress.value * metric.length);
    if (tangent == null) return;
    final lane = Offset(-tangent.vector.dy, tangent.vector.dx) * 17;
    canvas
      ..save()
      ..translate(tangent.position.dx + lane.dx, tangent.position.dy + lane.dy)
      ..rotate(-tangent.angle);
    _paintCar(canvas);
    canvas.restore();
  }

  /// Voiture de 36 × 18, avant tourné vers +x, centrée sur l'origine.
  static void _paintCar(Canvas canvas) {
    canvas.drawRRect(
      RRect.fromRectAndRadius(
        const Rect.fromLTWH(-17, -8, 36, 18),
        const Radius.circular(6),
      ),
      Paint()..color = const Color(0x22000000),
    );
    canvas.drawRRect(
      RRect.fromRectAndRadius(
        const Rect.fromLTWH(-18, -9, 36, 18),
        const Radius.circular(6),
      ),
      Paint()..color = AppColors.orange,
    );
    final glass = Paint()..color = const Color(0xFF3B4A54);
    // Pare-brise (avant) et lunette arrière.
    canvas.drawRRect(
      RRect.fromRectAndRadius(
        const Rect.fromLTWH(3, -6.5, 7, 13),
        const Radius.circular(2.5),
      ),
      glass,
    );
    canvas.drawRRect(
      RRect.fromRectAndRadius(
        const Rect.fromLTWH(-13, -6, 5, 12),
        const Radius.circular(2),
      ),
      glass,
    );
  }

  @override
  bool shouldRepaint(_RoadPainter oldDelegate) => false;
}
