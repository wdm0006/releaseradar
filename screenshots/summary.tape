Output screenshots/summary.gif

Set Shell "zsh"
Set FontSize 14
Set Width 1400
Set Height 800
Set Theme "Catppuccin Mocha"
Set Padding 20

Hide
Type "cd /Users/wmcginnis/Documents/GitHub/releaseradar"
Enter
Sleep 500ms
Type "RELEASERADAR_START_TAB=3 ./releaseradar"
Enter
Sleep 5s
Show

# Generate summary — send s twice with a gap for VHS reliability
Sleep 2s
Type "s"
Sleep 2s
Type "s"
Sleep 45s

Screenshot screenshots/summary.png

Sleep 1s
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 2s

Type "q"
Sleep 500ms
