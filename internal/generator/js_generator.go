package generator

import (
	"fmt"
	"strings"

	"github.com/immxrtalbeast/pipeline-gen/internal/analyzer"
)

func generateJavaScriptPipeline(info *analyzer.ProjectInfo) string {
	var pipeline strings.Builder

	pipeline.WriteString(`name: JavaScript/Node.js CI/CD Pipeline

on:
  push:
    branches: [ main, master, develop ]
  pull_request:
    branches: [ main, master, develop ]

jobs:
`)

	// Job для установки зависимостей и кеширования
	pipeline.WriteString(`  install:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - name: Setup Node.js
      uses: actions/setup-node@v3
      with:
        node-version: '`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("18")
	}

	pipeline.WriteString(`'
        cache: '`)

	// Определяем тип кеша в зависимости от инструмента сборки
	switch info.BuildTool {
	case "yarn":
		pipeline.WriteString("yarn")
	case "pnpm":
		pipeline.WriteString("pnpm")
	default:
		pipeline.WriteString("npm")
	}

	pipeline.WriteString(`'
`)

	// Установка зависимостей в зависимости от инструмента
	switch info.BuildTool {
	case "yarn":
		pipeline.WriteString(`    - name: Install dependencies with Yarn
      run: yarn install --frozen-lockfile
`)
	case "pnpm":
		pipeline.WriteString(`    - name: Install pnpm
      uses: pnpm/action-setup@v2
      with:
        version: latest
    - name: Install dependencies with pnpm
      run: pnpm install --frozen-lockfile
`)
	default:
		pipeline.WriteString(`    - name: Install dependencies with npm
      run: npm ci
`)
	}

	// Job для линтинга
	if containsDependency(info.Dependencies, "eslint") || containsDependency(info.Dependencies, "prettier") {
		pipeline.WriteString(`  lint:
    runs-on: ubuntu-latest
    needs: install
    steps:
    - uses: actions/checkout@v3
    - name: Setup Node.js
      uses: actions/setup-node@v3
      with:
        node-version: '`)

		if info.Version != "" {
			pipeline.WriteString(info.Version)
		} else {
			pipeline.WriteString("18")
		}

		pipeline.WriteString(`'
        cache: '`)

		switch info.BuildTool {
		case "yarn":
			pipeline.WriteString("yarn")
		case "pnpm":
			pipeline.WriteString("pnpm")
		default:
			pipeline.WriteString("npm")
		}

		pipeline.WriteString(`'
`)

		// Установка зависимостей для линтинга
		switch info.BuildTool {
		case "yarn":
			pipeline.WriteString(`    - name: Install dependencies
      run: yarn install --frozen-lockfile
`)
		case "pnpm":
			pipeline.WriteString(`    - name: Install pnpm
      uses: pnpm/action-setup@v2
      with:
        version: latest
    - name: Install dependencies
      run: pnpm install --frozen-lockfile
`)
		default:
			pipeline.WriteString(`    - name: Install dependencies
      run: npm ci
`)
		}

		pipeline.WriteString(`    - name: Run ESLint
      run: `)

		switch info.BuildTool {
		case "yarn":
			pipeline.WriteString("yarn lint")
		case "pnpm":
			pipeline.WriteString("pnpm lint")
		default:
			pipeline.WriteString("npm run lint")
		}

		pipeline.WriteString(`
    - name: Run Prettier check
      run: `)

		switch info.BuildTool {
		case "yarn":
			pipeline.WriteString("yarn format:check")
		case "pnpm":
			pipeline.WriteString("pnpm format:check")
		default:
			pipeline.WriteString("npm run format:check")
		}

		pipeline.WriteString(`
`)
	}

	// Job для тестов
	if info.HasTests {
		pipeline.WriteString(`  test:
    runs-on: ubuntu-latest
    needs: install
    strategy:
      matrix:
        node-version: [`)

		// Добавляем версии Node.js
		if info.Version != "" && info.Version != "18" {
			pipeline.WriteString(fmt.Sprintf(" '%s', '16', '18' ", info.Version))
		} else {
			pipeline.WriteString(" '16', '18', '20' ")
		}

		pipeline.WriteString(`]
    steps:
    - uses: actions/checkout@v3
    - name: Setup Node.js ${{ matrix.node-version }}
      uses: actions/setup-node@v3
      with:
        node-version: ${{ matrix.node-version }}
        cache: '`)

		switch info.BuildTool {
		case "yarn":
			pipeline.WriteString("yarn")
		case "pnpm":
			pipeline.WriteString("pnpm")
		default:
			pipeline.WriteString("npm")
		}

		pipeline.WriteString(`'
`)

		// Установка зависимостей для тестов
		switch info.BuildTool {
		case "yarn":
			pipeline.WriteString(`    - name: Install dependencies
      run: yarn install --frozen-lockfile
`)
		case "pnpm":
			pipeline.WriteString(`    - name: Install pnpm
      uses: pnpm/action-setup@v2
      with:
        version: latest
    - name: Install dependencies
      run: pnpm install --frozen-lockfile
`)
		default:
			pipeline.WriteString(`    - name: Install dependencies
      run: npm ci
`)
		}

		// Запуск тестов в зависимости от фреймворка
		pipeline.WriteString(`    - name: Run tests
      run: `)

		switch info.TestFramework {
		case "jest":
			switch info.BuildTool {
			case "yarn":
				pipeline.WriteString("yarn test")
			case "pnpm":
				pipeline.WriteString("pnpm test")
			default:
				pipeline.WriteString("npm test")
			}
		case "vitest":
			switch info.BuildTool {
			case "yarn":
				pipeline.WriteString("yarn test:vitest")
			case "pnpm":
				pipeline.WriteString("pnpm test:vitest")
			default:
				pipeline.WriteString("npm run test:vitest")
			}
		case "mocha":
			switch info.BuildTool {
			case "yarn":
				pipeline.WriteString("yarn test:mocha")
			case "pnpm":
				pipeline.WriteString("pnpm test:mocha")
			default:
				pipeline.WriteString("npm run test:mocha")
			}
		case "cypress":
			switch info.BuildTool {
			case "yarn":
				pipeline.WriteString("yarn cypress run")
			case "pnpm":
				pipeline.WriteString("pnpm cypress run")
			default:
				pipeline.WriteString("npm run cypress:run")
			}
		case "playwright":
			switch info.BuildTool {
			case "yarn":
				pipeline.WriteString("yarn playwright test")
			case "pnpm":
				pipeline.WriteString("pnpm playwright test")
			default:
				pipeline.WriteString("npm run test:playwright")
			}
		default:
			switch info.BuildTool {
			case "yarn":
				pipeline.WriteString("yarn test")
			case "pnpm":
				pipeline.WriteString("pnpm test")
			default:
				pipeline.WriteString("npm test")
			}
		}

		pipeline.WriteString(`
`)

		// Добавляем отчет о покрытии для Jest
		if info.TestFramework == "jest" {
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
    - name: Setup Node.js
      uses: actions/setup-node@v3
      with:
        node-version: '`)

		if info.Version != "" {
			pipeline.WriteString(info.Version)
		} else {
			pipeline.WriteString("18")
		}

		pipeline.WriteString(`'
        cache: '`)

		switch info.BuildTool {
		case "yarn":
			pipeline.WriteString("yarn")
		case "pnpm":
			pipeline.WriteString("pnpm")
		default:
			pipeline.WriteString("npm")
		}

		pipeline.WriteString(`'
`)

		switch info.BuildTool {
		case "yarn":
			pipeline.WriteString(`    - name: Install dependencies
      run: yarn install --frozen-lockfile
    - name: Verify build
      run: yarn build
`)
		case "pnpm":
			pipeline.WriteString(`    - name: Install pnpm
      uses: pnpm/action-setup@v2
      with:
        version: latest
    - name: Install dependencies
      run: pnpm install --frozen-lockfile
    - name: Verify build
      run: pnpm build
`)
		default:
			pipeline.WriteString(`    - name: Install dependencies
      run: npm ci
    - name: Verify build
      run: npm run build
`)
		}
	}

	// Job для сборки
	previousJob := "test"
	if !info.HasTests {
		previousJob = "verify"
	}
	if containsDependency(info.Dependencies, "eslint") || containsDependency(info.Dependencies, "prettier") {
		previousJob = "lint"
	}

	pipeline.WriteString(fmt.Sprintf(`  build:
    runs-on: ubuntu-latest
    needs: %s
    steps:
    - uses: actions/checkout@v3
    - name: Setup Node.js
      uses: actions/setup-node@v3
      with:
        node-version: '`, previousJob))

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("18")
	}

	pipeline.WriteString(`'
        cache: '`)

	switch info.BuildTool {
	case "yarn":
		pipeline.WriteString("yarn")
	case "pnpm":
		pipeline.WriteString("pnpm")
	default:
		pipeline.WriteString("npm")
	}

	pipeline.WriteString(`'
`)

	// Установка зависимостей для сборки
	switch info.BuildTool {
	case "yarn":
		pipeline.WriteString(`    - name: Install dependencies
      run: yarn install --frozen-lockfile
`)
	case "pnpm":
		pipeline.WriteString(`    - name: Install pnpm
      uses: pnpm/action-setup@v2
      with:
        version: latest
    - name: Install dependencies
      run: pnpm install --frozen-lockfile
`)
	default:
		pipeline.WriteString(`    - name: Install dependencies
      run: npm ci
`)
	}

	// Сборка в зависимости от инструмента сборки
	pipeline.WriteString(`    - name: Build application
      run: `)

	switch info.BuildTool {
	case "yarn":
		pipeline.WriteString("yarn build")
	case "pnpm":
		pipeline.WriteString("pnpm build")
	default:
		pipeline.WriteString("npm run build")
	}

	pipeline.WriteString(`
    - name: Upload build artifacts
      uses: actions/upload-artifact@v3
      with:
        name: build-files
        path: |`)

	// Определяем пути для артефактов в зависимости от типа приложения
	if containsDependency(info.Dependencies, "frontend-framework") {
		pipeline.WriteString(`
          dist/
          build/
          .next/
          out/`)
	} else if containsDependency(info.Dependencies, "backend-framework") {
		pipeline.WriteString(`
          dist/
          build/
          lib/`)
	} else {
		pipeline.WriteString(`
          dist/
          build/`)
	}

	pipeline.WriteString(`
`)

	return pipeline.String()
}