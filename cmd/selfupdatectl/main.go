package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

var version, genDir string

type current struct {
	Version string
	Sha256  []byte
}

func generateSha256(path string) []byte {
	h := sha256.New()
	b, err := os.ReadFile(path)
	if err != nil {
		fmt.Println(err)
	}
	h.Write(b)
	sum := h.Sum(nil)
	return sum
}

func createUpdate(path string, platform string) {
	c := current{Version: version, Sha256: generateSha256(path)}

	b, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		fmt.Println("error:", err)
	}
	err = os.WriteFile(filepath.Join(genDir, platform+".json"), b, 0755)
	if err != nil {
		panic(err)
	}
	err = os.MkdirAll(filepath.Join(genDir, version), 0755)
	if err != nil {
		panic(err)
	}
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)

	f, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	_, err = w.Write(f)
	if err != nil {
		panic(err)
	}

	w.Close() // You must close this first to flush the bytes to the buffer.

	err = os.WriteFile(filepath.Join(genDir, version, platform+".gz"), buf.Bytes(), 0755)
	if err != nil {
		fmt.Println(err)
	}
}

func printUsage() {
	fmt.Println("")
	fmt.Println("Positional arguments:")
	fmt.Println("\tSingle platform: go-selfupdate myapp 1.2")
	fmt.Println("\tCross platform: go-selfupdate /tmp/mybinares/ 1.2")
}

func main() {
	outputDirFlag := flag.String("o", "public", "Output directory for writing updates")

	var defaultPlatform string
	goos := os.Getenv("GOOS")
	goarch := os.Getenv("GOARCH")
	if goos != "" && goarch != "" {
		defaultPlatform = goos + "-" + goarch
	} else {
		defaultPlatform = runtime.GOOS + "-" + runtime.GOARCH
	}
	platformFlag := flag.String("platform", defaultPlatform,
		"Target platform in the form OS-ARCH. Defaults to running os/arch or the combination of the environment variables GOOS and GOARCH if both are set.")

	flag.Parse()
	if flag.NArg() < 2 {
		flag.Usage()
		printUsage()
		os.Exit(0)
	}

	platform := *platformFlag
	appPath := flag.Arg(0)
	version = flag.Arg(1)
	genDir = *outputDirFlag

	err := os.MkdirAll(genDir, 0755)
	if err != nil {
		panic(err)
	}
	// If dir is given create update for each file
	fi, err := os.Stat(appPath)
	if err != nil {
		panic(err)
	}

	if fi.IsDir() {
		files, err := os.ReadDir(appPath)
		if err == nil {
			var wg sync.WaitGroup
			wg.Add(len(files))

			for _, file := range files {
				go func(file fs.DirEntry) {
					defer wg.Done()
					createUpdate(filepath.Join(appPath, file.Name()), file.Name())
				}(file)
			}
			wg.Wait()
			os.Exit(0)
		}
	}

	createUpdate(appPath, platform)
}
