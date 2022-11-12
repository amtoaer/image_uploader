clean:
	rm -rf ./release
linux:
	GOOS=linux go build -ldflags "-w -s" -o release/iu-linux .
darwin:
	GOOS=darwin go build -ldflags "-w -s" -o release/iu-macos .
windows:
	GOOS=windows go build -ldflags "-w -s" -o release/iu-windows.exe .
all:clean linux darwin windows
