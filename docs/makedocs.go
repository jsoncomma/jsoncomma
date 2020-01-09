package main

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"time"
)

// +build ignore

// builds the documentation

func main() {
	tmpl := template.Must(template.New("index.html.template").Option("missingkey=error").ParseFiles("./docs/index.html.template"))
	f, err := os.Create("./docs/index.html")
	if err != nil {
		log.Fatal(err)
	}
	cssFile, err := os.Open("./docs/main.css")
	if err != nil {
		log.Fatal(err)
	}

	style, err := ioutil.ReadAll(cssFile)
	if err != nil {
		log.Fatalf("reading css file: %s", err)
	}

	// embed the CSS within the HTML because that means that the entire
	// webpage can be loaded at once (everything is contained in index.html).
	// It makes it feel so much faster <3
	// When writing the documenation, there's a dirty hack which allows you
	// to still preview the template file as if you saw the the original HTML file
	// TODO: maybe build a quick dev server?

	if err := tmpl.Execute(f, map[string]interface{}{
		"BotGenerated":        template.HTML(fmt.Sprintf("<!-- HTML Automatically build by makedocs.go on %s-->", time.Now())),
		"Version":             getVersion(),
		"JsonCommaServerHelp": getHelp(),
		"Style":               template.CSS(string(style)),
	}); err != nil {
		log.Fatal(err)
	}

	log.Print("done")
}

func getVersion() string {
	log.Printf("getting version from git")
	output, err := exec.Command("git", "describe", "--tags").Output()

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		log.Print("Stderr:")
		fmt.Println(string(exitErr.Stderr))
		log.Fatal(err)
	} else if err != nil {
		log.Fatal(err)
	}

	return string(bytes.TrimRight(output, "\n"))
}

func getHelp() string {
	log.Printf("getting help message from jsoncomma")
	output, err := exec.Command("./jsoncomma", "server", "-help").Output()

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if exitErr.ExitCode() == 2 {
			output = exitErr.Stderr
		} else {
			log.Print("Stderr:")
			fmt.Println(string(exitErr.Stderr))
			log.Fatal(err)
		}
	} else if err != nil {
		log.Fatal(err)
	}

	return string(output)
}
