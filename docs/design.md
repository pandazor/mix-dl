# dl-manager — conception validée en conversation, 8 novembre 2026

Application Windows personnelle, portable, Go/Wails 3/SQLite. Interface sobre inspirée de l'organisation de GalleryDL Beyond, sans reprise de son code. Un ZIP contenant l'exécutable et les moteurs doit permettre de coller des URL sans installer Python ou Go.

## Fonctionnement
- Quatre vues : téléchargements, historique, comptes/sites, paramètres. Ajout multiligne, choix images/vidéos/extensions, destination et profil. Héritage global > site > tâche, figé à l'ajout sauf secrets résolus à chaque tentative.
- File SQLite persistante ; 1–3 tâches par défaut configurable ; réordonnancement, pause, annulation, relance, suppression des entrées sans suppression des fichiers. Au démarrage aucune tâche ne démarre : clic explicite obligatoire.
- gallery-dl pour les galeries, yt-dlp pour YouTube ; autres URL http(s) acceptées via gallery-dl. Pas de réimplémentation d'extracteurs. Archives distinctes par moteur ; fichiers partiels conservés. Sérialiser les jobs qui partagent une archive/destination pour éviter les courses sur les doublons.
- Sites : ArtStation, Rule34, Instagram, TikTok, Tumblr, Patreon, DeviantArt, Pixiv, YouTube ; profil générique pour les autres. Dossiers par site/créateur, noms contenant un identifiant pour éviter collisions. Les templates utilisent les métadonnées réelles de chaque moteur, sans expressions Python libres fournies par l'utilisateur.
- Comptes avec instructions et liens officiels. L'utilisateur effectue les actions de navigateur et importe un cookies.txt Netscape ou renseigne des tokens. Aucun pilotage du navigateur, aucune récupération automatique des anciennes clés. Plusieurs profils/site, un profil par défaut.
- Secrets chiffrés avec DPAPI CurrentUser. Fichiers temporaires privés et nettoyage, pas de secrets en arguments ou logs. Les données portables se trouvent dans data près de l'exécutable ; les secrets ne sont pas transférables à une autre session Windows sans nouvelle configuration.
- Quota facultatif global, persistant, réinitialisable explicitement. Le reset remet uniquement le compteur à zéro, ne purge pas l'archive et ne relance pas la file. Pause de toutes les tâches au seuil ; dépassement limité possible avant arrêt. Mesure du volume de médias, pas du trafic réseau exact.
- Réduction dans la zone de notification, menu afficher/pause/quitter. Quitter arrête les processus enfants, conserve l'état et les .part. Job Object Windows pour éviter les processus orphelins.
- Erreurs distinctes : authentification, limitation 429, réseau, moteur absent, destination, téléchargement partiel. Un HTTP 403 seul ne prouve pas une session expirée.

## Distribution et validation
Archive portable avec moteurs officiels gallery-dl, yt-dlp, ffmpeg/ffprobe, Deno et leurs licences. Vérifier chaque téléchargement officiel avec les sommes publiées lorsqu'elles existent. Version des outils visible. WebView2 détecté ; fournir un runtime local ou signaler explicitement un prérequis restant. Aucun paquet incomplet ne doit être présenté comme autonome.

Tests : persistance/récupération, concurrence, pause/reprise/reset, filtres sûrs, URLs dangereuses, chemins Windows, secrets/logs, DPAPI, arrêt d'arbre de processus, réseau local avec téléchargement interrompu puis repris. Test réel public et interface ; tests de sites privés seulement avec identifiants saisis volontairement par l'utilisateur.

## Choix et alternatives
Go/Wails/SQLite retenus dans le cahier des charges. Python/Qt (fork de Beyond) changerait la pile et demanderait quand même comptes/quota/packaging. GDL-Web est orienté serveur/sessions ; l'application desktop correspond au besoin portable. ZIP privilégié au monofichier autoextractible pour mettre à jour les moteurs indépendamment.

## Emplacement de travail
La session actuelle permet d'écrire dans son espace de travail, pas dans G:\Documents\ChatGPT. Les sources et le paquet sont préparés ici et livrés pour extraction dans G:\Documents\ChatGPT\dl-manager. Aucun changement global du système ou du Python existant.

## Évolution 0.4.0 — 9 novembre 2026

Windows et Linux sont désormais ciblés. Connexion OAuth guidée pour Tumblr/DeviantArt et fiches de sites personnalisés nommées : voir docs/superpowers/specs/2026-11-09-0.4.0-design.md. La connexion et le consentement restent manuels dans le navigateur ; les nouveaux tokens sont récupérés par le moteur puis remplis dans le formulaire. Les fiches personnalisées restent séparées du catalogue d’extracteurs gallery-dl. Le projet est publié sous AGPL-3.0-only ; composants tiers sous leurs licences respectives.

## 0.5.0 — langues

Settings.Language conserve fr/en/es/zh-CN/ja (vide ancien = français). Une route dédiée modifie uniquement cette préférence sous le verrou du gestionnaire. Les appels de traduction explicites utilisent un catalogue JSON local partagé avec les dialogues natifs ; les données utilisateur ne passent jamais dans la traduction. Les paramètres non enregistrés sont restaurés après le rendu dans une autre langue. Les messages externes inconnus et journaux restent bruts.
