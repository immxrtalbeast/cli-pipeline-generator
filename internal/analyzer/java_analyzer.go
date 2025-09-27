package analyzer

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/immxrtalbeast/pipeline-gen/internal/git"
)

func analyzeJavaProjectFromMemory(remoteInfo *git.RemoteRepoInfo, info *ProjectInfo) {
	info.BuildTool = detectJavaBuildTool(remoteInfo)
	info.TestFramework = detectJavaTestFramework(remoteInfo)
	info.Version = detectJavaVersion(remoteInfo)
	info.Dependencies = detectJavaDependencies(remoteInfo)
	info.HasTests = detectJavaTestsFromMemory(remoteInfo)

	// Определяем, является ли это Gradle проектом
	if info.BuildTool == "gradle" {
		info.Modules = detectGradleModulesFromMemory(remoteInfo)
	}
}
func analyzeJavaProject(repoPath string, info *ProjectInfo) error {
	info.BuildTool = detectJavaBuildToolLocal(repoPath)
	info.TestFramework = detectJavaTestFrameworkLocal(repoPath)
	info.Version = detectJavaVersionLocal(repoPath)
	info.Dependencies = detectJavaDependenciesLocal(repoPath)
	info.HasTests = detectJavaTestsLocal(repoPath)

	if info.BuildTool == "gradle" {
		info.Modules = detectGradleModulesLocal(repoPath)
	}
	return nil
}

// Вспомогательные функции для анализа Java/Gradle
func detectJavaBuildTool(remoteInfo *git.RemoteRepoInfo) string {
	if remoteInfo.HasFile("build.gradle") || remoteInfo.HasFile("build.gradle.kts") {
		return "gradle"
	}
	if remoteInfo.HasFile("pom.xml") {
		return "maven"
	}
	return "unknown"
}

func detectJavaBuildToolLocal(repoPath string) string {
	if exists(filepath.Join(repoPath, "build.gradle")) || exists(filepath.Join(repoPath, "build.gradle.kts")) {
		return "gradle"
	}
	if exists(filepath.Join(repoPath, "pom.xml")) {
		return "maven"
	}
	return "unknown"
}

func detectJavaTestFramework(remoteInfo *git.RemoteRepoInfo) string {
	// Анализируем зависимости в build.gradle
	if content, exists := remoteInfo.GetFileContent("build.gradle"); exists {
		if strings.Contains(content, "junit") || strings.Contains(content, "JUnit") {
			return "junit"
		}
		if strings.Contains(content, "testng") {
			return "testng"
		}
		if strings.Contains(content, "mockito") {
			return "junit-mockito" // обычно используется с JUnit
		}
	}

	// Анализируем зависимости в pom.xml
	if content, exists := remoteInfo.GetFileContent("pom.xml"); exists {
		if strings.Contains(content, "junit") {
			return "junit"
		}
		if strings.Contains(content, "testng") {
			return "testng"
		}
	}

	return "junit" // по умолчанию
}

func detectJavaTestFrameworkLocal(repoPath string) string {
	// Анализ для локального репозитория
	gradlePath := filepath.Join(repoPath, "build.gradle")
	if exists(gradlePath) {
		content, err := os.ReadFile(gradlePath)
		if err == nil {
			text := string(content)
			if strings.Contains(text, "junit") || strings.Contains(text, "JUnit") {
				return "junit"
			}
			if strings.Contains(text, "testng") {
				return "testng"
			}
		}
	}

	pomPath := filepath.Join(repoPath, "pom.xml")
	if exists(pomPath) {
		content, err := os.ReadFile(pomPath)
		if err == nil {
			text := string(content)
			if strings.Contains(text, "junit") {
				return "junit"
			}
			if strings.Contains(text, "testng") {
				return "testng"
			}
		}
	}

	return "junit"
}

func detectJavaVersion(remoteInfo *git.RemoteRepoInfo) string {
	// Анализируем версию Java из build.gradle
	if content, exists := remoteInfo.GetFileContent("build.gradle"); exists {
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.Contains(line, "sourceCompatibility") {
				// Ищем что-то вроде: sourceCompatibility = JavaVersion.VERSION_11
				if strings.Contains(line, "VERSION_") {
					parts := strings.Split(line, "VERSION_")
					if len(parts) > 1 {
						version := strings.TrimRight(parts[1], " \")]")
						return strings.Trim(version, "\"'")
					}
				}
				// Или: sourceCompatibility = '11'
				if strings.Contains(line, "'") {
					parts := strings.Split(line, "'")
					if len(parts) > 1 {
						return parts[1]
					}
				}
			}
		}
	}

	// Анализируем версию из pom.xml
	if content, exists := remoteInfo.GetFileContent("pom.xml"); exists {
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.Contains(line, "<java.version>") {
				parts := strings.Split(line, "<java.version>")
				if len(parts) > 1 {
					version := strings.Split(parts[1], "<")[0]
					return version
				}
			}
			if strings.Contains(line, "<maven.compiler.source>") {
				parts := strings.Split(line, "<maven.compiler.source>")
				if len(parts) > 1 {
					version := strings.Split(parts[1], "<")[0]
					return version
				}
			}
		}
	}

	return "11" // версия по умолчанию
}

