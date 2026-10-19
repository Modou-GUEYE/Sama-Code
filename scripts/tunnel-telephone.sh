#!/bin/zsh
# Garde le téléphone relié à l'API locale par le câble USB : rétablit
# `adb reverse tcp:8080` dès qu'il saute (débranchement, changement de mode USB…).
# À laisser tourner dans un terminal pendant les tests. Ctrl+C pour arrêter.
ADB="${ADB:-$HOME/Library/Android/sdk/platform-tools/adb}"
echo "Tunnel API ↔ téléphone actif (Ctrl+C pour arrêter)"
while true; do
  if "$ADB" get-state >/dev/null 2>&1; then
    if ! "$ADB" reverse --list 2>/dev/null | grep -q "tcp:8080"; then
      "$ADB" reverse tcp:8080 tcp:8080 >/dev/null 2>&1 && echo "$(date +%H:%M:%S) lien rétabli"
    fi
  fi
  sleep 2
done
