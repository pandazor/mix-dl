# Vérifications de dl-manager 0.4.0

Validation du 9 novembre 2026, Windows x64.

- Suite Go Windows : **64 tests principaux réussis, aucun ignoré**, avec DPAPI réel et téléchargements gallery-dl sur serveur local.
- Connexion guidée : processus local de test, sortie fragmentée, paire Tumblr/token DeviantArt, validation des URL HTTPS et redirections, réponse partielle ou trop grande refusée, erreur sans sortie brute, session unique, annulation/fermeture, expiration des résultats non consommés, nouvelle tentative après expiration et lecture unique. L'API protège toutes les actions OAuth ; ni state ni status n'exposent les nouveaux secrets.
- Chrome réel, profil/base temporaires : bouton OAuth, remplissage puis enregistrement explicite, refus, réponse reçue après fermeture ignorée, annulation par Échap. Seul le retour du fournisseur OAuth est simulé.
- Sites personnalisés : détection exacte/sous-domaine, suffixes trompeurs, suffixes publics et domaines intégrés refusés, deux sites avec destinations/comptes distincts, renommage, suppression d'une fiche utilisée refusée, cookies d'un autre domaine rejetés et chemins stables.
- Sauvegardes chiffrées : import v1 conservé ; définitions personnalisées v2, remappage des comptes par domaine, import neuf et fusion, remplacement explicite. Régression reproduite puis corrigée : un site local sans surcharge garde son héritage global à l'import.
- Chrome : création de deux fiches, nom contenant des chevrons affiché comme texte, dossier global/custom, compte cookies, ajout de tâches sans téléchargement, renommage et rechargement. Capture inspectée à largeur 840 ; aucun message d'erreur JavaScript. Les parcours de destinations, export/import et erreurs des versions précédentes passent toujours.
- TypeScript compilé ; go vet Windows et Linux (purego) réussis ; exécutables Windows x64 et Linux x64 compilés.
- Un contrôle lancé en même temps que les deux compilations a rencontré deux délais de démarrage (gallery-dl et processus OAuth de test). La suite complète relancée seule réussit : 64 tests, aucun ignoré. Aucun délai de production n’a été augmenté pour masquer ces échecs.
- Revue indépendante ciblée des extraits de code : aucun blocage signalé. La lecture directe des fichiers par le réviseur a bloqué ; cette revue ne constitue pas une inspection indépendante complète des sources, tests ou archives.
- DeviantArt : l'utilisateur confirme le téléchargement avec le refresh-token obtenu manuellement avant 0.4.0. TikTok/ArtStation/Instagram/Rule34 : retours positifs antérieurs.

## Limites 0.4.0

La connexion réelle avec le nouveau bouton doit être essayée par l'utilisateur sur Tumblr et DeviantArt ; aucun compte réel ni token de la conversation n'a servi aux tests. Le quota partagé Tumblr reste applicable. Une seule autorisation à la fois, port local 6414 disponible, cinq minutes maximum. La saisie manuelle reste disponible.

Les tests de navigateur ne remplacent pas le contrôle de la fenêtre Wails et de son ouverture du navigateur natif. La fermeture native Windows reste à confirmer (test WebView2 inconclusif en 0.3.1). Le lancement et le trousseau Linux restent à valider dans une session Linux.

Le domaine des fiches personnalisées est immuable ; pour le changer, créer une autre fiche. Les cookies de domaines parents ou étrangers sont refusés pour ces fiches. Une fiche ne rend pas un site compatible avec gallery-dl. Les sauvegardes avec fiches personnalisées nécessitent la version 0.4.0 ou supérieure ; les anciennes sauvegardes restent importables.

---

# Vérifications de dl-manager 0.3.2

Validation du 9 novembre 2026, Windows x64.

