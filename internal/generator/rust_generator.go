package generator

import (
	"fmt"
	"strings"

	"github.com/immxrtalbeast/pipeline-gen/internal/analyzer"
)

func generateRustPipeline(info *analyzer.ProjectInfo) string {
	var pipeline strings.Builder

	pipeline.WriteString(`name: Rust CI/CD Pipeline

on:
  push:
    branches: [ main, master, develop ]
  pull_request:
    branches: [ main, master, develop ]

jobs:
`)

	// Job для проверки кода (clippy) и форматирования (fmt)
	pipeline.WriteString(`  check:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - name: Install Rust
      uses: actions-rust-lang/setup-rust-toolchain@v1
      with:
        toolchain: `)

	if info.Version != "" && info.Version != "stable" {
		pipeline.WriteString(fmt.Sprintf(" '%s' ", info.Version))
	} else {
		pipeline.WriteString(" 'stable' ")
	}

	pipeline.WriteString(`
        components: clippy, rustfmt
    - name: Check code format
      run: cargo fmt -- --check
    - name: Run clippy
      run: cargo clippy -- -D warnings
`)

	// Job для тестов
	if info.HasTests {
		pipeline.WriteString(`  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        os: [ubuntu-latest, windows-latest, macos-latest]
        rust: [`)

		// Добавляем версии Rust
		if info.Version != "" && info.Version != "stable" {
			pipeline.WriteString(fmt.Sprintf(" '%s', 'stable', 'nightly' ", info.Version))
		} else {
			pipeline.WriteString(" 'stable', 'nightly', '1.70.0' ")
		}

		pipeline.WriteString(`]
    steps:
    - uses: actions/checkout@v3
    - name: Install Rust ${{ matrix.rust }}
      uses: actions-rust-lang/setup-rust-toolchain@v1
      with:
        toolchain: ${{ matrix.rust }}
    - name: Run tests
      run: cargo test
    - name: Run tests with coverage
      run: cargo tarpaulin --out Xml
      if: matrix.os == 'ubuntu-latest' && matrix.rust == 'stable'
    - name: Upload coverage to Codecov
      uses: codecov/codecov-action@v3
      if: matrix.os == 'ubuntu-latest' && matrix.rust == 'stable'
      with:
        files: cobertura.xml
`)

		// Для workspace проектов
		if len(info.Modules) > 1 {
			pipeline.WriteString(`    - name: Run tests for all workspace members
      run: cargo test --workspace
`)
		}
	}

	// Job для сборки
	previousJob := "test"
	if !info.HasTests {
		previousJob = "check"
	}

	pipeline.WriteString(fmt.Sprintf(`  build:
    runs-on: ubuntu-latest
    needs: %s
    strategy:
      matrix:
        target: [x86_64-unknown-linux-gnu, x86_64-pc-windows-msvc, x86_64-apple-darwin]
    steps:
    - uses: actions/checkout@v3
    - name: Install Rust
      uses: actions-rust-lang/setup-rust-toolchain@v1
      with:
        toolchain: `, previousJob))

	if info.Version != "" {
		pipeline.WriteString(fmt.Sprintf(" '%s' ", info.Version))
	} else {
		pipeline.WriteString(" 'stable' ")
	}

	pipeline.WriteString(fmt.Sprintf(`
        target: ${{ matrix.target }}
    - name: Build release
      run: cargo build --release --target ${{ matrix.target }}
    - name: Upload artifacts
      uses: actions/upload-artifact@v3
      with:
        name: rust-${{ matrix.target }}
        path: target/${{ matrix.target }}/release/
`))

	// Job для проверки безопасности (audit)
	pipeline.WriteString(`  security:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - name: Install Rust
      uses: actions-rust-lang/setup-rust-toolchain@v1
      with:
        toolchain: stable
    - name: Install cargo-audit
      run: cargo install cargo-audit
    - name: Audit dependencies
      run: cargo audit
`)

	// Job для бенчмарков (если есть)
	if containsDependency(info.Dependencies, "bench") || detectRustBenchmarks(info) {
		pipeline.WriteString(`  benchmark:
    runs-on: ubuntu-latest
    needs: build
    steps:
    - uses: actions/checkout@v3
    - name: Install Rust
      uses: actions-rust-lang/setup-rust-toolchain@v1
      with:
        toolchain: stable
    - name: Run benchmarks
      run: cargo bench
`)
	}

	// Job для публикации в crates.io (если это библиотека)
	if containsDependency(info.Dependencies, "type:library") {
		pipeline.WriteString(`  publish:
    runs-on: ubuntu-latest
    needs: [check, test, security]
    if: github.event_name == 'push' && contains(github.ref, 'refs/tags/')
    steps:
    - uses: actions/checkout@v3
    - name: Install Rust
      uses: actions-rust-lang/setup-rust-toolchain@v1
      with:
        toolchain: stable
    - name: Publish to crates.io
      run: cargo publish
      env:
        CARGO_REGISTRY_TOKEN: ${{ secrets.CARGO_REGISTRY_TOKEN }}
`)
	}

	// Job для сборки документации
	pipeline.WriteString(`  docs:
    runs-on: ubuntu-latest
    needs: build
    steps:
    - uses: actions/checkout@v3
    - name: Install Rust
      uses: actions-rust-lang/setup-rust-toolchain@v1
      with:
        toolchain: stable
    - name: Build documentation
      run: cargo doc --no-deps
    - name: Upload documentation
      uses: actions/upload-artifact@v3
      with:
        name: rust-docs
        path: target/doc/
`)

	return pipeline.String()
}

// Вспомогательная функция для обнаружения бенчмарков
func detectRustBenchmarks(info *analyzer.ProjectInfo) bool {
	// Проверяем наличие бенчмарков по зависимостям или другим признакам
	return containsDependency(info.Dependencies, "criterion") || 
	       containsDependency(info.Dependencies, "bench") ||
	       info.TestFramework == "criterion"
}