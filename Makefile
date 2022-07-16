KIND_CONFIG := ./build/kind-cluster-config.yml
CDB_DSN ?= postgresql://root@localhost:26257/linkgraph?sslmode=disable
DB_NAME ?= linkgraph
DB_CONTAINER ?= cdb
ES_NODES ?= http://localhost:9200
API_PROTO_FILES=$(shell find api -name *.proto)

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