- Reproduction sur le binaire 0.3.1 : une tâche Tumblr en échec, injectée via une réponse /api/state synthétique, disparaît de Téléchargements. Le test échoue sur l'absence de sa ligne avant correction.
- Après correction, test Chrome réussi : échec visible, explication du quota et heure conservée, texte à la ligne, Détails et bouton de remise en attente disponibles ; erreur réseau inchangée ; tâche terminée réservée à l'Historique. Capture 1100 × 800 inspectée, aucune erreur JavaScript.
- Le même parcours vérifie les régressions de destination et sauvegarde de 0.3.1 dans une base et un profil temporaires. Aucun accès à Tumblr, cookie ou compte réel pendant les tests.
- Suite Go Windows : **49 tests principaux réussis (64 avec les sous-tests), aucun ignoré**, dont DPAPI réel et téléchargement gallery-dl sur un serveur local.
- TypeScript compilé ; go vet Windows et Linux (purego) réussis ; exécutables Windows x64 et Linux x64 compilés.
- Diagnostic utilisateur, lecture seule des métadonnées et journaux : trois essais Tumblr ont échoué sur le quota quotidien de l'API, code 4, aucun fichier ajouté. L'authentification du profil était bien associée aux tâches. Ce résultat ne valide pas encore un téléchargement Tumblr.
- TikTok et ArtStation : téléchargements confirmés par l'utilisateur sur 0.3.1. Instagram et Rule34 : retours positifs antérieurs.

## Limites 0.3.2

Ce correctif affiche les erreurs ; il n'augmente pas le quota partagé de Tumblr et ne relance aucune tâche automatiquement. L'heure de remise à zéro est celle fournie par le moteur, sans conversion de fuseau. Les clés d'une application Tumblr personnelle (consumer key et secret) ne sont pas encore prises en charge.

