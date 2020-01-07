test: $(shell fd --extension go)
	goimports -w .
	go test ./runereader

run: $(shell fd --extension go)
	goimports -w .
	go build
	./jsoncomma