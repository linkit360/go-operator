.PHONY: run build

run:
	export yondu_RESPONSE_LOG=response.log; \
  export yondu_REQUEST_LOG=response.log; \
  ./yondu

rm:
	rm bin/yondu-linux-amd64; rm ~/linkit/yondu-linux-amd64

build:
	export GOOS=linux; export GOARCH=amd64; \
  go build -ldflags "-s -w" -o bin/yondu-linux-amd64 ; cp bin/yondu-linux-amd64 ~/linkit; cp dev/yondu.yml ~/linkit/;

