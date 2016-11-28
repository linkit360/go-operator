.PHONY: dev run build

dev:
	rm ./service; go build ; \
	export YONDO_RESPONSE_LOG=response.log; \
	export YONDO_REQUEST_LOG=response.log; \
	./yondo;

run:
	export YONDO_RESPONSE_LOG=response.log; \
  export YONDO_REQUEST_LOG=response.log; \
  ./yondo

rm:
	rm bin/yondo-linux-amd64; rm ~/linkit/yondo-linux-amd64

build:
	export GOOS=linux; export GOARCH=amd64; \
  go build -ldflags "-s -w" -o bin/yondo-linux-amd64 ; cp bin/yondo-linux-amd64 ~/linkit

