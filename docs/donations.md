# Activer le soutien du projet

Le lien personnel du mainteneur n’a pas encore été fourni. Aucun bénéficiaire ni bouton de paiement actif n’est configuré.

1. Dans Revolut, ouvrir Paiements, Nouveau (+), puis la création d’un lien de paiement et la demande d’argent. Copier le lien Revolut.me générique qui laisse le montant au donateur. Vérifier les conditions et limites affichées dans l’application.
2. Fournir ce lien public au projet. Ne transmettre ni mot de passe, ni code de connexion, ni coordonnées bancaires.
3. Créer .github/FUNDING.yml sur une branche, avec le lien réel :

```yaml
# Remplacer la valeur avant de publier ; ceci est un exemple, pas un bénéficiaire.
custom: ['https://revolut.me/VOTRE_REVTAG']
```

4. Fusionner vers la branche par défaut. Dans les paramètres GitHub du dépôt, vérifier Features → Sponsorships. Ajouter ensuite le même lien dans les cinq guides.

Le bouton GitHub renvoie vers le prestataire choisi ; cette configuration n’inscrit pas automatiquement le mainteneur à GitHub Sponsors et ne collecte aucun paiement dans dl-manager.

Références officielles vérifiées le 9 novembre 2026 :
- [Bouton de soutien GitHub](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/displaying-a-sponsor-button-in-your-repository)
- [Lien Revolut.me](https://help.revolut.com/fr-FR/help/transfers/payment-links/revolut-me-link/)
