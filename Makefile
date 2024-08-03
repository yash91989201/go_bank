build:
	@go build -o bin/go_bank

run: build	
	@./bin/go_bank

test: 
	@go test -v ./..