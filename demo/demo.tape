# opsforge demo — regenerate with: vhs demo.tape
Output demo-v0.3.2.gif

Set Shell "zsh"
Set FontSize 16
Set Width 1200
Set Height 760
Set Padding 24
Set Theme "Catppuccin Mocha"
Set TypingSpeed 65ms
Set PlaybackSpeed 1.0

Hide
Type "clear" Enter
Show
Sleep 1s

# --- the cockpit -------------------------------------------------------
Type "opsforge status"  Sleep 500ms  Enter
Sleep 7s

Type "clear" Enter
Sleep 800ms

# --- the interactive picker (with tabs) --------------------------------
Type "opsforge"  Sleep 500ms  Enter
# let the full catalog view breathe — this is the richest screen
Sleep 6s
# scroll down a touch so the reader sees it is a long, browsable list
Down Down Down  Sleep 2s
Down Down Down  Sleep 2500ms
Up Up Up Up Up Up  Sleep 2s

# filter to k8s
Type "/"  Sleep 700ms
Type "k8s"  Sleep 3500ms
Escape  Sleep 2s

# tab 2: updates only
Type "2"  Sleep 4s
# tab 3: live CVE security scan
Type "3"  Sleep 6s
# back to tools
Type "1"  Sleep 2s
Type "q"  Sleep 1s

Type "clear" Enter
Sleep 800ms

# --- stack profiles ----------------------------------------------------
# 7 ready-made stacks; `opsforge install --profile <name>` installs one in one go
Type "opsforge profiles"  Sleep 500ms  Enter
# long screen: hold, then scroll through the stacks
Sleep 6s
Down Down Down Down  Sleep 3s
Down Down Down Down  Sleep 3s
Down Down Down Down  Sleep 3s

Type "clear" Enter
Sleep 800ms

# --- doctor ------------------------------------------------------------
Type "opsforge doctor"  Sleep 500ms  Enter
Sleep 8s
