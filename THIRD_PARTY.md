# Composants tiers

Les licences et notices distribuées dans `tools/licenses/` et `licenses/`
restent applicables à leurs composants respectifs. Le code de l'application
et les programmes externes sont distincts ; les moteurs sont lancés comme processus.

| Composant | Origine du code / des binaires |
| --- | --- |
| gallery-dl | https://codeberg.org/mikf/gallery-dl |
| yt-dlp | https://github.com/yt-dlp/yt-dlp |
| FFmpeg / ffprobe | https://github.com/yt-dlp/FFmpeg-Builds ; https://github.com/FFmpeg/FFmpeg |
| Deno | https://github.com/denoland/deno |
| Go | https://go.dev/ |
| Wails | https://github.com/wailsapp/wails |
| modernc SQLite et dépendances Go | Versions dans `go.mod` et `go.sum` des sources |
| Microsoft Edge WebView2 | Prérequis système, non redistribué dans ce paquet |

Les versions exactes, URL de téléchargement et empreintes des moteurs se trouvent
dans `tools/manifest.json`. Les versions des modules compilés figurent dans
`licenses/go-modules.json`. Les distributions gallery-dl et yt-dlp embarquent
leurs dépendances Python ; voir également les notices fournies par leurs projets.

Le binaire FFmpeg fourni est une construction GPL du projet FFmpeg-Builds de
yt-dlp. Le projet publie ses recettes de construction et les liens vers les
sources amont sur la page indiquée ci-dessus. La commande `tools/ffmpeg.exe -version`
affiche l'identifiant de révision et la configuration de cette construction.
Ce ZIP est destiné à un usage personnel ; avant toute redistribution publique,
conserver et satisfaire les conditions de chaque composant, notamment celles
portant sur la fourniture du code source correspondant.
