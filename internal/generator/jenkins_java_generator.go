package generator

import (
	"strings"

	"github.com/immxrtalbeast/pipeline-gen/internal/analyzer"
)

func generateJenkinsJavaPipeline(info *analyzer.ProjectInfo) string {
	var pipeline strings.Builder

	pipeline.WriteString(`pipeline {
    agent any
    
    environment {
        JAVA_VERSION = '`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("11")
	}

	pipeline.WriteString(`'
        MAVEN_OPTS = '-Dmaven.repo.local=.m2/repository'
    }
    
    tools {
        maven 'Maven 3.8.6'
        jdk '`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("11")
	}

	pipeline.WriteString(`'
    }
    
    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }
        
        stage('Build') {
            steps {`)

	if info.BuildTool == "gradle" {
		pipeline.WriteString(`
                sh './gradlew build'`)
	} else {
		pipeline.WriteString(`
                sh 'mvn clean compile'`)
	}

	pipeline.WriteString(`
            }
        }
`)

	if info.HasTests {
		pipeline.WriteString(`
        stage('Test') {
            steps {`)

		if info.BuildTool == "gradle" {
			pipeline.WriteString(`
                sh './gradlew test'
                sh './gradlew jacocoTestReport'`)
		} else {
			pipeline.WriteString(`
                sh 'mvn test'
                sh 'mvn jacoco:report'`)
		}

		pipeline.WriteString(`
            }
            post {
                always {
                    publishTestResults testResultsPattern: 'target/surefire-reports/*.xml'
                    publishCoverage adapters: [jacocoAdapter('target/site/jacoco/jacoco.xml')], sourceFileResolver: sourceFiles('STORE_LAST_BUILD')
                }
            }
        }
`)
	}

	pipeline.WriteString(`
        stage('Package') {
            steps {`)

	if info.BuildTool == "gradle" {
		pipeline.WriteString(`
                sh './gradlew bootJar'`)
	} else {
		pipeline.WriteString(`
                sh 'mvn package -DskipTests'`)
	}

	pipeline.WriteString(`
            }
            post {
                always {
                    archiveArtifacts artifacts: 'target/*.jar', fingerprint: true
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
