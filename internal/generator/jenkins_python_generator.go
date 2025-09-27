package generator

import (
	"strings"

	"github.com/immxrtalbeast/pipeline-gen/internal/analyzer"
)

func generateJenkinsPythonPipeline(info *analyzer.ProjectInfo) string {
	var pipeline strings.Builder

	pipeline.WriteString(`pipeline {
    agent any
    
    environment {
        PYTHON_VERSION = '`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("3.9")
	}

	pipeline.WriteString(`'
    }
    
    tools {
        python '`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("3.9")
	}

	pipeline.WriteString(`'
    }
    
    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }
        
        stage('Setup') {
            steps {
                sh 'python --version'
                sh 'pip --version'
            }
        }
        
        stage('Dependencies') {
            steps {`)

	// Установка зависимостей в зависимости от инструмента сборки
	switch info.BuildTool {
	case "poetry":
		pipeline.WriteString(`
                sh 'pip install poetry'
                sh 'poetry install'`)
	case "pipenv":
		pipeline.WriteString(`
                sh 'pip install pipenv'
                sh 'pipenv install'`)
	default:
		pipeline.WriteString(`
                sh 'pip install -r requirements.txt'`)
	}

	pipeline.WriteString(`
            }
        }
`)

	if info.HasTests {
		pipeline.WriteString(`
        stage('Test') {
            steps {`)

		switch info.BuildTool {
		case "poetry":
			pipeline.WriteString(`
                sh 'poetry run pytest --cov=. --cov-report=xml --cov-report=html'`)
		case "pipenv":
			pipeline.WriteString(`
                sh 'pipenv run pytest --cov=. --cov-report=xml --cov-report=html'`)
		default:
			pipeline.WriteString(`
                sh 'pytest --cov=. --cov-report=xml --cov-report=html'`)
		}

		pipeline.WriteString(`
            }
            post {
                always {
                    publishTestResults testResultsPattern: 'test-results.xml'
                    publishCoverage adapters: [coberturaAdapter('coverage.xml')], sourceFileResolver: sourceFiles('STORE_LAST_BUILD')
                }
            }
        }
`)
	}

	pipeline.WriteString(`
        stage('Build') {
            steps {`)

	switch info.BuildTool {
	case "poetry":
		pipeline.WriteString(`
                sh 'poetry build'`)
	case "pipenv":
		pipeline.WriteString(`
                sh 'pipenv run python setup.py sdist bdist_wheel'`)
	default:
		pipeline.WriteString(`
                sh 'python setup.py sdist bdist_wheel'`)
	}

	pipeline.WriteString(`
            }
            post {
                always {
                    archiveArtifacts artifacts: 'dist/*', fingerprint: true
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
