.PHONY: all windows

all: windows

windows:
	env CC=x86_64-w64-mingw32-gcc CGO_ENABLED=1 GOOS=windows go build -ldflags '-H=windowsgui -s -w' -o throwpro_v02.exe main/throwpro.go
	upx -9 throwpro_v02.exe
	go build main/throwpro.go