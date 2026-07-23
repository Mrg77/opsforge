<div align="center">

# opsforge 🔥

**Votre poste de travail DevOps, forgé en quelques minutes.**

Choisissez vos CLI depuis une interface terminal interactive, installez-les d'un
seul coup, et transformez votre zsh en un environnement DevOps sensible au
contexte — complétion en direct, prompt conscient de la prod, et des **guards
policy-as-code** qui vous empêchent de démolir le mauvais cluster.

opsforge est la **couche supply-chain + policy de votre propre poste de
travail** : il installe votre boîte à outils, garde-fou la façon dont *vous*
l'utilisez, et vous remet un SBOM corrélé aux CVE de l'ensemble. C'est un outil
personnel, pas une plateforme d'équipe — pas de serveur, pas de compte, pas de
verrouillage.

[English](README.md) · **Français**

[![CI](https://github.com/Mrg77/opsforge/actions/workflows/ci.yml/badge.svg)](https://github.com/Mrg77/opsforge/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/Mrg77/opsforge?sort=semver)](https://github.com/Mrg77/opsforge/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/Mrg77/opsforge)](https://goreportcard.com/report/github.com/Mrg77/opsforge)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
<br>
[![Tools](https://img.shields.io/badge/tools-287-blue)](#le-catalogue)
[![SBOM](https://img.shields.io/badge/SBOM-CycloneDX%201.6-orange)](#sbom--chaîne-dapprovisionnement)
[![Made with Go](https://img.shields.io/badge/made%20with-Go-00ADD8?logo=go&logoColor=white)](https://go.dev)

![opsforge demo](demo/demo-v0.3.2.gif)

**[Installation](#installation) · [Aperçu](#aperçu-rapide) · [Workflows](#workflows-courants) · [Shell](#lenvironnement-shell-devops) · [Guards](#guards-policy-as-code) · [Mode projet](#mode-projet) · [SBOM](#sbom--chaîne-dapprovisionnement) · [CI](#ci--intégrations) · [Catalogue](#le-catalogue) · [Sous le capot](#points-forts-dingénierie)**

</div>

---

## Ce que c'est

opsforge, c'est **trois outils dans un seul binaire** :

| | | |
|:--:|---|---|
| 📦 | **Installeur d'outils** | Un sélecteur interactif parmi **287 CLI triés sur le volet, couvrant tous les métiers IT** — dont une nouvelle catégorie **AI & LLM**. Il détecte ce que vous avez et ce qui est obsolète, puis installe le reste via Homebrew *ou* directement depuis les binaires de release GitHub — fonctionne sur une machine Linux nue sans gestionnaire de paquets. |
| 🐚 | **Shell DevOps** | Une seule commande transforme votre propre zsh en une expérience façon Warp/Fish : complétion en direct, aide inline `?`, prompt conscient de la prod, et des [**guards policy-as-code**](#guards-policy-as-code) sur les commandes destructrices. Aucun remplacement de shell, aucun verrouillage. |
| 📸 | **Poste de travail & projet as-code** | `opsforge snapshot` exporte toute votre config — outils, profils, shell, thème *et* politique de guards — dans un seul YAML ; un [`opsforge.yaml`](#mode-projet) committé déclare la boîte à outils d'un dépôt et `opsforge sync` la reproduit (avec un gate CVE). `apply --check` / `sync --check` vérifient une machine en CI, et [`opsforge sbom`](#sbom--chaîne-dapprovisionnement) en émet un SBOM corrélé aux CVE. |

> **Pourquoi :** reconstruire un poste de travail DevOps, c'est installer plus de
> 20 CLI, puis câbler complétions, alias et un prompt utile pour chacun — à la
> main, sur chaque machine. opsforge en fait une session de deux minutes et garde
> votre shell synchronisé à mesure que votre boîte à outils grandit.

---

## Installation

```sh
curl -fsSL https://raw.githubusercontent.com/Mrg77/opsforge/main/install.sh | sh
```

Télécharge le bon binaire pour votre OS/arch dans `~/.local/bin` (surchargeable
avec `OPSFORGE_INSTALL_DIR`, épinglable avec `OPSFORGE_VERSION=v1.2.3`). Depuis
les sources : `go install github.com/Mrg77/opsforge@latest`.

Gardez-le à jour avec `opsforge self update` — il télécharge la dernière release,
**vérifie son SHA-256 publié avant de remplacer le binaire en place**, et ne fait
rien quand vous êtes déjà à jour (`--check` pour cron/CI).

> **Windows :** utilisez WSL — le backend d'installation est Homebrew et la couche
> shell cible zsh. Le support natif winget/scoop + PowerShell est sur la feuille
> de route.

---

## Aperçu rapide

```sh
opsforge              # sélecteur interactif (onglets : 1 Outils · 2 Mises à jour · 3 Sécurité)
opsforge status       # cockpit de votre poste de travail en un coup d'œil
opsforge doctor       # bilan de santé complet — CVE & secrets exposés inclus
opsforge audit        # scan des CVE des outils installés (--secrets : creds exposés aussi)
opsforge guard test "terraform destroy" --context prod   # simule une règle de guard
opsforge apply --check my-setup.yaml                     # vérifie que cette machine correspond à votre snapshot (CI)
opsforge self update  # mise à jour, checksum vérifié avant le remplacement
```

<table>
<tr><th align="left">Commande</th><th align="left">Ce qu'elle fait</th></tr>
<tr><td><code>opsforge</code></td><td>Sélecteur interactif — parcourir, vérifier, installer</td></tr>
<tr><td><code>opsforge status</code></td><td>Cockpit : outils, mises à jour, shell, thème en un coup d'œil</td></tr>
<tr><td><code>opsforge notify [--json]</code></td><td>Un seul digest de ce qui mérite votre attention — CVE, mises à jour, secrets exposés, un opsforge plus récent (voir <a href="#le-digest-notify">notify</a>)</td></tr>
<tr><td><code>opsforge install kubectl helm</code></td><td>Installation non interactive par nom (scriptable)</td></tr>
<tr><td><code>opsforge install --profile aws-k8s</code></td><td>Installe tout un preset de stack en une commande</td></tr>
<tr><td><code>opsforge upgrade [-u] [outil…]</code></td><td>Met à jour tout, seulement l'obsolète (<code>-u</code>), ou des outils nommés</td></tr>
<tr><td><code>opsforge audit [--secrets] [--json]</code></td><td>Scan CVE des outils installés · scan de secrets exposés optionnel · <code>--json</code> + code de sortie non nul verrouille la CI</td></tr>
<tr><td><code>opsforge guard [init|list|test|lint]</code></td><td>Guards policy-as-code sur les commandes destructrices · <code>lint</code>/<code>test --json</code> les rendent vérifiables en CI (voir <a href="#guards-policy-as-code">Guards</a>)</td></tr>
<tr><td><code>opsforge use terraform@1.5</code></td><td>Épingle une version d'outil ici (délègue à mise/asdf)</td></tr>
<tr><td><code>opsforge sync [--check] [--init]</code></td><td>Installe les outils déclarés par un <code>opsforge.yaml</code> committé · <code>--check</code> reporte la dérive pour la CI · gate CVE optionnel (voir <a href="#mode-projet">Mode projet</a>)</td></tr>
<tr><td><code>opsforge sbom [--audit]</code></td><td>Émet un SBOM CycloneDX 1.6 des outils installés · <code>--audit</code> y embarque leurs CVE (voir <a href="#sbom--chaîne-dapprovisionnement">SBOM</a>)</td></tr>
<tr><td><code>opsforge snapshot</code> / <code>apply</code></td><td>Exporter / reconstruire tout un poste de travail</td></tr>
<tr><td><code>opsforge apply --check &lt;fichier-ou-url&gt;</code></td><td>Vérifie une machine par rapport à votre snapshot sans la modifier · code de sortie non nul en cas d'écart (<code>--json</code>)</td></tr>
<tr><td><code>opsforge self [version|update]</code></td><td>Affiche la version ou se met à jour — checksum vérifié avant le remplacement (<code>--check</code> pour CI/cron)</td></tr>
<tr><td><code>opsforge history [famille|outil]</code></td><td>Commandes shell récentes, groupées par famille d'outils (<code>kube</code>, <code>git</code>, <code>tf</code>… — voir <a href="#history">History</a>)</td></tr>
<tr><td><code>opsforge explain [--last] &lt;cmd&gt;</code></td><td>Demande à votre CLI IA d'expliquer une commande ou votre dernière erreur (le raccourci <code>??</code> du shell)</td></tr>
<tr><td><code>opsforge list [all] [-u]</code></td><td>Outils installés · catalogue complet · seulement les mises à jour (<code>--json</code> pour scripter)</td></tr>
<tr><td><code>opsforge list &lt;terme&gt;</code></td><td>Rechercher dans tout le catalogue par nom, description ou catégorie (ex. <code>list dns</code>)</td></tr>
<tr><td><code>opsforge profiles</code></td><td>Profils de stack avec statut d'installation</td></tr>
<tr><td><code>opsforge theme [set &lt;nom&gt;]</code></td><td>Lister/prévisualiser/persister les thèmes de couleurs</td></tr>
<tr><td><code>opsforge doctor</code></td><td>Bilan de santé complet — système, shell, boîte à outils, <strong>CVE &amp; secrets exposés</strong> (<code>--json</code>)</td></tr>
</table>

> **Exploitable par machine partout.** Un flag global `--json` fait émettre à
> `list`, `status`, `doctor` et `audit` du JSON structuré au lieu de la TUI —
> voir [CI & intégrations](#ci--intégrations).

### Le sélecteur

Lancez le binaire nu pour parcourir par catégorie et installer ce que vous cochez.

- **Onglets (façon k9s) :** `1` Outils · `2` Mises à jour (seulement l'obsolète) ·
  `3` Sécurité (scan CVE en direct des outils installés)
- **Touches :** `space` (co)cher · `u` toutes les mises à jour · `a` tout ce qui
  n'est pas installé · `s` sauver la sélection comme profil · `/` filtrer ·
  `i` installer · `q` quitter
- **Marqueurs :** `[✓]` vert installé · `[✓]` orange mise à jour disponible ·
  `[▸]` cyan sélectionné · `[ ]` gris non installé

---

## Workflows courants

Trois parcours qui montrent comment les pièces s'emboîtent.

### Configurer votre nouvelle machine

Vous changez de laptop ? Reconstruisez votre poste complet depuis un seul
fichier, au lieu d'une journée de config manuelle.

```sh
opsforge snapshot -o my-setup.yaml         # sur votre machine actuelle : outils + shell + thème + guards → un YAML
opsforge apply https://…/my-setup.yaml     # sur la nouvelle : passez le plan en revue, puis tout reconstruit
opsforge shell install && exec zsh         # allumez le shell DevOps
```

### Verrouiller votre CI sur les CVE & secrets

Transformez le binaire que vous utilisez en interactif en une barrière de sécurité
d'une seule ligne.

```sh
opsforge audit --json | tee cve-report.json   # code de sortie non nul sur toute CVE HIGH/CRITICAL — fait échouer le job tout seul
opsforge audit --secrets --json               # échoue aussi sur un identifiant exposé
```

Workflow prêt à l'emploi : [`examples/ci-security-gate.yml`](examples/ci-security-gate.yml).

### Versionner & valider votre politique de guards prod

Versionnez vos propres règles de sûreté prod dans un seul fichier et
gardez-les honnêtes dans le pipeline — comme vous versionneriez le reste de vos
dotfiles.

```sh
opsforge guard init                                            # écrit un guards.yaml de départ, puis committez-le
opsforge guard lint                                            # le valide — code de sortie non nul sur une règle invalide
opsforge guard test "terraform destroy" --context prod --json  # assertez en CI que les destroy prod sont refusés
```

---

## Au-delà des bases

### Profils de stack

Installez toute une stack en une commande — ou sauvez la vôtre :

```sh
opsforge install --profile aws-k8s   # aws, eksctl, kubectl, helm, k9s, terraform…
opsforge profiles                    # liste tout avec le statut d'installation
```

Intégrés : `core`, `k8s`, `aws-k8s`, `gcp-k8s`, `iac`, `observability`,
`security`, `sysadmin`, `netsec`, `secrets`, `ai`. Dans le sélecteur, sélectionnez vos outils et appuyez sur `s` pour
sauver un profil personnel dans `~/.config/opsforge/profiles.yaml` — ensuite
`opsforge install --profile my-stack` le reproduit n'importe où.

### Poste de travail as-code

La config de votre machine ne devrait pas être un flocon unique que vous
reconstruisez à la main :

```sh
opsforge snapshot -o my-setup.yaml    # outils + profils + shell + thème + guards + gestionnaire de versions → un fichier
opsforge apply <fichier-ou-url>       # la reconstruit sur n'importe quelle machine
opsforge apply --check <fichier-ou-url>  # vérifie une machine par rapport à elle, sans rien changer
```

Un snapshot capture **tout** le poste de travail géré — outils installés, vos
profils personnalisés, l'état de l'environnement shell, le **thème** actif, votre
**politique de guards** (le `guards.yaml` brut), et le **gestionnaire de versions**
détecté. `apply` affiche le plan complet et demande confirmation avant de changer
quoi que ce soit (`--yes` pour les scripts), restaurant le thème et les règles de
guards en même temps que les outils.

**Vérifier une machine contre un snapshot de référence.** `apply --check` compare
cette machine à un snapshot **que vous avez figé plus tôt**, **sans rien
modifier**, avec un **code de sortie non nul en cas d'écart** — un outil
manquant, ou un thème/guards/shell/gestionnaire de versions qui diffère. Avec
`--json`, il émet un rapport structuré — `{compliant, missing_tools, drift}` —
pour qu'un job CI puisse vérifier que votre laptop, ou une image de build,
correspond toujours à votre config de référence :

```sh
opsforge apply --check my-setup.yaml            # fait échouer le job au moindre écart
opsforge apply --check my-setup.yaml --json | jq '.compliant'
```

Les snapshots sont **compatibles vers l'avant** : le format a évolué de v1
(outils, profils, shell) à v2 (ajoute thème, guards, gestionnaire de versions), et
les anciens fichiers v1 se chargent toujours — les nouveaux champs restent
simplement non renseignés.

### Audit de sécurité

```sh
opsforge audit             # CVE dans vos outils installés
opsforge audit --secrets   # + identifiants exposés dans l'historique / rc / .env
```

Croise les versions installées avec [OSV.dev](https://osv.dev), triées par
sévérité, avec la version corrigée :

```
⚠ argocd         2.11.0
    [CRITICAL] CVE-2025-47933 Argo CD allows cross-site scripting…  → fixed in 2.13.8
    [HIGH]     CVE-2025-59531 Unauthenticated argocd-server panic…  → fixed in 2.14.20
✓ helm           4.2.3 — no known vulnerabilities
```

Le matching est fait côté client contre les plages affectées d'OSV, donc une CVE
corrigée avant votre version (ou seulement dans un futur majeur) n'est pas
signalée. `--secrets` scanne l'historique du shell, les fichiers rc et les `.env`
locaux à la recherche de tokens AWS/GitHub/GitLab/Slack, de clés privées, de
`--from-literal`, `docker login -p`… avec les valeurs toujours masquées.

### Épingler des versions d'outils

```sh
opsforge install mise
opsforge use terraform@1.5   # l'épingle dans ce répertoire
```

Délègue à **mise** (préféré) ou **asdf** — pas de réinvention du gestionnaire de
versions.

---

## L'environnement shell DevOps

```sh
opsforge shell install && exec zsh
```

Transforme votre **propre zsh** en un environnement sensible au DevOps (modules
sous `~/.config/opsforge/shell/`, `shell uninstall` restaure tout) :

- **Édition calme, à la demande** — rien ne surgit pendant que vous tapez : juste
  une suggestion inline grise issue de votre historique. `↑`/`↓` cherchent dans
  l'historique par le **préfixe de ligne entier** que vous avez tapé, `→` accepte
  toute la suggestion, `Tab` l'accepte mot à mot, et la ligne est colorée par
  syntaxe au fil de la frappe. Même terraform (qui ne fournit aucune complétion
  zsh) et opsforge lui-même sont couverts.

  <table>
  <tr><th align="left">Touche</th><th align="left">Ce qu'elle fait</th></tr>
  <tr><td><code>↑</code> / <code>↓</code></td><td>Parcourt l'historique par le préfixe de ligne tapé (<code>kubectl get pods -n s</code> + <code>↑</code> ne fait défiler que les lignes commençant ainsi)</td></tr>
  <tr><td><code>→</code></td><td>Accepte toute la suggestion grise</td></tr>
  <tr><td><code>Tab</code></td><td>Accepte la suggestion grise mot à mot (<code>ansible-play</code> + <code>Tab</code> → <code>ansible-playbook </code>)</td></tr>
  <tr><td><code>Ctrl-Space</code></td><td>Complétion fichier / commande</td></tr>
  <tr><td><code>Ctrl-R</code></td><td>Recherche dans tout votre historique</td></tr>
  </table>

  Vous préférez l'ancien menu live toujours ouvert (zsh-autocomplete) ? Mettez
  `OPSFORGE_AUTOMENU=1`. Désactivez toute la couche avec `OPSFORGE_INTERACTIVE=0`.
- **Aide `?`** — appuyez sur `?` sur une ligne vide pour une antisèche thémée ;
  tapez `kubectl get ?` pour l'aide de cette commande, rendue sous un en-tête
  encadré avec la syntaxe man colorée par `bat` ; tapez `??` pour qu'une IA
  explique votre dernière erreur.
- **Prompt de contexte** — le prompt de droite affiche le kube `cluster:namespace`
  et devient **rouge dès que le contexte ressemble à de la prod** — une alarme
  *visuelle* passive que vous voyez **avant même de commencer à taper**, aux côtés
  du compte cloud et du workspace terraform (chacun affiché seulement quand
  pertinent). Plus un prompt gauche épuré : répertoire relatif au dépôt, branche
  git avec marqueurs dirty/ahead/behind, durée de la dernière commande, et un `❯`
  qui rougit en cas d'échec. Tout est lu localement — jamais un cloud ou un
  cluster contacté.
- **Guards policy-as-code** — avant une commande destructrice (`kubectl delete`,
  `terraform destroy`, `helm uninstall`…) sur un contexte de production, opsforge
  peut confirmer, avertir ou bloquer — piloté par des [règles
  déclaratives](#guards-policy-as-code), pas des vérifications codées en dur.
  `OPSFORGE_GUARDS=0` pour désactiver.
- **Alias & assistants** — `k`, `tf`, `dc`, plus `kx`/`kn` pour changer de
  contexte/namespace kube (sélecteur fzf quand disponible). Le builtin `history`
  est élargi pour montrer les **200** dernières lignes (`history 1` pour tout), et
  `hg <terme>` grep tout votre historique — tandis que
  [`opsforge history`](#history) le groupe par famille d'outils DevOps.
- **Heads-up proactif** — une fois par session, opsforge affiche une one-line
  compacte dans votre propre shell quand quelque chose sur votre machine mérite
  votre attention : une CVE vient de toucher un outil installé, des mises à jour
  attendent, un secret fuit, ou un opsforge plus récent est sorti. Il lit un
  cache local (`~/.cache/opsforge/`, TTL 6h) et en rafraîchit un périmé en
  arrière-plan, pour que le prompt ne bloque jamais sur le réseau. Lancez
  [`opsforge notify`](#le-digest-notify) pour le détail complet ; coupez le
  heads-up avec `OPSFORGE_NOTIFY=0`.
- **Intégrations** — `fzf`, `zoxide`, `atuin` câblés quand présents.

**Trois couches, trois rôles :** le **prompt** est une alarme *passive* — il
rougit pour que vous **voyiez** que vous êtes en prod ; les
[**guards**](#guards-policy-as-code) sont une barrière *active* — ils
**arrêtent** une commande destructrice ; le
[heads-up **notify**](#le-digest-notify) est une veille *proactive* — il vous
**informe** quand une CVE, une mise à jour ou une fuite tombe sur votre machine.

Chaque module est validé avec `zsh -n` en CI, donc un script cassé ne peut jamais
atteindre votre shell.

<table>
<tr><th align="left">Commande shell</th><th align="left">Ce qu'elle fait</th></tr>
<tr><td><code>opsforge shell install</code></td><td>Installe l'environnement zsh dans <code>~/.zshrc</code> (idempotent)</td></tr>
<tr><td><code>opsforge shell uninstall</code></td><td>Le retire proprement (restaure <code>~/.zshrc</code>)</td></tr>
<tr><td><code>opsforge shell doctor</code></td><td>Montre ce qui est fourni et son état</td></tr>
<tr><td><code>opsforge shell sync</code></td><td>Rafraîchit les modules shell <em>et</em> les complétions en cache (à lancer après une mise à jour d'opsforge)</td></tr>
</table>

### History

Votre historique shell est plein des commandes exactes dont vous avez encore
besoin — enfouies sous tout le reste. `opsforge history` en extrait juste une
famille d'outils DevOps, pour que vous retrouviez le `kubectl port-forward` de la
semaine dernière sans scroller.

```sh
opsforge history              # vue d'ensemble : chaque famille, avec combien de commandes récentes chacune a
opsforge history kube         # commandes kubectl / helm / k9s / argocd… récentes
opsforge history tf           # terraform / tofu / terragrunt
opsforge history git -n 50    # plus de résultats (0 = sans limite)
opsforge history kube --json  # exploitable par machine
```

Les familles intégrées groupent les outils que vous associez naturellement — et
reflètent délibérément les domaines utilisés par les [guards](#guards-policy-as-code)
et les profils, pour que `kube`, `tf`, `cloud`… signifient la même chose partout
dans le produit :

<table>
<tr><th align="left">Famille</th><th align="left">Outils</th></tr>
<tr><td><code>kube</code></td><td>kubectl, helm, k9s, kubectx, kustomize, stern, kubeseal, flux, argocd…</td></tr>
<tr><td><code>git</code></td><td>git, gh, glab, lazygit, tig</td></tr>
<tr><td><code>tf</code></td><td>terraform, tofu, terragrunt, tflint, terraform-docs</td></tr>
<tr><td><code>docker</code></td><td>docker, docker-compose, podman, nerdctl, colima</td></tr>
<tr><td><code>cloud</code></td><td>aws, gcloud, az, doctl, eksctl, flyctl, vercel</td></tr>
<tr><td><code>ansible</code></td><td>ansible, ansible-playbook, ansible-galaxy, ansible-vault</td></tr>
</table>

Passez un nom de famille, ou n'importe quel exécutable pour filtrer sur ce seul
outil. Les résultats sont des commandes distinctes, les plus récentes en premier,
avec un compteur d'exécutions `×N` ; `--limit/-n` les plafonne (défaut 20, `0` =
tout) et `--json` les émet pour les scripts. L'historique est analysé
**passivement** — opsforge lit le fichier, n'exécute jamais rien.

---

## Guards policy-as-code

<div align="center">

![opsforge guard test — un terraform destroy prod refusé par la politique](demo/screenshots/guard.png)

</div>

C'est la partie qu'aucun autre outil ne fait. Homebrew Bundle, mise, chezmoi et
aqua installent vos CLI — aucun d'eux ne **garde-fou la façon dont vous les
utilisez**. opsforge transforme la couche de sûreté prod du shell en un petit
moteur de politique : un ensemble déclaratif de règles qui décide si une commande
destructrice doit s'exécuter, avertir, confirmer ou être refusée — en fonction du
contexte dans lequel vous êtes vraiment.

### La seule règle à comprendre

Un guard ne se déclenche que quand **deux choses sont réunies en même temps** :
une **commande destructrice** *et* un **marqueur de production**. S'il en manque
une → la commande passe intacte — les commandes en lecture seule ne vous
embêtent donc jamais, et les commandes destructrices sur staging ou dev ne vous
gênent pas. C'est un filet de sécurité contre le geste distrait, pas un mur
devant chaque commande.

| Commande | Contexte | Décision | Pourquoi |
|:--|:--|:--:|:--|
| `kubectl delete pod api` | `prod-eks` | ⚠️ confirm | destructrice + prod |
| `kubectl get pods` | `prod-eks` | ✓ allow | prod, mais lecture seule |
| `kubectl delete pod api` | `staging` | ✓ allow | destructrice, mais pas prod |
| `terraform destroy -var-file=prod.tfvars` | *(aucun)* | ⚠️ confirm | la prod est dans la commande elle-même |
| `terraform destroy -var-file=dev.tfvars` | *(aucun)* | ✓ allow | dev, pas prod |
| `terraform plan -var-file=prod.tfvars` | *(aucun)* | ✓ allow | plan est en lecture seule |
| `helm uninstall app` | `prod` | ⚠️ confirm | destructrice + prod |
| `ls` · `git status` · `cat` | `prod` | ✓ allow | rien de destructeur |

Simulez n'importe lequel de ces cas avec `opsforge guard test "<cmd>" --context <ctx>`.

Les règles vivent dans un seul fichier, `~/.config/opsforge/guards.yaml`. Chaque
règle matche une regex de **commande** et une regex de **contexte**, et choisit
une action :

| Action | Effet |
|:--|:--|
| `allow` | s'exécute normalement (aussi le résultat quand rien ne matche) |
| `warn` | affiche le message, puis s'exécute |
| `confirm` | exige de taper `yes` avant de s'exécuter |
| `deny` | bloque la commande purement et simplement |

```yaml
# ~/.config/opsforge/guards.yaml  (le premier match gagne)
version: 1
rules:
  - name: "confirm destructive kubectl on prod"
    match:
      command: "kubectl (delete|drain|cordon|apply|replace)"
      context: "prod|production"
    action: confirm
    message: "This changes PRODUCTION Kubernetes resources."

  - name: "never delete namespaces on prod"
    match:
      command: "kubectl delete (ns|namespace)"
      context: "prod"
    action: deny
    message: "Deleting a prod namespace is forbidden by policy."
```

```sh
opsforge guard init                                    # écrit un guards.yaml de départ commenté
opsforge guard list                                    # montre les règles actives (intégrées ou les vôtres)
opsforge guard test "terraform destroy" --context prod # simule : quelle règle se déclenche, et l'action
opsforge guard lint                                    # valide guards.yaml — code de sortie non nul en cas d'erreur
opsforge guard test "kubectl delete ns" --context prod --json  # {command, context, matched_rule, action, message}
```

**Une politique que vous pouvez versionner et valider en CI.** Comme les règles
vivent dans un seul fichier, vous pouvez committer `guards.yaml` à côté de vos
dotfiles et le garder honnête dans le pipeline :

- `opsforge guard lint` valide la politique active et **sort avec un code non nul**
  quand elle est cassée — une regex invalide, une action inconnue, ou une mauvaise
  version fait échouer le job au lieu de retomber silencieusement sur la politique
  par défaut à l'exécution.
- `opsforge guard test "<cmd>" --context prod --json` émet la décision sous forme
  `{command, context, matched_rule, action, message}`, pour qu'un pipeline puisse
  **vérifier** que, disons, `terraform destroy` est bien `deny`é en prod — le même
  appel `Evaluate` que celui du shell, donc le test ne peut pas diverger du
  comportement réel.

C'est le fossé défensif, prolongé : les guards s'appliquent sur votre propre
shell, et la politique qui les pilote est **testable et versionnable** comme le
reste de votre config — pas un flocon de neige propre à chaque machine.

### Comment opsforge sait que vous êtes en prod

Le « contexte » qu'une règle matche vient de **deux sources**, et opsforge est
délibérément honnête sur les compromis de chacune :

- **Lu passivement depuis votre environnement** — sans lancer une seule commande.
  opsforge récupère le `current-context` de la kubeconfig, `AWS_PROFILE`/`AWS_VAULT`
  (ou `CLOUDSDK_ACTIVE_CONFIG_NAME`), et le workspace terraform
  (`.terraform/environment`). Il **ne lance jamais `kubectl` ni `gcloud`** pour
  savoir où vous êtes, donc évaluer une règle ne peut pas déclencher un login OIDC
  dans le navigateur ni se bloquer sur un CLI wrapper.
- **Lu depuis la commande elle-même** — parce qu'en 2026, les équipes ciblent la
  prod bien plus souvent avec `-var-file=prod.tfvars` ou un dossier
  `environments/prod/` qu'avec un *workspace* terraform. La politique par défaut
  matche donc aussi ces marqueurs **dans la ligne de commande** pour
  `terraform`/`tofu`/`terragrunt`, si bien que
  `terraform destroy -var-file=prod.tfvars` confirme même sans workspace défini.
  `terraform plan …` reste autorisé — c'est en lecture seule.

> **Ayez les idées claires sur ce que c'est.** Les guards sont un **filet de
> sécurité contre le geste distrait** — ils vous rattrapent quand vous changez
> d'env sans le remarquer, pas une erreur délibérée. Ce **n'est pas** une barrière
> de sécurité. La vraie protection prod reste là où elle doit être :
> `prevent_destroy`, des comptes cloud séparés, et des approbations CI. opsforge
> **complète** cette couche, il ne la remplace pas.

### Ce que vous voyez quand un guard se déclenche

Sur un `confirm`, la commande est retenue au prompt jusqu'à ce que vous tapiez
`yes` :

```text
⚠  opsforge guard
   This changes PRODUCTION Kubernetes resources.
   kubectl delete pod api -n payments
   (to skip guards this session: OPSFORGE_GUARDS=0)
Type 'yes' to run this: ▏
```

Un `deny` affiche un **✗ Blocked by opsforge guard** rouge et efface la ligne ;
un `warn` affiche son message et s'exécute quand même.

### Tout est configurable dans un seul fichier

- **Zéro config par défaut.** Sans `guards.yaml`, la politique intégrée ci-dessus
  reproduit exactement l'ancien comportement de confirmation en prod — mettre à
  jour ne change rien tant que vous n'optez pas pour des règles personnalisées.
  Commencez à personnaliser avec `opsforge guard init`, qui dépose un `guards.yaml`
  entièrement commenté que vous pouvez éditer.
- **Rapide sur le chemin critique.** Le shell pré-filtre à moindre coût et n'appelle
  le moteur (`opsforge guard check`, utilisé en interne) que sur les commandes qui
  ont l'air destructrices, pour que votre prompt reste instantané.
- **Échoue en ouvrant, bruyamment.** Un `guards.yaml` cassé ne peut jamais coincer
  votre shell : la commande s'exécute, et l'erreur de parsing est affichée sur
  stderr pour que vous corrigiez votre YAML.

Désactivez tout pour une session avec `OPSFORGE_GUARDS=0`.

---

## Mode projet

<div align="center">

![opsforge sync --check — un rapport de dérive pour l'opsforge.yaml d'un projet](demo/screenshots/sync.png)

</div>

Un snapshot de poste de travail épingle toute une *machine*. Un **projet** a
souvent besoin de moins — juste la boîte à outils dont *ce dépôt* dépend.
Committez un `opsforge.yaml` à sa racine et n'importe qui le reproduit en une
commande — l'angle reproductibilité que possèdent mise et devbox, plus un gate
CVE qu'eux n'ont pas.

```yaml
# opsforge.yaml — à committer à la racine du dépôt
version: 1
tools:
  - kubectl
  - helm
  - terraform
profiles:
  - core          # tire aussi des profils de stack entiers
fail_on: high     # optionnel : sync échoue si un outil requis a une CVE HIGH/CRITICAL
```

```sh
opsforge sync           # installe ce que le manifest déclare et qui manque
opsforge sync --check   # reporte la dérive, code de sortie non nul si un outil requis manque (CI/pre-commit)
opsforge sync --init    # écrit un opsforge.yaml de départ ici
```

`sync` remonte depuis le répertoire courant jusqu'au `opsforge.yaml` le plus
proche, donc il marche depuis n'importe quel sous-répertoire. Il résout `tools` +
`profiles` en une seule liste dédoublonnée, n'installe que ce qui manque (via
Homebrew ou une release GitHub, par outil), et ignore avec un avertissement tout
ce qui n'est pas dans le catalogue.

**Le différenciateur : un gate CVE dans le même fichier.** Mettez `fail_on: high`
(ou `critical`) et `sync` audite *seulement les outils requis par ce projet*
contre [OSV.dev](https://osv.dev) et **échoue** quand l'un porte une CVE de ce
niveau — donc un seul fichier committé vous donne à la fois un **environnement
reproductible** *et* un **gate supply-chain**, ce que mise/devbox ne combinent
pas. Avec `--json`, `sync --check` émet `{compliant, missing, present, unknown,
cve_blocked, fail_on}` pour qu'un pipeline puisse vérifier :

```sh
opsforge sync --check --json | jq '.compliant'   # fait échouer le job en cas de dérive ou de CVE bloquante
```

---

## SBOM & chaîne d'approvisionnement

<div align="center">

![opsforge sbom --audit — un SBOM CycloneDX avec une CVE embarquée, passé dans jq](demo/screenshots/sbom.png)

</div>

opsforge est le seul gestionnaire d'outils qui émet un **SBOM corrélé aux CVE de
votre poste de travail** — un artefact supply-chain consommable par grype,
`trivy sbom`, ou un pipeline de conformité.

```sh
opsforge sbom                # JSON CycloneDX 1.6 de vos outils installés → stdout
opsforge sbom --audit > bom.json   # + CVE embarquées, capturé dans un fichier
```

- **`opsforge sbom`** construit un document **CycloneDX 1.6** où chaque outil
  installé est un composant avec sa **version** détectée et — quand le catalogue
  le mappe à un écosystème de paquets — un **PURL**.
- **`opsforge sbom --audit`** croise OSV.dev et embarque les CVE connues comme
  **vulnerabilities** CycloneDX, chacune liée à son composant avec une sévérité et
  la version de correctif recommandée. Le SBOM sort corrélé aux CVE d'emblée.

Le document part sur stdout (un court résumé sur stderr), donc
`opsforge sbom > bom.json` vous donne un fichier propre plus du retour. C'est un
différenciateur supply-chain 2026 : aucun autre installeur CLI ne vous remet un
inventaire signé de votre boîte à outils *avec* ses vulnérabilités, prêt à
alimenter un scanner ou un gate de conformité.

C'est toute la chaîne supply-chain dans un seul binaire : un **checksum** prouve
que chaque téléchargement est intact, une **signature cosign** prouve que la
release est authentique (voir [le catalogue](#le-catalogue)), et le **SBOM**
prouve ce que vous avez obtenu au final — CVE comprises.

### Le digest notify

opsforge n'attend pas que vous lanciez `audit` — `opsforge notify` est **un
seul digest de tout ce qui, sur *votre* machine, mérite votre attention**, au
même endroit :

- les outils installés porteurs d'une **CVE connue** (HIGH/CRITICAL en rouge),
- les outils qui **peuvent être mis à jour**,
- les **identifiants qui fuient** dans votre historique shell / rc / `.env`
  (quand scannés),
- un **opsforge plus récent** que celui que vous exécutez.

Chaque ligne s'accompagne de la commande exacte qui la corrige :

```
  ✗ 1 tool with a HIGH/CRITICAL CVE          → opsforge audit
  ✗ 6 critical secrets leaking in your shell → opsforge audit --secrets
  ⚠ 3 tools can be updated                   → opsforge upgrade -u
```

```sh
opsforge notify            # le digest complet, groupé par sévérité
opsforge notify --json     # le Digest structuré, pour les scripts
opsforge notify --refresh  # recalcule le cache maintenant
opsforge notify --quiet    # juste la one-line compacte (utilisée par le shell)
```

**Un heads-up dans votre shell, une fois par session.** Quand quelque chose
mérite votre attention, le [shell DevOps](#lenvironnement-shell-devops) affiche
une one-line compacte au démarrage — ex. *« opsforge: 1 tool with a
HIGH/CRITICAL CVE · 3 tools can be updated — run `opsforge notify` »* — puis
vous lancez `opsforge notify` pour le détail. Coupez-la avec `OPSFORGE_NOTIFY=0`.

**En cache, instantané, ne bloque jamais.** `notify` lit un cache local sous
`~/.cache/opsforge/` (TTL 6h) et ne fait jamais que le *lire* — un cache périmé
est rafraîchi en arrière-plan (ou à la demande avec `--refresh`), donc ni le
digest ni le heads-up du shell n'attendent sur le réseau. Le même constat
remonte aussi en un coup d'œil dans [`opsforge status`](#aperçu-rapide).

C'est le seul gestionnaire d'outils qui replie CVE, mises à jour, secrets
exposés *et* sa propre self-update dans un seul digest et le pousse,
proactivement, dans votre shell — dès qu'un advisory tombe sur votre boîte à
outils, vous le savez, sans rien lancer.

---

## CI & intégrations

opsforge n'est pas qu'une jolie TUI — un flag global `--json` fait émettre à
`list`, `status`, `doctor` et `audit` du JSON structuré, pour que le même binaire
que vous utilisez en interactif pilote aussi scripts et pipelines.

```sh
opsforge audit --json | jq '.tools[] | select(.vulnerable)'   # seulement les outils affectés
opsforge doctor --json | jq '.status'                         # "healthy" | "warnings" | "failing"
opsforge list all --json | jq '.[] | select(.outdated).name'  # les outils avec une mise à jour
```

Les commandes de sécurité définissent aussi des **codes de sortie qui ont du
sens**, ce qui transforme opsforge en une barrière d'une seule ligne :

- `opsforge audit` (et `--json`) sort avec un **code non nul sur toute CVE
  HIGH/CRITICAL**.
- `opsforge audit --secrets` ajoute les identifiants exposés au rapport ; une
  **fuite critique** sort aussi avec un code non nul.
- `opsforge doctor --json` retourne `{status, passed, warnings, failed, checks[]}`
  et échoue quand une vérification échoue.

Workflow GitHub Actions prêt à l'emploi : [`examples/ci-security-gate.yml`](examples/ci-security-gate.yml)
— il installe opsforge et fait échouer le pipeline sur toute CVE HIGH/CRITICAL ou
identifiant exposé, en téléversant les rapports JSON comme artefacts.

```yaml
# extrait — audit sort avec un code non nul sur HIGH/CRITICAL, faisant échouer le job tout seul
- name: CVE audit
  run: opsforge audit --json | tee cve-report.json
```

### GitHub Action officielle

Sautez le boilerplate d'installation — l'action composite le fait, puis lance les
gates que vous activez (`audit`, `secrets`, `guard-lint`, `sbom`, `baseline`) :

```yaml
- uses: Mrg77/opsforge@v1
  with:
    audit: 'true'          # échoue sur toute CVE HIGH/CRITICAL
    secrets: 'true'        # échoue aussi sur un identifiant exposé
    guard-lint: 'true'     # valide guards.yaml (policy-as-code)
    sbom: 'true'           # émet un SBOM CycloneDX, téléversé comme artefact
    baseline: my-setup.yaml   # vérifie que cette machine correspond à votre snapshot
```

Exemple complet : [`examples/github-action-usage.yml`](examples/github-action-usage.yml).

### Image Docker

Une image distroless (~20–30 Mo, sans gestionnaire de paquets) embarque le
binaire statique — lancez n'importe quelle commande contre une image de build qui
a vos CLI :

```sh
docker run --rm ghcr.io/mrg77/opsforge audit --json
```

### Hooks pre-commit

Verrouillez les commits avec le même moteur de politique, directement depuis
[`.pre-commit-hooks.yaml`](.pre-commit-hooks.yaml) :

```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/Mrg77/opsforge
    rev: v1.0.0
    hooks:
      - id: opsforge-guard-lint   # valide guards.yaml — échoue sur une règle invalide
      - id: opsforge-secrets      # bloque un commit qui expose un identifiant
```

---

## Le catalogue

**287 outils sur 16 catégories** — Kubernetes, Infrastructure as Code, CLI Cloud,
Conteneurs, Git & CI/CD, Observabilité & Monitoring, Logs, Réseau & HTTP,
**Système & SysAdmin**, Bases de données, Sécurité & Conformité, Secrets & Identité,
Serverless & PaaS, Runtime & Versions, Utilitaires, et une nouvelle catégorie
**AI & LLM**. Le catalogue couvre désormais **tous les métiers IT** — pas seulement
Kubernetes et le cloud, mais aussi le développement, le DevOps, le système, le
réseau, la sécurité, les bases de données et l'IA — pour qu'un dev, un ingénieur
DevOps, un ingénieur système, un ingénieur réseau, un DevSecOps ou un ingénieur IA
y trouvent tous leur boîte à outils :

- **Réseau** — `tcpdump`, `iperf3`, `nmap`, `wireguard`…
- **Système & SysAdmin** — `htop`, `tmux`, `zellij`, `rclone`…
- **Sécurité & pentest** — `nuclei`, `ffuf`, `semgrep`, `trivy`, `opa`…
- **Bases de données** — `mongosh`, `litecli`, `atlas`…
- **Observabilité, GitOps & pipelines** — `prometheus`, `otel-cli`, `grafana`,
  `argo`, `tekton`/`tkn`, `dagger`…
- **AI & LLM** — `ollama`, `llm`, `aichat`, `mods`, `aider`, `fabric`,
  `gemini-cli`, `promptfoo`, `codex`…

C'est un unique [fichier YAML](internal/catalog/catalog.yaml) embarqué — ajouter un
outil est une PR de cinq lignes.

**Deux backends d'installation, choisis par outil à l'exécution :**

- **Homebrew** (quand présent dans le PATH) — toujours la dernière release ;
  `opsforge upgrade` rafraîchit toute la boîte à outils.
- **Releases GitHub** — pour les hôtes sans Homebrew (Linux nu, images CI), les
  outils avec un bloc `github:` sont installés en téléchargeant leur binaire de
  release dans `~/.local/bin`. Aucun gestionnaire de paquets requis.

Forcez-en un avec `OPSFORGE_BACKEND=brew|github` ; définissez le répertoire cible
avec `OPSFORGE_BIN_DIR`.

**Chaîne d'approvisionnement : vérification de checksum.** Avant qu'un binaire de
release GitHub soit rendu exécutable, opsforge vérifie son **SHA-256 contre un
checksum publié** — `checksums.txt`, `<asset>.sha256`, ou un template déclaré par
outil via le champ `checksum:` du catalogue. Une non-correspondance **refuse
l'installation** ; une release qui ne publie aucun checksum est un avertissement,
pas un échec (au mieux, pour que les ~85 % de projets qui n'en fournissent aucun
s'installent quand même).

**Chaîne d'approvisionnement : provenance signée.** Les releases d'opsforge
elles-mêmes sont **signées keyless avec [cosign](https://github.com/sigstore/cosign)
(Sigstore)** — aucune clé longue durée, le certificat est lié à l'identité OIDC
GitHub du workflow de release — plus une **attestation de build-provenance SLSA**
GitHub native. La release publie `checksums.txt.sig` + `checksums.txt.pem` à côté
de `checksums.txt`. À la **self-update**, si `cosign` est installé localement,
opsforge vérifie cette signature contre l'identité attendue et affiche
*« signature verified (cosign, keyless) »* — un checksum valide dont la signature
ne vérifie **pas** est refusé comme une non-correspondance. Vérifiez-la
vous-même :

```sh
cosign verify-blob \
  --certificate checksums.txt.pem \
  --signature   checksums.txt.sig \
  --certificate-identity-regexp '^https://github.com/Mrg77/opsforge/\.github/workflows/release\.yml@.*' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  checksums.txt
```

### Ajouter vos propres outils

Le catalogue n'est pas une liste fermée. Pointez opsforge vers un **overlay** et
vos propres outils — CLI internes ou privés — apparaissent dans le sélecteur, les
profils et chaque commande, **sans aucune PR**. Deux façons d'en charger un :

- Déposez un ou plusieurs fichiers dans `~/.config/opsforge/catalog.d/*.yaml`
  (mergés par ordre alphabétique).
- Ou définissez `OPSFORGE_CATALOG=/chemin/vers/mon-catalogue.yaml`.

Le format est exactement celui du catalogue — des `categories:` avec des `tools:`
(`name`, `bin`, `brew`, `description`), et optionnellement des `profiles:` :

```yaml
# ~/.config/opsforge/catalog.d/internal.yaml
categories:
  - name: Internal
    tools:
      - name: acme-cli
        bin: acme
        brew: acmecorp/tap/acme-cli
        description: CLI de déploiement interne d'ACME Corp
```

La sémantique de merge est prévisible :

- Un outil au nom **nouveau** est **ajouté** au catalogue.
- Un outil au nom **existant** **remplace** celui du catalogue — épinglez une
  formule interne, changez de source, ajustez une description.
- Un profil au nom existant est de même **remplacé**.
- **Les champs YAML inconnus sont rejetés**, pour qu'une typo échoue bruyamment au
  lieu d'être silencieusement ignorée.

C'est ainsi que vous intégrez vos propres CLI internes ou privés dans opsforge :
gardez un overlay à côté de vos dotfiles et votre outillage maison s'installe de
la même façon que le catalogue public.

---

## Thèmes

Toute l'UI est thémable — une seule palette pilote chaque commande :

```sh
opsforge theme              # liste tous les thèmes avec un aperçu de couleurs
opsforge theme dracula      # en prévisualise un
opsforge theme set dracula  # le persiste — chaque commande suit, sans rechargement
```

Thèmes : `forge` (par défaut), `nord`, `dracula`, `gruvbox`, `light`, `mono`,
`auto`. `auto` s'accorde au fond de votre terminal ; `mono` est sans couleur pour
les logs/CI. Le thème pilote **chaque commande *et* le sélecteur interactif** —
une seule palette, aucune couleur par défaut égarée nulle part. Précédence :
`$OPSFORGE_THEME` › sauvé (`theme set`) › auto.

---

## Points forts d'ingénierie

Les parties vers lesquelles pointer un relecteur :

- **Un moteur de politique pour le shell.** Les guards prod ne sont pas des `if`
  codés en dur — c'est une politique déclarative (regex × contexte →
  allow/warn/confirm/deny), le premier match gagne, validée au chargement, avec un
  défaut intégré qui préserve le comportement. Le contexte est lu passivement
  (kubeconfig / env / workspace tf) donc l'évaluation ne déclenche jamais un login
  OIDC, et le shell n'appelle le moteur que sur les commandes qui ont l'air
  destructrices.
- **Audit CVE avec un vrai matching de version.** Interroge OSV.dev par outil,
  filtre les vulnérabilités *côté client* contre les plages affectées d'OSV (semver
  `introduced`/`fixed`) et dédoublonne les CVE listées sous plusieurs ID
  d'advisory — pour ne signaler que ce qui affecte la version que vous exécutez,
  avec le correctif sur votre branche. La sévérité vient d'un vrai **calcul de
  score de base CVSS v3.1** à partir du vecteur OSV, pour qu'une CVE critique ne
  soit jamais mal classée ni ratée.
- **Vérification de checksum de la chaîne d'approvisionnement.** Les binaires de
  release GitHub sont vérifiés en SHA-256 contre un checksum publié
  (`checksums.txt`, `<asset>.sha256`, ou un template `checksum:` du catalogue)
  *avant* d'être rendus exécutables — une non-correspondance refuse l'installation ;
  une release sans checksum se dégrade en avertissement.
- **Une mise à jour qui vérifie sa propre intégrité — et sa provenance.**
  `opsforge self update` récupère la dernière release, vérifie son SHA-256 publié,
  et seulement ensuite remplace le binaire en cours d'exécution — atomiquement. La
  même garantie de chaîne d'approvisionnement que l'installeur donne à vos outils,
  opsforge se l'applique à lui-même : un asset falsifié n'est jamais rendu
  exécutable. Comme nos releases sont **signées cosign keyless**, la self-update
  **vérifie aussi cette signature** (quand cosign est installé) contre l'identité
  OIDC du workflow de release — une signature publiée mais invalide est refusée
  comme une non-correspondance. `--check` signale la disponibilité avec un code de
  sortie pour cron/CI, et un build de dev (aucun tag de release à comparer) est un
  no-op sûr.
- **Releases signées keyless avec provenance SLSA.** Les releases sont signées
  avec **cosign keyless (Sigstore/Fulcio)** à partir de l'identité OIDC de GitHub
  Actions — aucune clé à stocker — et portent une **attestation de
  build-provenance SLSA** GitHub native. `checksums.txt.sig` + `checksums.txt.pem`
  accompagnent chaque release ; n'importe qui peut les `cosign verify-blob` contre
  l'identité du workflow.
- **Une seule source de vérité pour les familles d'outils.** Les « familles »
  DevOps (`kube`, `tf`, `cloud`…) sur lesquelles `history` filtre et dont le
  pré-filtre des guards dérive vivent désormais dans un seul package
  (`internal/families`) — la taxonomie autrefois codée en dur à trois endroits
  divergents. Ajouter un outil à une famille, ou un nouveau verbe dangereux, est un
  changement d'une ligne consommé partout d'un coup.
- **Exploitable par machine, avec des codes de sortie qui ont du sens.** Un flag
  global `--json` rend `list`/`status`/`doctor`/`audit` en JSON structuré ; `audit`
  sort avec un code non nul sur les CVE HIGH/CRITICAL (et les fuites de secrets
  critiques avec `--secrets`), pour qu'il s'insère en CI comme barrière de sécurité
  sans script d'enrobage.
- **Un SBOM corrélé aux CVE de votre poste de travail.** `opsforge sbom` construit
  un document CycloneDX 1.6 à partir des outils *détectés* — chacun un composant
  avec sa version et, quand mappé, un PURL — et `--audit` y embarque les CVE
  d'OSV.dev comme vulnerabilities CycloneDX liées. Aucun autre gestionnaire d'outils
  n'émet un inventaire signé de votre boîte à outils *avec* ses vulnérabilités,
  alimentable par grype/trivy ou un gate de conformité.
- **Un seul digest en cache, sans jamais bloquer.** `opsforge notify` agrège CVE,
  mises à jour disponibles, secrets exposés et un opsforge plus récent dans un
  unique digest en cache (`internal/notices`, `~/.cache/opsforge/`, TTL 6h). Le
  shell (une one-line une fois par session via `notify.zsh`) comme `opsforge
  status` le lisent *sans* appel réseau synchrone — un cache périmé est recalculé
  dans un processus détaché en arrière-plan — donc le chemin du heads-up ne peut
  jamais bloquer votre prompt. Aucun autre gestionnaire d'outils ne pousse une
  CVE, une mise à jour ou une fuite fraîche dans votre shell.
- **Env reproductible + gate CVE dans un seul fichier.** Un `opsforge.yaml`
  committé (`version`, `tools`, `profiles`, `fail_on`) fait reproduire à
  `opsforge sync` la boîte à outils d'un dépôt — et `fail_on: high|critical` audite
  *seulement les outils requis* et fait échouer le sync sur une CVE correspondante.
  C'est la reproductibilité que possèdent mise et devbox, plus un gate supply-chain
  qu'eux ne combinent pas.
- **Détection sûre pour l'auth.** Sonder `kubectl --version` là où kubectl est un
  dispatcher de SDK cloud câblé à un plugin OIDC peut faire surgir un login
  navigateur. Chaque sonde tourne avec un `KUBECONFIG` neutralisé et un
  `WaitDelay`, pour que la détection ne déclenche jamais d'auth ni ne se bloque sur
  un CLI wrapper retenant le pipe de sortie.
- **Le catalogue ne peut pas mentir.** Un job CI valide les 287 références brew
  contre l'API Homebrew et chaque template d'asset GitHub contre la vraie dernière
  release de l'outil (darwin/linux × amd64/arm64) — une formule renommée est
  attrapée avant qu'un utilisateur ne la rencontre en plein milieu d'une
  installation.
- **Cas limites Homebrew gérés.** Auto-tape les taps tiers et récupère des conflits
  de lien (`brew link --overwrite`) qui feraient échouer une mise à jour de docker.
- **Ne casse jamais votre shell.** Les modules sont vérifiés au `zsh -n` en CI ; le
  snippet `shell env` ne fait que des recherches de PATH (aucun sous-processus)
  pour un démarrage rapide.

### Architecture

```
cmd/                Commandes Cobra (install, status, audit, guard, sync, sbom, snapshot, apply, self, doctor, shell, theme…)
internal/catalog/   Catalogue YAML embarqué + validation brew/github
internal/project/   Manifest opsforge.yaml : résolution tools/profiles, plan de dérive, gate CVE (sync)
internal/sbom/      Builder CycloneDX 1.6 (composants + PURL + vulnerabilities CVE embarquées)
internal/detect/    Détection concurrente PATH + version + brew-outdated
internal/installer/ Routeur de backend : Homebrew + téléchargement releases GitHub (checksum.go : vérif SHA-256 ; auto-update)
internal/audit/     Client OSV.dev + matching de version côté client + scoring CVSS v3.1
internal/families/  Source de vérité unique des familles d'outils DevOps (consommée par history + pré-filtre des guards)
internal/history/   Lecteur passif d'historique shell + regroupement par famille d'outils DevOps
internal/secrets/   Scanner d'identifiants exposés
internal/notices/   Digest en cache derrière `opsforge notify` (CVE + mises à jour + secrets + self-update)
internal/output/    Émetteur JSON exploitable par machine pour le flag --json
internal/snapshot/  Capture / apply / rapport d'écart --check du poste de travail
internal/tui/       Sélecteur Bubble Tea avec onglets (stylé par le thème)
internal/shellcfg/  Modules d'environnement zsh (dont notify.zsh) + cache de complétions + moteur de politique des guards (policy.go)
internal/ui/        Identité visuelle partagée + thèmes
```

---

## Développement

```sh
go test ./...                                   # tests unitaires
OPSFORGE_SKIP_BREW_VALIDATION=1 go test ./...   # saute les vérifications réseau du catalogue
go build -o opsforge .
```

La CI lance gofmt, vet, les tests race sur Linux & macOS, valide le catalogue
contre l'amont, et cross-compile toutes les cibles. Les releases sont coupées par
GoReleaser sur tag.

## Feuille de route

- [ ] Support bash & fish pour la couche shell (actuellement zsh uniquement)
- [ ] Windows natif (winget/scoop + complétions PowerShell)
- [ ] Plus de templates `github:` pour une couverture sans brew complète

## Licence

MIT
