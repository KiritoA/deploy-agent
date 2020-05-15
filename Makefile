FILE_NAME_PREXIF=deploy-agent

default: build

build: build-linux build-windows

build-linux:
	GOOS=linux GOARCH=amd64 go build -o build/$(FILE_NAME_PREXIF)-linux-amd64 .

build-windows:
	GOOS=windows GOARCH=amd64 go build -o build/$(FILE_NAME_PREXIF)-windows-amd64.exe .