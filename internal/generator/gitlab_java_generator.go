package generator

import (
	"strings"

	"github.com/immxrtalbeast/pipeline-gen/internal/analyzer"
)

func generateGitLabJavaPipeline(info *analyzer.ProjectInfo) string {
	var pipeline strings.Builder

	pipeline.WriteString(`stages:
  - build
  - test
  - package
  - deploy

variables:
  MAVEN_OPTS: "-Dmaven.repo.local=$CI_PROJECT_DIR/.m2/repository"
  MAVEN_CLI_OPTS: "--batch-mode --errors --fail-at-end --show-version"

cache:
  paths:
    - .m2/repository/
    - target/

`)

	// Build job
	pipeline.WriteString(`build:
  stage: build
  image: maven:`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("11")
	}

	pipeline.WriteString(`-openjdk
  script:`)

	if info.BuildTool == "gradle" {
		pipeline.WriteString(`
    - ./gradlew build`)
	} else {
		pipeline.WriteString(`
    - mvn $MAVEN_CLI_OPTS compile`)
	}

	pipeline.WriteString(`
  artifacts:
    paths:
      - target/
    expire_in: 1 hour

`)

	if info.HasTests {
		pipeline.WriteString(`test:
  stage: test
  image: maven:`)

		if info.Version != "" {
			pipeline.WriteString(info.Version)
		} else {
			pipeline.WriteString("11")
		}

		pipeline.WriteString(`-openjdk
  script:`)

		if info.BuildTool == "gradle" {
			pipeline.WriteString(`
    - ./gradlew test
    - ./gradlew jacocoTestReport`)
		} else {
			pipeline.WriteString(`
    - mvn $MAVEN_CLI_OPTS test
    - mvn $MAVEN_CLI_OPTS jacoco:report`)
		}

		pipeline.WriteString(`
  artifacts:
    reports:
      junit:
        - target/surefire-reports/TEST-*.xml
      coverage_report:
        coverage_format: cobertura
        path: target/site/jacoco/jacoco.xml
    paths:
      - target/site/jacoco/
    expire_in: 1 week
  coverage: '/Total.*?(\d+%)/'

`)
	}

	pipeline.WriteString(`package:
  stage: package
  image: maven:`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("11")
	}

	pipeline.WriteString(`-openjdk
  script:`)

	if info.BuildTool == "gradle" {
		pipeline.WriteString(`
    - ./gradlew build
    - ./gradlew bootJar`)
	} else {
		pipeline.WriteString(`
    - mvn $MAVEN_CLI_OPTS package -DskipTests`)
	}

	pipeline.WriteString(`
  artifacts:
    paths:
      - target/*.jar
      - build/libs/*.jar
    expire_in: 1 week

deploy:
  stage: deploy
  image: alpine:latest
  script:
    - echo "Deploying Java application"
  only:
    - main
    - master
`)

	return pipeline.String()
}
