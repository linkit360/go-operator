.PHONY: run build


VERSION=$(shell git describe --always --long --dirty)

version:
	 @echo Version IS $(VERSION)


run:
	export MOBILINK_RESPONSE_LOG=response.log; \
  export MOBILINK_REQUEST_LOG=response.log; \
  ./mt_manager

rm:
	rm bin/mobilink-linux-amd64; rm ~/linkit/mobilink-linux-amd64

build:
	export GOOS=linux; export GOARCH=amd64; \
	sed -i "s/%VERSION%/$(VERSION)/g" /home/centos/vostrok/utils/metrics/metrics.go; \
  go build -ldflags "-s -w" -o bin/mobilink-linux-amd64 ; cp bin/mobilink-linux-amd64 ~/linkit; cp dev/mobilink.yml ~/linkit/;


