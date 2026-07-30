package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"filippo.io/age"

	"github.com/Mrg77/opsforge/internal/envcatalog"
	"github.com/Mrg77/opsforge/internal/envvault"
	"github.com/Mrg77/opsforge/internal/ui"
)

// envCmd groups the encrypted environment store. `opsforge env` with no
// subcommand shows the status (and whether a session is unlocked).
var envCmd = &cobra.Command{
	Use:   "env",
	Short: "An encrypted store for env vars — persist secrets without cleartext",
	Long: `A passphrase-locked vault of environment variables that survives across
shells WITHOUT writing a secret in cleartext to disk — and without retyping the
passphrase on every operation.

Re-exporting the same AWS/Vault/registry credentials in every new terminal is
tedious; putting them in ~/.zshrc persists them but leaks them in cleartext
(a dotfiles backup, or 'opsforge audit --secrets', will find them). This vault
keeps the convenience while the file at rest is encrypted with age.

Unlock ONCE, then work — like ssh-agent:

    opsforge env unlock          # type the passphrase once (session ~15 min)
    opsforge env set AWS_REGION  # no passphrase — session is open
    opsenv                       # load the vars into your shell (no passphrase)
    opsforge env lock            # forget the session now (or let it expire)

Honesty: once loaded into a shell the values are ordinary environment variables,
visible to child processes; and while unlocked, the vault key sits in a 0600
RAM file that self-expires. The win: nothing in cleartext at rest, no retyping.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(ui.Header("opsforge env", "your encrypted environment vault"))
		fmt.Println()
		if !envvault.Exists() {
			fmt.Printf("  %s %s\n", ui.Dim.Render("•"), ui.Dim.Render("no vault yet — add a variable with `opsforge env set NAME`"))
			return nil
		}
		fmt.Printf("  %s %s\n", ui.OKMark(), ui.Dim.Render("a vault exists (encrypted)"))
		if d := envvault.SessionRemaining(); d > 0 {
			fmt.Printf("  %s %s\n", ui.OKMark(), ui.OK.Render(fmt.Sprintf("unlocked — session ~%d min left", int(d.Minutes())+1)))
		} else {
			fmt.Printf("  %s %s\n", ui.Dim.Render("•"), ui.Dim.Render("locked — `opsforge env unlock` to open a session"))
		}
		fmt.Println()
		fmt.Println("  " + ui.Faint.Render("Load into your shell:  opsenv"))
		fmt.Println("  " + ui.Faint.Render("List variable names:   opsforge env list"))
		return nil
	},
}

// resolveIdentity is the heart of "one passphrase": it returns the vault's
// identity from the active RAM session if there is one, otherwise it prompts
// for the passphrase ONCE, unwraps the identity, AND opens a session so the
// next commands don't ask again. This is why `set`/`load`/`list` stop nagging.
func resolveIdentity() (*age.X25519Identity, error) {
	if id, ok := envvault.Session(); ok {
		return id, nil
	}
	pass, err := readPassphrase(false)
	if err != nil {
		return nil, err
	}
	// Open a session so subsequent operations reuse it; unwrap happens inside.
	if err := envvault.OpenSession(pass, envvault.DefaultTTL); err != nil {
		return nil, err
	}
	id, _ := envvault.Session()
	if id == nil {
		return nil, fmt.Errorf("failed to open a vault session")
	}
	return id, nil
}

// envUnlockCmd opens a session (passphrase once, cached in RAM with a TTL).
var envUnlockCmd = &cobra.Command{
	Use:   "unlock",
	Short: "Unlock the vault for a while — passphrase once, no re-prompt after",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !envvault.Exists() {
			return fmt.Errorf("no vault yet — `opsforge env set NAME` to create one")
		}
		if d := envvault.SessionRemaining(); d > 0 {
			fmt.Printf("  %s already unlocked (~%d min left)\n", ui.OKMark(), int(d.Minutes())+1)
			return nil
		}
		pass, err := readPassphrase(false)
		if err != nil {
			return err
		}
		if err := envvault.OpenSession(pass, envvault.DefaultTTL); err != nil {
			return err
		}
		fmt.Printf("  %s vault unlocked — session ~%d min\n", ui.OKMark(), int(envvault.DefaultTTL.Minutes()))
		return nil
	},
}

// envLockCmd forgets the session now.
var envLockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Forget the unlocked session now (instead of waiting for it to expire)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if envvault.CloseSession() {
			fmt.Printf("  %s vault locked\n", ui.OKMark())
		} else {
			fmt.Println(ui.Dim.Render("  no active session"))
		}
		return nil
	},
}

// envSetCmd adds/updates a variable. The value is read masked from the TTY so
// it never lands in argv (and thus never in shell history / ps output). A
// non-secret value can be passed inline as NAME=value for convenience.
var envSetCmd = &cobra.Command{
	Use:   "set NAME[=value]",
	Short: "Add or update a variable in the encrypted vault",
	Args:  cobra.ExactArgs(1),
	// Complete the NAME argument with the known DevOps env vars (same catalog
	// as `export <TAB>` in the shell layer), so `opsforge env set AWS_<TAB>`
	// offers AWS_SECRET_ACCESS_KEY etc. even for a var you've never set.
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return envcatalog.Completions(), cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		name, value, inline := strings.Cut(args[0], "=")
		if !envvault.ValidName(name) {
			return fmt.Errorf("invalid variable name %q", name)
		}
		if !inline {
			v, err := readSecret(fmt.Sprintf("Value for %s", name))
			if err != nil {
				return err
			}
			value = v
		}

		var id *age.X25519Identity
		if !envvault.Exists() {
			// First set CREATES the vault and FIXES the master passphrase, so
			// confirm it (a typo here would lock the vault forever — no
			// recovery by design). Then open a session immediately.
			fmt.Fprintln(os.Stderr, "Creating a new vault — choose a master passphrase (there is no recovery if you forget it).")
			pass, err := readPassphrase(true)
			if err != nil {
				return err
			}
			created, err := envvault.Create(pass)
			if err != nil {
				return err
			}
			id = created
			_ = envvault.OpenSession(pass, envvault.DefaultTTL) // best-effort: skip a re-prompt
		} else {
			var err error
			if id, err = resolveIdentity(); err != nil {
				return err
			}
		}

		vault, err := envvault.LoadWith(id)
		if err != nil {
			return err
		}
		if err := vault.Set(name, value); err != nil {
			return err
		}
		if err := envvault.SaveWith(vault, id); err != nil {
			return err
		}
		fmt.Printf("  %s %s stored in the encrypted vault\n", ui.OKMark(), ui.Heading.Render(name))
		return nil
	},
}

// envListCmd shows the variable NAMES only — never values.
var envListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the names of variables in the vault (never their values)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !envvault.Exists() {
			fmt.Println(ui.Dim.Render("  no vault yet — `opsforge env set NAME` to create one"))
			return nil
		}
		id, err := resolveIdentity()
		if err != nil {
			return err
		}
		vault, err := envvault.LoadWith(id)
		if err != nil {
			return err
		}
		names := vault.Names()
		if len(names) == 0 {
			fmt.Println(ui.Dim.Render("  the vault is empty"))
			return nil
		}
		fmt.Printf("  %s\n", ui.Dim.Render(fmt.Sprintf("%d variable(s) — values hidden", len(names))))
		for _, n := range names {
			fmt.Printf("  %s %s\n", ui.Accent.Render("•"), n)
		}
		return nil
	},
}

// envRmCmd removes a variable.
var envRmCmd = &cobra.Command{
	Use:   "rm NAME",
	Short: "Remove a variable from the vault",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !envvault.Exists() {
			return fmt.Errorf("no vault yet")
		}
		id, err := resolveIdentity()
		if err != nil {
			return err
		}
		vault, err := envvault.LoadWith(id)
		if err != nil {
			return err
		}
		if !vault.Remove(args[0]) {
			return fmt.Errorf("%q is not in the vault", args[0])
		}
		if err := envvault.SaveWith(vault, id); err != nil {
			return err
		}
		fmt.Printf("  %s removed %s\n", ui.OKMark(), args[0])
		return nil
	},
}

// envLoadCmd prints `export KEY=value` lines to STDOUT for the shell to eval.
// Prompts (if any) go to stderr, so `eval "$(opsforge env load)"` is clean.
var envLoadCmd = &cobra.Command{
	Use:   "load",
	Short: "Print `export` lines for the vault (for eval in your shell)",
	Long: `Decrypts the vault and prints shell 'export' lines on stdout, so your
shell can apply them:

    eval "$(opsforge env load)"

Prefer the 'opsenv' shell function (installed by the shell layer). If a session
is unlocked, no passphrase is asked; otherwise you're prompted once (on stderr,
never stdout) and a session is opened.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !envvault.Exists() {
			return fmt.Errorf("no vault yet — `opsforge env set NAME` to create one")
		}
		id, err := resolveIdentity()
		if err != nil {
			return err
		}
		vault, err := envvault.LoadWith(id)
		if err != nil {
			return err
		}
		fmt.Print(vault.ExportLines()) // the ONLY thing on stdout
		return nil
	},
}

