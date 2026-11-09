# Langues et guides — 0.5.0

L’utilisateur demande un sélecteur Langue dans Paramètres, une interface et une documentation en français, anglais, espagnol, chinois simplifié et japonais, sa phrase « je ne suis pas dev, mais j’ai une idée ! », des explications simples par site, ses captures et une possibilité de soutien.

Choix de réalisation : catalogues locaux complets, sans traduction en ligne. Langue persistée dans Settings, défaut français pour les installations existantes, changement immédiat sans perdre les réglages en cours de saisie. Les données personnelles, chemins, noms, secrets et journaux bruts ne sont pas traduits. Les messages connus de l’application, guides, boîtes de dialogue natives, dates et unités le sont. Les messages externes inconnus gardent leur texte technique.

Approches : traduction navigateur rejetée (dépendance externe et données privées) ; copies de l’interface rejetées (maintenance divergente) ; un catalogue partagé et des appels de traduction explicites retenus par l’implémenteur.

Documentation utilisateur complète dans cinq langues, liens croisés, captures françaises signalées. Tableau distinguant tests utilisateur confirmés et intégrations restant à valider. Extension recommandée : Get cookies.txt LOCALLY, export du site courant uniquement. Dons : lien Revolut.me fourni par le mainteneur, configuré dans .github/FUNDING.yml et les cinq guides. Soutien facultatif, montant libre ; aucun paiement intégré à l’application.

Contraintes : préserver data/downloads ; branche feat/0.5.0 depuis main contenant V0.4.0 ; pas de publication de release ni push forcé. AGPL inchangée et références techniques françaises conservées et identifiées.
