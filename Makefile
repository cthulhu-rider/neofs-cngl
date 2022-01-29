all: build

build:
	CGO_ENABLED=0 \
	GO111MODULE=on \
	go build -v -trimpath \
	-o ./bin/neofs-cngl ./src/$(notdir)