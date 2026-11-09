# Portage Linux x86-64

L’application utilise le backend Wails purego avec GTK4 et WebKitGTK 6.0. Elle conserve les données dans le dossier `data` à côté de l’exécutable. L’option `--data` permet de choisir un autre dossier. Un verrou `flock` protège chaque dossier de données. Le bouton de réduction utilise la barre des tâches du bureau. Fermer la fenêtre quitte l’application.

Le coffre Linux chiffre les comptes et les caches avec AES-256-GCM. Il conserve l’identifiant aléatoire du coffre et les fichiers chiffrés dans le dossier de données. La clé est stockée dans le trousseau de session via `secret-tool` et transmise par un flux privé, sans argument de commande. Il faut un service Secret Service disponible et déverrouillé pour enregistrer ou ouvrir un compte. Les fichiers Windows conservent leur format DPAPI. Pour utiliser le même compte sur Linux, configurez-le dans un dossier de données Linux distinct.

## Vérifications du portage

- `./build-linux.ps1` a compilé le TypeScript et l’exécutable Linux x86-64 avec `CGO_ENABLED=0` et `-tags purego`.
- Avec `GOOS=linux`, `GOARCH=amd64` et `CGO_ENABLED=0`, `go test -tags purego -c -o bin/linux-tests/ ./...` et `go vet -tags purego ./...` ont réussi.
- Les tests Go du chiffrement du coffre Linux ont été exécutés sur l’hôte Windows à partir de la liste explicite des sources du coffre. Ils ont vérifié la relecture après réouverture, la suppression, le stockage chiffré et la préservation d’un secret après un refus du trousseau. Le test remplace l’appel à Secret Service par un double de test.
- `python scripts/fetch-tools.py --platform linux` a récupéré gallery-dl v1.32.19, yt-dlp 2026.10.26, Deno v2.9.8 et FFmpeg/ffprobe autobuild-2026-11-08-18-52. Les téléchargements ont été contrôlés avec les SHA-256 publiés.
- Une archive d’essai `tar.gz` a été relue et contrôlée par SHA-256. Le lanceur et les cinq outils ont le mode `0755`. Les autres fichiers ont le mode `0644`.
- Les six binaires ont été examinés comme ELF x86-64 et ciblent le chargeur glibc. Les moteurs autonomes utilisent des bibliothèques système, dont zlib et le runtime GCC. Les imports FFmpeg atteignent GLIBC_2.28. GTK4 et WebKitGTK 6.0 sont chargés par Wails au démarrage.

L’hôte de développement est Windows. Le lancement du bureau Linux, l’interaction avec un vrai trousseau Secret Service et l’arrêt d’un groupe de processus sur un noyau Linux restent à vérifier. Le test Linux de groupe de processus est compilé et peut être exécuté avec `go test -tags purego ./internal/process` depuis une session Linux.
