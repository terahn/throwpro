package throwpro

//go:generate sh -c "(printf 'package throwpro\nvar icon string=`'; base64 eye.png; printf '`') >icon.go"

var name = lns(`ThrowPro Minecraft Assistant`, `Version 0.2`)
