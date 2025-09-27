package generator

import (
	"strings"

	"github.com/immxrtalbeast/pipeline-gen/internal/analyzer"
)

func generateJenkinsCSharpPipeline(info *analyzer.ProjectInfo) string {
	var pipeline strings.Builder

	pipeline.WriteString(`pipeline {
    agent any
    
    environment {
        DOTNET_VERSION = '`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("8.0")
	}

	pipeline.WriteString(`'
        DOTNET_CLI_TELEMETRY_OPTOUT = '1'
    }
    
    tools {
        dotnet '`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("8.0")
	}

	pipeline.WriteString(`'
    }
    
    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }
        
        stage('Restore') {
            steps {
                sh 'dotnet restore'
            }
        }
        
        stage('Build') {
            steps {
                sh 'dotnet build --no-restore --configuration Release'
            }
        }
`)

	if info.HasTests {
		pipeline.WriteString(`
        stage('Test') {
            steps {
                sh 'dotnet test --no-build --configuration Release --logger trx --collect:"XPlat Code Coverage"'
            }
            post {
                always {
                    publishTestResults testResultsPattern: '**/TestResults/*.trx'
                    publishCoverage adapters: [coberturaAdapter('**/coverage.cobertura.xml')], sourceFileResolver: sourceFiles('STORE_LAST_BUILD')
                }
            }
        }
`)
	}

	pipeline.WriteString(`
        stage('Package') {
            steps {`)

	// Publish для ASP.NET Core если обнаружен
	if containsDependency(info.Dependencies, "web-framework:aspnetcore") {
		pipeline.WriteString(`
                sh 'dotnet publish --no-build --configuration Release -o publish'`)
	} else {
		pipeline.WriteString(`
                sh 'dotnet pack --no-build --configuration Release -o packages'`)
	}

	pipeline.WriteString(`
            }
            post {
                always {
                    archiveArtifacts artifacts: 'publish/**,packages/**', fingerprint: true
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
