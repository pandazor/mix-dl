# Vérifications de dl-manager 0.2.0

Validation du 9 novembre 2026, hôte Windows x64.

## Vérifié pour 0.2.0

- TypeScript : compilation réussie. Exécutables Windows x64 et Linux x64 compilés.
- Windows : `DLM_RESTRICTED_TESTS=1 go test -work -count=1 -v ./...` :
  **34 tests réussis, 1 test DPAPI explicitement ignoré**. `go vet ./...` réussi.
- Test du parcours signalé : ajout avant le compte, sauvegarde de l'ID/API key,
  fermeture/réouverture SQLite, lancement de la tâche, puis vérification exacte
  de `extractor.rule34.user-id` et `api-key` fournis au moteur. Identifiants fictifs.
- Secrets conservés lors d'une modification avec champs vides ; réponses HTTP
  limitées au User ID et aux indicateurs de présence. Erreur du coffre affichée,
  jamais convertie en succès ou stockage en clair.
- Héritage au lancement et à la reprise ; choix explicites conservés, exclusion
  des comptes actifs. Case de compte par défaut testée dans les deux sens.
- Interface contrôlée dans le navigateur avec un coffre de test : compte rouvert,
  User ID visible, clé indiquée comme présente, renommage sans nouvelle clé,
  bouton de reprise accessible après une erreur d'authentification.
- Linux : compilation des tests et `go vet -tags purego ./...` réussis ; les tests
  Linux n'ont pas été exécutés sur cet hôte Windows.
- Moteurs Linux : téléchargements officiels épinglés, SHA-256 publiés vérifiés ;
  inspection ELF x86-64/glibc. Archive Linux de contrôle : SHA-256 et modes 0755 vérifiés.
- Revue indépendante : problèmes de reprise, case par défaut, fins de ligne du
  script Linux et environnement PowerShell corrigés.
- Script de transfert : clonage du bundle et push non forcé vérifiés contre un
  dépôt local isolé. Le dépôt GitHub utilisateur n'a pas été modifié.
- Workflow GitHub Windows/Linux linté ; aucun run distant effectué avant push.

## Limites de cette validation

Le test DPAPI est ignoré car le profil isolé du terminal ne charge pas le coffre
Windows (même échec reproduit avec .NET). Le stockage réel doit être confirmé dans
la session Windows normale. Le formulaire de test utilise des secrets fictifs et
un coffre en mémoire ; il ne prétend pas valider votre connexion Rule34 réelle.

Le bureau Linux, le trousseau Secret Service réel et l'arrêt de processus Linux
restent à valider sur Linux. Le backend graphique Wails purego est expérimental.
Les tests Linux et Windows s'exécuteront aussi via le workflow après le push.
Les autres limites de la version précédente ci-dessous restent applicables.

## Historique de validation 0.1.0

### Vérifications de dl-manager 0.1.0

Validation du 9 novembre 2026, Windows 10 x64.

## Vérifié

- Compilation TypeScript et compilation de l'exécutable Windows Go/Wails.
- Tests du modèle, filtres, URLs, chemins Windows, persistance SQLite et reprise
  après interruption sans démarrage automatique.
- Concurrence, exclusion des archives partagées, arrêt au quota et remise à zéro
  sans reprise automatique.
- Vrai processus gallery-dl contre un serveur HTTP local : interruption,
  reprise, intégrité des octets, comptage et seconde exécution sans doublon.
- Deux URL différentes avec le même nom de fichier : conservation des deux
  contenus, y compris avec le choix « identifiant + titre ».
- Reprise pendant l'arrêt d'une tâche : refus explicite, aucune reprise perdue.
- Remplacement d'un compte : sérialisation avec le démarrage des tâches.
- Suppression d'un compte par défaut : retrait de la référence dans les réglages.
- Secrets absents de la ligne de commande, masquage des secrets dans les logs,
  validation du format Netscape et arrêt de l'arbre des processus enfants.
- Application native démarrée dans un dossier neuf ; page principale rendue
  et contrôlée par son arbre d'accessibilité.
- Interface complète contrôlée dans son mode aperçu local : réglages, moteurs,
  ajout, démarrage, historique et formulaires de site. Téléchargement public
  réussi de https://www.python.org/static/img/python-logo.png (1 fichier, ~16 Ko).
- Revue indépendante du code ; corrections des quatre problèmes importants
  reproduites par des tests de régression.
- Vérification des empreintes des moteurs avant empaquetage ; test CRC des ZIP.

## Limites précises

- Le test DPAPI échoue dans le profil isolé du terminal de validation ; le même
  échec est reproduit avec .NET, avant toute écriture de secret. Il est signalé
  comme ignoré lors de la suite sous `DLM_RESTRICTED_TESTS=1`. Le chiffrement
  devra être confirmé en enregistrant puis vérifiant un compte dans une session
  Windows normale. Aucun stockage en clair de secours n'a été ajouté.
- Capture native indisponible dans l'outil de contrôle : l'affichage natif a été
  vérifié via l'accessibilité. Les sélecteurs natifs, le menu de notification et
  le cycle masquer/réafficher n'ont pas été validés de bout en bout.
- Les sites nécessitant un compte et un téléchargement YouTube réel n'ont pas
  été testés avec vos connexions. Leur disponibilité dépend des sites/moteurs.
- Les destinations avec alias, jonctions ou liens symboliques ne sont pas
  couvertes par les tests d'exclusion ; utiliser le même chemin canonique.
- Les deux choix de nommage sont identiques pour YouTube dans cette version.
- WebView2 Runtime reste un prérequis système et n'est pas livré dans le ZIP.

Cette première version est prête pour un essai personnel ; ces limites ne sont
pas présentées comme des validations accomplies.
