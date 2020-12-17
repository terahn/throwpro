.PHONY: all windows

all: windows

windows:
	env CC=x86_64-w64-mingw32-gcc CGO_ENABLED=1 GOOS=windows go build -ldflags '-H=windowsgui -s -w' -o artifacts/throwpro.exe ./gui
	go build -o artifacts/ThrowPro.app/Contents/MacOS/ThrowPro ./gui
	cd artifacts && zip throwpro_v06_windows.zip throwpro.exe
	cd artifacts && zip throwpro_v06_mac.zip ThrowPro.app
	rm artifacts/throwpro.exe
	rm -rf artifacts/ThrowPro.app