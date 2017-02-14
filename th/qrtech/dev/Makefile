.PHONY: dev run build

dev:
	rm ./service; go build ; \
	export RESPONSE_LOG=response.log; \
	export REQUEST_LOG=response.log; \
	./qrtech;

run:
	export RESPONSE_LOG=response.log; \
  export REQUEST_LOG=response.log; \
  ./qrtech

rm:
	rm bin/qrtech-linux-amd64; rm ~/linkit/qrtech-linux-amd64

build:
	export GOOS=linux; export GOARCH=amd64; \
  go build -ldflags "-s -w" -o bin/qrtech-linux-amd64 ; cp bin/qrtech-linux-amd64 ~/linkit

