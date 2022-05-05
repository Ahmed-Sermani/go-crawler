KIND_CONFIG := ./build/kind-cluster-config.yml

db-run-migrations: migrate-check-deps check-db-env
	migrate -source file://migrations -database '$(subst postgresql,cockroach,${DB_DSN})' up

migrate-check-deps: 
	@if [ -z `which migrate` ]; then \
		echo "installing go-migrate with cockroach support"; \
		go install -tags 'cockroachdb postgres' -u github.com/golang-migrate/migrate/v4/cmd/migrate; \
	fi

define dsn_missing_error

CDB_DSN envvar is undefined. To run the migrations this envvar
must point to a cockroach db instance. For example, if you are
running a local cockroachdb (with --insecure) and have created
a database called 'linkgraph' you can define the envvar by 
running:

export DB_DSN='postgresql://root@localhost:26257/linkgraph?sslmode=disable'

endef
export dsn_missing_error

check-db-env:
ifndef DB_DSN
	$(error ${dsn_missing_error})
endif

test: 
	@echo "[go test] running tests and collecting coverage metrics"
	@go test -v -tags all_tests -race -coverprofile=coverage.txt -covermode=atomic ./...

test_db:
	@echo "[go test] running tests and collecting coverage metrics"
	CDB_DSN='postgresql://root@localhost:26257/linkgraph?sslmode=disable' go test -v -tags all_tests -race -coverprofile=coverage.txt -covermode=atomic ./...

apply:
	@kubectl apply -f ./build/cdb

up-cluster:
	@kind create cluster --name search --config ${KIND_CONFIG}
