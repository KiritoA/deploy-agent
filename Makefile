FILE_NAME_PREXIF=deploy-agent

default: build

build: build-server build-client

build-server:
	GOOS=linux GOARCH=amd64 go build -o build/$(FILE_NAME_PREXIF)-server ./cmd/server

build-client:
	GOOS=linux GOARCH=amd64 go build -o build/$(FILE_NAME_PREXIF)-client ./cmd/client
