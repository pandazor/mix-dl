# dl-manager 0.1.0 — Windows x64

Gestionnaire personnel de téléchargements utilisant gallery-dl et yt-dlp.
Le paquet inclut les moteurs : aucune installation de Python, Go, FFmpeg ou Deno
n'est nécessaire pour l'utiliser.

## Démarrage

1. Extraire **tout** le ZIP dans un dossier où votre compte Windows peut écrire,
   par exemple `G:\Documents\ChatGPT\dl-manager`.
2. Lancer `dl-manager.exe`. Garder le dossier `tools` à côté de l'exécutable.
3. Dans **Paramètres**, choisir le dossier global de téléchargement, les médias
   et les extensions souhaitées, puis enregistrer.
4. Dans **Téléchargements**, coller une ou plusieurs URL, une par ligne, et cliquer
   sur **Ajouter**, puis sur **Démarrer / reprendre**.

Prérequis restant : **Microsoft Edge WebView2 Runtime** pour afficher la fenêtre.
Il est présent sur le PC où cette version a été vérifiée ; il n'est pas inclus
dans ce ZIP. Si le lancement en demande l'installation, utiliser la page officielle :
https://developer.microsoft.com/microsoft-edge/webview2/
L'application cible Windows 10/11 x64 ; ce paquet n'est pas une version Linux.

## Comptes et sites

Ouvrir **Comptes & sites**, puis choisir une plateforme.

| Plateforme | Connexion proposée |
| --- | --- |
| ArtStation, Instagram, TikTok, Patreon, YouTube | Fichier `cookies.txt` au format Netscape |
| Rule34.xxx | User ID + API Key, ou cookies pour les contenus compatibles |
| Pixiv | Refresh token |
| DeviantArt | Refresh token ou cookies selon le contenu |
| Tumblr | OAuth Token + Token Secret, ou cookies selon le contenu |
| Autres sites | Cookies facultatifs ; compatibilité dépendant de gallery-dl |

Les connexions et exports se font manuellement dans votre navigateur. Les guides
de chaque site sont accessibles depuis l'application. Aucun ancien token n'est
récupéré automatiquement. Les contenus publics compatibles restent accessibles
sans compte.

Après avoir enregistré un compte, le sélectionner comme **Compte par défaut**
dans les réglages du site, ou le choisir dans les options des liens ajoutés.
Pour renouveler une connexion, cliquer sur **Modifier**, fournir le nouveau
secret/fichier, puis relancer les tâches concernées. Un HTTP 403 seul n'est pas
considéré comme la preuve d'un token expiré.

Les secrets et les caches d'authentification sont chiffrés par Windows DPAPI pour
le compte Windows courant. Sur un autre PC ou sous un autre compte Windows,
reconfigurer les connexions. L'export de cookies original reste à votre charge :
l'import ne supprime pas ce fichier. **Vérifier le stockage** contrôle le
déchiffrement local ; seule une tentative de téléchargement vérifie la connexion
au site. Si le chiffrement échoue, le compte n'est pas enregistré en clair.

## File, filtres et dossiers

- Réglages : global → site → options du nouvel ajout. Les choix sont figés à
  l'ajout ; les secrets sont relus à chaque tentative.
- Dossiers : destination seule, `site`, ou `site/créateur`. Les recherches Rule34
  utilisent les tags comme nom de dossier. Un lien direct conserve son identité
  de domaine et de chemin pour éviter les collisions.
- Images et vidéos peuvent être cochées séparément. Les extensions filtrent le
  contenu fourni par le site ; aucun transcodage n'est effectué pour fabriquer
  un format qui n'existe pas. Une extension personnalisée inconnue est traitée
  comme une image.
- YouTube télécharge les formats compatibles disponibles. FFmpeg peut assembler
  les pistes audio/vidéo sans réencodage. Les deux choix de nommage utilisent
  actuellement le même modèle identifiant + titre pour YouTube.
