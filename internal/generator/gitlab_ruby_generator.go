package generator

import (
	"strings"

	"github.com/immxrtalbeast/pipeline-gen/internal/analyzer"
)

func generateGitLabRubyPipeline(info *analyzer.ProjectInfo) string {
	var pipeline strings.Builder

	pipeline.WriteString(`stages:
  - install
  - test
  - build
  - deploy

variables:
  RUBY_VERSION: '`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("2.7")
	}

	pipeline.WriteString(`'

cache:
  paths:
    - vendor/bundle/

install:
  stage: install
  image: ruby:`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("2.7")
	}

	pipeline.WriteString(`-alpine
  script:
    - ruby --version
    - gem install bundler
    - bundle config set --local path 'vendor/bundle'
    - bundle install
  artifacts:
    paths:
      - vendor/bundle/
    expire_in: 1 hour

`)

	// Lint job если есть RuboCop
	if containsDependency(info.Dependencies, "rubocop") {
		pipeline.WriteString(`lint:
  stage: install
  image: ruby:`)

		if info.Version != "" {
			pipeline.WriteString(info.Version)
		} else {
			pipeline.WriteString("2.7")
		}

		pipeline.WriteString(`-alpine
  script:
    - gem install bundler
    - bundle config set --local path 'vendor/bundle'
    - bundle install
    - bundle exec rubocop
  dependencies:
    - install

`)
	}

	if info.HasTests {
		pipeline.WriteString(`test:
  stage: test
  image: ruby:`)

		if info.Version != "" {
			pipeline.WriteString(info.Version)
		} else {
			pipeline.WriteString("2.7")
		}

		pipeline.WriteString(`-alpine
  script:
    - gem install bundler
    - bundle config set --local path 'vendor/bundle'
    - bundle install
    - `)

		// Запуск тестов в зависимости от фреймворка
		switch info.TestFramework {
		case "rspec":
			pipeline.WriteString("bundle exec rspec --format documentation")
		case "minitest":
			pipeline.WriteString("bundle exec rake test")
		case "test-unit":
			pipeline.WriteString("bundle exec rake test")
		case "cucumber":
			pipeline.WriteString("bundle exec cucumber")
		default:
			pipeline.WriteString("bundle exec rake test")
		}

		pipeline.WriteString(`
  dependencies:
    - install
  artifacts:
    reports:
      junit:
        - spec/reports/rspec.xml
    paths:
      - coverage/
    expire_in: 1 week
  coverage: '/\(\d+\.\d+%\)/'

`)
	}

	pipeline.WriteString(`build:
  stage: build
  image: ruby:`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("2.7")
	}

	pipeline.WriteString(`-alpine
  script:
    - gem install bundler
    - bundle config set --local path 'vendor/bundle'
    - bundle install`)

	// Сборка в зависимости от типа приложения
	if containsDependency(info.Dependencies, "web-framework:rails") {
		pipeline.WriteString(`
    - bundle exec rails assets:precompile
    - bundle exec rails db:create db:migrate`)
	} else if containsDependency(info.Dependencies, "gem") {
		pipeline.WriteString(`
    - gem build *.gemspec`)
	} else {
		pipeline.WriteString(`
    - bundle exec rake build`)
	}

	pipeline.WriteString(`
  dependencies:
    - install
  artifacts:
    paths:`)

	// Определяем пути для артефактов в зависимости от типа приложения
	if containsDependency(info.Dependencies, "web-framework:rails") {
		pipeline.WriteString(`
      - public/assets/
      - tmp/cache/
      - log/`)
	} else if containsDependency(info.Dependencies, "gem") {
		pipeline.WriteString(`
      - *.gem
      - pkg/`)
	} else {
		pipeline.WriteString(`
      - dist/
      - build/`)
	}

	pipeline.WriteString(`
    expire_in: 1 week

deploy:
  stage: deploy
  image: alpine:latest
  script:
    - echo "Deploying Ruby application"
  only:
    - main
    - master
`)

	return pipeline.String()
}
