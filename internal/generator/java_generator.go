package generator

import (
	"fmt"
	"strings"

	"github.com/immxrtalbeast/pipeline-gen/internal/analyzer"
)

func generateJavaPipeline(info *analyzer.ProjectInfo) string {
	var pipeline strings.Builder

	// Определяем, используем ли мы Gradle или Maven
	buildTool := "gradle"
	if info.BuildTool == "maven" {
		buildTool = "maven"
	}

	pipeline.WriteString(fmt.Sprintf(`name: Java CI/CD Pipeline (%s)

on:
  push:
    branches: [ main, master, develop ]
  pull_request:
    branches: [ main, master, develop ]

jobs:
`, strings.Title(buildTool)))

	// Job для тестов или проверки
	if info.HasTests {
		pipeline.WriteString(`  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        java-version: [`)

		// Добавляем версии Java
		if info.Version != "" && info.Version != "11" {
			pipeline.WriteString(fmt.Sprintf(" '%s', '11', '17' ", info.Version))
		} else {
			pipeline.WriteString(" '8', '11', '17' ")
		}

		pipeline.WriteString(`]
    steps:
    - uses: actions/checkout@v3
    - name: Set up JDK ${{ matrix.java-version }}
      uses: actions/setup-java@v3
      with:
        java-version: ${{ matrix.java-version }}
        distribution: 'temurin'
`)

		// Для Gradle проектов
		if buildTool == "gradle" {
			pipeline.WriteString(`    - name: Setup Gradle
      uses: gradle/actions/setup-gradle@v3
      
    - name: Grant execute permission for gradlew
      run: chmod +x gradlew
      
    - name: Run tests with Gradle
      run: ./gradlew test
`)

			// Для многомодульных проектов
			if len(info.Modules) > 0 {
				pipeline.WriteString(`    - name: Run tests for all modules
      run: ./gradlew test
`)
			}
		} else {
			// Для Maven проектов
			pipeline.WriteString(`    - name: Run tests with Maven
      run: mvn test
`)
		}

		// Добавляем отчет о покрытии если есть зависимости для этого
		if containsDependency(info.Dependencies, "jacoco") || containsDependency(info.Dependencies, "coverage") {
			pipeline.WriteString(`    - name: Upload coverage reports
      uses: codecov/codecov-action@v3
`)
		}

	} else {
		// Если тестов нет - простая проверка сборки
		pipeline.WriteString(`  verify:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - name: Set up JDK
      uses: actions/setup-java@v3
      with:
        java-version: '`)

		if info.Version != "" {
			pipeline.WriteString(info.Version)
		} else {
			pipeline.WriteString("11")
		}

		pipeline.WriteString(`'
        distribution: 'temurin'
`)

		if buildTool == "gradle" {
			pipeline.WriteString(`    - name: Setup Gradle
      uses: gradle/actions/setup-gradle@v3
      
    - name: Grant execute permission for gradlew
      run: chmod +x gradlew
      
    - name: Verify build
      run: ./gradlew build -x test
`)
		} else {
			pipeline.WriteString(`    - name: Verify build
      run: mvn compile -q
`)
		}
	}

	// Job для сборки
	previousJob := "test"
	if !info.HasTests {
		previousJob = "verify"
	}

	pipeline.WriteString(fmt.Sprintf(`  build:
    runs-on: ubuntu-latest
    needs: %s
    steps:
    - uses: actions/checkout@v3
    - name: Set up JDK
      uses: actions/setup-java@v3
      with:
        java-version: '`, previousJob))

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("11")
	}

	pipeline.WriteString(`'
        distribution: 'temurin'
`)

	if buildTool == "gradle" {
		pipeline.WriteString(`    - name: Setup Gradle
      uses: gradle/actions/setup-gradle@v3
      
    - name: Build with Gradle
      run: ./gradlew build -x test
      
    - name: Upload JAR artifacts
      uses: actions/upload-artifact@v3
      with:
        name: java-artifacts
        path: build/libs/
`)

		// Для Spring Boot приложений
		if containsDependency(info.Dependencies, "spring-boot") {
			pipeline.WriteString(`    - name: Build Spring Boot JAR
      run: ./gradlew bootJar
      
    - name: Upload Spring Boot JAR
      uses: actions/upload-artifact@v3
      with:
        name: spring-boot-app
        path: build/libs/*.jar
`)
		}
	} else {
		pipeline.WriteString(`    - name: Build with Maven
      run: mvn package -DskipTests
      
    - name: Upload JAR artifacts
      uses: actions/upload-artifact@v3
      with:
        name: java-artifacts
        path: target/*.jar
`)
	}

	// Job для линтинга/проверки качества кода
	if containsDependency(info.Dependencies, "checkstyle") || containsDependency(info.Dependencies, "pmd") {
		pipeline.WriteString(`  quality:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - name: Set up JDK
      uses: actions/setup-java@v3
      with:
        java-version: '11'
        distribution: 'temurin'
`)

		if buildTool == "gradle" {
			pipeline.WriteString(`    - name: Run quality checks
      run: ./gradlew checkstyleMain pmdMain
`)
		} else {
			pipeline.WriteString(`    - name: Run quality checks
      run: mvn checkstyle:check pmd:check
`)
		}
	}

	// Job для публикации в репозиторий пакетов (если настроено)
	if containsDependency(info.Dependencies, "maven-publish") || containsDependency(info.Dependencies, "publishing") {
		pipeline.WriteString(`  publish:
    runs-on: ubuntu-latest
    needs: build
    if: github.event_name == 'push' && contains(github.ref, 'refs/tags/')
    steps:
    - uses: actions/checkout@v3
    - name: Set up JDK
      uses: actions/setup-java@v3
      with:
        java-version: '11'
        distribution: 'temurin'
    - name: Publish to GitHub Packages
      run: ./gradlew publish
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
`)
	}

	return pipeline.String()
}
