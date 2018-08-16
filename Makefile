
test:
	./run_tests

clean:
	rm -rf bin/*
	rm -f faqaas*.zip

build_linux_amd64:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/linux_amd64/faqaas github.com/mat/faqaas/admin

build_docker_image: build_linux_amd64
	docker build -t matthiasluedtke/faqaas:latest -t matthiasluedtke/faqaas:`cat VERSION` .

docker_run:
	docker run -p 3000:8080 --env-file docker_run.env matthiasluedtke/faqaas:latest

push_docker_image:
	docker push matthiasluedtke/faqaas:latest
	docker push matthiasluedtke/faqaas:`cat VERSION`
