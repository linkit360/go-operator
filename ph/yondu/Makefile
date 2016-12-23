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


mo:
	curl -v "http://localhost:50311/api/mo?msisdn=9123123&transid=111111&timestamp=20161223170000&message=play1"


callback:
	curl -v "http://localhost:50311/api/callback?msisdn=9123123&transid=111111&timestamp=20161223170000&status_code=2006"