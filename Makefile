.PHONY: test run fuzz

GOFILES=$(shell fd --extension go)

run: $(GOFILES)
	goimports -w .
	go build
	./jsoncomma

test: $(GOFILES)
	goimports -w .
	go test ./...

fuzzing_workdir/jsoncomma.zip: $(GOFILES)
	go-fuzz-build -o fuzzing_workdir/jsoncomma.zip ./internals

fuzz: fuzzing_workdir/jsoncomma.zip
	go-fuzz -bin=fuzzing_workdir/jsoncomma.zip -workdir=fuzzing_workdir/ -func FuzzJson