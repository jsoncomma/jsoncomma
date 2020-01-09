package main

// builds the documentation

func main() {
	vars := map[string]string{
		"Version":             getVersion(),
		"JsonCommaServerHelp": getHelp(),
	}
}
