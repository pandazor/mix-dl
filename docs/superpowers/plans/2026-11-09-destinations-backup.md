# Destinations et sauvegarde chiffrée — plan

**Goal:** corriger l'héritage de destination et permettre la sauvegarde portable des sites/comptes.
**Architecture:** résolution au premier démarrage ; module de chiffrement indépendant, endpoints HTTP et import atomique SQLite/coffre.
**Tech Stack:** Go standard crypto, SQLite, TypeScript, Wails.
**Spec:** docs/superpowers/specs/2026-11-09-destinations-backup-design.md

## Contraintes
Préserver data/downloads et l'instance en cours. Aucun secret dans Git, les logs ou les réponses HTTP. Fichiers de sauvegarde chiffrés, 16 caractères minimum.

## Tâches
- [x] T1 Tests API/queue : priorité tâche/site/global, réglage changé avant départ, reprises, tâches anciennes ; ajouter destinationMode et RefreshDestinations([]string), route jobs/destinations.
- [x] T2 Tests chiffrement : intégrité, phrase, aléa et limites ; internal/backup Encrypt/Decrypt et format fixe borné.
- [x] T3 Tests export/import API : vraie sauvegarde disque, import vers coffre neuf, transaction et rollback, conservation des réglages, chemins étrangers ; internal/web/backups.go et Store.ImportSites.
- [x] T4 Interface destination effective et actualisation des tâches ; dialogues de sauvegarde/import, avertissements, sélection native des fichiers.
- [x] T5 Suite complète, vet Windows/Linux, compilation Windows/Linux, contrôle UI et revue indépendante ; documentation 0.3.0 et livrables.

## Review Focus
Secrets dans les erreurs ; fichier hostile/volumineux ; échec partiel coffre/SQLite ; import concurrent à une file active ; tâches reprises après changement de destination. Chaque classe doit être couverte par les tests de T1/T2/T3.
