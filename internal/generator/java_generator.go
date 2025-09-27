package generator

import (
	"fmt"
	"strings"

	"github.com/immxrtalbeast/pipeline-gen/internal/analyzer"
)

func generateJavaPipeline(info *analyzer.ProjectInfo) string {
	var pipeline strings.Builder

	// Умное определение билд-тула с проверкой структуры проекта
	buildTool := detectBuildTool(info)

	pipeline.WriteString(fmt.Sprintf(`name: Java CI/CD Pipeline (%s)

on:
  push:
    branches: [ main, master, develop ]
  pull_request:
    branches: [ main, master, develop ]

env:
  DEFAULT_JAVA_VERSION: '%s'

jobs:
`, strings.Title(buildTool), getDefaultJavaVersion(info)))

	// Job для тестов или проверки
	if info.HasTests {
		pipeline.WriteString(`  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        java-version: [`)

		// Умный подбор версий Java
		versions := getJavaTestVersions(info)
		for i, v := range versions {
			if i > 0 {
				pipeline.WriteString(", ")
			}
			pipeline.WriteString(fmt.Sprintf("'%s'", v))
		}

		pipeline.WriteString(`]
    steps:
    - uses: actions/checkout@v4
    
    - name: Validate project structure
      run: |
        if [ ! -f "build.gradle" ] && [ ! -f "pom.xml" ] && [ ! -f "build.gradle.kts" ]; then
          echo "❌ No build configuration found (build.gradle, pom.xml, build.gradle.kts)"
          exit 1
        fi
        echo "✅ Build configuration found"
    
    - name: Set up JDK ${{ matrix.java-version }}
      uses: actions/setup-java@v4
      with:
        java-version: ${{ matrix.java-version }}
        distribution: 'temurin'
        cache: '` + buildTool + `'
`)

		// Общие шаги для сборки
		pipeline.WriteString(buildToolSetupSteps(buildTool, true))

		// Запуск тестов
		if buildTool == "gradle" {
			pipeline.WriteString(`    - name: Run tests with Gradle
      run: |
        if [ -f "gradlew" ]; then
          ./gradlew test --no-daemon
        else
          gradle test --no-daemon
        fi
`)
		} else {
			pipeline.WriteString(`    - name: Run tests with Maven
      run: mvn test -B
`)
		}

		// Для многомодульных проектов - дополнительная проверка
		if len(info.Modules) > 0 {
			pipeline.WriteString(fmt.Sprintf(`    - name: Verify module structure
      run: echo "Project has %d modules" && ls -la
`, len(info.Modules)))
		}

		// Отчет о покрытии
		if hasCoverageTool(info.Dependencies) {
			pipeline.WriteString(`    - name: Upload coverage reports
      uses: codecov/codecov-action@v4
      with:
        file: ./build/reports/jacoco/test/jacocoTestReport.xml
        flags: unittests
        name: codecov-umbrella
`)
		}

		// Сохранение результатов тестов
		pipeline.WriteString(`    - name: Upload test results
      uses: actions/upload-artifact@v4
      if: always()
      with:
        name: test-results-${{ matrix.java-version }}
        path: |
          build/reports/tests/
          target/surefire-reports/
        retention-days: 7
`)

	} else {
		// Если тестов нет - простая проверка сборки
		pipeline.WriteString(`  verify:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    
    - name: Set up JDK
      uses: actions/setup-java@v4
      with:
        java-version: ${{ env.DEFAULT_JAVA_VERSION }}
        distribution: 'temurin'
        cache: '` + buildTool + `'
`)

		pipeline.WriteString(buildToolSetupSteps(buildTool, false))

		if buildTool == "gradle" {
			pipeline.WriteString(`    - name: Verify Gradle build
      run: |
        if [ -f "gradlew" ]; then
          ./gradlew assemble -x test --no-daemon
        else
          gradle assemble -x test --no-daemon
        fi
`)
		} else {
			pipeline.WriteString(`    - name: Verify Maven build
      run: mvn compile -B -q
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
    - uses: actions/checkout@v4
    
    - name: Set up JDK
      uses: actions/setup-java@v4
      with:
        java-version: ${{ env.DEFAULT_JAVA_VERSION }}
        distribution: 'temurin'
        cache: '`+buildTool+`'
`, previousJob))

	pipeline.WriteString(buildToolSetupSteps(buildTool, false))

	// Логика сборки
	if buildTool == "gradle" {
		pipeline.WriteString(`    - name: Build with Gradle
      run: |
        if [ -f "gradlew" ]; then
          ./gradlew build -x test --no-daemon
        else
          gradle build -x test --no-daemon
        fi
        
    - name: Verify artifacts existence
      run: |
        echo "Checking for artifacts in build/libs/:"
        ls -la build/libs/ || echo "No artifacts found in build/libs/"
        find . -name "*.jar" -o -name "*.war" | head -10 || echo "No JAR/WAR files found"
        
    - name: Upload JAR artifacts
      uses: actions/upload-artifact@v4
      with:
        name: java-artifacts
        path: |
          build/libs/*.jar
          build/libs/*.war
        retention-days: 30
`)

		// Для Spring Boot приложений
		if containsDependency(info.Dependencies, "spring-boot") {
			pipeline.WriteString(`    - name: Build Spring Boot executable
      run: |
        if [ -f "gradlew" ]; then
          ./gradlew bootJar --no-daemon
        else
          gradle bootJar --no-daemon
        fi
        
    - name: Upload Spring Boot JAR
      uses: actions/upload-artifact@v4
      with:
        name: spring-boot-app
        path: build/libs/*-boot.jar
        retention-days: 30
`)
		}
	} else {
		pipeline.WriteString(`    - name: Build with Maven
      run: mvn package -DskipTests -B
      
    - name: Verify artifacts existence  
      run: |
        echo "Checking for artifacts in target/:"
        ls -la target/ || echo "No artifacts found in target/"
        find . -name "*.jar" -o -name "*.war" | head -10 || echo "No JAR/WAR files found"
        
    - name: Upload JAR artifacts
      uses: actions/upload-artifact@v4
      with:
        name: java-artifacts  
        path: |
          target/*.jar
          target/*.war
        retention-days: 30
`)
	}

	// Job для линтинга/проверки качества кода
	if hasQualityTools(info.Dependencies) {
		pipeline.WriteString(`  quality:
    runs-on: ubuntu-latest
    needs: ` + previousJob + `
    steps:
    - uses: actions/checkout@v4
    
    - name: Set up JDK
      uses: actions/setup-java@v4
      with:
        java-version: ${{ env.DEFAULT_JAVA_VERSION }}
        distribution: 'temurin'
        cache: '` + buildTool + `'
`)

		pipeline.WriteString(buildToolSetupSteps(buildTool, false))

		if buildTool == "gradle" {
			pipeline.WriteString(`    - name: Run quality checks
      run: |
        if [ -f "gradlew" ]; then
          ./gradlew checkstyleMain spotbugsMain pmdMain --no-daemon
        else
          gradle checkstyleMain spotbugsMain pmdMain --no-daemon
        fi
`)
		} else {
			pipeline.WriteString(`    - name: Run quality checks
      run: mvn checkstyle:check pmd:check spotbugs:check -B
`)
		}

		pipeline.WriteString(`    - name: Upload quality reports
      uses: actions/upload-artifact@v4
      with:
        name: quality-reports
        path: |
          build/reports/
          target/site/
        retention-days: 7
`)
	}

	// Job для публикации
	if hasPublishingConfig(info.Dependencies) {
		pipeline.WriteString(`  publish:
    runs-on: ubuntu-latest
    needs: build
    if: github.event_name == 'push' && contains(github.ref, 'refs/tags/')
    steps:
    - uses: actions/checkout@v4
    
    - name: Set up JDK
      uses: actions/setup-java@v4
      with:
        java-version: ${{ env.DEFAULT_JAVA_VERSION }}
        distribution: 'temurin'
        cache: '` + buildTool + `'
`)

		pipeline.WriteString(buildToolSetupSteps(buildTool, false))

		pipeline.WriteString(`    - name: Validate publishing configuration
      run: |
        echo "Checking publishing config..."
        if [ -f "gradlew" ]; then
          ./gradlew tasks --all | grep publish || echo "No publish tasks found"
        else
          mvn help:describe -Dcmd=deploy || echo "No deploy goal found"
        fi
        
    - name: Publish to repository
      run: |
        if [ -f "gradlew" ]; then
          ./gradlew publish -x test --no-daemon
        else
          mvn deploy -DskipTests -B
        fi
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        # Дополнительные переменные для публикации
        MAVEN_USERNAME: ${{ github.actor }}
        MAVEN_PASSWORD: ${{ secrets.GITHUB_TOKEN }}
`)
	}

	return pipeline.String()
}

// Вспомогательные функции

func detectBuildTool(info *analyzer.ProjectInfo) string {
	// Приоритет: явное указание > анализ файлов
	if info.BuildTool == "maven" {
		return "maven"
	}
	if info.BuildTool == "gradle" {
		return "gradle"
	}

	// По умолчанию Gradle
	return "gradle"
}

func getDefaultJavaVersion(info *analyzer.ProjectInfo) string {
	if info.Version != "" {
		return info.Version
	}
	// Современные версии по умолчанию
	return "17"
}

func getJavaTestVersions(info *analyzer.ProjectInfo) []string {
	if info.Version != "" {
		// Если указана конкретная версия, тестируем её и LTS версии
		baseVersion := cleanJavaVersion(info.Version)
		versions := []string{baseVersion}

		// Добавляем LTS версии если их нет
		for _, v := range []string{"11", "17", "21"} {
			if v != baseVersion {
				versions = append(versions, v)
			}
		}
		return versions[:3] // Ограничиваем 3 версиями
	}

	// По умолчанию тестируем на актуальных LTS
	return []string{"11", "17", "21"}
}

func cleanJavaVersion(version string) string {
	// Убираем префиксы типа "1.8", оставляем только цифры
	if strings.HasPrefix(version, "1.") {
		return strings.TrimPrefix(version, "1.")
	}
	return version
}

func buildToolSetupSteps(buildTool string, needsTestSetup bool) string {
	var steps strings.Builder

	if buildTool == "gradle" {
		steps.WriteString(`    - name: Setup Gradle
      uses: gradle/actions/setup-gradle@v4
      
    - name: Validate Gradle wrapper
      run: |
        if [ -f "gradlew" ]; then
          chmod +x gradlew
          echo "✅ Using Gradle wrapper"
        else
          echo "ℹ️  Using system Gradle"
        fi
`)
	} else {
		steps.WriteString(`    - name: Cache Maven dependencies
      uses: actions/cache@v4
      with:
        path: ~/.m2/repository
        key: maven-${{ hashFiles('**/pom.xml') }}
        restore-keys: |
          maven-
`)
	}

	return steps.String()
}

func hasCoverageTool(deps []string) bool {
	tools := []string{"jacoco", "coverage", "jacoco-maven", "jacoco-gradle"}
	for _, tool := range tools {
		if containsDependency(deps, tool) {
			return true
		}
	}
	return false
}

func hasQualityTools(deps []string) bool {
	tools := []string{"checkstyle", "pmd", "spotbugs", "findbugs", "sonar"}
	for _, tool := range tools {
		if containsDependency(deps, tool) {
			return true
		}
	}
	return false
}

func hasPublishingConfig(deps []string) bool {
	tools := []string{"maven-publish", "publishing", "maven-deploy", "nexus", "artifactory"}
	for _, tool := range tools {
		if containsDependency(deps, tool) {
			return true
		}
	}
	return false
}
