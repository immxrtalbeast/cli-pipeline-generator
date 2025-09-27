pipeline {
    agent any
    
    environment {
        GO_VERSION = '1.23.0'
        GOPATH = "${WORKSPACE}"
        GOCACHE = "${WORKSPACE}/.cache"
    }
    
    tools {
        go '1.23.0'
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
}