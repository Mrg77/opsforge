package i18n

// fr is the French message table. A key absent here falls back to English via
// T(), so partial translation is safe — add keys as commands are localized.
var fr = map[string]string{
	// ── status ────────────────────────────────────────────────────────────
	"status.short":              "Un cockpit de votre poste DevOps en un coup d'œil",
	"status.header.tag":         "votre poste DevOps en un coup d'œil",
	"status.label.toolbox":      "Outils",
	"status.label.updates":      "Màj",
	"status.label.security":     "Sécurité",
	"status.label.posture":      "Posture",
	"status.label.shell":        "Shell",
	"status.label.versions":     "Versions",
	"status.label.backend":      "Backend",
	"status.label.theme":        "Thème",
	"status.label.profiles":     "Profils",
	"status.toolbox.installed":  "{n}/{total} installés",
	"status.updates.available":  "{n} disponible(s)",
	"status.updates.hint":       "— lancez `opsforge upgrade -u`",
	"status.updates.uptodate":   "tout est à jour",
	"status.security.pending":   "scan en attente — `opsforge audit` pour un rapport complet",
	"status.security.highcrit":  "{n} outil(s) avec des CVE HAUTES/CRITIQUES",
	"status.security.cves":      "{n} outil(s) avec des CVE",
	"status.security.audithint": "— `opsforge audit`",
	"status.security.clean":     "aucune CVE connue",
	"status.posture.hint":       "— posture de sécurité de votre poste · `opsforge advise` pour un plan",
	"status.shell.off":          "inactif — `opsforge shell install`",
	"status.shell.active":       "actif",
	"status.versions.none":      "aucun — installez mise pour `opsforge use`",
	"status.backend.github":     "Releases GitHub",
	"status.backend.brew":       "Homebrew + GitHub",
	"status.theme.fromenv":      " (depuis $OPSFORGE_THEME)",
	"status.theme.auto":         " (auto — `opsforge theme set <nom>` pour changer)",
	"status.footer":             "  Lancez `opsforge` pour le menu · `opsforge doctor` pour un bilan complet",
	"status.tip":                "  Astuce : `opsforge audit --secrets` scanne vos outils (CVE) et votre shell (secrets exposés)",

	// ── doctor ────────────────────────────────────────────────────────────
	"doctor.short":      "Bilan de santé complet de votre poste DevOps",
	"doctor.header.tag": "un bilan de santé complet de votre poste DevOps",

	// section headers
	"doctor.section.system":   "Système",
	"doctor.section.shell":    "Environnement shell",
	"doctor.section.toolbox":  "Outils",
	"doctor.section.security": "Sécurité",
	"doctor.section.health":   "Santé",

	// System section
	"doctor.system.homebrew":           "Homebrew",
	"doctor.system.homebrew.available": "disponible",
	"doctor.system.homebrew.notfound":  "introuvable",
	"doctor.system.homebrew.fix":       "installez depuis https://brew.sh (opsforge peut aussi installer via les releases GitHub)",
	"doctor.system.brewpath":           "bin Homebrew dans le PATH",
	"doctor.system.brewpath.fix":       "ajoutez `eval \"$(/opt/homebrew/bin/brew shellenv)\"` à ~/.zprofile",
	"doctor.system.localbin":           "~/.local/bin dans le PATH",
	"doctor.system.localbin.detail":    "(les outils installés via GitHub arrivent ici)",
	"doctor.system.localbin.fix":       "ajoutez `export PATH=\"$HOME/.local/bin:$PATH\"` à ~/.zshrc",
	"doctor.system.vm":                 "Gestionnaire de versions",
	"doctor.system.vm.works":           "{mgr} — `opsforge use <outil>@<ver>` fonctionne",
	"doctor.system.vm.none":            "non installé (optionnel — `opsforge install mise` active `opsforge use`)",

	// Shell section
	"doctor.shell.layer":           "couche shell opsforge",
	"doctor.shell.active":          "active dans ~/.zshrc",
	"doctor.shell.inactive":        "non installée",
	"doctor.shell.layer.fix":       "lancez `opsforge shell install`",
	"doctor.shell.completions":     "Complétions en cache",
	"doctor.shell.completions.n":   "{n} outil(s)",
	"doctor.shell.completions.fix": "lancez `opsforge shell sync`",
	"doctor.shell.plugin.fix":      "installé par `opsforge shell install`",

	// Toolbox section
	"doctor.toolbox.installed":        "Outils installés",
	"doctor.toolbox.installed.n":      "{n} sur {total} outils du catalogue",
	"doctor.toolbox.updates.avail":    "Mises à jour disponibles",
	"doctor.toolbox.updates.detail":   "{n} outil(s) : {list}",
	"doctor.toolbox.updates.fix":      "lancez `opsforge upgrade -u` pour tout mettre à jour",
	"doctor.toolbox.updates.ok":       "Mises à jour",
	"doctor.toolbox.updates.uptodate": "tout est à jour",
	"doctor.toolbox.probe":            "Sonde de version",
	"doctor.toolbox.probe.detail":     "{n} outil(s) ne renvoient pas de version (cosmétique) : {list}",

	// Security section
	"doctor.sec.cves":             "CVE connues",
	"doctor.sec.cves.skipped":     "ignoré (--skip-security)",
	"doctor.sec.cves.noauditable": "aucun outil auditable installé",
	"doctor.sec.cves.scanning":    "  scan des CVE sur OSV.dev…",
	"doctor.sec.cves.clean":       "{n} outil(s) vérifié(s), aucun vulnérable",
	"doctor.sec.cves.affected":    "{n} outil(s) touché(s) : {list}",
	"doctor.sec.cves.fix":         "lancez `opsforge audit` pour le détail, puis `opsforge upgrade` sur les outils touchés",
	"doctor.sec.secrets":          "Secrets exposés",
	"doctor.sec.secrets.clean":    "aucun dans l'historique, les fichiers rc ou les .env locaux",
	"doctor.sec.secrets.found":    "{n} fuite(s) potentielle(s) détectée(s)",
	"doctor.sec.secrets.critical": " ({n} critique(s))",
	"doctor.sec.secrets.fix":      "lancez `opsforge audit --secrets`, puis révoquez et supprimez les identifiants exposés",

	// Health summary
	"doctor.health.passed":   "{n} réussis",
	"doctor.health.warnings": "{n} avertissements",
	"doctor.health.failed":   "{n} échecs",
	"doctor.health.failing":  "Certaines vérifications ont échoué — traitez les indications → ci-dessus.",
	"doctor.health.warned":   "Sain, avec quelques améliorations optionnelles ci-dessus.",
	"doctor.health.allgood":  "Tout est bon. Bon déploiement ! 🔥",
	"doctor.checks.failed":   "{n} vérification(s) en échec",

	// flag help
	"doctor.flag.skipsecurity": "ignorer le scan CVE en ligne (hors-ligne / plus rapide)",

	// ── audit ──
	"audit.short": "Scanne les CVE de vos outils installés — et les secrets exposés sur votre poste",
	"audit.long": `Croise les versions de vos outils installés avec la base de vulnérabilités
OSV.dev et signale ceux qui présentent des CVE connues et devraient être mis à
jour. Seuls les outils dotés d'un mapping OSV dans le catalogue sont vérifiés.

Avec --secrets, scanne aussi les endroits où les identifiants fuient
habituellement — historique du shell, fichiers rc, fichiers .env locaux — et
signale les résultats masqués.`,
	"audit.header.sub.cves":       "versions des outils installés vs base de vulnérabilités OSV.dev",
	"audit.header.sub.secrets":    "CVE de vos outils + identifiants exposés sur votre poste",
	"audit.noauditable":           "Aucun outil auditable installé (aucun outil installé ne porte de mapping OSV).",
	"audit.scanning":              "Audit de {n} outil(s) installé(s) via OSV.dev…",
	"audit.tool.clean":            "{version} — aucune vulnérabilité connue",
	"audit.vuln.fixedin":          "  → corrigé dans {version}",
	"audit.allclean":              "Tous les outils audités sont exempts de vulnérabilités connues.",
	"audit.found":                 "Vulnérabilités détectées",
	"audit.found.hint":            "dans {n} outil(s). Lancez `opsforge upgrade` ou mettez à jour les outils concernés.",
	"audit.secrets.scanning":      "Recherche de secrets exposés sur votre poste…",
	"audit.secrets.scanning.note": "  (historique du shell, fichiers rc, fichiers .env locaux — valeurs masquées)",
	"audit.secrets.clean":         "✓ Aucun identifiant exposé trouvé.",
	"audit.secrets.line":          "ligne",
	"audit.secrets.found":         "{n} fuite(s) potentielle(s) détectée(s)",
	"audit.secrets.found.hint":    "dans {n} emplacement(s).",
	"audit.secrets.cleanup": `  Nettoyage : renouvelez les identifiants réels, puis supprimez les lignes
  (historique : éditez ~/.zsh_history · préférez 'read -s' ou un gestionnaire de secrets)`,
	"audit.flag.secrets": "scanne aussi l'historique du shell, les fichiers rc et les .env locaux à la recherche d'identifiants exposés",

	// ── ai ────────────────────────────────────────────────────────────────
	"ai.short":        "Montre quel moteur d'IA opsforge utilise — ou comment en configurer un",
	"ai.header.tag":   "le moteur d'IA piloté par opsforge",
	"ai.none":         "Aucun moteur d'IA détecté",
	"ai.backend":      "Moteur : ",
	"ai.backend.note": "      opsforge lui transmet un prompt et diffuse la réponse — rien n'est exécuté.",
	"ai.try":          "Essayez :",
	"ai.try.advise":   "opsforge advise    # priorisation IA de vos CVE, mises à jour & secrets",
	"ai.override":     "Vous pouvez forcer le moteur à tout moment via $OPSFORGE_AI_CMD.",

	// ── advise ────────────────────────────────────────────────────────────
	"advise.short":      "Demande à l'IA de prioriser les CVE, mises à jour & secrets de votre poste",
	"advise.header.tag": "plan priorisé par l'IA pour votre poste",
	"advise.nobackend":  "aucun moteur d'IA disponible",
	"advise.scanning":   "  Analyse en cours (CVE · mises à jour · identifiants)…",
	"advise.clean":      "Rien à signaler — votre poste semble propre.",
	"advise.asking":     "  Interrogation de {backend}…",
	"advise.replylang":  "Réponds en français.",
}
