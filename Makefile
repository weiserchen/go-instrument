
.PHONY: test
test:
	go test -v -count=1 ./...

.PHONY: benchmark
benchmark:
	go test -v ./... -bench=. -run=^$ -benchmem -count=10 | benchstat -