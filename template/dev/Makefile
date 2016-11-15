.PHONY: dev run build

dev:
	rm ./service; go build ; \
	export MOBILINK_RESPONSE_LOG=response.log; \
	export MOBILINK_REQUEST_LOG=response.log; \
	./mt_manager;

run:
	export MOBILINK_RESPONSE_LOG=response.log; \
  export MOBILINK_REQUEST_LOG=response.log; \
  ./mt_manager

rm:
	rm bin/mt_manager-linux-amd64; rm ~/linkit/mt_manager-linux-amd64

build:
	export GOOS=linux; export GOARCH=amd64; \
  go build -ldflags "-s -w" -o bin/mt_manager-linux-amd64 ; cp bin/mt_manager-linux-amd64 ~/linkit

