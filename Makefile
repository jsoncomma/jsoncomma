.PHONY: test run fuzz

GOFILES=$(shell fd --extension go)

test: $(GOFILES)
	goimports -w .
	go test ./...

run: $(GOFILES)
	goimports -w .
	go build
	./jsoncomma

fuzzing_workdir/jsoncomma.zip: $(GOFILES)
	go-fuzz-build -o fuzzing_workdir/jsoncomma.zip ./internals

fuzz: fuzzing_workdir/jsoncomma.zip
	go-fuzz -bin=fuzzing_workdir/jsoncomma.zip -workdir=fuzzing_workdir/