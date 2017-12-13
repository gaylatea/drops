bin:
	mkdir -p bin

bin/server-darwin: bin
	cd cmd/server; \
	GOOS=darwin GOARCH=amd64 go build -o ../../bin/server-darwin

bin/server-linux: bin
	cd cmd/server; \
	GOOS=linux GOARCH=amd64 go build -o ../../bin/server-linux

bin/shell-darwin: bin
	cd cmd/shell; \
	GOOS=darwin GOARCH=amd64 go build -o ../../bin/shell-darwin

bin/shell-linux: bin
	cd cmd/shell; \
	GOOS=linux GOARCH=amd64 go build -o ../../bin/shell-linux

clean:
	rm -rf bin

release: bin/server-linux bin/shell-darwin

all: bin/server-darwin bin/server-linux bin/shell-darwin bin/shell-linux

.PHONY: clean