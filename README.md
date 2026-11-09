# dl-manager 0.3.2 — Windows et Linux x64

Gestionnaire personnel de téléchargements utilisant gallery-dl et yt-dlp.
Le paquet inclut les moteurs : aucune installation de Python, Go, FFmpeg ou Deno
n'est nécessaire pour l'utiliser.

## Licence

Copyright (C) 2026 pandazor et contributeurs.

Le code propre à dl-manager est distribué sous **GNU Affero General Public License,
version 3 uniquement (AGPL-3.0-only)**. Voir [LICENSE](LICENSE) pour le texte complet.
Ce logiciel est fourni sans garantie. Les composants tiers conservent leurs licences
respectives, décrites dans [THIRD_PARTY.md](THIRD_PARTY.md) et les notices jointes.

Sources : https://github.com/pandazor/mix-dl

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
Le ZIP Windows cible Windows 10/11 x64.

### Linux x64

Le TAR.GZ Linux inclut aussi les cinq moteurs. Sur Debian/Ubuntu avec interface graphique :

```sh
sudo apt install libgtk-4-1 libwebkitgtk-6.0-4 libsecret-tools
# Décompresser dans un dossier appartenant à votre utilisateur :
tar -xzf dl-manager-0.3.2-linux-x64.tar.gz
cd dl-manager
./dl-manager
```

Une distribution glibc récente est nécessaire (Debian 12+/Ubuntu 24.04+ ; zlib et
libgcc, normalement présents). Un service de trousseau compatible Secret Service
(KWallet compatible ou GNOME Keyring) doit être actif et déverrouillé pour les
comptes. L'application refuse le stockage de secrets sans ce coffre.
La compilation Linux est vérifiée depuis Windows ; le lancement natif reste à
valider sur votre bureau Linux. Le backend Wails purego est expérimental.


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

Le premier compte est proposé comme **Compte par défaut**. La case peut être
décochée. Les tâches réglées sur « Compte par défaut du site » utilisent ce compte
au lancement, même si elles ont été ajoutées avant lui. Dans **Détails**, le compte
d'une tâche peut être changé lorsqu'elle est en pause.

En rouvrant un compte, le User ID est visible et la présence de la clé/cookies
est indiquée. Les secrets restent masqués. Les champs secrets laissés vides
conservent leurs valeurs. Pour renouveler une connexion, saisir le nouveau secret,
enregistrer puis cliquer sur **Démarrer / reprendre**. Un changement de méthode
nécessite de nouveaux identifiants. Un HTTP 403 seul n'est pas
considéré comme la preuve d'un token expiré.

Les secrets et les caches d'authentification sont chiffrés par Windows DPAPI pour
le compte Windows courant. Sous Linux, AES-256-GCM protège les fichiers avec une
clé conservée dans Secret Service. Les coffres Windows/Linux ne sont pas interchangeables. Sur un autre PC ou sous un autre compte Windows,
reconfigurer les connexions. L'export de cookies original reste à votre charge :
l'import ne supprime pas ce fichier. **Vérifier le stockage** contrôle le
déchiffrement local ; seule une tentative de téléchargement vérifie la connexion
au site. Si le chiffrement échoue, le compte n'est pas enregistré en clair.

## File, filtres et dossiers

- Réglages : global → site → options du nouvel ajout. Les choix sont figés à
  l'ajout, sauf la destination héritée résolue au premier départ et le compte
  par défaut résolu à chaque lancement. Les secrets
  sont relus à chaque tentative.
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

### Erreurs de téléchargement et quota Tumblr

Les tâches en erreur restent visibles dans **Téléchargements**, ainsi que dans
**Historique**. **Détails** affiche le message et le journal du moteur.
Le bouton **↻ Remettre en attente**, puis **Démarrer / reprendre** si la file est
arrêtée, relance une tâche existante ; il n'est pas nécessaire de recoller son URL.
Une file déjà active peut reprendre la tâche dès sa remise en attente.

Pour Tumblr, saisir `access-token` dans **Token / API Key** et
`access-token-secret` dans **OAuth Token Secret**. L'autorisation OAuth peut réussir
alors que le quota de la clé d'application partagée de gallery-dl est épuisé.
Dans ce cas, attendre l'heure de remise à zéro annoncée dans le journal avant
une nouvelle tentative. Cette heure est affichée telle que fournie par le moteur.
Renouveler les tokens ne réinitialise pas ce quota, partagé avec d'autres utilisateurs.
Les identifiants d'une application Tumblr personnelle (consumer key et secret)
ne sont pas encore pris en charge par dl-manager.

