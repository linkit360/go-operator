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
	curl -v "http://localhost:50306/api/mo?msisdn=639153712383&transid=111111&timestamp=20161227101000&message=play1"

callback:
	curl -v "http://localhost:50306/api/callback?msisdn=639153712383&transid=111111&timestamp=20161227101000&status_code=2006"

charge:
	curl -v "http://52.76.96.115/m360api/v1/charging/639175256925/P1"

# curl -H 'Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMyIsImlzcyI6Imh0dHA6XC9cLzUyLjc2Ljk2LjExNVwvbTM2MGFwaVwvYXBpXC9hdXRoZW50aWNhdGUiLCJpYXQiOjE0NzA2MzQ2NjcsImV4cCI6MTUwMjE3MDY2NywibmJmIjoxNDcwNjM0NjY3LCJqdGkiOiJkMTBjZGFjYmZmNjVhYzZmOThkNTE4OGMyZWRmY2VkMiJ9.5U1tkrbjw3mn1Y7fZt2mouSHTLZTVx6EFauIpG3ovok'  -v "http://52.76.96.115/m360api/v1/charging/639175256925/P1"