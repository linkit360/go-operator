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

ais:
	curl -v "http://localhost:50306/api/mo/ais?ref=61201000015999095326&msn=66911111110&svk=450455555&acs=REG_SUCCESS&chn=SMS&mdt=2016-12-01%2000:00:18.227"