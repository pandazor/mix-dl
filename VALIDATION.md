# Vérifications de dl-manager 0.1.0

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
