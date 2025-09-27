package generator

import (
	"strings"

	"github.com/immxrtalbeast/pipeline-gen/internal/analyzer"
)

func generateJenkinsRubyPipeline(info *analyzer.ProjectInfo) string {
	var pipeline strings.Builder

	pipeline.WriteString(`pipeline {
    agent any
    
    environment {
        RUBY_VERSION = '`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("2.7")
	}

	pipeline.WriteString(`'
    }
    
    tools {
        ruby '`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("2.7")
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
                sh 'ruby --version'
                sh 'gem install bundler'
            }
        }
        
        stage('Dependencies') {
            steps {
                sh 'bundle config set --local path vendor/bundle'
                sh 'bundle install'
            }
        }
`)

	// Lint job если есть RuboCop
	if containsDependency(info.Dependencies, "rubocop") {
		pipeline.WriteString(`
        stage('Lint') {
            steps {
                sh 'bundle exec rubocop'
            }
        }
`)
	}

	if info.HasTests {
		pipeline.WriteString(`
        stage('Test') {
            steps {`)

		// Запуск тестов в зависимости от фреймворка
		switch info.TestFramework {
		case "rspec":
			pipeline.WriteString(`
                sh 'bundle exec rspec --format documentation'`)
		case "minitest":
			pipeline.WriteString(`
                sh 'bundle exec rake test'`)
		case "test-unit":
			pipeline.WriteString(`
                sh 'bundle exec rake test'`)
		case "cucumber":
			pipeline.WriteString(`
                sh 'bundle exec cucumber'`)
		default:
			pipeline.WriteString(`
                sh 'bundle exec rake test'`)
		}

		pipeline.WriteString(`
            }
            post {
                always {
                    publishTestResults testResultsPattern: 'spec/reports/*.xml'
                }
            }
        }
`)
	}

	pipeline.WriteString(`
        stage('Build') {
            steps {`)

	// Сборка в зависимости от типа приложения
	if containsDependency(info.Dependencies, "web-framework:rails") {
		pipeline.WriteString(`
                sh 'bundle exec rails assets:precompile'
                sh 'bundle exec rails db:create db:migrate'`)
	} else if containsDependency(info.Dependencies, "gem") {
		pipeline.WriteString(`
                sh 'gem build *.gemspec'`)
	} else {
		pipeline.WriteString(`
                sh 'bundle exec rake build'`)
	}

	pipeline.WriteString(`
            }
            post {
                always {
                    archiveArtifacts artifacts: 'public/assets/**,tmp/cache/**,*.gem,pkg/**', fingerprint: true
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
