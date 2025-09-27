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
  GO_VERSION: `)

	if info.Version != "" {
		pipeline.WriteString(fmt.Sprintf(" '%s'", info.Version))
	} else {
		pipeline.WriteString(" '1.19'")
	}
	pipeline.WriteString(fmt.Sprintf(`
  MAIN_PACKAGE_PATH: '%s'`, info.MainFilePath))

	pipeline.WriteString(`
   jobs:`)

	// Job test (если есть тесты)
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

		if strings.Contains(info.Architecture, "standard-go-layout") {
			pipeline.WriteString(`    - name: Build all commands
      run: go build ./cmd/...
`)
		} else {
			pipeline.WriteString(`    - name: Build main package
      run: go build ${{ env.MAIN_PACKAGE_PATH }}
`)
		}

		// Добавляем линтеры если нужно
		if containsDependency(info.Dependencies, "web-framework") {
			pipeline.WriteString(`    - name: Security scan
      run: go vet ./...
`)
		}
	}

	// Job build
	pipeline.WriteString(`
    build:
      runs-on: ubuntu-latest
`)

	// Добавляем зависимость только если есть тесты
	if info.HasTests {
		pipeline.WriteString("    needs: test\n")
	}

	pipeline.WriteString(`    steps:
    - uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: ${{ env.GO_VERSION }}
        
    - name: Download dependencies
      run: go mod download
`)

	pipeline.WriteString(`    - name: Build main package
      run: |
        mkdir -p bin
        go build -o bin/app ${{ env.MAIN_PACKAGE_PATH }}
`)

	pipeline.WriteString(`    - name: Upload artifacts
      uses: actions/upload-artifact@v4
      with:
        name: go-binaries
        path: bin/
`)

	return pipeline.String()
}

func containsDependency(deps []string, depType string) bool {
	for _, dep := range deps {
		if strings.Contains(dep, depType) {
			return true
		}
	}
	return false
}
