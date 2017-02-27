.PHONY: run build


VERSION=$(shell git describe --always --long --dirty)

version:
	 @echo Version IS $(VERSION)


run:
	export yondu_RESPONSE_LOG=response.log; \
  export yondu_REQUEST_LOG=response.log; \
  ./yondu

rm:
	rm bin/yondu-linux-amd64; rm ~/linkit/yondu-linux-amd64

build:
	export GOOS=linux; export GOARCH=amd64; \
	sed -i "s/%VERSION%/$(VERSION)/g" /home/centos/vostrok/utils/metrics/metrics.go; \
  go build -ldflags "-s -w" -o bin/yondu-linux-amd64 ; cp bin/yondu-linux-amd64 ~/linkit; cp dev/yondu.yml ~/linkit/;

mou:
	curl -v "http://localhost:50306/api/mo?message=Play%20off&msisdn=9951502420&telco=globe&rrn=1488219393"

mo:
	curl -v "http://localhost:50306/api/mo?message=Play%20on&msisdn=9951502420&telco=globe&rrn=1488219393"

dnok:
	curl -v "http://localhost:50306/api/dn?telco=globe&msisdn=9951502420&status=success&code=200&rrn=1488219393&timestamp=1486527412"

dn401:
	curl -v "http://localhost:50306/api/dn?telco=globe&msisdn=9951502420&status=fail&code=401&rrn=1488219393&timestamp=1486527412"

dnpostpaid:
	curl -v "http://localhost:50306/api/dn?telco=globe&msisdn=9951502420&status=fail&code=414&rrn=1488219393&timestamp=1486527412"

mt:
	curl -v "http://localhost:50306/yondu/mt?key=26a339971a34cc3f4c835e16ccc642a77a3a0063373dd9cd58bf26d2c02309d8&msisdn=9951502420&keyword=play_00&message=Here+is+the+link%3A+http%3A%2F%2Fdev.pk.linkit360.ru%2Funiq%2FTNZPMRlB8&rrn=1488218310"

# curl -v "http://162.209.29.190/xportal/partner-2656.php?key=26a339971a34cc3f4c835e16ccc642a77a3a0063373dd9cd58bf26d2c02309d8&msisdn=9178626134&keyword=play_250&message=Here+is+the+link%3A+http%3A%2F%2Fdev.pk.linkit360.ru%2Funiq%2FTNZPMRlB8&rrn=1488214508"
# //curl -v "http://162.209.29.190/xportal/partner-2656.php?key=26a339971a34cc3f4c835e16ccc642a77a3a0063373dd9cd58bf26d2c02309d8&msisdn=9951502420&keyword=play_00&message=Here+is+the+link%3A+http%3A%2F%2Fdev.pk.linkit360.ru%2Funiq%2FTNZPMRlB8&rrn=1488218310"
# http://platform.ph.linkit360.ru/api/mo?telco=globe&msisdn=9171234567&message=sample%20keyword&rrn=abcd1234
# http://platform.ph.linkit360.ru/api/dn?telco=globe&msisdn=9171234567&status=success&code=200&rrn=abcd1234&timestamp=1486527412