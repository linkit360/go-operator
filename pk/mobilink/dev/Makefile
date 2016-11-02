.PHONY: dev run build

dev:
	rm ./mobilink; go build ; ./mobilink;

run:
	./mobilink

build:
	rm -v bin/mobilink-linux-amd64; \
	export GOOS=linux; export GOARCH=amd64; \
  go build -ldflags "-s -w" -o bin/mobilink-linux-amd64

sendprod:
        ssh -i ~/.ssh/mobilink_centos.pem centos@52.66.23.201 rm /home/centos/go/src/github.com/vostrok/mobilink/bin/mobilink-linux-amd64
		scp -i ~/.ssh/mobilink_centos.pem bin/mobilink-linux-amd64 centos@52.66.23.201:/home/centos/go/src/github.com/vostrok/mobilink/bin/mobilink-linux-amd64
		scp -i ~/.ssh/mobilink_centos.pem dev/appconfig.yml centos@52.66.23.201:/home/centos/go/src/github.com/vostrok/mobilink/dev/

senddev:
        ssh -i ~/.ssh/mobilink_centos_testing_stage.pem  centos@35.154.8.158 rm /home/centos/go/src/github.com/vostrok/mobilink/bin/mobilink-linux-amd64
		ssh -i ~/.ssh/mobilink_centos_testing_stage.pem bin/mobilink-linux-amd64 centos@35.154.8.158:/home/centos/go/src/github.com/vostrok/mobilink/bin/mobilink-linux-amd64
		ssh -i ~/.ssh/mobilink_centos_testing_stage.pem dev/appconfig.yml centos@35.154.8.158:/home/centos/go/src/github.com/vostrok/mobilink/dev/
