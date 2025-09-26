package generator

import (
	"fmt"
	"strings"

	"github.com/immxrtalbeast/pipeline-gen/internal/analyzer"
)

func generateGoPipelineActions(info *analyzer.ProjectInfo) string {
	var pipeline strings.Builder

	pipeline.WriteString(`name: Go CI/CD Pipeline

on:
  push:
    branches: [ main, master ]
  pull_request:
    branches: [ main, master ]

env:
	GO_VERSION:`)

	if info.Version != "" {
		pipeline.WriteString(fmt.Sprintf(" '%s' ", info.Version))
	} else {
		pipeline.WriteString(" '1.19', '1.20' ")
	}

	pipeline.WriteString(`

jobs:`)
	if info.HasTests {
		pipeline.WriteString(`
  test:
    runs-on: ubuntu-latest
	steps:
	- uses: actions/checkout@v3
	- name: Set up Go
      uses: actions/setup-go@v3
	with:
        go-version: ${{ env.GO_VERSION }}
	- name: Download dependencies
      run: go mod download
    
    - name: Run tests
      run: go test -v ./...`)

		// Добавляем шаги в зависимости от архитектуры
		if strings.Contains(info.Architecture, "standard-go-layout") {
			pipeline.WriteString("\n    - name: Build commands\n      run: go build ./cmd/...")
		} else {
			pipeline.WriteString("\n    - name: Build\n      run: go build -v ./...")
		}

		// Добавляем линтеры если нужно
		if containsDependency(info.Dependencies, "web-framework") {
			pipeline.WriteString("\n    - name: Security scan\n      run: go vet ./...")
		}
	}
	if info.HasTests {
		pipeline.WriteString("\n  build:\n    runs-on: ubuntu-latest\n    needs: test\n    steps:")

	} else {
		pipeline.WriteString("\n  build:\n    runs-on: ubuntu-latest\n		steps:")

	}

	pipeline.WriteString(`
    	- uses: actions/checkout@v3
    	- name: Set up Go
     	  uses: actions/setup-go@v3
		  with:
			  go-version:`)

	if info.Version != "" {
		pipeline.WriteString(fmt.Sprintf(" '%s' ", info.Version))
	} else {
		pipeline.WriteString(" '1.19', '1.20' ")
	}

	pipeline.WriteString(`
    - name: Build
      run: go build -o bin/app ${{ env.MAIN_PACKAGE_PATH }}
    - name: Upload artifact
      uses: actions/upload-artifact@v3
      with:
        name: app-binary
        path: bin/app`)

	return pipeline.String()
}

func containsDependency(deps []string, depType string) bool {
	for _, dep := range deps {
		if strings.HasPrefix(dep, depType) {
			return true
		}
	}
	return false
}
