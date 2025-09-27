package generator

import (
	"fmt"
	"strings"

	"github.com/immxrtalbeast/pipeline-gen/internal/analyzer"
)

func generateRubyPipeline(info *analyzer.ProjectInfo) string {
	var pipeline strings.Builder

	pipeline.WriteString(`name: Ruby CI/CD Pipeline

on:
  push:
    branches: [ main, master, develop ]
  pull_request:
    branches: [ main, master, develop ]

jobs:
`)

	// Job для установки зависимостей
	pipeline.WriteString(`  install:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - name: Setup Ruby
      uses: ruby/setup-ruby@v1
      with:
        ruby-version: '`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("2.7")
	}

	pipeline.WriteString(`'
        bundler-cache: true
`)

	// Установка зависимостей в зависимости от инструмента сборки
	switch info.BuildTool {
	case "bundler":
		pipeline.WriteString(`    - name: Install dependencies
      run: bundle install
`)
	case "rake":
		pipeline.WriteString(`    - name: Install dependencies
      run: |
        gem install bundler
        bundle install
`)
	default:
		pipeline.WriteString(`    - name: Install dependencies
      run: |
        gem install bundler
        bundle install
`)
	}

	// Job для линтинга (если есть RuboCop)
	if containsDependency(info.Dependencies, "rubocop") {
		pipeline.WriteString(`  lint:
    runs-on: ubuntu-latest
    needs: install
    steps:
    - uses: actions/checkout@v3
    - name: Setup Ruby
      uses: ruby/setup-ruby@v1
      with:
        ruby-version: '`)

		if info.Version != "" {
			pipeline.WriteString(info.Version)
		} else {
			pipeline.WriteString("2.7")
		}

		pipeline.WriteString(`'
        bundler-cache: true
    - name: Install dependencies
      run: bundle install
    - name: Run RuboCop
      run: bundle exec rubocop
`)
	}

	// Job для тестов
	if info.HasTests {
		pipeline.WriteString(`  test:
    runs-on: ubuntu-latest
    needs: install
    strategy:
      matrix:
        ruby-version: [`)

		// Добавляем версии Ruby
		if info.Version != "" && info.Version != "2.7" {
			pipeline.WriteString(fmt.Sprintf(" '%s', '2.6', '2.7', '3.0' ", info.Version))
		} else {
			pipeline.WriteString(" '2.6', '2.7', '3.0' ")
		}

		pipeline.WriteString(`]
    steps:
    - uses: actions/checkout@v3
    - name: Setup Ruby ${{ matrix.ruby-version }}
      uses: ruby/setup-ruby@v1
      with:
        ruby-version: ${{ matrix.ruby-version }}
        bundler-cache: true
    - name: Install dependencies
      run: bundle install
    - name: Run tests
      run: `)

		// Запуск тестов в зависимости от фреймворка
		switch info.TestFramework {
		case "rspec":
			pipeline.WriteString("bundle exec rspec")
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
`)

		// Добавляем отчет о покрытии для RSpec
		if info.TestFramework == "rspec" {
			pipeline.WriteString(`    - name: Upload coverage reports
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage/lcov.info
        flags: unittests
        name: codecov-umbrella
`)
		}

	} else {
		// Если тестов нет - простая проверка
		pipeline.WriteString(`  verify:
    runs-on: ubuntu-latest
    needs: install
    steps:
    - uses: actions/checkout@v3
    - name: Setup Ruby
      uses: ruby/setup-ruby@v1
      with:
        ruby-version: '`)

		if info.Version != "" {
			pipeline.WriteString(info.Version)
		} else {
			pipeline.WriteString("2.7")
		}

		pipeline.WriteString(`'
        bundler-cache: true
    - name: Install dependencies
      run: bundle install
    - name: Verify syntax
      run: |
        find . -name "*.rb" -exec ruby -c {} \;
`)

		// Проверка сборки для Rails приложений
		if containsDependency(info.Dependencies, "web-framework:rails") {
			pipeline.WriteString(`    - name: Verify Rails app
      run: |
        bundle exec rails db:create db:migrate
        bundle exec rails assets:precompile
`)
		}

		pipeline.WriteString(`
`)
	}

	// Job для сборки
	previousJob := "test"
	if !info.HasTests {
		previousJob = "verify"
	}
	if containsDependency(info.Dependencies, "rubocop") {
		previousJob = "lint"
	}

	pipeline.WriteString(fmt.Sprintf(`  build:
    runs-on: ubuntu-latest
    needs: %s
    steps:
    - uses: actions/checkout@v3
    - name: Setup Ruby
      uses: ruby/setup-ruby@v1
      with:
        ruby-version: '`, previousJob))

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("2.7")
	}

	pipeline.WriteString(`'
        bundler-cache: true
    - name: Install dependencies
      run: bundle install
`)

	// Сборка в зависимости от типа приложения
	if containsDependency(info.Dependencies, "web-framework:rails") {
		pipeline.WriteString(`    - name: Build Rails application
      run: |
        bundle exec rails assets:precompile
        bundle exec rails db:create db:migrate
`)
	} else if containsDependency(info.Dependencies, "gem") {
		pipeline.WriteString(`    - name: Build gem
      run: |
        gem build *.gemspec
`)
	} else {
		pipeline.WriteString(`    - name: Build application
      run: bundle exec rake build
`)
	}

	pipeline.WriteString(`    - name: Upload build artifacts
      uses: actions/upload-artifact@v3
      with:
        name: build-files
        path: |`)

	// Определяем пути для артефактов в зависимости от типа приложения
	if containsDependency(info.Dependencies, "web-framework:rails") {
		pipeline.WriteString(`
          public/assets/
          tmp/cache/
          log/`)
	} else if containsDependency(info.Dependencies, "gem") {
		pipeline.WriteString(`
          *.gem
          pkg/`)
	} else {
		pipeline.WriteString(`
          dist/
          build/`)
	}

	pipeline.WriteString(`
`)

	return pipeline.String()
}