package generator

import (
	"strings"

	"github.com/immxrtalbeast/pipeline-gen/internal/analyzer"
)

func generateGitLabCSharpPipeline(info *analyzer.ProjectInfo) string {
	var pipeline strings.Builder

	pipeline.WriteString(`stages:
  - restore
  - build
  - test
  - package
  - deploy

variables:
  DOTNET_VERSION: '`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("8.0")
	}

	pipeline.WriteString(`'
  DOTNET_CLI_TELEMETRY_OPTOUT: 1

cache:
  paths:
    - .nuget/packages/
    - **/bin/
    - **/obj/

restore:
  stage: restore
  image: mcr.microsoft.com/dotnet/sdk:`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("8.0")
	}

	pipeline.WriteString(`-alpine
  script:
    - dotnet restore
  artifacts:
    paths:
      - .nuget/packages/
    expire_in: 1 hour

build:
  stage: build
  image: mcr.microsoft.com/dotnet/sdk:`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("8.0")
	}

	pipeline.WriteString(`-alpine
  script:
    - dotnet build --no-restore --configuration Release
  dependencies:
    - restore
  artifacts:
    paths:
      - **/bin/Release/
    expire_in: 1 hour

`)

	if info.HasTests {
		pipeline.WriteString(`test:
  stage: test
  image: mcr.microsoft.com/dotnet/sdk:`)

		if info.Version != "" {
			pipeline.WriteString(info.Version)
		} else {
			pipeline.WriteString("8.0")
		}

		pipeline.WriteString(`-alpine
  script:
    - dotnet test --no-build --configuration Release --logger trx --collect:"XPlat Code Coverage"
  dependencies:
    - build
  artifacts:
    reports:
      junit:
        - **/TestResults/*.trx
      coverage_report:
        coverage_format: cobertura
        path: **/coverage.cobertura.xml
    paths:
      - **/TestResults/
    expire_in: 1 week
  coverage: '/Total\s*\|\s*(\d+(?:\.\d+)?%)/'

`)
	}

	pipeline.WriteString(`package:
  stage: package
  image: mcr.microsoft.com/dotnet/sdk:`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("8.0")
	}

	pipeline.WriteString(`-alpine
  script:`)

	// Publish для ASP.NET Core если обнаружен
	if containsDependency(info.Dependencies, "web-framework:aspnetcore") {
		pipeline.WriteString(`
    - dotnet publish --no-build --configuration Release -o publish`)
	} else {
		pipeline.WriteString(`
    - dotnet pack --no-build --configuration Release -o packages`)
	}

	pipeline.WriteString(`
  dependencies:
    - build
  artifacts:
    paths:`)

	if containsDependency(info.Dependencies, "web-framework:aspnetcore") {
		pipeline.WriteString(`
      - publish/`)
	} else {
		pipeline.WriteString(`
      - packages/`)
	}

	pipeline.WriteString(`
    expire_in: 1 week

deploy:
  stage: deploy
  image: alpine:latest
  script:
    - echo "Deploying .NET application"
  only:
    - main
    - master
`)

	return pipeline.String()
}
