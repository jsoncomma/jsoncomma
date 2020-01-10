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

// builds the documentation

func main() {

    buildToGithubPages := false

    if len(os.Args) == 2 && os.Args[1] == "gh-pages" {
        buildToGithubPages = true
    } else if len(os.Args) != 1 {
        log.Fatalf("expect one optional argument, 'gh-pages'. Given %v", os.Args[1:])
    }

    if buildToGithubPages {
        log.Print("making sure git status is clean before building to github pages")
        output, err := exec.Command("git", "status", "--short").Output()
        if err != nil {
            log.Fatalf("%v", err)
        }
        if len(output) > 0 {
            log.Fatalf("expected clean state. Please commit your changes first")
        }
    }

    log.Print("building documentation...")

    // embed the CSS within the HTML because that means that the entire
    // webpage can be loaded at once (everything is contained in index.html).
    // It makes it feel so much faster <3
    // When writing the documenation, there's a dirty hack which allows you
    // to still preview the template file as if you saw the the original HTML file
    // TODO: maybe build a quick dev server?

    tmpl := template.Must(template.New("index.html.template").Option("missingkey=error").ParseFiles("./docs/index.html.template"))

    cssFile, err := os.Open("./docs/main.css")
    if err != nil {
        log.Fatal(err)
    }

    style, err := ioutil.ReadAll(cssFile)
    if err != nil {
        log.Fatalf("reading css file: %s", err)
    }

    f, err := os.Create("./docs/build.html")
    if err != nil {
        log.Fatal(err)
    }

    version := getVersion()

    if err := tmpl.Execute(f, map[string]interface{}{
        "BotGenerated":        template.HTML(fmt.Sprintf("<!-- HTML Automatically build by makedocs.go on %s-->", time.Now())),
        "Version":             version,
        "JsonCommaServerHelp": getHelp(),
        "Style":               template.CSS(string(style)),
    }); err != nil {
        log.Fatal(err)
    }

    if !buildToGithubPages {
        log.Print("done")
        return
    }

    log.Print("moving build to github pages branch")

    // the reason I build to ./docs/build.html and then move to index.html is 
    // so that when you just run the build by itself (without pushing to
    // github pages branch), the build is ignored (docs/build.html in .gitignore).
    // renaming it index.html allows us to use git's stash (because index.html
    // isn't ignore)

    log.Print("  renaming ./docs/build.html to index.html")
    if err := os.Rename("./docs/build.html", "./index.html"); err != nil {
    	log.Fatalf("err: %s", err)
    }

    log.Print("  creating build stash [git stash --include-untracked]")
    if err := exec.Command("git", "stash", "--include-untracked").Run(); err != nil {
        var exitErr *exec.ExitError
        if errors.As(err, &exitErr) {
            log.Printf("Stderr: %s", exitErr.Stderr)
        }
        log.Fatalf("err: %s", err)
    }

    log.Printf("  checkout gh-pages [git checkout gh-pages]")
    if err := exec.Command("git", "checkout", "gh-pages").Run(); err != nil {
        var exitErr *exec.ExitError
        if errors.As(err, &exitErr) {
            log.Printf("Stderr: %s", exitErr.Stderr)
        }
        log.Fatalf("err: %s", err)
    }

    log.Printf("  poping build stash [git stash pop]")
    if err := exec.Command("git", "stash", "pop").Run(); err != nil {
        var exitErr *exec.ExitError
        if errors.As(err, &exitErr) {
            log.Printf("Stderr: %s", exitErr.Stderr)
        }
        log.Fatalf("err: %s", err)
    }

    log.Printf("  add index.html to git index [git add index.html]")
    if err := exec.Command("git", "add", "index.html").Run(); err != nil {
    	var exitErr *exec.ExitError
    	if errors.As(err, &exitErr) {
    	    log.Printf("Stderr: %s", exitErr.Stderr)
    	}
    	log.Fatalf("err: %s", err)
    }

    log.Printf("  commiting build [git commit -m \"build %s\"]", version)
    if err := exec.Command("git", "commit", "-m", fmt.Sprintf("build %s", version)).Run(); err != nil {
        var exitErr *exec.ExitError
        if errors.As(err, &exitErr) {
            log.Printf("Stderr: %s", exitErr.Stderr)
        }
        log.Fatalf("err: %s", err)
    }

    log.Printf("  return to original branch [git checkout -]")
    if err := exec.Command("git", "checkout", "-").Run(); err != nil {
        var exitErr *exec.ExitError
        if errors.As(err, &exitErr) {
            log.Printf("Stderr: %s", exitErr.Stderr)
        }
        log.Fatalf("err: %s", err)
    }

    log.Print("done")
}

func getVersion() string {
    log.Printf("  getting jsoncomma version from git")
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
    log.Printf("  getting help message from jsoncomma")

    if _, err := exec.LookPath("./jsoncomma"); err != nil {
        log.Fatalf  ("./jsoncomma doesn't exists. Run go build first (error: %s)", err)
    }

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
