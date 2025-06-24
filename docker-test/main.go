// generate.go
package main

import (
	"os"
	"text/template"

	"example.com/docker-test/docker"
)

func main() {
	// 1. load manifest
	// manifestPath := filepath.Join("modules", "hello-world", "module.yml")
	manifestPath := "./module.yml"
	manifest, err := docker.LoadManifest(manifestPath)
	if err != nil {
		panic(err)
	}

	// 2. parse template
	tmpl, err := template.ParseFiles("./docker/compose.tmpl")
	if err != nil {
		panic(err)
	}

	// 3. render to stdout (or write to file)
	if err := tmpl.Execute(os.Stdout, manifest); err != nil {
		panic(err)
	}
}
