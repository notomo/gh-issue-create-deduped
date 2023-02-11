GH_NAME:=issue-create-deduped

build:
	go build -o gh-${GH_NAME} main.go

install: build
	gh extension remove ${GH_NAME} || echo
	gh extension install .

test:
	go test -v ./...
