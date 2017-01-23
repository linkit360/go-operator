.PHONY: run build

run:
	export RESPONSE_LOG=response.log; \
  export REQUEST_LOG=response.log; \
  ./cheese

rm:
	rm bin/cheese-linux-amd64; rm ~/linkit/cheese-linux-amd64

build:
	export GOOS=linux; export GOARCH=amd64; \
  go build -ldflags "-s -w" -o bin/cheese-linux-amd64 ; cp bin/cheese-linux-amd64 ~/linkit; cp dev/cheese.yml ~/linkit/;


mo:
	curl -v "http://localhost:50306/api/mo?msisdn=639055209652&transid=2910DMP9055209652148403120057&timestamp=20161227101000&message=play%20on"