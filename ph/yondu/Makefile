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


mo:
	curl -v "http://localhost:50306/api/mo?msisdn=639055209652&transid=2910DMP9055209652148403120057&timestamp=20161227101000&message=play%20on"

unsubscribe:
	curl -v "http://localhost:50306/api/mo?msisdn=639055209652&transid=2910DMP9055209652148403120057&timestamp=20161227101000&message=play%20off"

callbackpaid:
	curl -v "http://localhost:50306/api/callback?msisdn=639055209652&transid=2910KRE9055209652148403127752002&timestamp=20161227101000&status_code=0"

callback:
	curl -v "http://localhost:50306/api/callback?msisdn=639055209652&transid=2910KRE9055209652148403127752002&timestamp=20161227101000&status_code=1"

charge:
	curl -v "http://52.76.96.115/m360api/v1/charging/9175256925/P1"

# curl -H 'Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMyIsImlzcyI6Imh0dHA6XC9cLzUyLjc2Ljk2LjExNVwvbTM2MGFwaVwvYXBpXC9hdXRoZW50aWNhdGUiLCJpYXQiOjE0NzA2MzQ2NjcsImV4cCI6MTUwMjE3MDY2NywibmJmIjoxNDcwNjM0NjY3LCJqdGkiOiJkMTBjZGFjYmZmNjVhYzZmOThkNTE4OGMyZWRmY2VkMiJ9.5U1tkrbjw3mn1Y7fZt2mouSHTLZTVx6EFauIpG3ovok'  -v "http://52.76.96.115/m360api/v1/charging/639175256925/P1"
# curl -v "http://52.76.96.115/?key=26a339971a34cc3f4c835e16ccc642a77a3a0063373dd9cd58bf26d2c02309d8&msisdn=9171234567&keyword=play_250&message=Here+is+the+link%3A+http%3A%2F%2Fdev.pk.linkit360.ru%2Funiq%2FTNZPMRlB8&rrn=abcd1234"

# 9178626134
# curl -v "http://162.209.29.190/xportal/partner-2656.php?key=26a339971a34cc3f4c835e16ccc642a77a3a0063373dd9cd58bf26d2c02309d8&msisdn=9178626134&keyword=play_250&message=Here+is+the+link%3A+http%3A%2F%2Fdev.pk.linkit360.ru%2Funiq%2FTNZPMRlB8&rrn=1488214508"
# //curl -v "http://162.209.29.190/xportal/partner-2656.php?key=26a339971a34cc3f4c835e16ccc642a77a3a0063373dd9cd58bf26d2c02309d8&msisdn=9951502420&keyword=play_00&message=Here+is+the+link%3A+http%3A%2F%2Fdev.pk.linkit360.ru%2Funiq%2FTNZPMRlB8&rrn=1488218310"
# http://platform.ph.linkit360.ru/api/mo?telco=globe&msisdn=9171234567&message=sample%20keyword&rrn=abcd1234
# http://platform.ph.linkit360.ru/api/dn?telco=globe&msisdn=9171234567&status=success&code=200&rrn=abcd1234&timestamp=1486527412
