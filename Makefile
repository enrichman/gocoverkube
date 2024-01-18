
dev-setup: dev-cluster-create dev-sample-server-build dev-sample-server-copy

dev-cluster-create:
	k3d registry create gocoverkube-registry.localhost
	k3d cluster create gocoverkube --registry-use gocoverkube-registry.localhost
	k3d kubeconfig merge -ad

dev-sample-server-build:
	CGO_ENABLED=0 go build -o sample-server -coverpkg=./... -cover ./tests/sample-server
	docker build -t enrichman/gocoverkube-sample-server:dev -f tests/sample-server/Dockerfile .

dev-sample-server-copy:
	k3d image import -c gocoverkube enrichman/gocoverkube-sample-server:dev

dev-delete:
	k3d cluster delete gocoverkube
	k3d cluster delete --all