func init() {
	envCmd.AddCommand(envUnlockCmd, envLockCmd, envSetCmd, envListCmd, envRmCmd, envLoadCmd)
	rootCmd.AddCommand(envCmd)
}

// --- TTY helpers -----------------------------------------------------------

// readPassphrase reads the master passphrase masked from the terminal. Prompts
// go to STDERR so they never contaminate `env load`'s stdout. With confirm, it
// asks twice and checks they match (used when creating the vault).
func readPassphrase(confirm bool) (string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return "", fmt.Errorf("a passphrase is required, but stdin is not a terminal")
	}
	fmt.Fprint(os.Stderr, "Vault passphrase: ")
	b, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	pass := string(b)
	if pass == "" {
		return "", fmt.Errorf("empty passphrase")
	}
	if confirm {
		fmt.Fprint(os.Stderr, "Confirm passphrase: ")
		b2, err := term.ReadPassword(fd)
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", err
		}
		if string(b2) != pass {
			return "", fmt.Errorf("passphrases don't match")
		}
	}
	return pass, nil
}

// readSecret reads a single value masked from the terminal (used for `set`),
// so a secret value never appears in argv/history.
func readSecret(label string) (string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return "", fmt.Errorf("a value is required, but stdin is not a terminal (pass NAME=value instead)")
	}
	fmt.Fprintf(os.Stderr, "%s: ", label)
	b, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
