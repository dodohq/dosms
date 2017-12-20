setup_githook:
	rm -f .git/hooks/pre-commit.sample
	curl https://gist.githubusercontent.com/stanleynguyen/dde089f7728f2ad74a5d1489c10cde83/raw/cdf07988d0ba5c5d69a562bbe7b8a08bb1716bf4/pre-commit.go.sh > .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit

setup_dbtool:
	go get -u -d github.com/mattes/migrate/cli github.com/lib/pq
	go build -tags 'postgres' -o $(GOPATH)/bin/migrate github.com/mattes/migrate/cli

db_migrate:
	migrate -database $(DB) -path migrations/ up

db_make:
	migrate create -ext sql -dir migrations $(NAME)

db_rollback:
	migrate -database $(DB) -path migrations/ down

dev:
	GO_ENV=development fresh