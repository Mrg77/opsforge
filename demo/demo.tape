# opsforge demo — regenerate with: vhs demo.tape
Output demo-v0.3.gif

Set Shell "zsh"
Set FontSize 16
Set Width 1200
Set Height 760
Set Padding 24
Set Theme "Catppuccin Mocha"
Set TypingSpeed 60ms
Set PlaybackSpeed 1.0

Hide
Type "clear" Enter
Show
Sleep 1s

# --- the cockpit -------------------------------------------------------
Type "opsforge status"  Sleep 500ms  Enter
Sleep 6s

Type "clear" Enter
Sleep 700ms

# --- the interactive picker (with tabs) --------------------------------
Type "opsforge"  Sleep 500ms  Enter
Sleep 3s
# filter
Type "/"  Sleep 600ms
Type "k8s"  Sleep 2500ms
Escape  Sleep 1500ms
# tabs: updates then security
Type "2"  Sleep 3s
Type "3"  Sleep 5s
Type "1"  Sleep 1500ms
Type "q"  Sleep 1s

Type "clear" Enter
Sleep 700ms

# --- themes ------------------------------------------------------------
Type "opsforge theme"  Sleep 500ms  Enter
Sleep 6s

Type "clear" Enter
Sleep 700ms

# --- doctor ------------------------------------------------------------
Type "opsforge doctor"  Sleep 500ms  Enter
Sleep 7s
