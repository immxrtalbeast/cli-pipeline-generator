package generator

import (
	"strings"

	"github.com/immxrtalbeast/pipeline-gen/internal/analyzer"
)

func generateGitLabPythonPipeline(info *analyzer.ProjectInfo) string {
	var pipeline strings.Builder

	pipeline.WriteString(`stages:
  - install
  - test
  - build
  - deploy

variables:
  PYTHON_VERSION: '`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("3.9")
	}

	pipeline.WriteString(`'

install:
  stage: install
  image: python:`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("3.9")
	}

	pipeline.WriteString(`-alpine
  script:
    - python --version
    - pip install --upgrade pip`)

	// Установка зависимостей в зависимости от инструмента сборки
	switch info.BuildTool {
	case "poetry":
		pipeline.WriteString(`
    - pip install poetry
    - poetry install`)
	case "pipenv":
		pipeline.WriteString(`
    - pip install pipenv
    - pipenv install`)
	default:
		pipeline.WriteString(`
    - pip install -r requirements.txt`)
	}

	pipeline.WriteString(`
  cache:
    paths:
      - .venv/
      - venv/
      - __pycache__/

`)

	if info.HasTests {
		pipeline.WriteString(`test:
  stage: test
  image: python:`)

		if info.Version != "" {
			pipeline.WriteString(info.Version)
		} else {
			pipeline.WriteString("3.9")
		}

		pipeline.WriteString(`-alpine
  script:`)

		switch info.BuildTool {
		case "poetry":
			pipeline.WriteString(`
    - pip install poetry
    - poetry install
    - poetry run pytest --cov=. --cov-report=xml --cov-report=html`)
		case "pipenv":
			pipeline.WriteString(`
    - pip install pipenv
    - pipenv install
    - pipenv run pytest --cov=. --cov-report=xml --cov-report=html`)
		default:
			pipeline.WriteString(`
    - pip install -r requirements.txt
    - pytest --cov=. --cov-report=xml --cov-report=html`)
		}

		pipeline.WriteString(`
  artifacts:
    reports:
      coverage_report:
        coverage_format: cobertura
        path: coverage.xml
    paths:
      - coverage.xml
      - htmlcov/
    expire_in: 1 week
  coverage: '/TOTAL.*\s+(\d+%)$/'

`)
	}

	pipeline.WriteString(`build:
  stage: build
  image: python:`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("3.9")
	}

	pipeline.WriteString(`-alpine
  script:`)

	switch info.BuildTool {
	case "poetry":
		pipeline.WriteString(`
    - pip install poetry
    - poetry install
    - poetry build`)
	case "pipenv":
		pipeline.WriteString(`
    - pip install pipenv
    - pipenv install
    - pipenv run python setup.py sdist bdist_wheel`)
	default:
		pipeline.WriteString(`
    - pip install -r requirements.txt
    - python setup.py sdist bdist_wheel`)
	}

	pipeline.WriteString(`
  artifacts:
    paths:
      - dist/
    expire_in: 1 week

deploy:
  stage: deploy
  image: alpine:latest
  script:
    - echo "Deploying Python application"
  only:
    - main
    - master
`)

	return pipeline.String()
}
