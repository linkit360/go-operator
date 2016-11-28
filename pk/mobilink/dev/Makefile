.PHONY: run build

run:
	export MOBILINK_RESPONSE_LOG=response.log; \
  export MOBILINK_REQUEST_LOG=response.log; \
  ./mt_manager

rm:
	rm bin/mobilink-linux-amd64; rm ~/linkit/mobilink-linux-amd64

build:
	export GOOS=linux; export GOARCH=amd64; \
  go build -ldflags "-s -w" -o bin/mobilink-linux-amd64 ; cp bin/mobilink-linux-amd64 ~/linkit; cp dev/mobilink.yml ~/linkit/;


