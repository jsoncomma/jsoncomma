.PHONY: test run fuzz

test: $(shell fd --extension go)
	goimports -w .
	go test ./...

run: $(shell fd --extension go)
	goimports -w .
	go build
	./jsoncomma

fuzzing_workdir/jsoncomma.zip:
	go-fuzz-build -o fuzzing_workdir/jsoncomma.zip ./internals

fuzz: fuzzing_workdir/jsoncomma.zip
	go-fuzz -bin=fuzzing_workdir/jsoncomma.zip -workdir=fuzzing_workdir/