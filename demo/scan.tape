# opsforge scan screenshot — regenerate with: vhs scan.tape
# Runs the REAL `opsforge scan` (needs opsforge + syft on PATH) and captures
# it to demo/screenshots/scan.gif.
Output demo/screenshots/scan.gif

Set FontSize 20
Set Width 1180
Set Height 640
Set Padding 26
Set Theme "Dracula"

# Trim the long CVE list to keep the shot readable; the run itself is real.
Type "opsforge scan node:16-alpine --diff 2>/dev/null | sed -n '1,9p;30,47p'" Enter
Sleep 40s
Sleep 500ms
