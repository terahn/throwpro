.PHONY: all windows

all: windows

windows:
	env CC=x86_64-w64-mingw32-gcc CGO_ENABLED=1 GOOS=windows go build -ldflags '-H=windowsgui -s -w' -o artifacts/throwpro.exe ./gui
	go build -o artifacts/ThrowPro.app/Contents/MacOS/ThrowPro ./gui
	zip -r artifacts/throwpro_v04_windows.zip artifacts/throwpro.exe
	zip -r artifacts/throwpro_v04_mac.zip artifacts/ThrowPro.app
	rm artifacts/throwpro.exe
	rm -rf artifacts/ThrowPro.app