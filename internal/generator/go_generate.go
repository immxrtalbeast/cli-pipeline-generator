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
		pipeline.WriteString(fmt.Sprintf("'%s'\n\n", info.Version))
	} else {
		pipeline.WriteString("'1.21'\n\n")
	}

	pipeline.WriteString("jobs:\n")

	// Job test (если есть тесты)
	if info.HasTests {
		pipeline.WriteString(`  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: ${{ env.GO_VERSION }}
    
    - name: Download dependencies
      run: go mod download
      
    - name: Run tests
      run: go test -v ./...
`)

		// Дополнительные шаги в зависимости от архитектуры
		if strings.Contains(info.Architecture, "standard-go-layout") {
			pipeline.WriteString(`    - name: Build all commands
      run: go build ./cmd/...
`)
		} else {
			pipeline.WriteString(`    - name: Build all packages
      run: go build ./...
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
	pipeline.WriteString(`  build:
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

	// Умная логика сборки в зависимости от структуры проекта
	if strings.Contains(info.Architecture, "standard-go-layout") {
		// Для стандартной Go структуры с cmd/ папкой
		pipeline.WriteString(`    - name: Build all binaries from cmd/
      run: |
        mkdir -p bin
        for dir in ./cmd/*/; do
          if [ -d "$dir" ]; then
            binary_name=$(basename "$dir")
            echo "Building $binary_name from $dir"
            go build -o "bin/${binary_name}" "./cmd/${binary_name}"
          fi
        done
`)
	} else {
		// Универсальный подход для любого проекта
		pipeline.WriteString(`    - name: Build using recursive approach
      run: |
        mkdir -p bin
        # Пытаемся найти и собрать main пакеты
        if [ -f "go.mod" ]; then
          # Используем go list чтобы найти все main пакеты
          main_packages=$(go list -f '{{.ImportPath}} {{.Name}}' ./... | grep ' main$' | cut -d' ' -f1)
          if [ -n "$main_packages" ]; then
            for pkg in $main_packages; do
              binary_name=$(basename "$pkg")
              echo "Building $binary_name from $pkg"
              go build -o "bin/${binary_name}" "$pkg"
            done
          else
            # Fallback: пытаемся собрать стандартным способом
            go build -o bin/app ./...
          fi
        else
          # Для проектов без go.mod
          go build -o bin/app .
        fi
`)
	}

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
