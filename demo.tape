# VHS tape file — run `vhs demo.tape` to generate demo.gif
# https://github.com/charmbracelet/vhs
#
# Prerequisites:
#   brew install vhs
#   make build   (or install portctl via Homebrew)

Output demo.gif

Require portctl

Set Shell "bash"
Set FontSize 14
Set Width 800
Set Height 500
Set Padding 20
Set TypingSpeed 40ms

# Clean slate for recording
Hide
Type "portctl stop 2>/dev/null; pkill -f 'port-(registry|server)' 2>/dev/null; sleep 1; rm -rf ~/.port-registry; clear"
Enter
Sleep 3s
Show

Type "portctl start"
Enter
Sleep 2s

Type "portctl allocate --app myapp --service postgres"
Enter
Sleep 1.5s

Type "portctl allocate --app myapp --service redis"
Enter
Sleep 1.5s

Type "portctl allocate --app myapp --service web --port 8080"
Enter
Sleep 1.5s

Type "portctl list --app myapp"
Enter
Sleep 2.5s

Type "portctl check --port 8080"
Enter
Sleep 1.5s

Type "portctl release --app myapp --service web"
Enter
Sleep 1.5s

Type "portctl list --app myapp"
Enter
Sleep 2.5s