func detectJavaVersionLocal(repoPath string) string {
	// Аналогично для локального анализа
	gradlePath := filepath.Join(repoPath, "build.gradle")
	if exists(gradlePath) {
		content, err := os.ReadFile(gradlePath)
		if err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.Contains(line, "sourceCompatibility") {
					if strings.Contains(line, "VERSION_") {
						parts := strings.Split(line, "VERSION_")
						if len(parts) > 1 {
							version := strings.TrimRight(parts[1], " \")]")
							return strings.Trim(version, "\"'")
						}
					}
					if strings.Contains(line, "'") {
						parts := strings.Split(line, "'")
						if len(parts) > 1 {
							return parts[1]
						}
					}
				}
			}
		}
	}

	pomPath := filepath.Join(repoPath, "pom.xml")
	if exists(pomPath) {
		content, err := os.ReadFile(pomPath)
		if err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.Contains(line, "<java.version>") {
					parts := strings.Split(line, "<java.version>")
					if len(parts) > 1 {
						version := strings.Split(parts[1], "<")[0]
						return version
					}
				}
			}
		}
	}

	return "11"
}

func detectJavaDependencies(remoteInfo *git.RemoteRepoInfo) []string {
	deps := []string{}

	// Анализируем build.gradle на наличие популярных зависимостей
	if content, exists := remoteInfo.GetFileContent("build.gradle"); exists {
		if strings.Contains(content, "spring-boot") {
			deps = append(deps, "framework:spring-boot")
		}
		if strings.Contains(content, "spring-core") {
			deps = append(deps, "framework:spring")
		}
		if strings.Contains(content, "hibernate") {
			deps = append(deps, "orm:hibernate")
		}
		if strings.Contains(content, "jackson") {
			deps = append(deps, "json:jackson")
		}
		if strings.Contains(content, "mysql") || strings.Contains(content, "postgresql") {
			deps = append(deps, "database")
		}
		if strings.Contains(content, "web") || strings.Contains(content, "spring-web") {
			deps = append(deps, "web-framework")
		}
	}

	// Анализируем pom.xml
	if content, exists := remoteInfo.GetFileContent("pom.xml"); exists {
		if strings.Contains(content, "spring-boot") {
			deps = append(deps, "framework:spring-boot")
		}
		if strings.Contains(content, "hibernate") {
			deps = append(deps, "orm:hibernate")
		}
	}

	return deps
}

func detectJavaDependenciesLocal(repoPath string) []string {
	deps := []string{}

	gradlePath := filepath.Join(repoPath, "build.gradle")
	if exists(gradlePath) {
		content, err := os.ReadFile(gradlePath)
		if err == nil {
			text := string(content)
			if strings.Contains(text, "spring-boot") {
				deps = append(deps, "framework:spring-boot")
			}
			if strings.Contains(text, "hibernate") {
				deps = append(deps, "orm:hibernate")
			}
			if strings.Contains(text, "mysql") || strings.Contains(text, "postgresql") {
				deps = append(deps, "database")
			}
		}
	}

	pomPath := filepath.Join(repoPath, "pom.xml")
	if exists(pomPath) {
		content, err := os.ReadFile(pomPath)
		if err == nil {
			text := string(content)
			if strings.Contains(text, "spring-boot") {
				deps = append(deps, "framework:spring-boot")
			}
		}
	}

	return deps
}

func detectJavaTestsFromMemory(remoteInfo *git.RemoteRepoInfo) bool {
	// Ищем тестовые файлы в Java проектах
	for _, file := range remoteInfo.Structure {
		if strings.HasSuffix(file, "Test.java") ||
			strings.Contains(file, "/test/") && strings.HasSuffix(file, ".java") ||
			strings.Contains(file, "/src/test/") {
			return true
		}
	}
	return false
}

func detectJavaTestsLocal(repoPath string) bool {
	// Ищем тесты в локальной директории
	patterns := []string{
		filepath.Join(repoPath, "src/test/java/**/*Test.java"),
		filepath.Join(repoPath, "**/test/**/*.java"),
		filepath.Join(repoPath, "**/*Test.java"),
	}

	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		if len(matches) > 0 {
			return true
		}
	}
	return false
}

// Функции для обнаружения Gradle модулей
func detectGradleModulesFromMemory(remoteInfo *git.RemoteRepoInfo) []string {
	modules := []string{}

	// Ищем settings.gradle или settings.gradle.kts
	if content, exists := remoteInfo.GetFileContent("settings.gradle"); exists {
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.Contains(line, "include") {
				// Пример: include 'module1', 'module2'
				parts := strings.Split(line, "'")
				for i := 1; i < len(parts); i += 2 {
					if i < len(parts) {
						modules = append(modules, parts[i])
					}
				}
			}
		}
	}

	// Также ищем поддиректории с build.gradle
	for _, file := range remoteInfo.Structure {
		if strings.Contains(file, "/build.gradle") && file != "build.gradle" {
			dir := filepath.Dir(file)
			modules = append(modules, dir)
		}
	}

	return modules
}

func detectGradleModulesLocal(repoPath string) []string {
	modules := []string{}

	// Проверяем settings.gradle
	settingsPath := filepath.Join(repoPath, "settings.gradle")
	if exists(settingsPath) {
		content, err := os.ReadFile(settingsPath)
		if err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.Contains(line, "include") {
					parts := strings.Split(line, "'")
					for i := 1; i < len(parts); i += 2 {
						if i < len(parts) {
							modules = append(modules, parts[i])
						}
					}
				}
			}
		}
	}

	// Ищем поддиректории с build.gradle
	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.Name() == "build.gradle" && path != filepath.Join(repoPath, "build.gradle") {
			relPath, _ := filepath.Rel(repoPath, filepath.Dir(path))
			modules = append(modules, relPath)
		}
		return nil
	})

	if err != nil {
		// Игнорируем ошибки обхода
	}

	return modules
}
