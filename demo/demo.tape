# opsforge demo — regenerate with: vhs demo.tape
Output opsforge.gif

Set Shell "zsh"
Set FontSize 16
Set Width 1200
Set Height 760
Set Padding 24
Set Theme "Catppuccin Mocha"
Set TypingSpeed 55ms
Set PlaybackSpeed 1.0

Hide
Type "clear" Enter
Show
Sleep 800ms

# --- the cockpit -------------------------------------------------------
Type "opsforge status"  Sleep 400ms  Enter
Sleep 3s

Type "clear" Enter
Sleep 400ms

# --- the interactive picker (with tabs) --------------------------------
Type "opsforge"  Sleep 400ms  Enter
Sleep 2s
# filter
Type "/"  Sleep 400ms
Type "k8s"  Sleep 1500ms
Escape  Sleep 700ms
# tabs: updates then security
Type "2"  Sleep 1800ms
Type "3"  Sleep 3500ms
Type "1"  Sleep 800ms
Type "q"  Sleep 800ms

Type "clear" Enter
Sleep 400ms

# --- themes ------------------------------------------------------------
Type "opsforge theme"  Sleep 400ms  Enter
Sleep 3s

Type "clear" Enter
Sleep 400ms

# --- doctor ------------------------------------------------------------
Type "opsforge doctor"  Sleep 400ms  Enter
Sleep 3s
