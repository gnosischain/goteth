#!make

GOCC=go
MKDIR_P=mkdir -p
GIT_SUBM=git submodule

BIN_PATH=./build
BIN="./build/goteth"

.PHONY: check build dependencies install run clean

build: 
    $(GOCC) get
	$(GOCC) build -o $(BIN)

dependencies:
	$(GIT_SUBM) update --init 
	cd go-relay-client && git checkout "origin/goteth" && git pull origin goteth
	cd ..

install:
	$(GOCC) install
	$(GOCC) download

clean:
	rm -r $(BIN_PATH)


