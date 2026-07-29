package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/Mrg77/opsforge/internal/envvault"
	"github.com/Mrg77/opsforge/internal/ui"
)

// envCmd groups the encrypted environment store. `opsforge env` with no
// subcommand shows the status + how to load the vault into a shell.
var envCmd = &cobra.Command{
	Use:   "env",
	Short: "An encrypted store for env vars — persist secrets without cleartext",
	Long: `A passphrase-locked vault of environment variables that survives across
shells WITHOUT writing a secret in cleartext to disk.

Re-exporting the same AWS/Vault/registry credentials in every new terminal is
tedious; putting them in ~/.zshrc persists them but leaks them in cleartext
(a dotfiles backup, or 'opsforge audit --secrets', will find them). This vault
keeps the convenience — set once, load anywhere — while the file at rest is
encrypted with age. Unlock it once per session:

    opsenv                       # prompts for your passphrase, exports the vars
    # (opsenv is installed by the opsforge shell layer)

Honesty: once unlocked, the values are ordinary environment variables in that
session, visible to child processes. The win is nothing in cleartext on disk,
no retyping, and dotfiles you can back up without leaking.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(ui.Header("opsforge env", "your encrypted environment vault"))
		fmt.Println()
		if !envvault.Exists() {
			fmt.Printf("  %s %s\n", ui.Dim.Render("•"), ui.Dim.Render("no vault yet — add a variable with `opsforge env set NAME`"))
			return nil
		}
		fmt.Printf("  %s %s\n", ui.OKMark(), ui.Dim.Render("a vault exists (encrypted)"))
		fmt.Println()
		fmt.Println("  " + ui.Faint.Render("Load it into your shell:  opsenv"))
		fmt.Println("  " + ui.Faint.Render("List variable names:      opsforge env list"))
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
	RunE: func(cmd *cobra.Command, args []string) error {
		name, value, inline := strings.Cut(args[0], "=")
		if !envvault.ValidName(name) {
			return fmt.Errorf("invalid variable name %q", name)
		}

		if !inline {
			// Read the value masked — the whole point is to keep secrets out
			// of argv/history.
			v, err := readSecret(fmt.Sprintf("Value for %s", name))
			if err != nil {
				return err
			}
			value = v
		}

		pass, err := readPassphrase(false)
		if err != nil {
			return err
		}
		vault, err := envvault.Load(pass)
		if err != nil {
			return err
		}
		if err := vault.Set(name, value); err != nil {
			return err
		}
		if err := envvault.Save(vault, pass); err != nil {
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
		pass, err := readPassphrase(false)
		if err != nil {
			return err
		}
		vault, err := envvault.Load(pass)
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
		name := args[0]
		pass, err := readPassphrase(false)
		if err != nil {
			return err
		}
		vault, err := envvault.Load(pass)
		if err != nil {
			return err
		}
		if !vault.Remove(name) {
			return fmt.Errorf("%q is not in the vault", name)
		}
		if err := envvault.Save(vault, pass); err != nil {
			return err
		}
		fmt.Printf("  %s removed %s\n", ui.OKMark(), name)
		return nil
	},
}

// envLoadCmd prints `export KEY=value` lines to STDOUT for the shell to eval.
// This is the machine-facing half: it writes nothing but the export lines to
// stdout (prompts go to stderr), so `eval "$(opsforge env load)"` works.
var envLoadCmd = &cobra.Command{
	Use:   "load",
	Short: "Print `export` lines for the vault (for eval in your shell)",
	Long: `Decrypts the vault and prints shell 'export' lines on stdout, so your
shell can apply them:

    eval "$(opsforge env load)"

Prefer the 'opsenv' shell function (installed by the shell layer) — it does
exactly this. The passphrase prompt is written to stderr, never stdout, so it
can't end up in the eval'd text.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !envvault.Exists() {
			return fmt.Errorf("no vault yet — `opsforge env set NAME` to create one")
		}
		pass, err := readPassphrase(false)
		if err != nil {
			return err
		}
		vault, err := envvault.Load(pass)
		if err != nil {
			return err
		}
		// The ONLY thing on stdout is the export lines.
		fmt.Print(vault.ExportLines())
		return nil
	},
}

func init() {
	envCmd.AddCommand(envSetCmd, envListCmd, envRmCmd, envLoadCmd)
	rootCmd.AddCommand(envCmd)
}

// --- TTY helpers -----------------------------------------------------------

// readPassphrase reads the master passphrase masked from the terminal. Prompts
// go to STDERR so they never contaminate `env load`'s stdout. With confirm, it
// asks twice and checks they match (used only if we later add init flows).
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
