.PHONY: test run fuzz profs-memory profs-cpu

GOFILES=$(shell fd --extension go)

run: $(GOFILES)
	goimports -w .
	go build
	./jsoncomma

test: $(GOFILES)
	goimports -w .
	go test ./...

profs: $(GOFILES)
	mkdir -p profs
	go test ./internals/ -o profs/internals.test -bench 'Fix$' -cpuprofile=profs/cpu.fix | tee profs/bench.fix
	touch profs

profs-memory:
	go tool pprof --alloc_space profs/internals.test profs/mem.fix

profs-cpu:
	go tool pprof profs/internals.test profs/cpu.fix

fuzzing_workdir/jsoncomma.zip: $(GOFILES)
	go-fuzz-build -o fuzzing_workdir/jsoncomma.zip ./internals

fuzz: fuzzing_workdir/jsoncomma.zip
	go-fuzz -bin=fuzzing_workdir/jsoncomma.zip -workdir=fuzzing_workdir/ -func FuzzJson