# Erreurs de téléchargement visibles — 0.3.2

Objectif : après un échec, conserver la tâche dans Téléchargements avec son erreur lisible ; expliquer le quota Tumblr et conserver l'heure de remise à zéro fournie par le moteur.

Diagnostic : trois tâches Tumblr sont failed, code moteur 4, fichiers/octet/ignorés à zéro. Journal : Daily API rate limit exceeded et Rate limit will reset at 19:06:46. Compte OAuth associé. La liste Téléchargements filtre failed, ce qui fait disparaître la tâche et son erreur ; l'Historique les contient.

Choix : conserver failed dans Téléchargements comme les tâches demandant une reconnexion. Garder l'Historique et les journaux. Afficher un message de quota lisible, sans interpréter un HTTP 429 comme jeton invalide. Aucune relance automatique, modification du coffre ou nouvelle tentative réelle. Les clés personnelles d'application Tumblr ne sont pas prises en charge par ce correctif.

- [x] Test navigateur : instantané complet fictif avec échec Tumblr et tâche terminée ; vérifier que l'échec reste dans Téléchargements, que son heure/message sont visibles et que Détails expose le message.
- [x] Observer l'échec avec le binaire 0.3.1 puis corriger frontend/app.ts et style.css.
- [x] Version 0.3.2, guide et notes : quota partagé, accès à l'Historique dans 0.3.1, essais TikTok/ArtStation confirmés par l'utilisateur.
- [x] Compiler TypeScript, suite Go, contrôles navigateur et builds Windows/Linux.

Contraintes : préserver data/downloads et les comptes réels. Aucun secret de la conversation dans les fichiers/tests. Validation native/Linux reste celle documentée pour 0.3.1.

Résultats : échec reproduit sur 0.3.1 ; parcours Chrome corrigé, capture inspectée, 49 tests Go principaux (64 avec sous-tests), aucun ignoré. TypeScript, vet Windows/Linux et les deux builds réussis.

Livraison : scripts/package.py vérifie les moteurs, le contenu et les modes des archives ; pousser fix/visible-download-errors pour revue, sans tag ni publication automatique.
