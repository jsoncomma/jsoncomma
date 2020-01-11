package main

import (
    "bytes"
    "errors"
    "fmt"
    "html/template"
    "io"
    "io/ioutil"
    "log"
    "net/http"
    "os"
    "os/exec"
    "time"
)

// builds the documentation

func main() {

    buildToGithubPages := false

    if len(os.Args) == 2 {
        if os.Args[1] == "gh-pages" {
            buildToGithubPages = true
        } else if os.Args[1] == "dev-server" {
            server()
            return
        } else {
            log.Fatalf("expect 'gh-pages' or 'dev-server', got %q", os.Args[1])
        }
    } else if len(os.Args) != 1 {
        log.Fatalf("expect one optional argument. Given %v", os.Args[1:])
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

    f, err := os.Create("./docs/build.html")
    if err != nil {
        log.Fatal(err)
    }

    version := getVersion()

    buildTo(f, version)

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

    // remove index.html before poping the stash because index.html
    // already exists on this branch, and poping doesn't want to overwrite
    // because master:index.html was stashed with --untracked-files
    log.Printf("  remove index.html")
    if err := os.Remove("index.html"); err != nil && !os.IsNotExist(err) {
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

func server() {
    version := getVersion()
    reload := 0
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        reload++
        if r.URL.Path != "/" && r.URL.Path != "/index.html" {
            http.Error(w, "404 Not Found", http.StatusNotFound)
            return
        }
        if err := buildTo(w, fmt.Sprintf("%s (reload %d)", version, reload)); err != nil {
            http.Error(w, fmt.Sprintf("err building: %s", err), http.StatusInternalServerError)
        }
    })
    log.Printf("Starting server on port 8080")
    if err := http.ListenAndServe(":8080", nil); err != nil {
        log.Fatal(err)
    }
}

func buildTo(w io.Writer, version string) error {
    log.Print("building documentation...")

    // embed the CSS within the HTML because that means that the entire
    // webpage can be loaded at once (everything is contained in index.html).
    // It makes it feel so much faster <3

    tmpl := template.Must(template.New("index.html.template").Option("missingkey=error").ParseFiles("./docs/index.html.template"))

    cssFile, err := os.Open("./docs/main.css")
    if err != nil {
        return err
    }

    style, err := ioutil.ReadAll(cssFile)
    if err != nil {
        return fmt.Errorf("reading css file: %s", err)
    }

    if err := tmpl.Execute(w, map[string]interface{}{
        "BotGenerated":        template.HTML(fmt.Sprintf("<!-- HTML Automatically build by makedocs.go on %s-->", time.Now())),
        "Version":             version,
        "JsonCommaServerHelp": getHelp(),
        "Style":               template.CSS(string(style)),
    }); err != nil {
        return err
    }

    return nil
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
        log.Fatalf("./jsoncomma doesn't exists. Run go build first (error: %s)", err)
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
