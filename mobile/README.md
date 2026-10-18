# Sama Code — application mobile

Application Flutter (Android, iOS) organisée en MVVM : `lib/core` (API, stockage,
traductions fr/wo, thème), `lib/data` (modèles, repositories), `lib/features`
(un dossier par écran, avec `views/` et `viewmodels/`), `lib/shared/widgets`.

## Lancer sur un téléphone Android branché en USB

L'API doit tourner sur le Mac (`cd ../backend && make run`).

```bash
adb reverse tcp:8080 tcp:8080     # le téléphone joint l'API par le câble
flutter run --dart-define=API_BASE_URL=http://localhost:8080/api/v1
```

Sans `--dart-define`, l'app vise `http://10.0.2.2:8080/api/v1` (API locale vue
depuis l'émulateur Android).

## Tests et qualité

```bash
flutter analyze
flutter test
```
