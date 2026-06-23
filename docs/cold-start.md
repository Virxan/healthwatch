# Relancer Healthwatch après extinction du PC

Tout ce qui a déjà été construit (cluster k3d, image importée, Argo CD,
dev shell Nix) persiste sur disque. Pas besoin de tout reconstruire après
un redémarrage - juste de "réveiller" les bons morceaux dans l'ordre.

Projet : `~/projets/healthwatch`

## 1. Ouvrir un terminal WSL

Ouvre Ubuntu (ou ton terminal WSL habituel) depuis le menu Windows, ou
`wsl` depuis un terminal Windows. Ça démarre la VM WSL si elle ne
l'était pas déjà.

```sh
cd ~/projets/healthwatch
```

## 2. Vérifier que Docker tourne

```sh
sudo systemctl status docker
```

Si ce n'est pas `active (running)` :

```sh
sudo systemctl start docker
```

(Normalement automatique grâce à `systemctl enable` fait à l'installation -
ce contrôle est juste une sécurité.)

## 3. Réveiller le cluster k3d

```sh
k3d cluster list
```

Si le cluster `healthwatch` apparaît mais que ses nodes ne sont pas actifs :

```sh
k3d cluster start healthwatch
```

C'est plus rapide que `k3d cluster create` - ça redémarre les containers
existants (cluster, image déjà importée, état Argo CD) au lieu de tout
recréer. Inutile de refaire `just import-image` à ce stade : l'image est
toujours dans le node.

Si la commande dit que le cluster n'existe pas du tout (suppression
précédente, ou nouvelle machine), repars du
[README](../README.md#4-deploy-to-a-local-kubernetes-cluster-k3d-via-argo-cd)
pour le recréer.

## 4. Entrer dans le dev shell

```sh
nix develop
```

Rapide : tout est déjà dans le store Nix, rien à retélécharger.

## 5. Vérifier que tout est en bonne santé

Laisse une minute ou deux à Kubernetes pour reprogrammer les pods après le
redémarrage des nodes, puis :

```sh
kubectl -n argocd get pods
kubectl -n healthwatch get pods
kubectl -n argocd get applications
```

Attendu :

- les pods Argo CD en `Running`
- le pod `healthwatch-xxxxx` en `1/1 Running`
- l'Application en `Synced` / `Healthy`

Si un pod reste bloqué plus de 2-3 minutes, force un redémarrage :

```sh
kubectl -n healthwatch delete pod -l app.kubernetes.io/name=healthwatch
```

## 6. Ouvrir le dashboard

```sh
just dashboard
```

Puis ouvre `http://localhost:8080` dans le navigateur Windows.

**Si le port 8080 est déjà pris** (un vieux `just run` resté ouvert dans un
autre terminal, par exemple) :

```sh
lsof -i :8080        # repère le PID fautif
kill <PID>
```

ou, plus simple, change juste le port local :

```sh
kubectl -n healthwatch port-forward svc/healthwatch 8081:8080
```

→ `http://localhost:8081`

## Pour l'UI Argo CD (optionnel)

```sh
kubectl -n argocd port-forward svc/argocd-server 8443:443
just argocd-password   # si tu as oublié le mot de passe admin
```

→ `https://localhost:8443`

## Tout arrêter proprement en fin de session

Pas obligatoire (Docker/WSL peuvent rester allumés sans souci), mais si tu
veux libérer les ressources de la machine :

```sh
k3d cluster stop healthwatch
```

Au prochain démarrage, reprends directement à l'étape 3 ci-dessus
(`k3d cluster start healthwatch`).
