.PHONY: run build

VERSION=$(shell git describe --always --long --dirty)

version:
	 @echo Version IS $(VERSION)

run:
	export RESPONSE_LOG=response.log; \
  export REQUEST_LOG=response.log; \
  ./cheese

rm:
	rm bin/cheese-linux-amd64; rm ~/linkit/cheese-linux-amd64

build:
	export GOOS=linux; export GOARCH=amd64; \
	sed -i "s/%VERSION%/$(VERSION)/g" /home/centos/vostrok/utils/metrics/metrics.go; \
  go build -ldflags "-s -w" -o bin/cheese-linux-amd64 ; cp bin/cheese-linux-amd64 ~/linkit; cp dev/cheese.yml ~/linkit/;


ais:
	curl -v "http://localhost:50306/api/mo/ais?ref=61201000015999095326&msn=66911111110&svk=777&acs=REG_SUCCESS&chn=SMS&mdt=2016-12-01%2000:00:18.227"

dtac:
	curl -v "http://localhost:50306/api/mo/dtac?ref=61201000015999095326&msn=669XXXXXXX0&svk=450455555&acs=REG_SUCCESS&chn=SMS&mdt=2016-12-01%2000:00:18.227"

trueh:
	curl -v "http://localhost:50306/api/mo/trueh?ref=61201000015999095326&msn=669XXXXXXX0&svk=450455555&acs=REG_SUCCESS&chn=SMS&mdt=2016-12-01%2000:00:18.227"

pixel:
	curl -L "http://localhost:50300/mobilink-p2?msisdn=928777777777&aff_sub=hIDMA1511170000000001035050071575WF0TPC79c000723PZ02345"