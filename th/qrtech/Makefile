.PHONY: run build

run:
	export RESPONSE_LOG=response.log; \
  export REQUEST_LOG=response.log; \
  ./qrtech

rm:
	rm bin/qrtech-linux-amd64; rm ~/linkit/qrtech-linux-amd64

build:
	export GOOS=linux; export GOARCH=amd64; \
  go build -ldflags "-s -w" -o bin/qrtech-linux-amd64 ; cp bin/qrtech-linux-amd64 ~/linkit; cp dev/qrtech.yml ~/linkit/;

mo:
	curl -v -X POST --data "motoken=1234&msgid=_msgid123&msisdn=66819197088&message=P1&shortcode=4219112&&productid=&operator=1&keyword=P1"  "http://localhost:50306/api/mo"

dnpaid:
	curl -v -X POST --data "motoken=1234&dnid=_msgid123&msisdn=66819197088&message=P1&shortcode=4219112&&productid=&operator=1&keyword=P1&bcdate=20170215&dnerrorcode=200"  "http://localhost:50306/api/dn"

dnerror:
	curl -v -X POST --data "motoken=1234&dnid=_msgid123&msisdn=66819197088&message=P1&shortcode=4219112&&productid=&operator=1&keyword=P1&bcdate=20170215&dnerrorcode=550"  "http://localhost:50306/api/dn"