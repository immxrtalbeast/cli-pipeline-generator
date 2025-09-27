package generator

import (
	"strings"

	"github.com/immxrtalbeast/pipeline-gen/internal/analyzer"
)

func generateJenkinsRustPipeline(info *analyzer.ProjectInfo) string {
	var pipeline strings.Builder

	pipeline.WriteString(`pipeline {
    agent any
    
    environment {
        RUST_VERSION = '`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("stable")
	}

	pipeline.WriteString(`'
        CARGO_HOME = "${WORKSPACE}/.cargo"
        RUSTUP_HOME = "${WORKSPACE}/.rustup"
    }
    
    tools {
        rust 'Rust'
    }
    
    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }
        
        stage('Setup') {
            steps {
                sh 'rustc --version'
                sh 'cargo --version'
                sh 'rustup component add clippy rustfmt'
            }
        }
        
        stage('Check') {
            steps {
                sh 'cargo check'
                sh 'cargo fmt -- --check'
                sh 'cargo clippy -- -D warnings'
            }
        }
`)

	if info.HasTests {
		pipeline.WriteString(`
        stage('Test') {
            steps {
                sh 'cargo test'
                sh 'cargo tarpaulin --out Xml'
            }
            post {
                always {
                    publishTestResults testResultsPattern: '**/test-results.xml'
                    publishCoverage adapters: [coberturaAdapter('cobertura.xml')], sourceFileResolver: sourceFiles('STORE_LAST_BUILD')
                }
            }
        }
`)

		// Для workspace проектов
		if len(info.Modules) > 1 {
			pipeline.WriteString(`
        stage('Test Workspace') {
            steps {
                sh 'cargo test --workspace'
            }
        }
`)
		}
	}

	pipeline.WriteString(`
        stage('Build') {
            steps {
                sh 'cargo build --release'
                archiveArtifacts artifacts: 'target/release/*', fingerprint: true
            }
        }
        
        stage('Build Linux') {
            steps {
                sh 'rustup target add x86_64-unknown-linux-gnu'
                sh 'cargo build --release --target x86_64-unknown-linux-gnu'
                archiveArtifacts artifacts: 'target/x86_64-unknown-linux-gnu/release/*', fingerprint: true
            }
        }
        
        stage('Build Windows') {
            steps {
                sh 'apt-get update && apt-get install -y gcc-mingw-w64'
                sh 'rustup target add x86_64-pc-windows-gnu'
                sh 'cargo build --release --target x86_64-pc-windows-gnu'
                archiveArtifacts artifacts: 'target/x86_64-pc-windows-gnu/release/*', fingerprint: true
            }
        }
`)

	pipeline.WriteString(`
        stage('Security') {
            steps {
                sh 'cargo install cargo-audit'
                sh 'cargo audit'
            }
        }
`)

	// Benchmarks stage если есть
	if containsDependency(info.Dependencies, "bench") || detectRustBenchmarks(info) {
		pipeline.WriteString(`
        stage('Benchmark') {
            steps {
                sh 'cargo bench'
            }
        }
`)
	}

	// Documentation stage
	pipeline.WriteString(`
        stage('Documentation') {
            steps {
                sh 'cargo doc --no-deps'
                archiveArtifacts artifacts: 'target/doc/**', fingerprint: true
            }
        }
`)

	// Publish stage для библиотек
	if containsDependency(info.Dependencies, "type:library") {
		pipeline.WriteString(`
        stage('Publish') {
            when {
                expression { return env.BRANCH_NAME == 'main' || env.BRANCH_NAME == 'master' }
            }
            steps {
                sh 'cargo publish'
            }
        }
`)
	}

	pipeline.WriteString(`    }
    
    post {
        always {
            cleanWs()
        }
        success {
            echo 'Rust Pipeline completed successfully!'
        }
        failure {
            echo 'Rust Pipeline failed!'
        }
    }
}`)

	return pipeline.String()
}
