.PHONY: test run fuzz

test: $(shell fd --extension go)
	goimports -w .
	go test ./runereader

run: $(shell fd --extension go)
	goimports -w .
	go build
	./jsoncomma

fuzz:
	go-fuzz-build -o fuzzing_workdir/jsoncomma.zip ./internals
