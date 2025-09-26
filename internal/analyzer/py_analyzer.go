package analyzer

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/immxrtalbeast/pipeline-gen/internal/git"
)

func analyzePythonProjectFromMemory(remoteInfo *git.RemoteRepoInfo, info *ProjectInfo) {
	info.BuildTool = detectPythonBuildTool(remoteInfo)
	info.TestFramework = detectPythonTestFramework(remoteInfo)
	info.Version = detectPythonVersion(remoteInfo)
	info.Dependencies = detectPythonDependencies(remoteInfo)
	info.HasTests = detectPythonTestsFromMemory(remoteInfo)
}

func analyzePythonProject(repoPath string, info *ProjectInfo) error {
	info.BuildTool = detectPythonBuildToolLocal(repoPath)
	info.TestFramework = detectPythonTestFrameworkLocal(repoPath)
	info.Version = detectPythonVersionLocal(repoPath)
	info.Dependencies = detectPythonDependenciesLocal(repoPath)
	info.HasTests = detectPythonTestsLocal(repoPath)
	return nil
}

// Вспомогательные функции для анализа Python
func detectPythonBuildTool(remoteInfo *git.RemoteRepoInfo) string {
	if remoteInfo.HasFile("pyproject.toml") {
		return "poetry"
	}
	if remoteInfo.HasFile("Pipfile") {
		return "pipenv"
	}
	if remoteInfo.HasFile("setup.py") {
		return "setuptools"
	}
	return "pip"
}

func detectPythonBuildToolLocal(repoPath string) string {
	if exists(filepath.Join(repoPath, "pyproject.toml")) {
		return "poetry"
	}
	if exists(filepath.Join(repoPath, "Pipfile")) {
		return "pipenv"
	}
	if exists(filepath.Join(repoPath, "setup.py")) {
		return "setuptools"
	}
	return "pip"
}

func detectPythonTestFramework(remoteInfo *git.RemoteRepoInfo) string {
	// Проверяем конфигурационные файлы тестов
	if content, exists := remoteInfo.GetFileContent("pytest.ini"); exists {
		if strings.Contains(content, "pytest") {
			return "pytest"
		}
	}
	if remoteInfo.HasFile("tox.ini") {
		return "pytest" // часто используется с tox
	}

	// Проверяем зависимости
	if content, exists := remoteInfo.GetFileContent("requirements.txt"); exists {
		if strings.Contains(content, "pytest") {
			return "pytest"
		}
		if strings.Contains(content, "unittest") {
			return "unittest"
		}
	}

	return "unittest" // по умолчанию
}

func detectPythonTestFrameworkLocal(repoPath string) string {
	if exists(filepath.Join(repoPath, "pytest.ini")) {
		return "pytest"
	}
	if exists(filepath.Join(repoPath, "tox.ini")) {
		return "pytest"
	}

	// Читаем requirements.txt
	reqPath := filepath.Join(repoPath, "requirements.txt")
	if exists(reqPath) {
		content, err := os.ReadFile(reqPath)
		if err == nil {
			if strings.Contains(string(content), "pytest") {
				return "pytest"
			}
		}
	}

	return "unittest"
}

func detectPythonVersion(remoteInfo *git.RemoteRepoInfo) string {
	// Проверяем различные файлы с версией Python
	if content, exists := remoteInfo.GetFileContent(".python-version"); exists {
		return strings.TrimSpace(content)
	}
	if content, exists := remoteInfo.GetFileContent("runtime.txt"); exists {
		if strings.HasPrefix(content, "python-") {
			return strings.TrimPrefix(strings.TrimSpace(content), "python-")
		}
	}

	// Анализируем pyproject.toml
	if content, exists := remoteInfo.GetFileContent("pyproject.toml"); exists {
		if strings.Contains(content, "requires-python") {
			// Упрощенный парсинг для примера
			lines := strings.Split(content, "\n")
			for _, line := range lines {
				if strings.Contains(line, "requires-python") {
					parts := strings.Split(line, "=")
					if len(parts) > 1 {
						return strings.Trim(strings.TrimSpace(parts[1]), "\"'")
					}
				}
			}
		}
	}

	return "3.9" // версия по умолчанию
}

func detectPythonVersionLocal(repoPath string) string {
	// Аналогично для локального анализа
	if exists(filepath.Join(repoPath, ".python-version")) {
		content, err := os.ReadFile(filepath.Join(repoPath, ".python-version"))
		if err == nil {
			return strings.TrimSpace(string(content))
		}
	}

	if exists(filepath.Join(repoPath, "runtime.txt")) {
		content, err := os.ReadFile(filepath.Join(repoPath, "runtime.txt"))
		if err == nil {
			version := strings.TrimSpace(string(content))
			if strings.HasPrefix(version, "python-") {
				return strings.TrimPrefix(version, "python-")
			}
		}
	}

	return "3.9"
}

func detectPythonDependencies(remoteInfo *git.RemoteRepoInfo) []string {
	deps := []string{}

	// Анализируем requirements.txt
	if content, exists := remoteInfo.GetFileContent("requirements.txt"); exists {
		if strings.Contains(content, "django") {
			deps = append(deps, "web-framework:django")
		}
		if strings.Contains(content, "flask") {
			deps = append(deps, "web-framework:flask")
		}
		if strings.Contains(content, "fastapi") {
			deps = append(deps, "web-framework:fastapi")
		}
		if strings.Contains(content, "sqlalchemy") || strings.Contains(content, "django.db") {
			deps = append(deps, "database")
		}
		if strings.Contains(content, "numpy") || strings.Contains(content, "pandas") {
			deps = append(deps, "data-science")
		}
	}

	return deps
}

func detectPythonDependenciesLocal(repoPath string) []string {
	deps := []string{}

	reqPath := filepath.Join(repoPath, "requirements.txt")
	if exists(reqPath) {
		content, err := os.ReadFile(reqPath)
		if err == nil {
			text := string(content)
			if strings.Contains(text, "django") {
				deps = append(deps, "web-framework:django")
			}
			if strings.Contains(text, "flask") {
				deps = append(deps, "web-framework:flask")
			}
			if strings.Contains(text, "fastapi") {
				deps = append(deps, "web-framework:fastapi")
			}
			if strings.Contains(text, "sqlalchemy") || strings.Contains(text, "django.db") {
				deps = append(deps, "database")
			}
		}
	}

	return deps
}

func detectPythonTestsFromMemory(remoteInfo *git.RemoteRepoInfo) bool {
	// Ищем файлы с тестами в Python проектах
	for _, file := range remoteInfo.Structure {
		if strings.HasPrefix(filepath.Base(file), "test_") ||
			strings.HasSuffix(file, "_test.py") ||
			strings.Contains(file, "/tests/") {
			return true
		}
	}
	return false
}

func detectPythonTestsLocal(repoPath string) bool {
	// Ищем тесты в локальной директории
	patterns := []string{
		filepath.Join(repoPath, "test_*.py"),
		filepath.Join(repoPath, "*_test.py"),
		filepath.Join(repoPath, "tests", "*.py"),
		filepath.Join(repoPath, "**", "test_*.py"),
		filepath.Join(repoPath, "**", "*_test.py"),
	}

	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		if len(matches) > 0 {
			return true
		}
	}
	return false
}
