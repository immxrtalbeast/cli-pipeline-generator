package generator

import (
	"fmt"
	"strings"

	"github.com/immxrtalbeast/pipeline-gen/internal/analyzer"
)

func generatePythonPipeline(info *analyzer.ProjectInfo) string {
	var pipeline strings.Builder

	pipeline.WriteString(`name: Python CI/CD Pipeline

on:
  push:
    branches: [ main, master, develop ]
  pull_request:
    branches: [ main, master, develop ]

jobs:
`)

	// Job для тестов или проверки
	if info.HasTests {
		pipeline.WriteString(`  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        python-version: [`)

		// Добавляем версии Python
		if info.Version != "" && info.Version != "3.9" {
			pipeline.WriteString(fmt.Sprintf(" '%s', '3.9' ", info.Version))
		} else {
			pipeline.WriteString(" '3.8', '3.9', '3.10' ")
		}

		pipeline.WriteString(`]
    steps:
    - uses: actions/checkout@v3
    - name: Set up Python ${{ matrix.python-version }}
      uses: actions/setup-python@v3
      with:
        python-version: ${{ matrix.python-version }}
    - name: Install dependencies
`)

		// Выбираем способ установки зависимостей
		switch info.BuildTool {
		case "poetry":
			pipeline.WriteString(`      run: |
        pip install poetry
        poetry install
`)
		case "pipenv":
			pipeline.WriteString(`      run: |
        pip install pipenv
        pipenv install --dev
`)
		default:
			pipeline.WriteString(`      run: |
        python -m pip install --upgrade pip
        pip install -r requirements.txt
`)
		}

		// Добавляем запуск тестов
		pipeline.WriteString(`    - name: Run tests
`)
		switch info.TestFramework {
		case "pytest":
			pipeline.WriteString(`      run: pytest
`)
		default:
			pipeline.WriteString(`      run: python -m unittest discover
`)
		}

	} else {
		// Если тестов нет - простая проверка
		pipeline.WriteString(`  verify:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - name: Set up Python
      uses: actions/setup-python@v3
      with:
        python-version: '`)

		if info.Version != "" {
			pipeline.WriteString(info.Version)
		} else {
			pipeline.WriteString("3.9")
		}

		pipeline.WriteString(`'
    - name: Install dependencies
`)

		switch info.BuildTool {
		case "poetry":
			pipeline.WriteString(`      run: |
        pip install poetry
        poetry install
`)
		case "pipenv":
			pipeline.WriteString(`      run: |
        pip install pipenv
        pipenv install
`)
		default:
			pipeline.WriteString(`      run: |
        python -m pip install --upgrade pip
        if [ -f requirements.txt ]; then pip install -r requirements.txt; fi
`)
		}

		pipeline.WriteString(`    - name: Verify imports
      run: python -c "import sys; print('Python path:', sys.path)"
`)
	}

	if info.BuildTool == "poetry" || info.BuildTool == "setuptools" {
		previousJob := "test"
		if !info.HasTests {
			previousJob = "verify"
		}

		pipeline.WriteString(fmt.Sprintf(`  build:
    runs-on: ubuntu-latest
    needs: %s
    steps:
    - uses: actions/checkout@v3
    - name: Set up Python
      uses: actions/setup-python@v3
      with:
        python-version: '`, previousJob))

		if info.Version != "" {
			pipeline.WriteString(info.Version)
		} else {
			pipeline.WriteString("3.9")
		}

		pipeline.WriteString(`'
    - name: Build package
`)

		switch info.BuildTool {
		case "poetry":
			pipeline.WriteString(`      run: |
        pip install poetry
        poetry build
`)
		case "setuptools":
			pipeline.WriteString(`      run: |
        pip install setuptools wheel
        python setup.py sdist bdist_wheel
`)
		default:
			pipeline.WriteString(`      run: |
        echo "No specific build step configured"
`)
		}

		pipeline.WriteString(`    - name: Upload package
      uses: actions/upload-artifact@v3
      with:
        name: python-package
        path: dist/
`)
	}

	// Job для линтинга (если есть зависимости для линтинга)
	if containsDependency(info.Dependencies, "web-framework") {
		pipeline.WriteString(`  lint:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - name: Set up Python
      uses: actions/setup-python@v3
    - name: Install linters
      run: |
        pip install flake8 black mypy
    - name: Run linters
      run: |
        flake8 .
        black --check .
        mypy .
`)
	}

	return pipeline.String()
}
