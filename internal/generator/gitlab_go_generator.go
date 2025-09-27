package generator

import (
	"strings"

	"github.com/immxrtalbeast/pipeline-gen/internal/analyzer"
)

func generateGitLabGoPipeline(info *analyzer.ProjectInfo) string {
	var pipeline strings.Builder

	pipeline.WriteString(`stages:
  - build
  - test
  - deploy

variables:
  GO_VERSION: '`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("1.21")
	}

	pipeline.WriteString(`'

build:
  stage: build
  image: golang:`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("1.21")
	}

	pipeline.WriteString(`-alpine
  script:
    - go mod download
    - go build -v ./...
  artifacts:
    paths:
      - "*.exe"
      - "*.bin"
    expire_in: 1 hour

`)

	if info.HasTests {
		pipeline.WriteString(`test:
  stage: test
  image: golang:`)

		if info.Version != "" {
			pipeline.WriteString(info.Version)
		} else {
			pipeline.WriteString("1.21")
		}

		pipeline.WriteString(`-alpine
  script:
    - go mod download
    - go test -v -race -coverprofile=coverage.out ./...
    - go tool cover -html=coverage.out -o coverage.html
  artifacts:
    reports:
      coverage_report:
        coverage_format: cobertura
        path: coverage.xml
    paths:
      - coverage.out
      - coverage.html
    expire_in: 1 week
  coverage: '/coverage: \d+\.\d+%/'

`)
	}

	pipeline.WriteString(`deploy:
  stage: deploy
  image: alpine:latest
  script:
    - echo "Deploying application"
  only:
    - main
    - master
`)

	return pipeline.String()
}
