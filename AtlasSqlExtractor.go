//go:build ignore

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

type AtlasSchema struct {
	Spec struct {
		Schema struct {
			SQL string `yaml:"sql"`
		} `yaml:"schema"`
	} `yaml:"spec"`
}

func main() {
	_, filename, _, _ := runtime.Caller(0)
	projectDir := filepath.Dir(filename)

	data, err := os.ReadFile(filepath.Join(projectDir, "k8s", "atlas-schema", "atlas-schema.yaml"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read atlas-schema.yaml: %v\n", err)
		os.Exit(1)
	}

	var schema AtlasSchema
	if err := yaml.Unmarshal(data, &schema); err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse YAML: %v\n", err)
		os.Exit(1)
	}

	sql := schema.Spec.Schema.SQL
	if sql == "" {
		fmt.Fprintln(os.Stderr, "no SQL found in spec.schema.sql")
		os.Exit(1)
	}

	outPath := filepath.Join(projectDir, "testdata", "schema.sql")
	if err := os.WriteFile(outPath, []byte(sql), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write %s: %v\n", outPath, err)
		os.Exit(1)
	}

	fmt.Printf("Extracted SQL to %s\n", outPath)
}