package generator

import (
	"strings"

	"github.com/immxrtalbeast/pipeline-gen/internal/analyzer"
)

func generateJenkinsJavaScriptPipeline(info *analyzer.ProjectInfo) string {
	var pipeline strings.Builder

	pipeline.WriteString(`pipeline {
    agent any
    
    environment {
        NODE_VERSION = '`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("18")
	}

	pipeline.WriteString(`'
    }
    
    tools {
        nodejs '`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("18")
	}

	pipeline.WriteString(`'
    }
    
    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }
        
        stage('Dependencies') {
            steps {`)

	// Установка зависимостей в зависимости от инструмента сборки
	switch info.BuildTool {
	case "yarn":
		pipeline.WriteString(`
                sh 'yarn install --frozen-lockfile'`)
	case "pnpm":
		pipeline.WriteString(`
                sh 'npm install -g pnpm'
                sh 'pnpm install --frozen-lockfile'`)
	default:
		pipeline.WriteString(`
                sh 'npm ci'`)
	}

	pipeline.WriteString(`
            }
        }
`)

	// Lint job если есть ESLint/Prettier
	if containsDependency(info.Dependencies, "eslint") || containsDependency(info.Dependencies, "prettier") {
		pipeline.WriteString(`
        stage('Lint') {
            steps {`)

		switch info.BuildTool {
		case "yarn":
			pipeline.WriteString(`
                sh 'yarn lint'
                sh 'yarn format:check'`)
		case "pnpm":
			pipeline.WriteString(`
                sh 'pnpm lint'
                sh 'pnpm format:check'`)
		default:
			pipeline.WriteString(`
                sh 'npm run lint'
                sh 'npm run format:check'`)
		}

		pipeline.WriteString(`
            }
        }
`)
	}

	if info.HasTests {
		pipeline.WriteString(`
        stage('Test') {
            steps {`)

		switch info.BuildTool {
		case "yarn":
			pipeline.WriteString(`
                sh 'yarn test --coverage'`)
		case "pnpm":
			pipeline.WriteString(`
                sh 'pnpm test --coverage'`)
		default:
			pipeline.WriteString(`
                sh 'npm test --coverage'`)
		}

		pipeline.WriteString(`
            }
            post {
                always {
                    publishTestResults testResultsPattern: 'test-results.xml'
                    publishCoverage adapters: [coberturaAdapter('coverage/cobertura-coverage.xml')], sourceFileResolver: sourceFiles('STORE_LAST_BUILD')
                }
            }
        }
`)
	}

	pipeline.WriteString(`
        stage('Build') {
            steps {`)

	switch info.BuildTool {
	case "yarn":
		pipeline.WriteString(`
                sh 'yarn build'`)
	case "pnpm":
		pipeline.WriteString(`
                sh 'pnpm build'`)
	default:
		pipeline.WriteString(`
                sh 'npm run build'`)
	}

	pipeline.WriteString(`
            }
            post {
                always {
                    archiveArtifacts artifacts: 'dist/**,build/**,.next/**,out/**', fingerprint: true
                }
            }
        }
    }
    
    post {
        always {
            cleanWs()
        }
        success {
            echo 'Pipeline completed successfully!'
        }
        failure {
            echo 'Pipeline failed!'
        }
    }
}`)

	return pipeline.String()
}
