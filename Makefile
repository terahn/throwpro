.PHONY: all windows

all: windows

windows:
	env CC=x86_64-w64-mingw32-gcc CGO_ENABLED=1 GOOS=windows go build -ldflags '-H=windowsgui -s -w' -o throwpro.exe main/throwpro.go
	go build main/throwpro.go
	zip -r throwpro_v04.zip throwpro.exe
