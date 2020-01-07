run: $(shell fd --extension go)
	goimports -w .
	go build
	./jsoncomma