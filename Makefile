KUBECONFIG=$(HOME)/.kube/dev
tag=dev
image=paskalmaksim/sre-metrics-exporter:$(tag)
config=config.yaml

run:
	go run --race ./cmd/main.go -config=$(config) -log.level=DEBUG -web.listen-address=127.0.0.1:8080
test:
	go mod tidy
	go fmt ./...
	go vet ./...
	go test ./...
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run -v
build:
	go run github.com/goreleaser/goreleaser@latest build --clean --skip-validate --snapshot
	mv ./dist/sre-metrics-exporter_linux_amd64_v1/sre-metrics-exporter sre-metrics-exporter
	docker build --pull . -t $(image) --load
push:
	docker push $(image)
testChart:
	ct lint --charts ./charts/sre-metrics-exporter
deploy:
	helm upgrade sre-metrics-exporter \
	--install \
	--namespace sre-metrics-exporter \
	--create-namespace \
	./charts/sre-metrics-exporter \
	--set podLabels.hash=`git rev-parse --short HEAD` \
	--set registry.image=$(image) \
	--set registry.imagePullPolicy=Always \
	--set-file config=$(config)