
VERSION=$(shell git describe --always)

build:
	CGO_ENABLED=0 go build -ldflags="-X 'github.com/enrichman/gocoverkube/internal/cli.Version=${VERSION}'"

dev-setup: dev-cluster-create dev-sample-server-build dev-sample-server-deploy

dev-cluster-create:
	k3d cluster create gocoverkube --agents 3
	k3d kubeconfig merge -ad
	kubectl config use-context k3d-gocoverkube

dev-cluster-delete:
	k3d cluster delete gocoverkube

dev-sample-server-build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o sample-server -coverpkg=./... -cover ./tests/sample-server
	docker build -t sample-server:local -f tests/sample-server/Dockerfile .

dev-sample-server-deploy:
	k3d image import -c gocoverkube sample-server:local
	kubectl apply -f ./tests/sample-server/deployment.yaml
	kubectl apply -f ./tests/sample-server/pod.yaml
