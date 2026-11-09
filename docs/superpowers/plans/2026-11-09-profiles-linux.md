# Comptes persistants et version Linux

> Execution: correction native des comptes et sous-tache Linux isolee; revue independante finale.

**Goal:** fiabiliser les comptes sauvegardes, livrer Windows/Linux 0.2.0 et preparer le depot mix-dl.
**Architecture:** secrets proteges conserves lors des edits vides; choix de compte herite resolu au lancement. Adaptateurs OS pour coffre, processus et fenetre.
**Tech Stack:** Go, Wails v3.1.0, TypeScript, SQLite.
**Spec:** demande utilisateur du 2026-11-09 et docs/design.md existant.

## Contraintes
- Jamais de token/cookie retourne au navigateur ou versionne.
- Ne pas ecraser les donnees existantes ni les choix explicites sans compte.
- Premiere creation: compte par defaut si aucun; taches heritees utilisent le defaut courant.
- Linux x64: compilation purego, coffre Secret Service, arret du groupe de processus.
- Copie G: et push autorises mais refus de droits local actuellement. Aucun contournement.

## Verification ciblee
- Edition avec token vide conserve le token; lecture masque secrets mais affiche user ID.
- Erreur du coffre ne produit pas de faux succes.
- Tache creee avant le compte: heritage a l execution, choix explicite respecte.
- Compte actif non modifiable; valeur par defaut valide pour son site.
- Linux: pas de secrets en clair, cancellation tue enfants, noms executables sans .exe.

## Taches
- [ ] T1 Comptes/API/frontend: tests regression puis correction, association par defaut et affichage.
- [ ] T2 Linux: adaptateurs OS, coffre, outils, compilation/package et tests cibles.
- [ ] T3 Livraison: tests Windows, build des deux plateformes, revue, archives, depot local et script de copie/push si acces reste bloque.

## Journal
- Depot local initialise. Depot distant et dossier G: inaccessibles avec les droits de cette session.
- Ruling: executer sans demande intermediaire, comme demande par utilisateur; seul le transfert bloque restera manuel si necessaire.
