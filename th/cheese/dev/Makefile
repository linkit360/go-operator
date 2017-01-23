.PHONY: dev run build

dev:
	rm ./service; go build ; \
	export RESPONSE_LOG=response.log; \
	export REQUEST_LOG=response.log; \
	./cheese;

run:
	export RESPONSE_LOG=response.log; \
  export REQUEST_LOG=response.log; \
  ./cheese

rm:
	rm bin/cheese-linux-amd64; rm ~/linkit/cheese-linux-amd64

build:
	export GOOS=linux; export GOARCH=amd64; \
  go build -ldflags "-s -w" -o bin/cheese-linux-amd64 ; cp bin/cheese-linux-amd64 ~/linkit

