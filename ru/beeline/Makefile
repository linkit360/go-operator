.PHONY: run build

run:
	export RESPONSE_LOG=response.log; \
  export REQUEST_LOG=response.log; \
  ./beeline

rm:
	rm bin/beeline-linux-amd64; rm ~/linkit/beeline-linux-amd64

build:
	export GOOS=linux; export GOARCH=amd64; \
  go build -ldflags "-s -w" -o bin/beeline-linux-amd64 ; cp bin/beeline-linux-amd64 ~/linkit; cp dev/beeline.yml ~/linkit/;