Référence : https://github.com/mikf/gallery-dl/issues/5664

## Quota de volume

Dans **Paramètres**, renseigner la limite en Go (0 = désactivée). Le compteur
survit aux redémarrages. Le seuil provoque la pause de toutes les tâches.
**Réinitialiser** attend leur arrêt puis remet seulement le compteur à zéro ;
les fichiers et archives sont conservés et la reprise reste manuelle.

Le compteur mesure les octets de médias ajoutés, pas le trafic réseau exact
(requêtes, chiffrement, retransmissions, etc.). Un petit dépassement reste possible
avant que les moteurs s'arrêtent : ce n'est pas un plafond réseau strict.

## Fermeture et sauvegarde

La croix **quitte l'application**, sous Windows comme sous Linux, avec le même
arrêt propre que **Quitter l'application**. **Réduire en arrière-plan** conserve
l'application ouverte : dans la zone de notification sous Windows, minimisée
sous Linux. Le menu de l'icône Windows propose également **Quitter**.
Les processus de téléchargement enfants sont arrêtés et les fichiers partiels
restent disponibles pour la prochaine reprise.

Le dossier `data`, créé au premier lancement, contient l'état SQLite, les
comptes chiffrés, les archives, les journaux et le cache WebView2. Sauvegarder ce
dossier **application fermée**, avec les répertoires de téléchargement.
Les téléchargements sont par défaut placés dans `downloads`, à côté du programme.
Pour mettre à jour le programme, le quitter et conserver `data` et vos fichiers.

## Contenu du paquet

- `dl-manager.exe` (Windows) ou `dl-manager` (Linux) : interface et cœur Go intégrés.
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
sommes publiées. `scripts/package.py --platform all` crée les ZIP et le TAR.GZ sous `../../outputs` (ou `--output dist` pour le dossier local).
`build-linux.ps1` permet la compilation Linux depuis Windows ; `sh build-linux.sh`
compile sous Linux, et `CGO_ENABLED=0 go test -tags purego ./...` y lance les tests.
Le test d'intégration réel requiert `tools/gallery-dl.exe`.
`npm run test:ui` vérifie le parcours des destinations, sauvegardes et erreurs dans Chrome
avec un profil temporaire et le binaire compilé. `DLM_TEST_BROWSER` permet de choisir
le navigateur et `DLM_TEST_BINARY` un autre binaire. Sous Linux, installer le navigateur
Playwright (`npx playwright install chromium`) ou renseigner `DLM_TEST_BROWSER`.
`scripts/test-window-close.ps1` contrôle la fermeture native dans une session Windows
avec WebView2 opérationnel ; il utilise uniquement un répertoire temporaire.

Dans un environnement de test isolé où le profil DPAPI n'est pas chargé,
`DLM_RESTRICTED_TESTS=1` permet de signaler explicitement le seul test DPAPI comme
ignoré. Ne pas utiliser cette option pour valider le chiffrement en session normale.

Sources et documentations des moteurs :
https://codeberg.org/mikf/gallery-dl — https://gdl-org.github.io/docs/
https://github.com/yt-dlp/yt-dlp — https://github.com/yt-dlp/FFmpeg-Builds
https://github.com/denoland/deno

## Mise à jour depuis 0.1.0 à 0.3.1

Quitter l'ancienne application et sauvegarder son dossier `data`. Extraire 0.3.2
puis conserver/copier ce dossier `data` à côté du nouvel exécutable Windows ; les
comptes DPAPI existants restent lisibles dans la même session Windows. Le paquet
ne contient aucun `data` et ne remplace pas les téléchargements.

Les anciennes tâches sans compte enregistré sont interprétées comme utilisant
le compte par défaut du site, car 0.1.0 ne distinguait pas héritage et choix
« Sans compte ». Choisir explicitement « Sans compte » dans les détails si voulu.
Ne pas copier le coffre Windows sous Linux : utiliser une sauvegarde chiffrée
ou reconfigurer les comptes Linux.

## Dépôt

Projet : https://github.com/pandazor/mix-dl
Le kit de transfert contient un Git bundle des sources et `installer-mix-dl.ps1`.
Ce script clone les sources vers `G:\Documents\ChatGPT\mix-dl`, copie les paquets
sous `dist`, réutilise la configuration SSH du projet `maison-work/nouveau-jeu`
si disponible, puis pousse `main`. Aucun push forcé, aucune suppression de dossier,
aucun secret ou téléchargement personnel dans le dépôt.

