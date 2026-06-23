# Lancer Healthwatch après un `git clone`

Suppose que Nix (flakes) et Docker sont déjà installés sur la machine. Si
ce n'est pas le cas, voir d'abord le
[README - Setup](README.md#setup-on-windows-wsl).

## 1. Cloner et entrer dans le projet

```sh
git clone https://github.com/Virxan/healthwatch.git
cd healthwatch
```

## 2. Entrer dans le dev shell

```sh
nix develop
just            # liste toutes les tâches disponibles
```

Premier `nix develop` sur une machine neuve : ça télécharge tout le
toolchain (go, k3d, kubectl, argocd, just, lefthook...), ça prend
quelques minutes. Les fois suivantes, c'est instantané.

## 3. Test rapide, sans Kubernetes

```sh
just test       # unit tests + Cucumber/godog
just run        # http://localhost:8080
```

## 4. Construire le conteneur

```sh
just container
```

Le tout premier `just container` sur une machine donnée échoue
volontairement une fois, avec un message du type :

```text
error: hash mismatch ... got: sha256-XXXXXXX...
```

C'est normal (mécanisme Nix). Copie le hash affiché après `got:` dans le
champ `vendorHash` de `flake.nix`, puis relance :

```sh
just container
```

(Si tu clones le repo tel qu'il est sur GitHub à ce stade, le `vendorHash`
correct est probablement déjà commité - dans ce cas cette étape passe du
premier coup.)

## 5. Déployer sur k3d via Argo CD

```sh
just k3d-up               # crée le cluster + installe Argo CD
just argocd-password      # mot de passe admin initial
just import-image          # charge l'image construite à l'étape 4
just argocd-app             # branche Argo CD sur deploy/
```

`argocd/application.yaml` pointe déjà vers `Virxan/healthwatch` - si tu
clones ton propre fork, mets à jour `spec.source.repoURL` dans ce fichier
avant `just argocd-app`.

## 6. Vérifier et accéder au dashboard

```sh
kubectl -n healthwatch get pods       # attends 1/1 Running
kubectl -n argocd get applications    # attends Synced / Healthy

just dashboard
```

→ `http://localhost:8080`

## Au prochain démarrage

Pas besoin de tout refaire à chaque fois - voir
[`docs/cold-start.md`](cold-start.md) pour juste réveiller ce qui existe
déjà après extinction du PC.
