package generator

import (
	"strings"

	"github.com/immxrtalbeast/pipeline-gen/internal/analyzer"
)

func generateJenkinsGoPipeline(info *analyzer.ProjectInfo) string {
	var pipeline strings.Builder

	pipeline.WriteString(`pipeline {
    agent any
    
    environment {
        GO_VERSION = '`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("1.21")
	}

	pipeline.WriteString(`'
        GOPATH = "${WORKSPACE}"
        GOCACHE = "${WORKSPACE}/.cache"
    }
    
    tools {
        go '`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("1.21")
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
            steps {
                sh 'go mod download'
            }
        }
        
        stage('Build') {
            steps {
                sh 'go build -v ./...'
            }
        }
`)

	if info.HasTests {
		pipeline.WriteString(`
        stage('Test') {
            steps {
                sh 'go test -v -race -coverprofile=coverage.out ./...'
            }
            post {
                always {
                    publishTestResults testResultsPattern: '*.xml'
                    publishCoverage adapters: [goCoberturaAdapter(path: 'coverage.out')], sourceFileResolver: sourceFiles('STORE_LAST_BUILD')
                }
            }
        }
`)
	}

	pipeline.WriteString(`
        stage('Package') {
            steps {
                sh 'go build -o app ./...'
                archiveArtifacts artifacts: 'app', fingerprint: true
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
