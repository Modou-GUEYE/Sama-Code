import 'package:flutter_test/flutter_test.dart';
import 'package:sama_code/data/models/course.dart';
import 'package:sama_code/data/models/exam.dart';

// D25 : le texte est toujours en français ; le wolof n'existe qu'en audio.
void main() {
  const contents = {
    'fr': LocalizedContent(
      text: 'Que signifie ce panneau ?',
      audioUrl: 'fr.mp3',
      imageUrl: 'img.png',
    ),
    'wo': LocalizedContent(text: '', audioUrl: 'wo.mp3'),
  };

  test(
    'question : texte et image en français, audio dans la langue d’écoute',
    () {
      final wo = pick(contents, 'wo');
      expect(wo.text, 'Que signifie ce panneau ?');
      expect(wo.imageUrl, 'img.png');
      expect(wo.audioUrl, 'wo.mp3');
      expect(pick(contents, 'fr').audioUrl, 'fr.mp3');
    },
  );

  test('sans audio wolof, l’audio français est proposé', () {
    expect(pick({'fr': contents['fr']!}, 'wo').audioUrl, 'fr.mp3');
  });

  test('titre : en français, avec l’audio wolof', () {
    final label = pickLabel(const {
      'fr': Label(title: 'Le feu rouge', audioUrl: 'fr.mp3'),
      'wo': Label(title: '', audioUrl: 'wo.mp3'),
    }, 'wo');
    expect(label.title, 'Le feu rouge');
    expect(label.audioUrl, 'wo.mp3');
  });

  test('leçon : les audios wolof remplacent les audios français, en tête', () {
    const lesson = Lesson(
      id: 'l',
      courseId: 'c',
      labels: {},
      completed: false,
      blocks: {
        'fr': [
          LessonBlock(type: 'image', mediaUrl: 'img.png'),
          LessonBlock(type: 'audio', mediaUrl: 'fr.mp3'),
          LessonBlock(type: 'text', text: 'Le feu rouge impose l’arrêt.'),
        ],
        'wo': [LessonBlock(type: 'audio', mediaUrl: 'wo.mp3')],
      },
    );
    expect(lesson.blocksFor('wo').map((b) => b.mediaUrl ?? b.text), [
      'wo.mp3',
      'img.png',
      'Le feu rouge impose l’arrêt.',
    ]);
    expect(lesson.blocksFor('fr').length, 3);
  });
}
