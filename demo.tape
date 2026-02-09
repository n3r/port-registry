# VHS tape file — run `vhs demo.tape` to generate demo.gif
# https://github.com/charmbracelet/vhs

Output demo.gif

Set Shell "bash"
Set FontSize 14
Set Width 800
Set Height 500
Set Padding 20
Set TypingSpeed 40ms

Type "portctl start"
Enter
Sleep 2s

Type "portctl allocate --service postgres"
Enter
Sleep 1.5s

Type "portctl allocate --service redis"
Enter
Sleep 1.5s

Type "portctl allocate --service web --port 8080"
Enter
Sleep 1.5s

Type "portctl list"
Enter
Sleep 2.5s

Type "portctl check --port 8080"
Enter
Sleep 1.5s

Type "portctl release --service web"
Enter
Sleep 1.5s

Type "portctl list"
Enter
Sleep 2.5s
