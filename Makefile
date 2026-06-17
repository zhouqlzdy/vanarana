.PHONY: build run tidy clean

build:
	go build -o vanarana ./cmd/server/

run: build
	./vanarana

tidy:
	go mod tidy

clean:
	rm -f vanarana