## Destinations et sauvegardes — 0.3.1

Le dossier effectif suit cet ordre : **dossier saisi pour la tâche > dossier du site > dossier global**.

Pour activer la prise en compte du dossier global :

1. Dans **Paramètres**, renseigner le **Dossier global de téléchargement**.
   Il s'enregistre automatiquement à la sortie du champ, au choix par le sélecteur
   ou avant de changer de page. Le message sous le champ confirme l'enregistrement
   ou indique l'erreur. Les autres options nécessitent **Enregistrer les paramètres**.
2. Dans **Comptes & sites**, sélectionner le site, cocher **Utiliser le dossier
   global**, puis cliquer sur **Enregistrer les réglages du site**. Le champ affiche
   le chemin global actuel. Décocher pour choisir un dossier propre au site.
   Les dossiers personnalisés déjà enregistrés sont conservés lors de la mise à jour :
   cocher cette case pour les remettre explicitement à l'héritage.
3. À l'ajout des liens, laisser aussi le champ **Destination** vide dans
   **Options pour ces téléchargements** pour conserver cet héritage.

Pour les nouvelles tâches héritées, le dossier est résolu au premier démarrage :
un changement de réglage avant ce départ est donc pris en compte. Dès qu'une tâche
a commencé, son dossier reste fixe pour préserver ses fichiers partiels et sa reprise.

Les tâches créées en 0.1/0.2 ne distinguaient pas un dossier hérité d'un choix
personnel. Après avoir enregistré les réglages ci-dessus, cliquer sur **Tout
mettre en pause**, attendre l'arrêt des tâches actives, puis utiliser
**Actualiser les destinations en attente** pour réappliquer site/global aux tâches
jamais démarrées. Cliquer ensuite sur **Démarrer / reprendre** pour relancer la file.
Les tâches commencées, les téléchargements et les destinations explicitement
personnalisées par 0.3.0 sont conservés. L'actualisation ne déplace aucun fichier
et ne relance pas les téléchargements automatiquement.

Avec **Site / Créateur**, choisir par exemple `G:\dl-a-trier` comme dossier global :
Rule34 ajoutera `rule34\nom_artiste`. Si le global se termine déjà par `rule34`,
le nom sera répété. Les autres choix (classement, filtres, nommage) restent fixés
à l'ajout.

Dans **Comptes & sites**, **Exporter les sites et comptes** crée une sauvegarde
`.dlmbackup`, chiffrée par une phrase secrète de **16 caractères minimum**.
Il faut la confirmer et reconnaître l'avertissement : toute personne disposant
du fichier et de cette phrase peut récupérer les clés, tokens et cookies.
Préférer une longue phrase unique et la conserver séparément ; aucun mécanisme
de récupération n'existe. Le programme refuse d'écraser une sauvegarde existante.

**Importer une sauvegarde** ajoute les comptes dans le coffre de la session
courante, avec de nouveaux identifiants internes. Le fichier est portable entre
Windows et Linux, contrairement aux fichiers du coffre `data/credentials`.
Les comptes déjà présents sont conservés ; réimporter le même fichier peut créer
des doublons. Les réglages des sites existants restent inchangés sauf si la case
de remplacement est cochée. Les chemins incompatibles avec l'OS sont remis
à l'héritage du dossier global : contrôler les destinations après l'import.

**L'export fonctionne pendant les téléchargements**, après une pause individuelle
ou avec la file entièrement en pause. Il capture les comptes et leurs réglages
ensemble, puis chiffre le fichier sans interrompre les moteurs.

**L'import nécessite une file en pause** : son dialogue propose **Mettre la file
en pause** et indique quand les tâches ont fini de s'arrêter. Il ne relance pas
la file. La sauvegarde inclut les réglages des sites et leurs comptes ; elle
n'inclut ni téléchargements, ni tâches/historique,
ni réglages globaux, ni caches moteurs. Une sauvegarde ne renouvelle pas les
identifiants expirés. Aucun téléchargement n'est lancé par l'import.

Format version 1 : AES-256-GCM, PBKDF2-HMAC-SHA256 à 600 000 itérations,
sel de 16 octets et nonce de 12 octets aléatoires, en-tête authentifié,
32 Mio maximum. Aucun export en clair ni stockage de la phrase secrète.
Le format et le coût de dérivation sont bornés avant déchiffrement.
