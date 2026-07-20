# vhs script for the opsforge demo GIF.
# Regenerate with:  vhs demo/demo.tape   (from the repo root)
# Requires the `opsforge` binary on PATH or in the repo root.

Output demo/opsforge.gif

Set Shell "zsh"
Set FontSize 18
Set Width 1100
Set Height 750
Set Padding 20
Set Theme "Catppuccin Mocha"
Set TypingSpeed 60ms

# --- profiles: show the stack presets ---
Type "opsforge profiles"  Sleep 400ms  Enter
Sleep 3s

# --- list: catalog with live installed/version status ---
Type "opsforge list | head -20"  Sleep 400ms  Enter
Sleep 3s

# --- the interactive picker ---
Type "opsforge"  Sleep 400ms  Enter
Sleep 1500ms

# filter to kubernetes tools
Type "/"  Sleep 500ms
Type "k9s"  Sleep 1500ms
Escape  Sleep 800ms

# browse and toggle a couple of tools
Down Sleep 300ms
Down Sleep 300ms
Down Sleep 300ms
Type " "  Sleep 600ms
Down Sleep 300ms
Type " "  Sleep 800ms

# quit without installing (demo stays fast and side-effect free)
Type "q"  Sleep 1s
