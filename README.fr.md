<div align="center">

# opsforge 🔥

**Votre poste de travail DevOps, forgé en quelques minutes.**

Choisissez vos CLI depuis une interface terminal interactive, installez-les d'un
seul coup, et transformez votre zsh en un environnement DevOps qui connaît votre
contexte — complétion en direct, un prompt qui vous alerte quand vous passez en
prod, et des **guards policy-as-code** qui vous empêchent de flinguer le mauvais
cluster.

opsforge, c'est la **couche supply-chain + policy de votre poste de travail** :
il installe vos outils, encadre la façon dont *vous* les utilisez, et vous remet
un SBOM corrélé aux CVE ainsi qu'un document **OpenVEX** de l'ensemble — priorisé
par le catalogue des vulnérabilités activement exploitées de la CISA. Un outil
perso, pas une plateforme d'équipe — pas de serveur, pas de compte, rien qui
vous enferme.

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

**[Essayer](#essayer-dans-une-sandbox) · [Installation](#installation) · [Aperçu](#aperçu-rapide) · [Workflows](#workflows-courants) · [Shell](#lenvironnement-shell-devops) · [Guards](#guards-policy-as-code) · [Mode projet](#mode-projet) · [SBOM & VEX](#sbom--chaîne-dapprovisionnement) · [Agents IA (MCP)](#agents-ia-mcp) · [CI](#ci--intégrations) · [Catalogue](#le-catalogue) · [Sous le capot](#points-forts-dingénierie)**

</div>

---

## Ce que c'est

opsforge, c'est **trois outils dans un seul binaire** :

| | | |
|:--:|---|---|
| 📦 | **Installeur d'outils** | Un sélecteur interactif parmi **287 CLI triés sur le volet, couvrant tous les métiers de l'IT** — dont une nouvelle catégorie **AI & LLM**. Il détecte ce que vous avez déjà et ce qui a vieilli, puis installe le reste via Homebrew *ou* directement depuis les binaires de release GitHub — même sur une machine Linux nue, sans gestionnaire de paquets. |
| 🐚 | **Shell DevOps** | Une seule commande transforme votre zsh en une expérience façon Warp/Fish : complétion en direct, aide inline via `?`, un prompt qui vous signale la prod, et des [**guards policy-as-code**](#guards-policy-as-code) sur les commandes destructrices. On ne remplace pas votre shell, on ne vous enferme nulle part. |
| 📸 | **Poste de travail & projet as-code** | `opsforge snapshot` exporte toute votre config — outils, profils, shell, thème *et* politique de guards — dans un seul YAML ; un [`opsforge.yaml`](#mode-projet) committé déclare la boîte à outils d'un dépôt et `opsforge sync` la reproduit (avec un gate CVE). `apply --check` / `sync --check` vérifient une machine en CI, et [`opsforge sbom`](#sbom--chaîne-dapprovisionnement) en tire un SBOM corrélé aux CVE. |

### Pourquoi ça existe

Trois frictions récurrentes sur une machine DevOps, chacune résolue d'ordinaire
par un outil différent — ou à la main :

- **Reconstruire un poste**, c'est installer une vingtaine de CLI, puis brancher
  pour chacun les complétions, les alias et un prompt qui tient la route, sur
  chaque machine.
- **Un `kubectl delete` étourdi sur le mauvais contexte** n'a aucune ceinture de
  sécurité : les outils l'exécutent qu'on soit sur staging ou sur prod.
- **Personne ne sait vraiment ce qu'il y a sur la machine** — quelles versions,
  quelles CVE, lesquelles sont activement exploitées.

opsforge réunit tout ça dans un seul binaire parce que ces problèmes partagent
la même donnée (la boîte à outils détectée) et le même lieu (votre shell). C'est
délibérément un **outil perso, pas une plateforme d'équipe** — pas de serveur,
pas de compte, rien qui vous enferme — pour que ça reste quelque chose que vous
lancez, pas quelque chose que vous opérez.

---

## Essayer dans une sandbox

Envie de voir les guards se déclencher sans rien installer ni toucher à de la
vraie infra ? Lancez l'image de démo jetable — un shell zsh déjà forgé, placé
dans un **faux contexte de prod**, avec des stubs no-op `kubectl`/`terraform`/`helm` :

```sh
docker run --rm -it ghcr.io/mrg77/opsforge-demo
```

Elle ouvre un court tour guidé (status → guards → SBOM), puis vous rend la main
dans le shell : tapez vous-même `kubectl delete namespace payments` et regardez
le guard prod l'intercepter. Rien ne peut atteindre un vrai cluster — le contexte
« prod » est un faux kubeconfig d'une ligne, lu passivement, et les outils sont
des stubs.

Vous préférez le navigateur ? Ouvrez-la dans un Codespace — même image, zéro
installation locale :

[![Ouvrir dans GitHub Codespaces](https://github.com/codespaces/badge.svg)](https://codespaces.new/Mrg77/opsforge?quickstart=1)

---

## Installation

```sh
curl -fsSL https://raw.githubusercontent.com/Mrg77/opsforge/main/install.sh | sh
```

Télécharge le bon binaire pour votre OS/arch dans `~/.local/bin` (à surcharger
avec `OPSFORGE_INSTALL_DIR`, à épingler avec `OPSFORGE_VERSION=v1.2.3`). Depuis
les sources : `go install github.com/Mrg77/opsforge@latest`.

Pour rester à jour, `opsforge self update` télécharge la dernière release,
**vérifie son SHA-256 publié avant de remplacer le binaire en place**, et ne fait
rien si vous êtes déjà à jour (`--check` pour cron/CI).

> **Windows :** passez par WSL — l'installation s'appuie sur Homebrew et la couche
> shell vise zsh. Le support natif winget/scoop + PowerShell est prévu.

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
<tr><td><code>opsforge notify [--json]</code></td><td>Un seul digest de ce qui réclame votre attention — CVE, mises à jour, secrets exposés, un opsforge plus récent (voir <a href="#le-digest-notify">notify</a>)</td></tr>
<tr><td><code>opsforge install kubectl helm</code></td><td>Installation non interactive par nom (scriptable)</td></tr>
<tr><td><code>opsforge install --profile aws-k8s</code></td><td>Installe toute une stack prédéfinie en une commande</td></tr>
<tr><td><code>opsforge upgrade [-u] [outil…]</code></td><td>Met tout à jour, seulement l'obsolète (<code>-u</code>), ou les outils nommés</td></tr>
<tr><td><code>opsforge audit [--secrets] [--json]</code></td><td>Scan CVE des outils installés · scan de secrets exposés en option · <code>--json</code> + code de sortie non nul pour verrouiller la CI</td></tr>
<tr><td><code>opsforge guard [init|list|test|lint]</code></td><td>Guards policy-as-code sur les commandes destructrices · <code>lint</code>/<code>test --json</code> les rendent vérifiables en CI (voir <a href="#guards-policy-as-code">Guards</a>)</td></tr>
<tr><td><code>opsforge use terraform@1.5</code></td><td>Épingle une version d'outil ici (délègue à mise/asdf)</td></tr>
<tr><td><code>opsforge sync [--check] [--init]</code></td><td>Installe les outils déclarés par un <code>opsforge.yaml</code> committé · <code>--check</code> signale la dérive pour la CI · gate CVE en option (voir <a href="#mode-projet">Mode projet</a>)</td></tr>
<tr><td><code>opsforge sbom [--audit] [--sign]</code></td><td>Émet un SBOM CycloneDX 1.6 des outils installés · <code>--audit</code> y embarque leurs CVE · <code>--sign</code> ajoute un bundle Sigstore (voir <a href="#sbom--chaîne-dapprovisionnement">SBOM</a>)</td></tr>
<tr><td><code>opsforge vex [--kev] [--sign]</code></td><td>Émet un document OpenVEX des CVE de vos outils · <code>--kev</code> signale celles activement exploitées (CISA KEV) · <code>--sign</code> le signe (voir <a href="#vex--cisa-kev">VEX</a>)</td></tr>
<tr><td><code>opsforge scan &lt;image&gt; [--diff]</code></td><td>Scanne une image de conteneur (via syft/trivy + le moteur OSV d'opsforge) · <code>--diff</code> la corrèle avec votre poste (voir <a href="#scanner-une-image-de-conteneur">scan</a>)</td></tr>
<tr><td><code>opsforge mcp</code></td><td>Lance un serveur MCP en lecture seule pour qu'un agent IA interroge votre poste de travail (voir <a href="#agents-ia-mcp">MCP</a>)</td></tr>
<tr><td><code>opsforge snapshot</code> / <code>apply</code></td><td>Exporter / reconstruire tout un poste de travail</td></tr>
<tr><td><code>opsforge apply --check &lt;fichier-ou-url&gt;</code></td><td>Vérifie une machine par rapport à votre snapshot sans la modifier · code de sortie non nul en cas d'écart (<code>--json</code>)</td></tr>
<tr><td><code>opsforge self [version|update]</code></td><td>Affiche la version ou se met à jour — checksum vérifié avant le remplacement (<code>--check</code> pour CI/cron)</td></tr>
<tr><td><code>opsforge history [famille|outil]</code></td><td>Commandes shell récentes, regroupées par famille d'outils (<code>kube</code>, <code>git</code>, <code>tf</code>… — voir <a href="#history">History</a>)</td></tr>
<tr><td><code>opsforge explain [--last] &lt;cmd&gt;</code></td><td>Demande à votre CLI IA d'expliquer une commande ou votre dernière erreur (le raccourci <code>??</code> du shell)</td></tr>
<tr><td><code>opsforge list [all] [-u]</code></td><td>Outils installés · catalogue complet · seulement les mises à jour (<code>--json</code> pour scripter)</td></tr>
<tr><td><code>opsforge list &lt;terme&gt;</code></td><td>Cherche dans tout le catalogue par nom, description ou catégorie (ex. <code>list dns</code>)</td></tr>
<tr><td><code>opsforge profiles</code></td><td>Profils de stack avec leur statut d'installation</td></tr>
<tr><td><code>opsforge theme [set &lt;nom&gt;]</code></td><td>Lister, prévisualiser ou fixer les thèmes de couleurs</td></tr>
<tr><td><code>opsforge doctor</code></td><td>Bilan de santé complet — système, shell, boîte à outils, <strong>CVE &amp; secrets exposés</strong> (<code>--json</code>)</td></tr>
</table>

> **Lisible par une machine, partout.** Un flag global `--json` fait sortir à
> `list`, `status`, `doctor` et `audit` du JSON structuré au lieu de la TUI —
> voir [CI & intégrations](#ci--intégrations).

### Le sélecteur

Lancez le binaire seul pour parcourir le catalogue par catégorie et installer ce
que vous cochez.

- **Onglets (façon k9s) :** `1` Outils · `2` Mises à jour (uniquement l'obsolète) ·
  `3` Sécurité (scan CVE en direct des outils installés)
- **Touches :** `space` cocher/décocher · `u` toutes les mises à jour · `a` tout ce
  qui n'est pas installé · `s` enregistrer la sélection comme profil · `/` filtrer ·
  `i` installer · `q` quitter
- **Marqueurs :** `[✓]` vert : installé · `[✓]` orange : mise à jour disponible ·
  `[▸]` cyan : sélectionné · `[ ]` gris : non installé

---

## Workflows courants

Trois parcours qui montrent comment les pièces s'emboîtent.

### Mettre en route une nouvelle machine

Vous changez de laptop ? Reconstruisez votre poste complet à partir d'un seul
fichier, au lieu d'une journée de config manuelle.

```sh
opsforge snapshot -o my-setup.yaml         # sur votre machine actuelle : outils + shell + thème + guards → un YAML
opsforge apply https://…/my-setup.yaml     # sur la nouvelle : passez le plan en revue, puis reconstruisez tout
opsforge shell install && exec zsh         # activez le shell DevOps
```

### Faire de votre CI une barrière contre les CVE & les secrets

Le binaire que vous utilisez en interactif devient une barrière de sécurité en une
seule ligne.

```sh
opsforge audit --json | tee cve-report.json   # code de sortie non nul sur toute CVE HIGH/CRITICAL — fait échouer le job à lui seul
opsforge audit --secrets --json               # échoue aussi sur un identifiant exposé
```

Workflow prêt à l'emploi : [`examples/ci-security-gate.yml`](examples/ci-security-gate.yml).

### Versionner & valider votre politique de guards prod

Versionnez vos propres règles de sûreté prod dans un seul fichier et faites-les
respecter en CI — comme vous versionneriez le reste de vos dotfiles.

```sh
opsforge guard init                                            # génère un guards.yaml de départ, puis committez-le
opsforge guard lint                                            # le valide — code de sortie non nul sur une règle invalide
opsforge guard test "terraform destroy" --context prod --json  # vérifiez en CI que les destroy en prod sont bien refusés
```

---

## Au-delà des bases

### Profils de stack

Installez toute une stack en une commande — ou créez la vôtre :

```sh
opsforge install --profile aws-k8s   # aws, eksctl, kubectl, helm, k9s, terraform…
opsforge profiles                    # liste tout avec le statut d'installation
```

Intégrés : `core`, `k8s`, `aws-k8s`, `gcp-k8s`, `iac`, `observability`,
`security`, `sysadmin`, `netsec`, `secrets`, `ai`. Dans le sélecteur, cochez vos
outils et appuyez sur `s` pour enregistrer un profil personnel dans
`~/.config/opsforge/profiles.yaml` — ensuite `opsforge install --profile my-stack`
le reproduit n'importe où.

### Poste de travail as-code

La config de votre machine ne devrait pas être un montage artisanal, différent sur
chaque poste et refait à la main :

```sh
opsforge snapshot -o my-setup.yaml    # outils + profils + shell + thème + guards + gestionnaire de versions → un fichier
opsforge apply <fichier-ou-url>       # le reconstruit sur n'importe quelle machine
opsforge apply --check <fichier-ou-url>  # compare une machine à ce fichier, sans rien changer
```

Un snapshot capture **tout** le poste de travail géré — outils installés, profils
personnalisés, état de l'environnement shell, **thème** actif, **politique de
guards** (le `guards.yaml` brut) et **gestionnaire de versions** détecté. `apply`
affiche le plan complet et demande confirmation avant de toucher à quoi que ce
soit (`--yes` pour les scripts) ; il restaure le thème et les règles de guards en
même temps que les outils.

**Comparer une machine à un snapshot de référence.** `apply --check` compare cette
machine à un snapshot **que vous avez figé plus tôt**, **sans rien modifier**, et
sort avec un **code non nul dès qu'il y a un écart** — un outil manquant, ou un
thème / des guards / un shell / un gestionnaire de versions qui diffère. Avec
`--json`, il produit un rapport structuré — `{compliant, missing_tools, drift}` —
pour qu'un job CI puisse vérifier que votre laptop, ou une image de build,
correspond toujours à votre config de référence :

```sh
opsforge apply --check my-setup.yaml            # fait échouer le job au moindre écart
opsforge apply --check my-setup.yaml --json | jq '.compliant'
```

Les snapshots sont **compatibles vers l'avant** : le format est passé de la v1
(outils, profils, shell) à la v2 (qui ajoute thème, guards, gestionnaire de
versions), et les anciens fichiers v1 se chargent toujours — les nouveaux champs
restent simplement vides.

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

Le matching se fait côté client contre les plages affectées d'OSV : une CVE
corrigée avant votre version (ou seulement dans un futur majeur) n'est donc pas
signalée. `--secrets` passe au crible l'historique du shell, les fichiers rc et
les `.env` locaux pour y débusquer des tokens AWS/GitHub/GitLab/Slack, des clés
privées, des `--from-literal`, `docker login -p`… en masquant toujours les
valeurs.

### Épingler des versions d'outils

```sh
opsforge install mise
opsforge use terraform@1.5   # l'épingle dans ce répertoire
```

Délègue à **mise** (préféré) ou **asdf** — inutile de réinventer un gestionnaire
de versions.

---

## L'environnement shell DevOps

```sh
opsforge shell install && exec zsh
```

Transforme votre **zsh** en un environnement taillé pour le DevOps (modules sous
`~/.config/opsforge/shell/`, `shell uninstall` restaure tout) :

- **Une édition qui reste discrète, à la demande** — rien ne surgit pendant que
  vous tapez : juste une suggestion grise en ligne, issue de votre historique.
  `↑`/`↓` parcourent l'historique en filtrant sur le **début de ligne entier** que
  vous avez tapé, `→` accepte toute la suggestion, `Tab` l'accepte mot à mot, et la
  ligne se colore au fil de la frappe. Même terraform (qui ne fournit aucune
  complétion zsh) et opsforge lui-même sont couverts.

  <table>
  <tr><th align="left">Touche</th><th align="left">Ce qu'elle fait</th></tr>
  <tr><td><code>↑</code> / <code>↓</code></td><td>Parcourt l'historique en filtrant sur le début de ligne tapé (<code>kubectl get pods -n s</code> + <code>↑</code> ne fait défiler que les lignes qui commencent ainsi)</td></tr>
  <tr><td><code>→</code></td><td>Accepte toute la suggestion grise</td></tr>
  <tr><td><code>Tab</code></td><td>Accepte la suggestion grise mot à mot (<code>ansible-play</code> + <code>Tab</code> → <code>ansible-playbook </code>)</td></tr>
  <tr><td><code>Ctrl-Space</code></td><td>Complétion fichier / commande</td></tr>
  <tr><td><code>Ctrl-R</code></td><td>Recherche dans tout votre historique</td></tr>
  </table>

  Vous préférez l'ancien menu toujours ouvert (zsh-autocomplete) ? Mettez
  `OPSFORGE_AUTOMENU=1`. Pour désactiver toute la couche, `OPSFORGE_INTERACTIVE=0`.
- **Aide `?`** — appuyez sur `?` sur une ligne vide pour une antisèche aux couleurs
  du thème ; tapez `kubectl get ?` pour l'aide de cette commande, rendue sous un
  en-tête encadré avec la syntaxe man colorée par `bat` ; tapez `??` pour qu'une IA
  vous explique votre dernière erreur.
- **Prompt contextuel** — le prompt de droite affiche le `cluster:namespace` kube
  et vire au **rouge dès que le contexte ressemble à de la prod** — une alarme
  *visuelle* passive, que vous voyez **avant même de commencer à taper**, à côté du
  compte cloud et du workspace terraform (affichés chacun seulement quand c'est
  pertinent). Et à gauche, un prompt épuré : répertoire relatif au dépôt, branche
  git avec ses marqueurs dirty/ahead/behind, durée de la dernière commande, et un
  `❯` qui rougit en cas d'échec. Tout est lu en local — jamais un cloud ni un
  cluster contacté.
- **Guards policy-as-code** — avant une commande destructrice (`kubectl delete`,
  `terraform destroy`, `helm uninstall`…) dans un contexte de production, opsforge
  peut confirmer, avertir ou bloquer — le tout piloté par des [règles
  déclaratives](#guards-policy-as-code), et non par des vérifications codées en
  dur. `OPSFORGE_GUARDS=0` pour désactiver.
- **Alias & raccourcis** — `k`, `tf`, `dc`, plus `kx`/`kn` pour changer de
  contexte/namespace kube (sélecteur fzf quand il est là). Le builtin `history`
  est élargi pour afficher les **200** dernières lignes (`history 1` pour tout), et
  `hg <terme>` grep l'intégralité de votre historique — tandis que
  [`opsforge history`](#history) le regroupe par famille d'outils DevOps.
- **Un signalement proactif** — une fois par session, opsforge affiche une ligne
  compacte dans votre shell quand quelque chose sur votre machine réclame votre
  attention : une CVE vient de toucher un outil installé, des mises à jour
  attendent, un secret fuit, ou un opsforge plus récent est sorti. Il s'appuie sur
  un cache local (`~/.cache/opsforge/`, TTL 6h) et rafraîchit un cache périmé en
  arrière-plan, pour que le prompt ne bloque jamais sur le réseau. Lancez
  [`opsforge notify`](#le-digest-notify) pour le détail complet ; coupez ce
  signalement avec `OPSFORGE_NOTIFY=0`.
- **Intégrations** — `fzf`, `zoxide`, `atuin` branchés quand ils sont présents.

**Trois couches, trois rôles :** le **prompt** est une alarme *passive* — il
rougit pour que vous **voyiez** que vous êtes en prod ; les
[**guards**](#guards-policy-as-code) sont une barrière *active* — ils
**arrêtent** une commande destructrice ; le
[signalement **notify**](#le-digest-notify) est une veille *proactive* — il vous
**prévient** quand une CVE, une mise à jour ou une fuite tombe sur votre machine.

Chaque module est validé au `zsh -n` en CI : un script cassé ne peut donc jamais
arriver jusqu'à votre shell.

<table>
<tr><th align="left">Commande shell</th><th align="left">Ce qu'elle fait</th></tr>
<tr><td><code>opsforge shell install</code></td><td>Installe l'environnement zsh dans <code>~/.zshrc</code> (idempotent)</td></tr>
<tr><td><code>opsforge shell uninstall</code></td><td>Le retire proprement (restaure <code>~/.zshrc</code>)</td></tr>
<tr><td><code>opsforge shell doctor</code></td><td>Montre ce qui est fourni et dans quel état</td></tr>
<tr><td><code>opsforge shell sync</code></td><td>Rafraîchit les modules shell <em>et</em> les complétions en cache (à lancer après avoir mis opsforge à jour)</td></tr>
</table>

### History

Votre historique shell regorge des commandes exactes dont vous aurez à nouveau
besoin — noyées sous tout le reste. `opsforge history` en isole une seule famille
d'outils DevOps, pour que vous retrouviez le `kubectl port-forward` de la semaine
dernière sans faire défiler des pages.

```sh
opsforge history              # vue d'ensemble : chaque famille, avec son nombre de commandes récentes
opsforge history kube         # commandes kubectl / helm / k9s / argocd… récentes
opsforge history tf           # terraform / tofu / terragrunt
opsforge history git -n 50    # plus de résultats (0 = sans limite)
opsforge history kube --json  # lisible par une machine
```

Les familles intégrées regroupent les outils que vous associez naturellement — et
reprennent volontairement les domaines utilisés par les [guards](#guards-policy-as-code)
et les profils, pour que `kube`, `tf`, `cloud`… veuillent dire la même chose
partout dans le produit :

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
outil. Les résultats sont des commandes distinctes, les plus récentes en tête,
avec un compteur d'exécutions `×N` ; `--limit/-n` les plafonne (20 par défaut,
`0` = tout) et `--json` les sort pour les scripts. L'historique est analysé
**passivement** — opsforge lit le fichier, il n'exécute jamais rien.

---

## Guards policy-as-code

<div align="center">

![opsforge guard test — un terraform destroy prod refusé par la politique](demo/screenshots/guard.png)

</div>

Des outils comme Homebrew Bundle, mise, chezmoi et aqua installent vos CLI ;
opsforge ajoute une couche au-dessus — il **pose des garde-fous sur leur usage**.
Il transforme la couche de sûreté prod du shell en un petit moteur de politique :
un jeu de règles déclaratives qui décide si une commande destructrice doit
s'exécuter, avertir, demander confirmation ou être refusée — selon le contexte
dans lequel vous vous trouvez réellement.

### La seule règle à retenir

Un guard ne se déclenche que lorsque **deux conditions sont réunies en même
temps** : une **commande destructrice** *et* un **marqueur de production**. S'il en
manque une, la commande passe sans être touchée — les commandes en lecture seule
ne vous embêtent donc jamais, et les commandes destructrices sur staging ou dev ne
vous ralentissent pas. C'est un filet de sécurité pour le geste étourdi, pas un mur
devant chaque commande.

| Commande | Contexte | Décision | Pourquoi |
|:--|:--|:--:|:--|
| `kubectl delete pod api` | `prod-eks` | ⚠️ confirm | destructrice + prod |
| `kubectl get pods` | `prod-eks` | ✓ allow | prod, mais en lecture seule |
| `kubectl delete pod api` | `staging` | ✓ allow | destructrice, mais pas en prod |
| `terraform destroy -var-file=prod.tfvars` | *(aucun)* | ⚠️ confirm | la prod est dans la commande elle-même |
| `terraform destroy -var-file=dev.tfvars` | *(aucun)* | ✓ allow | dev, pas prod |
| `terraform plan -var-file=prod.tfvars` | *(aucun)* | ✓ allow | un plan est en lecture seule |
| `helm uninstall app` | `prod` | ⚠️ confirm | destructrice + prod |
| `ls` · `git status` · `cat` | `prod` | ✓ allow | rien de destructeur |

Simulez n'importe lequel de ces cas avec `opsforge guard test "<cmd>" --context <ctx>`.

La politique intégrée va bien au-delà de Kubernetes et Terraform : elle rattrape
aussi un **`git push --force` / `reset --hard` sur `main`**, les appels **cloud**
destructeurs (`aws s3 rm --recursive`, `ec2 terminate`, `eks/rds/cloudformation
delete`, `gcloud`/`az … delete` en prod), les pièges **conteneurs** (`docker
system prune`, `volume rm`, `rm -f`) et les effacements de **bases de données**
(`FLUSHALL`, `DROP DATABASE` en prod) — les commandes du quotidien qui méritent
un second coup d'œil, pas seulement les plus évidentes.

Les règles tiennent dans un seul fichier, `~/.config/opsforge/guards.yaml`. Chaque
règle matche une regex de **commande** et une regex de **contexte**, et choisit
une action :

| Action | Effet |
|:--|:--|
| `allow` | s'exécute normalement (c'est aussi le résultat quand rien ne matche) |
| `warn` | affiche le message, puis s'exécute |
| `confirm` | exige de taper `yes` avant de s'exécuter |
| `deny` | bloque purement et simplement la commande |

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
tiennent dans un seul fichier, vous pouvez committer `guards.yaml` à côté de vos
dotfiles et le faire respecter en CI :

- `opsforge guard lint` valide la politique active et **sort avec un code non nul**
  quand elle est cassée — une regex invalide, une action inconnue ou une mauvaise
  version fait échouer le job, au lieu de retomber en silence sur la politique par
  défaut à l'exécution.
- `opsforge guard test "<cmd>" --context prod --json` renvoie la décision sous la
  forme `{command, context, matched_rule, action, message}`, pour qu'un pipeline
  puisse **vérifier** que, mettons, `terraform destroy` est bien `deny`é en prod —
  c'est le même appel `Evaluate` que celui du shell, le test ne peut donc pas
  diverger du comportement réel.

Les guards s'appliquent sur votre propre shell, et la politique qui les pilote
est **testable et versionnable** comme le reste de votre config — au lieu d'être
bricolée à la main, différente sur chaque machine.

### Comment opsforge sait que vous êtes en prod

Le « contexte » sur lequel une règle matche provient de **deux sources**, et
opsforge assume ouvertement les compromis de chacune :

- **Lu passivement depuis votre environnement** — sans lancer une seule commande.
  opsforge récupère le `current-context` de la kubeconfig, `AWS_PROFILE`/`AWS_VAULT`
  (ou `CLOUDSDK_ACTIVE_CONFIG_NAME`) et le workspace terraform
  (`.terraform/environment`). Il **ne lance jamais `kubectl` ni `gcloud`** pour
  savoir où vous êtes : évaluer une règle ne peut donc pas déclencher un login OIDC
  dans le navigateur ni rester bloqué sur un CLI wrapper.
- **Lu depuis la commande elle-même** — parce qu'en 2026, les équipes ciblent bien
  plus souvent la prod avec `-var-file=prod.tfvars` ou un dossier
  `environments/prod/` qu'avec un *workspace* terraform. La politique par défaut
  matche donc aussi ces marqueurs **dans la ligne de commande** pour
  `terraform`/`tofu`/`terragrunt` : `terraform destroy -var-file=prod.tfvars`
  demande confirmation même sans workspace défini. `terraform plan …` reste
  autorisé — c'est en lecture seule.

> **Voyez clairement ce que c'est.** Les guards sont un **filet de sécurité contre
> le geste étourdi** — ils vous rattrapent quand vous changez d'env sans le
> remarquer, pas face à une erreur délibérée. Ce **n'est pas** une barrière de
> sécurité. La vraie protection prod reste à sa place : `prevent_destroy`, des
> comptes cloud séparés, des approbations en CI. opsforge **complète** cette
> couche, il ne la remplace pas.

### Ce que vous voyez quand un guard se déclenche

Sur un `confirm`, la commande reste bloquée au prompt jusqu'à ce que vous tapiez
`yes` :

```text
⚠  opsforge guard
   This changes PRODUCTION Kubernetes resources.
   kubectl delete pod api -n payments
   (to skip guards this session: OPSFORGE_GUARDS=0)
Type 'yes' to run this: ▏
```

Un `deny` affiche un **✗ Blocked by opsforge guard** en rouge et efface la ligne ;
un `warn` affiche son message et s'exécute quand même.

### Tout se configure dans un seul fichier

- **Zéro config par défaut.** Sans `guards.yaml`, la politique intégrée ci-dessus
  reproduit à l'identique l'ancien comportement de confirmation en prod — une mise
  à jour ne change rien tant que vous n'adoptez pas de règles personnalisées.
  Lancez `opsforge guard init` pour commencer : il dépose un `guards.yaml`
  entièrement commenté, prêt à être édité.
- **Rapide sur le chemin critique.** Le shell pré-filtre à moindre coût et n'appelle
  le moteur (`opsforge guard check`, utilisé en interne) que sur les commandes qui
  semblent destructrices : votre prompt reste instantané.
- **En cas d'erreur, il laisse passer — mais le fait savoir.** Un `guards.yaml`
  cassé ne peut jamais bloquer votre shell : la commande s'exécute, et l'erreur de
  parsing part sur stderr pour que vous corrigiez votre YAML.

Désactivez tout le temps d'une session avec `OPSFORGE_GUARDS=0`.

---

## Mode projet

<div align="center">

![opsforge sync --check — un rapport de dérive pour l'opsforge.yaml d'un projet](demo/screenshots/sync.png)

</div>

Un snapshot de poste de travail épingle toute une *machine*. Un **projet** a
souvent besoin de moins — juste la boîte à outils dont *ce dépôt-là* dépend.
Committez un `opsforge.yaml` à sa racine et n'importe qui le reproduit en une
commande : la même reproductibilité que mise ou devbox, avec un gate CVE en plus.

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
opsforge sync --check   # signale la dérive, code de sortie non nul si un outil requis manque (CI/pre-commit)
opsforge sync --init    # écrit un opsforge.yaml de départ ici
```

`sync` remonte depuis le répertoire courant jusqu'au `opsforge.yaml` le plus
proche : il marche donc depuis n'importe quel sous-répertoire. Il fusionne `tools`
et `profiles` en une seule liste dédoublonnée, n'installe que ce qui manque (via
Homebrew ou une release GitHub, selon l'outil), et laisse de côté, avec un
avertissement, tout ce qui n'est pas dans le catalogue.

**Un gate CVE dans le même fichier.** Mettez `fail_on: high` (ou
`critical`) et `sync` audite *uniquement les outils requis par ce projet* contre
[OSV.dev](https://osv.dev), et **échoue** dès que l'un porte une CVE de ce
niveau — un seul fichier committé vous donne donc à la fois un **environnement
reproductible** *et* un **gate supply-chain**, au même endroit. Avec `--json`,
`sync --check` renvoie `{compliant, missing, present, unknown,
cve_blocked, fail_on}` pour qu'un pipeline puisse s'appuyer dessus :

```sh
opsforge sync --check --json | jq '.compliant'   # fait échouer le job en cas de dérive ou de CVE bloquante
```

**Un lockfile pour une reproductibilité vérifiable.** `opsforge sync` écrit aussi
un **`opsforge.lock`** à côté du manifest, qui épingle chaque outil installé à sa
version exacte — la même idée que `package-lock.json` ou `mise.lock`. Committez-le,
et `sync --check` ne se contente plus de vérifier qu'un outil est *présent* : il
vérifie que c'est la *version épinglée*, et signale toute **dérive de version**,
dans la sortie humaine comme en JSON :

```yaml
# opsforge.lock — écrit par sync, vérifié par sync --check (à committer)
version: 1
tools:
  - name: helm
    version: 3.14.0
  - name: kubectl
    version: 1.29.3
```

```sh
opsforge sync --check --json | jq '.version_drift'
# [{"name":"helm","expected":"3.14.0","got":"3.15.1"}]  → code de sortie non nul
```

C'est non-cassant : sans lockfile, `--check` se comporte exactement comme avant ;
un outil épinglé à une version inconnue n'est jamais signalé. C'est ce qui fait
passer le « poste-de-travail-as-code » du vœu pieux à une reproductibilité qu'un
relecteur peut croire — `opsforge.yaml` déclare le *quoi*, `opsforge.lock` prouve
*quelle version exacte*.

---

## SBOM & chaîne d'approvisionnement

<div align="center">

![opsforge sbom --audit — un SBOM CycloneDX avec une CVE embarquée, passé dans jq](demo/screenshots/sbom.png)

</div>

opsforge émet un **SBOM de votre poste de travail corrélé aux CVE** — un artefact
supply-chain exploitable par grype, `trivy sbom` ou un pipeline de conformité.

```sh
opsforge sbom                # JSON CycloneDX 1.6 de vos outils installés → stdout
opsforge sbom --audit > bom.json   # + CVE embarquées, capturé dans un fichier
```

- **`opsforge sbom`** construit un document **CycloneDX 1.6** où chaque outil
  installé est un composant, avec sa **version** détectée et — quand le catalogue
  le rattache à un écosystème de paquets — un **PURL**.
- **`opsforge sbom --audit`** croise OSV.dev et embarque les CVE connues comme
  **vulnerabilities** CycloneDX, chacune reliée à son composant avec sa sévérité et
  la version de correctif recommandée. Le SBOM sort corrélé aux CVE d'emblée.

Le document part sur stdout (un court résumé sur stderr) : `opsforge sbom > bom.json`
vous donne donc un fichier propre, plus un retour à l'écran — un inventaire signé
de votre boîte à outils *avec* ses vulnérabilités, prêt à alimenter un scanner ou un
gate de conformité.

Voilà toute la chaîne supply-chain dans un seul binaire : un **checksum** prouve que
chaque téléchargement est intact, une **signature cosign** prouve que la release est
authentique (voir [le catalogue](#le-catalogue)), et le **SBOM** prouve ce que vous
avez obtenu au final — CVE comprises.

### VEX & CISA KEV

Une simple liste de CVE vous dit qu'une vulnérabilité *existe*. Elle ne vous dit
pas laquelle corriger **en premier** — et comme le NVD a cessé d'enrichir la
plupart des CVE en 2026, le score CVSS sur lequel vous trieriez est souvent
absent ou périmé. `opsforge vex` répond aux deux questions.

```sh
opsforge vex                 # document OpenVEX → stdout (va de pair avec `opsforge sbom`)
opsforge vex --kev           # + met en avant les CVE activement exploitées (CISA KEV)
opsforge vex > vex.json      # capture l'artefact machine
```

- **`opsforge vex`** transforme l'audit en un document **[OpenVEX](https://openvex.dev)
  v0.2.0** : une déclaration lisible par une machine par couple (composant, CVE),
  avec un statut (`affected`) et une **action** — passer à la version corrigée,
  ou surveiller l'advisory quand il n'en existe pas encore. Chaque composant est
  identifié par le **même PURL que le SBOM**, si bien qu'un scanner ou un auditeur
  en aval corrèle les deux d'emblée. La sortie est triée de façon déterministe :
  elle se diffe et se signe proprement.
- **`opsforge vex --kev`** croise le catalogue des **Known Exploited
  Vulnerabilities de la CISA** et met en évidence les CVE **exploitées dans la
  nature** — la poignée à corriger *tout de suite*, avant le reste. Le catalogue
  est récupéré une fois puis mis en cache (`~/.cache/opsforge/kev.json`, TTL 24h) ;
  c'est du best-effort : un accroc réseau se dégrade en « pas de données KEV »,
  jamais en commande en échec.

Prioriser par **exploitabilité** plutôt que par un score qui peut ne pas exister,
c'est une façon sensée de trier en 2026 — et le VEX est l'artefact qui porte ce
verdict jusqu'à ce qui le consomme ensuite.

### Signer les artefacts

Le SBOM comme le VEX peuvent être signés dans un **bundle
[Sigstore](https://www.sigstore.dev)** auto-contenu, que vous remettez à qui
vous voulez :

```sh
opsforge sbom --sign > bom.json      # + un bundle bom.sigstore.json
opsforge vex  --sign > vex.json      # + un bundle vex.sigstore.json
cosign verify-blob --key ~/.config/opsforge/signing.pub \
  --bundle bom.sigstore.json bom.json
```

`--sign` signe le document **par clé** avec une clé opsforge locale persistante
(une clé ECDSA P-256 générée au premier usage sous `~/.config/opsforge/`) et
écrit un bundle Sigstore que `cosign verify-blob` — ou n'importe quel vérifieur
Sigstore — accepte. C'est entièrement **hors-ligne** : pas de login OIDC, pas de
certificat, aucune entrée dans un log de transparence public.

Ce dernier point est un choix assumé, et il mérite d'être précis :

- **La signature locale est par clé, volontairement.** La signature keyless de
  Sigstore publierait l'identité OIDC du signataire (votre email) dans Rekor —
  un log public et immuable — à *chaque* signature, et elle ne prouverait rien
  sur la *provenance* supply-chain d'un document généré à la main sur un poste.
  opsforge signe donc avec une clé locale à la place.
- **Soyez clair sur ce que ça prouve.** Une signature par clé prouve
  l'**intégrité** du document et son **attribution à votre clé** — pas qu'il a
  été produit par un pipeline de confiance. La provenance est une propriété de
  CI : ce sont les *releases* d'opsforge qui sont signées **keyless avec
  provenance SLSA** (voir [le catalogue](#le-catalogue)), parce que là,
  l'identité *est* le pipeline.

Les mêmes primitives, le bon outil pour chaque tâche — l'intégrité locale pour
les artefacts que vous générez, la provenance keyless pour les binaires que vous
livrez.

### Scanner une image de conteneur

`opsforge scan` étend le même moteur OSV à une image de conteneur — et y ajoute
ce qu'un scanner isolé ne fait pas : **la corrélation avec votre propre poste**.

```sh
opsforge scan node:16-alpine          # CVE dans l'image
opsforge scan mon-image-ci --diff     # + son écart avec votre machine
opsforge scan mon-image --json        # lisible machine, non-zéro sur HIGH/CRITICAL
```

opsforge **ne réimplémente pas l'extraction du SBOM d'image** — c'est le travail
de syft/trivy, et les importer en bibliothèque gonflerait le binaire sans rien
apporter. Il pilote donc celui qui est installé (comme il délègue déjà l'épinglage
de versions à mise/asdf), relit le SBOM CycloneDX, et fait passer ces composants
par le moteur OSV **maison** d'opsforge — exactement le matcher qu'utilise
`opsforge audit` sur votre machine, scoring CVSS et versions de correctif par
branche compris.

Avec **`--diff`**, il répond à une question que trivy ne pose pas : *un outil que
je lance en local est-il embarqué dans une version différente dans cette image ?*
Il corrèle les composants de l'image avec votre boîte à outils installée et
signale la dérive de version — l'écart poste↔CI que le « ça marche chez moi »
masque. Comme `audit`, il sort avec un code non nul sur une CVE HIGH/CRITICAL :
il s'insère donc dans un pipeline comme barrière.

> Nécessite `syft` ou `trivy` dans le PATH (`opsforge install syft`). opsforge
> apporte la corrélation et le verdict OSV partagé, pas un scanner d'image de plus.

### Le digest notify

opsforge n'attend pas que vous lanciez `audit` — `opsforge notify` rassemble **au
même endroit tout ce qui, sur *votre* machine, réclame votre attention** :

- les outils installés porteurs d'une **CVE connue** (HIGH/CRITICAL signalés en
  rouge),
- les outils qui **peuvent être mis à jour**,
- les **identifiants qui fuient** dans votre historique shell / rc / `.env`
  (quand vous les scannez),
- un **opsforge plus récent** que celui que vous exécutez.

Chaque ligne s'accompagne de la commande exacte qui règle le problème :

```
  ✗ 1 tool with a HIGH/CRITICAL CVE          → opsforge audit
  ✗ 6 critical secrets leaking in your shell → opsforge audit --secrets
  ⚠ 3 tools can be updated                   → opsforge upgrade -u
```

```sh
opsforge notify            # le digest complet, groupé par sévérité
opsforge notify --json     # le Digest structuré, pour les scripts
opsforge notify --refresh  # recalcule le cache tout de suite
opsforge notify --quiet    # juste la ligne compacte (celle qu'utilise le shell)
```

**Un signalement dans votre shell, une fois par session.** Quand quelque chose
réclame votre attention, le [shell DevOps](#lenvironnement-shell-devops) affiche
une ligne compacte au démarrage — ex. *« opsforge: 1 tool with a HIGH/CRITICAL CVE
· 3 tools can be updated — run `opsforge notify` »* — puis vous lancez
`opsforge notify` pour le détail. Coupez-le avec `OPSFORGE_NOTIFY=0`.

**En cache, instantané, ne bloque jamais.** `notify` s'appuie sur un cache local
sous `~/.cache/opsforge/` (TTL 6h) et se contente toujours de le *lire* — un cache
périmé est rafraîchi en arrière-plan (ou à la demande avec `--refresh`), si bien
que ni le digest ni le signalement du shell n'attendent sur le réseau. Le même
constat remonte aussi d'un coup d'œil dans [`opsforge status`](#aperçu-rapide).

Il réunit CVE, mises à jour, secrets exposés *et* sa propre self-update dans un
seul digest, et le fait remonter de lui-même dans votre shell — dès qu'un advisory
tombe sur votre boîte à outils, vous le savez, sans avoir rien à lancer.

---

## Agents IA (MCP)

opsforge parle le **[Model Context Protocol](https://modelcontextprotocol.io)** —
si bien qu'un agent IA (Claude Code, Cursor, n'importe quel client MCP) peut
*interroger votre poste de travail* via les mêmes données que le CLI calcule,
sans scraping ni devinette.

```sh
claude mcp add opsforge -- opsforge mcp   # enregistre le serveur stdio une fois
```

`opsforge mcp` lance un serveur MCP stdio qui expose **cinq outils en lecture
seule** :

| Outil | Ce que l'agent obtient |
|:--|:--|
| `list_installed_tools` | chaque outil installé, sa version, sa catégorie, et s'il est obsolète |
| `audit_vulnerabilities` | les CVE de ces outils (sévérité max + version corrigée), directement depuis OSV.dev |
| `generate_sbom` | un SBOM CycloneDX 1.6 (avec CVE embarquées en option) |
| `workstation_status` | résumé en un coup d'œil : nombre d'outils installés/obsolètes, état du shell, contexte kube/cloud/tf |
| `check_guard_policy` | évalue une commande contre votre politique de guards — `allow`/`warn`/`confirm`/`deny` — *avant* que l'agent ne propose de la lancer |

> **En lecture seule par conception.** Chaque outil dérive de sources en lecture
> seule — **rien, via MCP, n'installe, ne met à jour ni ne modifie la machine**.
> C'est une frontière assumée : un agent peut *inspecter* votre poste et
> *raisonner* dessus (ce qui est obsolète, ce qui porte une CVE, si une commande
> déclencherait un guard prod), mais les actions qui modifient l'état restent
> derrière le CLI interactif, là où *vous* les confirmez. `check_guard_policy`
> n'exécute jamais la commande et — comme le shell — lire le contexte n'invoque
> jamais `kubectl`/`gcloud`.

opsforge devient ainsi une **source de vérité ancrée** sur laquelle un agent peut
s'appuyer : au lieu d'halluciner vos versions d'outils ou de deviner si
`terraform destroy` est sans risque ici, il pose la question.

---

## CI & intégrations

opsforge n'est pas qu'une jolie TUI — un flag global `--json` fait sortir à
`list`, `status`, `doctor` et `audit` du JSON structuré, pour que le binaire que
vous utilisez en interactif pilote aussi vos scripts et vos pipelines.

```sh
opsforge audit --json | jq '.tools[] | select(.vulnerable)'   # seulement les outils affectés
opsforge doctor --json | jq '.status'                         # "healthy" | "warnings" | "failing"
opsforge list all --json | jq '.[] | select(.outdated).name'  # les outils avec une mise à jour
```

Les commandes de sécurité renvoient aussi des **codes de sortie qui veulent dire
quelque chose**, et c'est ce qui fait d'opsforge une barrière en une seule ligne :

- `opsforge audit` (et `--json`) sort avec un **code non nul sur toute CVE
  HIGH/CRITICAL**.
- `opsforge audit --secrets` ajoute les identifiants exposés au rapport ; une
  **fuite critique** fait aussi sortir avec un code non nul.
- `opsforge doctor --json` renvoie `{status, passed, warnings, failed, checks[]}`
  et échoue dès qu'une vérification échoue.

Workflow GitHub Actions prêt à l'emploi : [`examples/ci-security-gate.yml`](examples/ci-security-gate.yml)
— il installe opsforge et fait échouer le pipeline sur toute CVE HIGH/CRITICAL ou
identifiant exposé, en téléversant les rapports JSON comme artefacts.

```yaml
# extrait — audit sort avec un code non nul sur HIGH/CRITICAL, faisant échouer le job tout seul
- name: CVE audit
  run: opsforge audit --json | tee cve-report.json
```

### GitHub Action officielle

Faites l'économie du boilerplate d'installation — l'action composite s'en charge,
puis lance les gates que vous activez (`audit`, `secrets`, `guard-lint`, `sbom`,
`baseline`) :

```yaml
- uses: Mrg77/opsforge@v1
  with:
    audit: 'true'          # échoue sur toute CVE HIGH/CRITICAL
    secrets: 'true'        # échoue aussi sur un identifiant exposé
    guard-lint: 'true'     # valide guards.yaml (policy-as-code)
    sbom: 'true'           # émet un SBOM CycloneDX, téléversé comme artefact
    vex: 'true'            # émet un document OpenVEX (priorisé KEV), téléversé aussi
    baseline: my-setup.yaml   # vérifie que cette machine correspond à votre snapshot
```

Exemple complet : [`examples/github-action-usage.yml`](examples/github-action-usage.yml).

### Image Docker

Une image distroless (~20–30 Mo, sans gestionnaire de paquets) embarque le binaire
statique — lancez n'importe quelle commande sur une image de build qui contient vos
CLI :

```sh
docker run --rm ghcr.io/mrg77/opsforge audit --json
```

C'est l'image de production — minimale, non interactive. Pour un *bac à sable*
avec un shell et les guards branchés, voir [Essayer dans une sandbox](#essayer-dans-une-sandbox)
(`ghcr.io/mrg77/opsforge-demo`).

### Hooks pre-commit

Filtrez les commits avec le même moteur de politique, directement depuis
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

**287 outils répartis en 16 catégories** — Kubernetes, Infrastructure as Code, CLI
Cloud, Conteneurs, Git & CI/CD, Observabilité & Monitoring, Logs, Réseau & HTTP,
**Système & SysAdmin**, Bases de données, Sécurité & Conformité, Secrets & Identité,
Serverless & PaaS, Runtime & Versions, Utilitaires, et une nouvelle catégorie
**AI & LLM**. Le catalogue couvre désormais **tous les métiers de l'IT** — pas
seulement Kubernetes et le cloud, mais aussi le développement, le DevOps, le
système, le réseau, la sécurité, les bases de données et l'IA — pour qu'un dev, un
ingénieur DevOps, un sysadmin, un ingénieur réseau, un profil DevSecOps ou un
ingénieur IA y trouvent tous leur boîte à outils :

- **Réseau** — `tcpdump`, `iperf3`, `nmap`, `wireguard`…
- **Système & SysAdmin** — `htop`, `tmux`, `zellij`, `rclone`…
- **Sécurité & pentest** — `nuclei`, `ffuf`, `semgrep`, `trivy`, `opa`…
- **Bases de données** — `mongosh`, `litecli`, `atlas`…
- **Observabilité, GitOps & pipelines** — `prometheus`, `otel-cli`, `grafana`,
  `argo`, `tekton`/`tkn`, `dagger`…
- **AI & LLM** — `ollama`, `llm`, `aichat`, `mods`, `aider`, `fabric`,
  `gemini-cli`, `promptfoo`, `codex`…

C'est un unique [fichier YAML](internal/catalog/catalog.yaml) embarqué — ajouter un
outil tient dans une PR de cinq lignes.

**Deux backends d'installation, choisis outil par outil à l'exécution :**

- **Homebrew** (quand il est dans le PATH) — toujours la dernière release ;
  `opsforge upgrade` rafraîchit toute la boîte à outils.
- **Releases GitHub** — pour les hôtes sans Homebrew (Linux nu, images CI), les
  outils dotés d'un bloc `github:` sont installés en téléchargeant leur binaire de
  release dans `~/.local/bin`. Aucun gestionnaire de paquets requis.

Forcez-en un avec `OPSFORGE_BACKEND=brew|github` ; fixez le répertoire cible avec
`OPSFORGE_BIN_DIR`.

**Supply-chain : vérification de checksum.** Avant de rendre exécutable un binaire
de release GitHub, opsforge vérifie son **SHA-256 par rapport à un checksum
publié** — `checksums.txt`, `<asset>.sha256`, ou un template déclaré par outil via
le champ `checksum:` du catalogue. Une non-correspondance **refuse l'installation** ;
une release qui ne publie aucun checksum donne lieu à un avertissement, pas à un
échec (best-effort, pour que les ~85 % de projets qui n'en fournissent aucun
s'installent quand même).

**Supply-chain : provenance signée.** Les releases d'opsforge elles-mêmes sont
**signées keyless avec [cosign](https://github.com/sigstore/cosign) (Sigstore)** —
aucune clé à longue durée de vie, le certificat est lié à l'identité OIDC GitHub du
workflow de release — plus une **attestation de build-provenance SLSA** native de
GitHub. La release publie `checksums.txt.sig` + `checksums.txt.pem` à côté de
`checksums.txt`. Lors de la **self-update**, si `cosign` est installé localement,
opsforge vérifie cette signature par rapport à l'identité attendue et affiche
*« signature verified (cosign, keyless) »* — un checksum valide dont la signature ne
se vérifie **pas** est refusé comme une non-correspondance. Vérifiez-la
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
profils et chaque commande, **sans la moindre PR**. Deux façons d'en charger un :

- Déposez un ou plusieurs fichiers dans `~/.config/opsforge/catalog.d/*.yaml`
  (fusionnés par ordre alphabétique).
- Ou définissez `OPSFORGE_CATALOG=/chemin/vers/mon-catalogue.yaml`.

Le format est exactement celui du catalogue — des `categories:` avec des `tools:`
(`name`, `bin`, `brew`, `description`), et éventuellement des `profiles:` :

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

Les règles de fusion sont prévisibles :

- Un outil au nom **inédit** est **ajouté** au catalogue.
- Un outil au nom **déjà pris** **remplace** celui du catalogue — épinglez une
  formule interne, changez de source, ajustez une description.
- Un profil au nom existant est **remplacé** de la même façon.
- **Les champs YAML inconnus sont rejetés**, pour qu'une typo échoue franchement
  au lieu de passer inaperçue.

C'est comme ça que vous intégrez vos propres CLI internes ou privés dans opsforge :
gardez un overlay à côté de vos dotfiles, et votre outillage maison s'installe
exactement comme le catalogue public.

---

## Thèmes

Toute l'interface est thémable — une seule palette pilote chaque commande :

```sh
opsforge theme              # liste tous les thèmes avec un aperçu de couleurs
opsforge theme dracula      # en prévisualise un
opsforge theme set dracula  # le persiste — chaque commande suit, sans rechargement
```

Thèmes : `forge` (par défaut), `nord`, `dracula`, `gruvbox`, `light`, `mono`,
`auto`. `auto` s'accorde au fond de votre terminal ; `mono` est sans couleur, pour
les logs/CI. Le thème pilote **chaque commande *et* le sélecteur interactif** — une
seule palette, aucune couleur par défaut qui traîne quelque part. Ordre de
priorité : `$OPSFORGE_THEME` › thème enregistré (`theme set`) › auto.

---

## Points forts d'ingénierie

Les points sur lesquels attirer l'œil d'un relecteur :

- **Un moteur de politique pour le shell.** Les guards prod ne sont pas des `if`
  codés en dur — c'est une politique déclarative (regex × contexte →
  allow/warn/confirm/deny), premier match gagnant, validée au chargement, avec un
  défaut intégré qui préserve le comportement. Le contexte est lu passivement
  (kubeconfig / env / workspace tf), donc l'évaluation ne déclenche jamais de login
  OIDC, et le shell n'appelle le moteur que sur les commandes qui semblent
  destructrices.
- **Audit CVE avec un vrai matching de version.** Interroge OSV.dev outil par outil,
  filtre les vulnérabilités *côté client* contre les plages affectées d'OSV (semver
  `introduced`/`fixed`) et dédoublonne les CVE listées sous plusieurs ID d'advisory
  — pour ne signaler que ce qui affecte la version que vous exécutez, avec le
  correctif situé sur votre branche. La sévérité vient d'un vrai **calcul de score
  de base CVSS v3.1** à partir du vecteur OSV, pour qu'une CVE critique ne soit
  jamais mal classée ni oubliée.
- **Vérification de checksum côté supply-chain.** Les binaires de release GitHub sont
  vérifiés en SHA-256 contre un checksum publié (`checksums.txt`, `<asset>.sha256`,
  ou un template `checksum:` du catalogue) *avant* d'être rendus exécutables — une
  non-correspondance refuse l'installation ; une release sans checksum se dégrade en
  simple avertissement.
- **Une mise à jour qui vérifie sa propre intégrité — et sa provenance.**
  `opsforge self update` récupère la dernière release, vérifie son SHA-256 publié,
  et seulement après remplace le binaire en cours d'exécution — de façon atomique.
  La garantie supply-chain que l'installeur offre à vos outils, opsforge se
  l'applique à lui-même : un asset falsifié n'est jamais rendu exécutable. Comme nos
  releases sont **signées cosign keyless**, la self-update **vérifie aussi cette
  signature** (quand cosign est installé) contre l'identité OIDC du workflow de
  release — une signature publiée mais invalide est refusée comme une
  non-correspondance. `--check` signale la disponibilité avec un code de sortie pour
  cron/CI, et un build de dev (aucun tag de release à comparer) ne fait rien, sans
  risque.
- **Releases signées keyless avec provenance SLSA.** Les releases sont signées avec
  **cosign keyless (Sigstore/Fulcio)** à partir de l'identité OIDC de GitHub
  Actions — aucune clé à stocker — et portent une **attestation de build-provenance
  SLSA** native de GitHub. `checksums.txt.sig` + `checksums.txt.pem` accompagnent
  chaque release ; n'importe qui peut les passer à `cosign verify-blob` contre
  l'identité du workflow.
- **Une seule source de vérité pour les familles d'outils.** Les « familles »
  DevOps (`kube`, `tf`, `cloud`…) sur lesquelles `history` filtre et dont dérive le
  pré-filtre des guards vivent désormais dans un seul package (`internal/families`)
  — la taxonomie autrefois codée en dur à trois endroits qui divergeaient. Ajouter
  un outil à une famille, ou un nouveau verbe dangereux, tient en une ligne, prise
  en compte partout d'un coup.
- **Lisible par une machine, avec des codes de sortie qui veulent dire quelque
  chose.** Un flag global `--json` rend `list`/`status`/`doctor`/`audit` en JSON
  structuré ; `audit` sort avec un code non nul sur les CVE HIGH/CRITICAL (et les
  fuites de secrets critiques avec `--secrets`), de sorte qu'il s'insère en CI comme
  barrière de sécurité sans script d'enrobage.
- **Un SBOM de votre poste de travail corrélé aux CVE.** `opsforge sbom` construit
  un document CycloneDX 1.6 à partir des outils *détectés* — chacun un composant
  avec sa version et, quand il est rattaché, un PURL — et `--audit` y embarque les
  CVE d'OSV.dev comme vulnerabilities CycloneDX liées — un inventaire signé de
  votre boîte à outils *avec* ses vulnérabilités, à donner en pâture à grype/trivy
  ou à un gate de conformité.
- **OpenVEX + tri par exploitabilité.** `opsforge vex` réutilise l'audit pour
  émettre un document OpenVEX v0.2.0 — une déclaration `affected` par couple
  (PURL, CVE) avec une action — en partageant le PURL *exact* qu'utilise le SBOM,
  pour que les deux se corrèlent. `--kev` croise le catalogue Known-Exploited de
  la CISA (en cache, TTL 24h, best-effort) pour faire ressortir ce qui est
  exploité *dans la nature* — une façon sensée de prioriser en 2026, maintenant
  que l'enrichissement CVSS n'est plus fiable. Le builder est pur (id/timestamp
  injectés) et trié de façon déterministe : le document se diffe et se signe.
- **Signature Sigstore par clé, un choix assumé.** `sbom --sign` / `vex --sign`
  produisent un bundle Sigstore auto-contenu via `sigstore-go` (une dépendance
  légère — pas cosign-as-library, qui gonflerait le go.mod), en implémentant
  l'interface `Keypair` sur une clé ECDSA P-256 locale persistante : la signature
  est entièrement hors-ligne et la clé publique reste stable d'une signature à
  l'autre. C'est par clé, pas keyless, volontairement : le keyless publierait
  l'identité du signataire dans un log Rekor public et ne prouverait rien sur la
  provenance d'un document généré à la main — la signature locale prouve donc
  intégrité + attribution à la clé, et la provenance keyless reste sur les
  releases signées en CI. Vérifiable avec `cosign verify-blob` ; les octets
  signés sont exactement ceux écrits, pour que la vérification corresponde au
  fichier.
- **Un serveur MCP en lecture seule.** `opsforge mcp` expose le poste de travail
  aux agents IA via le Model Context Protocol, avec cinq outils (outils installés,
  audit CVE, SBOM, statut, vérification de la politique de guards). Les builders
  de payload sont des fonctions pures sur des données qu'opsforge calcule déjà,
  testées unitairement sans client réel ; chaque outil est `ReadOnlyHint` et
  dérive de sources en lecture seule — les commandes qui modifient l'état restent
  derrière le CLI interactif par conception, si bien qu'un agent peut inspecter
  la machine mais jamais la changer.
- **Un lockfile pour une reproductibilité vérifiable.** `opsforge sync` écrit un
  `opsforge.lock` qui épingle la version exacte résolue de chaque outil
  (normalisée, triée par nom pour des diffs propres) ; `sync --check` compare la
  machine à ce fichier et signale la dérive de *version* — pas seulement les
  outils manquants — en JSON comme en sortie humaine, avec un code non nul en cas
  d'écart. `opsforge.yaml` déclare le *quoi*, `opsforge.lock` prouve *quelle
  version* — et il se dégrade proprement (pas de lock → comportement d'avant).
- **Scan d'image par corrélation, pas par réinvention.** `opsforge scan` pilote
  un syft/trivy installé pour le SBOM de l'image (les importer en bibliothèque
  triplerait le go.mod), puis fait passer les composants par le matcher OSV
  *maison* d'opsforge et — avec `--diff` — les corrèle avec la boîte à outils du
  poste pour révéler une dérive de version qu'un scanner isolé ne voit pas. Les
  pièces réutilisables (`internal/imagescan` : un parseur purl→OSV, la
  corrélation) sont testées unitairement ; l'extraction du SBOM est déléguée,
  volontairement.
- **Transport OSV en batch.** L'audit trouve tous les outils affectés en un seul
  appel `/v1/querybatch`, puis récupère chaque CVE distincte une fois — moins de
  requêtes sur le chemin sain et l'endpoint respectueux du rate-limit d'OSV, avec
  un repli par outil si le batch est indisponible. Le moteur de matching
  CVSS/semver est inchangé.
- **Un seul digest en cache, sans jamais bloquer.** `opsforge notify` agrège CVE,
  mises à jour disponibles, secrets exposés et un opsforge plus récent dans un seul
  digest en cache (`internal/notices`, `~/.cache/opsforge/`, TTL 6h). Le shell (une
  ligne une fois par session via `notify.zsh`) comme `opsforge status` le lisent
  *sans* appel réseau synchrone — un cache périmé est recalculé dans un processus
  détaché en arrière-plan — donc le signalement ne peut jamais bloquer votre prompt.
  Une CVE, une mise à jour ou une fuite fraîche remonte dans votre shell sans que
  vous ayez à la demander.
- **Env reproductible + gate CVE dans un seul fichier.** Un `opsforge.yaml` committé
  (`version`, `tools`, `profiles`, `fail_on`) fait reproduire à `opsforge sync` la
  boîte à outils d'un dépôt — et `fail_on: high|critical` audite *uniquement les
  outils requis* et fait échouer le sync sur une CVE correspondante. C'est la
  même reproductibilité que mise et devbox, plus un gate supply-chain dans le même
  fichier.
- **Une détection qui ne casse pas l'auth.** Sonder `kubectl --version` là où
  kubectl est un dispatcher de SDK cloud branché à un plugin OIDC peut faire surgir
  un login navigateur. Chaque sonde tourne avec un `KUBECONFIG` neutralisé et un
  `WaitDelay`, pour que la détection ne déclenche jamais d'auth et ne reste jamais
  bloquée sur un CLI wrapper qui retient le pipe de sortie.
- **Le catalogue ne peut pas mentir.** Un job CI valide les 287 références brew
  contre l'API Homebrew et chaque template d'asset GitHub contre la vraie dernière
  release de l'outil (darwin/linux × amd64/arm64) — une formule renommée est
  repérée avant qu'un utilisateur ne tombe dessus en pleine installation.
- **Les cas tordus de Homebrew sont gérés.** Auto-tap des taps tiers et
  récupération sur les conflits de lien (`brew link --overwrite`) qui, sinon, font
  échouer une mise à jour de docker.
- **Ne casse jamais votre shell.** Les modules sont vérifiés au `zsh -n` en CI ; le
  snippet `shell env` ne fait que des recherches dans le PATH (aucun sous-processus)
  pour un démarrage rapide.

### Architecture

```
cmd/                Commandes Cobra (install, status, audit, guard, sync, sbom, vex, scan, mcp, snapshot, apply, self, doctor, shell, theme…)
internal/catalog/   Catalogue YAML embarqué + validation brew/github + mappings d'écosystème OSV
internal/project/   Manifest opsforge.yaml : résolution tools/profiles, plan de dérive, gate CVE (sync) + opsforge.lock (lock.go)
internal/sbom/      Builder CycloneDX 1.6 (composants + PURL + vulnerabilities CVE embarquées)
internal/vex/       Builder OpenVEX v0.2.0 + récupération/cache du catalogue CISA KEV (kev.go)
internal/attest/    Signature Sigstore par clé du SBOM/VEX (clé ECDSA locale → bundle Sigstore)
internal/imagescan/ Scan d'image de conteneur : SBOM syft/trivy → moteur OSV d'opsforge → corrélation poste
internal/mcp/        Builders de payload MCP en lecture seule (fonctions pures sur catalog/detect/audit/guard)
internal/detect/    Détection concurrente PATH + version + brew-outdated
internal/installer/ Routeur de backend : Homebrew + téléchargement releases GitHub (checksum.go : vérif SHA-256 ; self-update)
internal/audit/     Client OSV.dev + matching de version côté client + scoring CVSS v3.1
internal/families/  Source de vérité unique des familles d'outils DevOps (consommée par history + pré-filtre des guards)
internal/history/   Lecteur passif d'historique shell + regroupement par famille d'outils DevOps
internal/secrets/   Scanner d'identifiants exposés
internal/notices/   Digest en cache derrière `opsforge notify` (CVE + mises à jour + secrets + self-update)
internal/output/    Émetteur JSON lisible par une machine pour le flag --json
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
contre l'amont et cross-compile toutes les cibles. Les releases sont produites par
GoReleaser sur tag.

## Feuille de route

- [ ] Support bash & fish pour la couche shell (zsh uniquement pour l'instant)
- [ ] Windows natif (winget/scoop + complétions PowerShell)
- [ ] Davantage de templates `github:` pour une couverture complète sans brew

## Licence

MIT
