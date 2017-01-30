.PHONY: dev run build

dev:
	rm ./service; go build ; \
	export RESPONSE_LOG=response.log; \
	export REQUEST_LOG=response.log; \
	./yondu;

run:
	export RESPONSE_LOG=response.log; \
  export REQUEST_LOG=response.log; \
  ./yondu

rm:
	rm bin/yondu-linux-amd64; rm ~/linkit/yondu-linux-amd64

build:
	export GOOS=linux; export GOARCH=amd64; \
  go build -ldflags "-s -w" -o bin/yondu-linux-amd64 ; cp bin/yondu-linux-amd64 ~/linkit

