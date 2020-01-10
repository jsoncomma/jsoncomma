package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"sync"
)

// generates PNG images
// note that you need inkscape installed

func main() {
	var wg sync.WaitGroup

	if err := os.MkdirAll("./imgs/png/", 0755); err != nil {
		log.Fatal(err)
	}

	for _, size := range []int{16, 32, 64, 128, 256, 512} {
		wg.Add(1)
		go func(size int) {
			defer wg.Done()
			strSize := strconv.Itoa(size)
			cmd := exec.Command("inkscape", "--without-gui", "--file", "./imgs/jsoncomma.svg", "--export-width", strSize, "--export-height", strSize, "--export-png", fmt.Sprintf("./imgs/png/%sx%s.png", strSize, strSize))
			if err := cmd.Run(); err != nil {
				log.Printf("size: %d: %s", size, err)
			}
		}(size)
	}

	wg.Wait()
}
