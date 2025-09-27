package generator

import (
	"strings"

	"github.com/immxrtalbeast/pipeline-gen/internal/analyzer"
)

func generateGitLabRustPipeline(info *analyzer.ProjectInfo) string {
	var pipeline strings.Builder

	pipeline.WriteString(`stages:
  - check
  - test
  - build
  - security
  - benchmark
  - docs
  - deploy

variables:
  CARGO_HOME: $CI_PROJECT_DIR/.cargo
  RUST_VERSION: '`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("stable")
	}

	pipeline.WriteString(`'

cache:
  paths:
    - .cargo/registry/
    - .cargo/git/
    - target/

check:
  stage: check
  image: rust:`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("latest")
	}

	pipeline.WriteString(`-slim
  before_script:
    - rustc --version
    - cargo --version
  script:
    - cargo check
    - cargo fmt -- --check
    - cargo clippy -- -D warnings
  artifacts:
    paths:
      - target/
    expire_in: 1 hour

`)

	if info.HasTests {
		pipeline.WriteString(`test:
  stage: test
  image: rust:`)

		if info.Version != "" {
			pipeline.WriteString(info.Version)
		} else {
			pipeline.WriteString("latest")
		}

		pipeline.WriteString(`-slim
  script:
    - cargo test
    - cargo tarpaulin --out Xml
  artifacts:
    reports:
      coverage_report:
        coverage_format: cobertura
        path: cobertura.xml
    paths:
      - cobertura.xml
    expire_in: 1 week
  coverage: '/\d+\.\d+%/'

`)

		// Для workspace проектов
		if len(info.Modules) > 1 {
			pipeline.WriteString(`test_workspace:
  stage: test
  image: rust:`)

			if info.Version != "" {
				pipeline.WriteString(info.Version)
			} else {
				pipeline.WriteString("latest")
			}

			pipeline.WriteString(`-slim
  script:
    - cargo test --workspace
  dependencies:
    - check
`)
		}
	}

	pipeline.WriteString(`build:
  stage: build
  image: rust:`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("latest")
	}

	pipeline.WriteString(`-slim
  script:
    - cargo build --release
  artifacts:
    paths:
      - target/release/
    expire_in: 1 week

build_linux:
  stage: build
  image: rust:`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("latest")
	}

	pipeline.WriteString(`-slim
  script:
    - cargo build --release --target x86_64-unknown-linux-gnu
  artifacts:
    paths:
      - target/x86_64-unknown-linux-gnu/release/
    expire_in: 1 week

build_windows:
  stage: build
  image: rust:`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("latest")
	}

	pipeline.WriteString(`-slim
  before_script:
    - apt-get update && apt-get install -y gcc-mingw-w64
  script:
    - rustup target add x86_64-pc-windows-gnu
    - cargo build --release --target x86_64-pc-windows-gnu
  artifacts:
    paths:
      - target/x86_64-pc-windows-gnu/release/
    expire_in: 1 week

`)

	pipeline.WriteString(`security:
  stage: security
  image: rust:`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("latest")
	}

	pipeline.WriteString(`-slim
  script:
    - cargo install cargo-audit
    - cargo audit
  dependencies:
    - check

`)

	// Job для бенчмарков (если есть)
	if containsDependency(info.Dependencies, "bench") || detectRustBenchmarks(info) {
		pipeline.WriteString(`benchmark:
  stage: benchmark
  image: rust:`)

		if info.Version != "" {
			pipeline.WriteString(info.Version)
		} else {
			pipeline.WriteString("latest")
		}

		pipeline.WriteString(`-slim
  script:
    - cargo bench
  dependencies:
    - build

`)
	}

	// Job для сборки документации
	pipeline.WriteString(`docs:
  stage: docs
  image: rust:`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("latest")
	}

	pipeline.WriteString(`-slim
  script:
    - cargo doc --no-deps
  artifacts:
    paths:
      - target/doc/
    expire_in: 1 week

`)

	// Job для публикации в crates.io (если это библиотека)
	if containsDependency(info.Dependencies, "type:library") {
		pipeline.WriteString(`publish:
  stage: deploy
  image: rust:`)

		if info.Version != "" {
			pipeline.WriteString(info.Version)
		} else {
			pipeline.WriteString("latest")
		}

		pipeline.WriteString(`-slim
  script:
    - cargo publish
  only:
    - tags
  dependencies:
    - check
    - test
    - security

`)
	}

	pipeline.WriteString(`deploy:
  stage: deploy
  image: alpine:latest
  script:
    - echo "Deploying Rust application"
  only:
    - main
    - master
`)

	return pipeline.String()
}