- Une à trois tâches simultanées. Les tâches partageant un site et une
  destination, ou un compte, sont sérialisées.
- Pause, annulation, relance, ordre de priorité, détails/journal et historique.
  Retirer une entrée de la liste ne supprime pas ses fichiers.
- Les fichiers `.part` et les archives des moteurs permettent la reprise et
  évitent les téléchargements déjà terminés. Conserver `data` et les destinations.
  Les réglages de dossier/nommage doivent rester cohérents pour les reprises.
- À chaque démarrage, **aucun téléchargement ne part automatiquement**.
  En revanche, une file déjà démarrée traite les liens que vous lui ajoutez.

## Quota de volume

Dans **Paramètres**, renseigner la limite en Go (0 = désactivée). Le compteur
survit aux redémarrages. Le seuil provoque la pause de toutes les tâches.
**Réinitialiser** attend leur arrêt puis remet seulement le compteur à zéro ;
les fichiers et archives sont conservés et la reprise reste manuelle.

Le compteur mesure les octets de médias ajoutés, pas le trafic réseau exact
(requêtes, chiffrement, retransmissions, etc.). Un petit dépassement reste possible
avant que les moteurs s'arrêtent : ce n'est pas un plafond réseau strict.

## Fermeture et sauvegarde

La croix réduit l'application dans la zone de notification. Pour l'arrêter,
utiliser **Quitter l'application** ou **Quitter** dans le menu de l'icône.
Les processus de téléchargement enfants sont arrêtés et les fichiers partiels
restent disponibles pour la prochaine reprise.

Le dossier `data`, créé au premier lancement, contient l'état SQLite, les
comptes chiffrés, les archives, les journaux et le cache WebView2. Sauvegarder ce
dossier **application fermée**, avec les répertoires de téléchargement.
Les téléchargements sont par défaut placés dans `downloads`, à côté du programme.
Pour mettre à jour le programme, le quitter et conserver `data` et vos fichiers.

## Contenu du paquet

- `dl-manager.exe` : interface et cœur Go intégrés.
- `tools/` : gallery-dl, yt-dlp, FFmpeg, ffprobe et Deno.
- `tools/manifest.json` : versions, origines et SHA-256 des moteurs.
- `tools/licenses/` et `licenses/` : licences des composants tiers.
- `SHA256SUMS.txt` : empreintes de tous les fichiers livrés.
- `VALIDATION.md` : vérifications réalisées et limites connues.

Ce paquet ne contient aucun compte, cookie, historique ou téléchargement de test.
La compatibilité et les restrictions des sites restent celles des moteurs et des
comptes utilisés. Les sites privés n'ont pas été testés avec vos identifiants.

## Recompiler depuis les sources

Installer Go 1.26 ou supérieur et Node.js. Dans le dossier des sources, exécuter
`npm install` pour installer le compilateur TypeScript, puis `build.ps1` avec
PowerShell. Le script peut réutiliser
la chaîne locale `../toolchain`, sinon il utilise Go et TypeScript disponibles.
La compilation produit `bin/dl-manager.exe`.

`test.ps1` lance les tests Go. `scripts/fetch-tools.py` télécharge les moteurs
officiels (Python nécessaire uniquement pour cette préparation) et vérifie leurs
sommes publiées. `scripts/package.py` crée les ZIP sous `../../outputs`.
Le test d'intégration réel requiert `tools/gallery-dl.exe`.

Dans un environnement de test isolé où le profil DPAPI n'est pas chargé,
`DLM_RESTRICTED_TESTS=1` permet de signaler explicitement le seul test DPAPI comme
ignoré. Ne pas utiliser cette option pour valider le chiffrement en session normale.

Sources et documentations des moteurs :
https://codeberg.org/mikf/gallery-dl — https://gdl-org.github.io/docs/
https://github.com/yt-dlp/yt-dlp — https://github.com/yt-dlp/FFmpeg-Builds
https://github.com/denoland/deno