Le lancement et le trousseau Linux restent à valider sur Linux. Le test de fermeture native Windows était inconclusif en 0.3.1 (WebView2 dans la session d'automatisation) ; ce parcours natif n'a pas été retesté pour ce correctif d'affichage.

---

# Vérifications de dl-manager 0.3.1

Validation du 9 novembre 2026, Windows x64.

- Suite Windows : **49 tests réussis, aucun ignoré**, avec DPAPI réel et téléchargement local gallery-dl.
- Régressions reproduites avant correction : global perdu à la navigation sans bouton de sauvegarde ; export refusé avec téléchargement actif.
- Route partielle de destination : préservation des réglages et dossiers propres aux sites, refus des chemins relatifs/vides, héritage de la nouvelle destination.
- Export chiffré testé avec un worker actif, après sa pause individuelle alors que la file reste active, et avec une modification de compte concurrente. Le fichier déchiffré contient un instantané cohérent et l'export ne change ni la file ni les réglages.
- Revue indépendante ciblée des extraits modifiés : aucun défaut critique ou important relevé. L’accès direct du réviseur aux fichiers a bloqué ; cette revue ne vaut pas inspection indépendante des tests, des archives ou du fonctionnement natif.
- Interface Chrome réelle, profil/base temporaires : navigation immédiate après saisie du global, case d'héritage et chemin effectif, dossier personnalisé conservé, nouvelle tâche Instagram héritée, export avec file active, pause depuis le dialogue d'import, import sans reprise automatique, chemin invalide signalé sans perte de l'ancien réglage, chemin retourné par le sélecteur enregistré automatiquement (seul le sélecteur OS est simulé). Capture de l’héritage inspectée à 840 × 620. Aucune erreur JavaScript.
- TypeScript compilé ; go vet Windows et Linux (purego) réussis ; exécutables Windows x64 et Linux x64 compilés. Tests Linux compilés, non exécutés sur cet hôte Windows.
- Instagram : l'utilisateur confirme sur 0.3.0 la lecture du fichier cookies et le téléchargement des images. Rule34 : retours positifs également. Aucun téléchargement réel sur ces comptes n'a été effectué par les tests de 0.3.1.

## Limites 0.3.1

La croix Windows appelle désormais Quitter au lieu de masquer la fenêtre. Son test natif isolé n'a pas pu conclure dans la session d'automatisation : WebView2 ne termine pas la navigation initiale. Une première fermeture trop précoce avait interrompu la création du contrôleur (80004004) ; elle ne valide pas le comportement de fermeture. Le script attend maintenant le chargement initial. **Le clic sur la croix, les sélecteurs natifs et le lancement/trousseau Linux restent à confirmer dans les sessions graphiques normales.**

L'export ne nécessite plus de pause. L'import de sauvegarde exige toujours la pause de la file entière et la fin des tâches actives ; le dialogue permet de déclencher cette pause. Les chemins propres aux sites déjà enregistrés et les destinations des tâches commencées sont conservés. Aucun fichier n'est déplacé.

---

# Vérifications de dl-manager 0.3.0

Validation du 9 novembre 2026, Windows x64.

- `go test -count=1 -v ./...` : **46 tests réussis, aucun ignoré**.
- `go vet ./...` et `GOOS=linux go vet -tags purego ./...` réussis.
- Tests Linux compilés avec `go test -c -o bin/linux-tests/ -tags purego ./...` ; non exécutés sur Windows.

- Héritage de destination reproduit avant correction : les tâches conservaient
  la destination initiale. Les tests vérifient maintenant site/global au premier
  départ, priorité du dossier explicite, prévisualisation effective et reprise
  dans le dossier initial.
- Actualisation explicite des anciennes tâches jamais démarrées, validation
  de l'ensemble de la sélection avant écriture, refus des tâches commencées.
- Sauvegarde chiffrée : aléa, aller-retour, 16 caractères Unicode minimum,
  phrase incorrecte, altération de l'en-tête et du contenu, bornes de taille.
- Export/import : fichier existant conservé, validation intégrale avant écriture,
  rejet des IDs malformés, des méthodes/sites inconnus, des références invalides ;
  rollback du coffre et transaction SQLite ; conservation de la configuration
  existante, réaffectation des comptes par défaut, chemins étrangers réinitialisés.
- Conservation des cookies, tokens et secrets OAuth. Import testé dans le vrai
  coffre Windows DPAPI avec données fictives ; absence du secret en clair sur disque.
- Interface testée dans Chrome headless, avec un profil et une base séparés :
  réglage global changé après ajout, détails actualisés, bouton de réapplication,
  export chiffré, refus d'une phrase courte ou incorrecte, import ajoutant un compte,
  champs secrets effacés à la fermeture ; aucune erreur JavaScript.
- Revue indépendante ciblée des extraits de code de chiffrement, import et
  destinations : aucun blocage relevé. Le premier accès aux fichiers du réviseur
  a échoué ; cette revue ne constitue pas une exécution indépendante des tests.
- Archives portables : sommes des moteurs contrôlées, ZIP vérifiés et TAR.GZ
  relu pour contrôler les SHA-256 et les modes exécutables Linux.
- TypeScript et compilations Windows/Linux vérifiés. Aucun changement de l'instance
  utilisateur en cours, de son coffre ou de ses téléchargements.

## Retours d'essais utilisateur — 9 novembre 2026

- Rule34 : premier retour positif sur les téléchargements ; les essais continuent.
- Dossier global : nouvel essai utilisateur encore à effectuer après activation
  de l'héritage et, pour les anciennes tâches jamais démarrées, actualisation
  des destinations. Les tests automatisés ci-dessus ne remplacent pas cet essai.
- Instagram : prochains essais annoncés ; aucun résultat utilisateur confirmé
  à ce stade.

## Limites 0.3.0

Le contrôle UI utilise le mode aperçu HTTP local. Les nouveaux sélecteurs de
fichiers natifs et le bureau/trousseau Linux doivent encore être essayés dans
leurs sessions graphiques normales. Les tests de chiffrement portable s'exécutent
sur Windows ; la compilation/vet Linux ne vaut pas exécution Linux.
Les formats expirés ou refusés par un site restent dépendants de ce site.

Les fichiers d'une tâche déjà commencée ne sont pas déplacés. Les anciennes tâches
sans provenance nécessitent l'action explicite d'actualisation. Les archives moteur
et les téléchargements ne sont pas transportés par la sauvegarde de comptes.

---

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
