package generator

import (
	"strings"

	"github.com/immxrtalbeast/pipeline-gen/internal/analyzer"
)

func generateGitLabJavaScriptPipeline(info *analyzer.ProjectInfo) string {
	var pipeline strings.Builder

	pipeline.WriteString(`stages:
  - install
  - test
  - build
  - deploy

variables:
  NODE_VERSION: '`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("18")
	}

	pipeline.WriteString(`'

cache:
  paths:
    - node_modules/
    - .npm/

install:
  stage: install
  image: node:`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("18")
	}

	pipeline.WriteString(`-alpine
  script:`)

	// Установка зависимостей в зависимости от инструмента сборки
	switch info.BuildTool {
	case "yarn":
		pipeline.WriteString(`
    - yarn install --frozen-lockfile`)
	case "pnpm":
		pipeline.WriteString(`
    - npm install -g pnpm
    - pnpm install --frozen-lockfile`)
	default:
		pipeline.WriteString(`
    - npm ci`)
	}

	pipeline.WriteString(`

`)

	// Lint job если есть ESLint/Prettier
	if containsDependency(info.Dependencies, "eslint") || containsDependency(info.Dependencies, "prettier") {
		pipeline.WriteString(`lint:
  stage: install
  image: node:`)

		if info.Version != "" {
			pipeline.WriteString(info.Version)
		} else {
			pipeline.WriteString("18")
		}

		pipeline.WriteString(`-alpine
  script:`)

		switch info.BuildTool {
		case "yarn":
			pipeline.WriteString(`
    - yarn install --frozen-lockfile
    - yarn lint
    - yarn format:check`)
		case "pnpm":
			pipeline.WriteString(`
    - npm install -g pnpm
    - pnpm install --frozen-lockfile
    - pnpm lint
    - pnpm format:check`)
		default:
			pipeline.WriteString(`
    - npm ci
    - npm run lint
    - npm run format:check`)
		}

		pipeline.WriteString(`

`)
	}

	if info.HasTests {
		pipeline.WriteString(`test:
  stage: test
  image: node:`)

		if info.Version != "" {
			pipeline.WriteString(info.Version)
		} else {
			pipeline.WriteString("18")
		}

		pipeline.WriteString(`-alpine
  script:`)

		switch info.BuildTool {
		case "yarn":
			pipeline.WriteString(`
    - yarn install --frozen-lockfile
    - yarn test --coverage`)
		case "pnpm":
			pipeline.WriteString(`
    - npm install -g pnpm
    - pnpm install --frozen-lockfile
    - pnpm test --coverage`)
		default:
			pipeline.WriteString(`
    - npm ci
    - npm test --coverage`)
		}

		pipeline.WriteString(`
  artifacts:
    reports:
      coverage_report:
        coverage_format: cobertura
        path: coverage/cobertura-coverage.xml
    paths:
      - coverage/
    expire_in: 1 week
  coverage: '/All files[^|]*\|[^|]*\s+([\d\.]+)/'

`)
	}

	pipeline.WriteString(`build:
  stage: build
  image: node:`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("18")
	}

	pipeline.WriteString(`-alpine
  script:`)

	switch info.BuildTool {
	case "yarn":
		pipeline.WriteString(`
    - yarn install --frozen-lockfile
    - yarn build`)
	case "pnpm":
		pipeline.WriteString(`
    - npm install -g pnpm
    - pnpm install --frozen-lockfile
    - pnpm build`)
	default:
		pipeline.WriteString(`
    - npm ci
    - npm run build`)
	}

	pipeline.WriteString(`
  artifacts:
    paths:
      - dist/
      - build/
      - .next/
      - out/
    expire_in: 1 week

deploy:
  stage: deploy
  image: alpine:latest
  script:
    - echo "Deploying JavaScript application"
  only:
    - main
    - master
`)

	return pipeline.String()
}
