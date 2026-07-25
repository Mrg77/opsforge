package credscan

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"encoding/pem"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"
)

// scanSSH inspects ~/.ssh: the directory permissions, then every private key,
// flagging keys stored WITHOUT a passphrase (usable as-is if stolen), keys
// readable beyond their owner, and weak algorithms. It only reads files and
// parses PEM/OpenSSH headers — it never decrypts a key or runs ssh.
func scanSSH() []Finding {
	dir := expand("~/.ssh")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil // no ~/.ssh — nothing to say
	}

	var out []Finding

	// The directory itself should be 0700.
	if fi, err := os.Stat(dir); err == nil && worldOrGroupReadable(fi.Mode()) {
		out = append(out, mkFinding("SSH", "key directory", "~/.ssh",
			SevMedium, "~/.ssh is accessible beyond your user ("+fi.Mode().Perm().String()+")",
			"Run `chmod 700 ~/.ssh`."))
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		// Public keys, config, and host lists are not secrets.
		if strings.HasSuffix(name, ".pub") ||
			name == "config" || name == "known_hosts" || name == "authorized_keys" {
			continue
		}
		full := filepath.Join(dir, name)
		b, err := os.ReadFile(full)
		if err != nil {
			continue
		}
		if !looksLikePrivateKey(b) {
			continue // not a key file (README, .bak of a pubkey, etc.)
		}
		out = append(out, sshKeyFindings("~/.ssh/"+name, full, b)...)
	}
	return out
}

// looksLikePrivateKey reports whether a blob's first PEM block is a private key.
func looksLikePrivateKey(b []byte) bool {
	block, _ := pem.Decode(b)
	return block != nil && strings.Contains(block.Type, "PRIVATE KEY")
}

// sshKeyFindings reports on a single private key file: permissions, whether it
// is unencrypted, and whether the algorithm is weak. displayPath is the tilde
// form for the report; full is the resolved path used to stat.
func sshKeyFindings(displayPath, full string, b []byte) []Finding {
	var out []Finding

	if fi, err := os.Lstat(full); err == nil && worldOrGroupReadable(fi.Mode()) {
		out = append(out, mkFinding("SSH", "private key", displayPath,
			SevHigh, "private key is readable beyond its owner ("+fi.Mode().Perm().String()+")",
			"A private key must be 0600. Run `chmod 600 "+displayPath+"`."))
	}

	switch keyProtection(b) {
	case protUnencrypted:
		out = append(out, mkFinding("SSH", "unencrypted private key", displayPath,
			SevHigh, "private key has no passphrase",
			"If this file is copied or leaked it's usable as-is. Add a passphrase: "+
				"`ssh-keygen -p -f "+displayPath+"`."))
	case protEncrypted:
		out = append(out, mkFinding("SSH", "private key", displayPath,
			SevOK, "private key is passphrase-protected", "Good hygiene."))
	}

	if kind, bits, weak := keyStrength(b); weak {
		detail := "Prefer Ed25519: `ssh-keygen -t ed25519`."
		title := kind + " key is weak"
		if bits > 0 {
			title = kind + " key is only " + strconv.Itoa(bits) + " bits"
		}
		out = append(out, mkFinding("SSH", "weak key", displayPath, SevMedium, title, detail))
	}
	return out
}

type protection int

const (
	protUnknown protection = iota
	protUnencrypted
	protEncrypted
)

// keyProtection decides whether a private key is encrypted WITHOUT decrypting
// it. ParseRawPrivateKey returns *ssh.PassphraseMissingError for an encrypted
// key (the reliable signal); success means it was unencrypted. We fall back to
// PEM headers for formats x/crypto doesn't parse.
func keyProtection(b []byte) protection {
	_, err := ssh.ParseRawPrivateKey(b)
	if err == nil {
		return protUnencrypted
	}
	if _, ok := err.(*ssh.PassphraseMissingError); ok {
		return protEncrypted
	}
	// Fallback to header inspection for anything x/crypto couldn't handle.
	block, _ := pem.Decode(b)
	if block == nil {
		return protUnknown
	}
	if block.Type == "ENCRYPTED PRIVATE KEY" {
		return protEncrypted
	}
	if _, ok := block.Headers["Proc-Type"]; ok { // "Proc-Type: 4,ENCRYPTED"
		return protEncrypted
	}
	return protUnknown
}

// keyStrength returns the algorithm, bit size (0 when N/A), and whether the key
// is weak: RSA < 2048, or DSA (deprecated). Only computable for an unencrypted,
// parsable key — an encrypted one returns weak=false (we can't and shouldn't).
func keyStrength(b []byte) (kind string, bits int, weak bool) {
	key, err := ssh.ParseRawPrivateKey(b)
	if err != nil {
		// Encrypted or unknown; strength isn't assessable here.
		if block, _ := pem.Decode(b); block != nil && block.Type == "DSA PRIVATE KEY" {
			return "DSA", 0, true
		}
		return "", 0, false
	}
	switch k := key.(type) {
	case *rsa.PrivateKey:
		return "RSA", k.N.BitLen(), k.N.BitLen() < 2048
	case *ecdsa.PrivateKey:
		return "ECDSA", 0, false
	default:
		// Ed25519 and others: not weak.
		return "", 0, false
	}
}
