# Destinations héritées et sauvegarde chiffrée

Demande du 9 novembre : respecter la destination globale quand un site reste par défaut ; exporter les réglages des sites et leurs comptes avec une phrase secrète de 16 caractères minimum.

## Diagnostic
Les tâches actuelles enregistrent leur destination à l'ajout. Les 17 tâches de l'instance consultée en lecture seule pointent vers downloads malgré une destination globale désormais différente.

## Choix de réalisation
- Résoudre site > global au premier démarrage des nouvelles tâches héritées, puis conserver le chemin pour toutes les reprises. Une destination explicite reste prioritaire. Les anciennes tâches ne portent pas cette information : action explicite pour réappliquer les dossiers aux tâches jamais démarrées, file en pause. Ne déplacer aucun fichier.
- Export chiffré uniquement, de tous les sites configurés et tous les comptes. Pas de tâches, historique, téléchargements, réglages globaux ou caches moteurs. AES-256-GCM et PBKDF2-HMAC-SHA256 600 000 itérations (bibliothèque standard Go), sel et nonce aléatoires, format versionné authentifié, maximum 32 Mio. Alternative DPAPI seul non portable ; JSON en clair écarté pour les secrets.
- Phrase secrète : 16 caractères Unicode minimum, 1024 octets maximum ; confirmation et avertissement sur l'accès aux comptes et l'absence de récupération. Aucun fichier en clair, aucune phrase secrète persistée ou journalisée.
- Export vers un nouveau fichier .dlmbackup (ne remplace pas un fichier existant). Import authentifié avant toute modification, file en pause, vérification intégrale, nouveaux IDs de comptes, chiffrement via le coffre local, transaction SQLite pour publier comptes et réglages ensemble. En cas d'échec du coffre/transaction, nettoyage des nouveaux secrets.
- Import conserve les réglages des sites déjà présents par défaut ; option explicite pour les remplacer. Les comptes existants ne sont jamais remplacés. Réimporter peut créer des doublons. Les chemins incompatibles avec l'OS sont vidés pour hériter du dossier global. Aucun téléchargement démarré par l'import.

## Vérification
Tests d'héritage au premier départ, priorité tâche/site/global, reprise figée, traitement prudent des tâches anciennes ; aller-retour réel export/import, mauvais mot de passe, fichier altéré ou trop gros, coffre indisponible, conservation de l'existant, réimport Windows/Linux. Recompiler TypeScript, Windows et Linux ; contrôle UI avec données fictives.
