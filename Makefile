CDB_DSN ?= postgresql://root@localhost:26257/linkgraph?sslmode=disable
DB_NAME ?= linkgraph
DB_CONTAINER ?= cdb
ES_NODES ?= http://localhost:9200
API_PROTO_FILES=$(shell find api -name *.proto)
IMAGE ?= search-monolith
CDB_MIGRATIONS_IMAGE ?= cdb-migrations
SHA = $(shell git rev-parse --short HEAD)
KIND_IMSGE ?= kindest/node:v1.24.6

kind-up: 
	@echo "[KinD] bootstrapping cluster with kubernetes 1.24" 
	@chmod +x ./hack/kind-with-registry.sh && ./hack/kind-with-registry.sh
	@echo "[helm] adding required repos"
	@helm repo add cockroachdb https://charts.cockroachdb.com/
	@helm repo add elastic https://helm.elastic.co

deploy:
	@echo "[kubectl apply] deploying namespaces"
	@kubectl apply -f ./build/k8s/namespaces.yaml
	@kubectl apply -f ./build/k8s/statefullset.yaml
	@kubectl apply -f ./build/k8s/svc.yaml
	@kubectl apply -f ./build/k8s/svc-headless.yaml
	@kubectl apply -f ./build/k8s/ingress.yaml
	@echo "[helm install] deploying dataplane components"
	@echo "[helm install] deploying cockroachdb"
	@helm upgrade cdb -i \
	    -n dataplane \
	    -f ./build/k8s/cdb.yaml \
	    cockroachdb/cockroachdb
	@echo "[helm install] deploying elasticsearch"
	@helm upgrade es -i \
	    -n dataplane \
	    -f ./build/k8s/es.yaml \
	    elastic/elasticsearch
	@echo "[kubectl apply] applying migrations job"
	@kubectl apply -f ./build/k8s/cdb-migrations.yaml


build-image:
	echo "[docker build] building localhost:5001/${IMAGE}:${SHA}"
	@docker build --file ./Dockerfile.search-monilith \
		--tag localhost:5001/${IMAGE}:${SHA} \
		--tag localhost:5001/${IMAGE}:latest \
		-t localhost:5001/${IMAGE} \
		.
	@echo "[docker build] building localhost:5001/${CDB_MIGRATIONS_IMAGE}:${SHA}"
	@docker build --file ./Dockerfile.cdb-migrations \
		--tag localhost:5001/${CDB_MIGRATIONS_IMAGE}:${SHA} \
		--tag localhost:5001/${CDB_MIGRATIONS_IMAGE}:latest \
		-t localhost:5001/${CDB_MIGRATIONS_IMAGE} \
		.
	
push-image:
	@echo "[load image] loading localhost:5001/${IMAGE}:latest"	
	@docker push localhost:5001/${IMAGE}
	@echo "[load image] loading localhost:5001/${CDB_MIGRATIONS_IMAGE}:latest"	
	@docker push localhost:5001/${CDB_MIGRATIONS_IMAGE}

build-push: build-image push-image

db-run-migrations: migrate-check-deps check-db-env
	migrate -source file://migrations -database '$(subst postgresql,cockroach,${CDB_DSN})' up

migrate-check-deps: 
	@if [ -z `which migrate` ]; then \
		echo "installing go-migrate with cockroach support"; \
		go install -tags 'cockroachdb postgres' -u github.com/golang-migrate/migrate/v4/cmd/migrate; \
	fi

create-db:
	@docker exec -it ${DB_CONTAINER} ./cockroach sql --insecure --execute="CREATE DATABASE IF NOT EXISTS ${DB_NAME}"

define dsn_missing_error

CDB_DSN envvar is undefined. To run the migrations this envvar
must point to a cockroach db instance. For example, if you are
running a local cockroachdb (with --insecure) and have created
a database called 'linkgraph' you can define the envvar by 
running:

export CDB_DSN='postgresql://root@localhost:26257/linkgraph?sslmode=disable'

endef
export dsn_missing_error

check-db-env:
ifndef CDB_DSN
	$(error ${dsn_missing_error})
endif

test: 
	@echo "[go test] running tests and collecting coverage metrics"
	CDB_DSN=${CDB_DSN} ES_NODES=${ES_NODES} go test -v -tags all_tests -race -coverprofile=coverage.txt -covermode=atomic ./...

test_unit:
	@echo "[go test] running unit tests and collection coverage metrics"
	go test -v -tags all_tests -race -coverprofile=coverage.txt -covermode=atomic ./...

test-debug:
	@read -p "Enter Test Package You Wish To Debug:" package_name; \
	CDB_DSN=${CDB_DSN} ES_NODES=${ES_NODES} dlv test $$package_name;


apply:
	@kubectl apply -f ./build/cdb

up-cluster:
	@kind create cluster --name search --config ${KIND_CONFIG}

proto: ensure-proto-deps
	@echo "[protoc] generating protos for API"
	@protoc --proto_path=. \
 		   	   --go_out=paths=source_relative:. \
 		   	   --go-grpc_out=paths=source_relative:. \
		   	   $(API_PROTO_FILES)

ensure-proto-deps:
	@echo "[go get] ensuring protoc packages are available"
	@go get google.golang.org/grpc
	@go get google.golang.org/protobuf
