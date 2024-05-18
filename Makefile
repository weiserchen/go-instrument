
.PHONY: test
test:
	go test -v -count=1 ./...

.PHONY: bench
bench:
	go test -v ./... -bench=. -run=^$ -benchmem -count=10 | benchstat -