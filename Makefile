.PHONY: install test build serve clean pack deploy ship

TAG?=$(shell git rev-list HEAD --max-count=1 --abbrev-commit)

export TAG

install:
	go get .

test: install
	go test ./...

build: install
	go build -ldflags "-X main.version=$(TAG)" -o bin/api .

serve: build
	bin/api -config config.yml

clean:
	rm bin/api

pack:
	GOOS=linux make build
	docker build -t gcr.io/revolut-sre-challenge/simple-api-cloud:$(TAG) .

upload:
	docker push gcr.io/revolut-sre-challenge/simple-api-cloud:$(TAG)

deploy:
	envsubst < k8s/deployment.yml | kubectl apply -f -

ship: test pack upload deploy clean