# Makefile for F1 Telemetry Backend

.PHONY: all build run test lint clean docker-build docker-build-all docker-push k8s-deploy k8s-delete k8s-logs

# Variables
BINARY_NAME=f1-telemetry-service
DOCKER_IMAGE_BACKEND=f1-telemetry-backend
DOCKER_IMAGE_SIDECAR=f1-telemetry-sidecar
K8S_NAMESPACE=f1-telemetry

all: build

## Build the binary
build:
	@echo "Building..."
	go build -o bin/$(BINARY_NAME) ./cmd/server

## Run the server locally
run: build
	@echo "Running..."
	./bin/$(BINARY_NAME)

## Run tests
test:
	@echo "Running tests..."
	go test -v -race ./...

## Run linter
lint:
	@echo "Linting..."
	golangci-lint run

## Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/

## Build backend Docker image
docker-build:
	@echo "Building backend Docker image..."
	docker build -t $(DOCKER_IMAGE_BACKEND):latest -f Dockerfile.backend .

## Build sidecar Docker image
docker-build-sidecar:
	@echo "Building sidecar Docker image..."
	docker build -t $(DOCKER_IMAGE_SIDECAR):latest -f Dockerfile.sidecar .

## Build all Docker images
docker-build-all: docker-build docker-build-sidecar
	@echo "All Docker images built successfully"

## Push Docker images (customize registry as needed)
docker-push:
	@echo "Pushing Docker images..."
	docker push $(DOCKER_IMAGE_BACKEND):latest
	docker push $(DOCKER_IMAGE_SIDECAR):latest

## Deploy to Kubernetes
k8s-deploy:
	@echo "Deploying to Kubernetes..."
	kubectl apply -f k8s/namespace.yaml
	kubectl apply -f k8s/sidecar-deployment.yaml
	kubectl apply -f k8s/backend-deployment.yaml
	@echo "Deployment complete!"
	@echo "Waiting for pods to be ready..."
	kubectl wait --for=condition=ready pod -l app=sidecar -n $(K8S_NAMESPACE) --timeout=120s
	kubectl wait --for=condition=ready pod -l app=backend -n $(K8S_NAMESPACE) --timeout=120s
	@echo "All pods are ready!"

## Delete Kubernetes deployment
k8s-delete:
	@echo "Deleting Kubernetes resources..."
	kubectl delete -f k8s/backend-deployment.yaml --ignore-not-found
	kubectl delete -f k8s/sidecar-deployment.yaml --ignore-not-found
	kubectl delete -f k8s/namespace.yaml --ignore-not-found
	@echo "Cleanup complete!"

## View Kubernetes logs
k8s-logs:
	@echo "Sidecar logs:"
	kubectl logs -n $(K8S_NAMESPACE) -l app=sidecar --tail=50
	@echo "\nBackend logs:"
	kubectl logs -n $(K8S_NAMESPACE) -l app=backend --tail=50

## Get Kubernetes status
k8s-status:
	@echo "Checking deployment status..."
	kubectl get all -n $(K8S_NAMESPACE)

## Port forward for local access
k8s-forward:
	@echo "Port forwarding backend service to localhost:8080..."
	kubectl port-forward -n $(K8S_NAMESPACE) svc/backend-service 8080:80
