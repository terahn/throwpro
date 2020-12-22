.PHONY: all

all: windows macos lambda

windows:
	env CC=x86_64-w64-mingw32-gcc CGO_ENABLED=1 GOOS=windows go build -ldflags '-H=windowsgui -s -w' -o artifacts/throwpro.exe ./gui
	cd artifacts && zip throwpro_v07_windows.zip throwpro.exe
	rm artifacts/throwpro.exe

macos:
	go build -o artifacts/ThrowPro.app/Contents/MacOS/ThrowPro ./gui
	cd artifacts && zip throwpro_v07_mac.zip ThrowPro.app
	rm -rf artifacts/ThrowPro.app
	
lambda:
	GOARCH=amd64 GOOS=linux go build -o artifacts/throwpro_api ./api
	chmod +x artifacts/throwpro_api
	
deploy: lambda
	sls deploy -c api/serverless.yml