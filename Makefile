.PHONY: test run fuzz build

GOFILES=$(shell fd --extension go)

run: $(GOFILES)
	goimports -w .
	go build
	./jsoncomma

build: $(GOFILES)
	goimports -w .
	go generate
	go build 

test: $(GOFILES)
	goimports -w .
	go test ./...

profs: $(GOFILES)
	mkdir -p profs
	go test ./internals/ -o profs/internals.test -bench 'Fix$' -cpuprofile=profs/cpu.fix | tee profs/bench.fix
	touch profs

fuzzing_workdir/jsoncomma.zip: $(GOFILES)
	go-fuzz-build -o fuzzing_workdir/jsoncomma.zip ./internals

fuzz: fuzzing_workdir/jsoncomma.zip
	go-fuzz -bin=fuzzing_workdir/jsoncomma.zip -workdir=fuzzing_workdir/ -func FuzzJson