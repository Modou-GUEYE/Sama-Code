import 'dart:async';

import 'package:flutter/material.dart';
import 'package:just_audio/just_audio.dart';

import '../../core/config.dart';
import '../../core/theme/app_theme.dart';

/// Gros bouton « Écouter » : lit un audio de la question ou de l'explication.
/// [autoPlay] lance la lecture dès l'affichage (mode audio, décision D20).
class ListenButton extends StatefulWidget {
  const ListenButton({
    super.key,
    required this.url,
    required this.label,
    this.autoPlay = false,
  });

  final String url;
  final String label;
  final bool autoPlay;

  @override
  State<ListenButton> createState() => _ListenButtonState();
}

class _ListenButtonState extends State<ListenButton> {
  final _player = AudioPlayer();
  StreamSubscription<PlayerState>? _sub;
  bool _playing = false;
  bool _failed = false;

  @override
  void initState() {
    super.initState();
    _sub = _player.playerStateStream.listen((s) {
      final playing =
          s.playing && s.processingState != ProcessingState.completed;
      if (mounted && playing != _playing) setState(() => _playing = playing);
    });
    _load();
  }

  @override
  void didUpdateWidget(ListenButton old) {
    super.didUpdateWidget(old);
    if (old.url != widget.url) _load();
  }

  Future<void> _load() async {
    try {
      await _player.setUrl(AppConfig.mediaUrl(widget.url));
      if (widget.autoPlay) await _player.play();
    } catch (_) {
      if (mounted) setState(() => _failed = true);
    }
  }

  Future<void> _toggle() async {
    if (_playing) {
      await _player.pause();
    } else {
      await _player.seek(Duration.zero);
      await _player.play();
    }
  }

  @override
  void dispose() {
    _sub?.cancel();
    _player.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (_failed) return const SizedBox.shrink();
    return Semantics(
      button: true,
      label: widget.label,
      excludeSemantics: true,
      child: Material(
        color: AppColors.orange.withValues(alpha: 0.14),
        shape: const StadiumBorder(),
        child: InkWell(
          customBorder: const StadiumBorder(),
          onTap: _toggle,
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 12),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(
                  _playing ? Icons.pause_rounded : Icons.volume_up_rounded,
                  size: 28,
                  color: const Color(0xFFB45F00),
                ),
                const SizedBox(width: 10),
                Text(
                  widget.label,
                  style: const TextStyle(
                    fontSize: 18,
                    fontWeight: FontWeight.w700,
                    color: Color(0xFFB45F00),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
