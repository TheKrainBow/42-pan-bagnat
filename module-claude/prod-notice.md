# ⚠️ ENVIRONNEMENT DE PRODUCTION

Tu travailles sur le module `<slug>` de Pan Bagnat. Le repo monté ici et les
conteneurs Docker que tu peux piloter sont ceux qui servent de vrais
utilisateurs, **en production**. Il n'existe pas d'environnement de staging.

Règles :
- Commence par observer (lecture, logs, `git diff`, `docker compose ps`) avant
  de modifier quoi que ce soit.
- Avant toute action destructive ou difficile à annuler (suppression de
  fichiers, `git reset --hard`, `git push --force`, `docker compose down`,
  suppression de volumes ou d'images, migration ou modification de base,
  redémarrage de conteneurs), explique ce que tu vas faire et attends la
  confirmation explicite de l'utilisateur.
- Ne touche jamais aux conteneurs, volumes ou repos d'un autre module.
- Ne lis, n'affiche et ne copie jamais de secrets (`.env`, clés, tokens) ni de
  dumps de base (`*.dump`).
- Fais des commits petits et lisibles. Montre le diff avant de redéployer.
- Si tu as un doute sur l'impact d'une action en production, ne l'exécute pas
  et demande.
- Quand tu résumes ton travail, rappelle ce qui a été modifié sur la
  production et ce qui reste à déployer.

Environnement technique :
- Docker est limité au projet compose du module : `docker compose ps`, `logs`,
  `restart`, `stop`, `start`, `exec`… fonctionnent depuis la racine du repo ;
  tout ce qui vise un autre module est refusé.
- Les build et déploiements passent par Pan Bagnat, pour qu'ils apparaissent
  dans les logs du module : `pb deploy` (build + up -d), `pb rebuild`
  (build --no-cache + up -d), `pb down` (sans supprimer les volumes).
  `docker compose up/build/down`, `docker build` et `docker run` sont refusés.
- `git fetch`, `pull` et `push` utilisent la clé de déploiement du module, déjà
  configurée. Ne cherche pas à la lire.
