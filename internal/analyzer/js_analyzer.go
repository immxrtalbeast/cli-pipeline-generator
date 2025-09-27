package analyzer

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/immxrtalbeast/pipeline-gen/internal/git"
)

func analyzeJavaScriptProjectFromMemory(remoteInfo *git.RemoteRepoInfo, info *ProjectInfo) {
	info.BuildTool = detectJavaScriptBuildTool(remoteInfo)
	info.TestFramework = detectJavaScriptTestFramework(remoteInfo)
	info.Version = detectJavaScriptVersion(remoteInfo)
	info.Dependencies = detectJavaScriptDependencies(remoteInfo)
	info.HasTests = detectJavaScriptTestsFromMemory(remoteInfo)
	info.Modules = detectJavaScriptModulesFromMemory(remoteInfo)
}

func analyzeJavaScriptProject(repoPath string, info *ProjectInfo) error {
	info.BuildTool = detectJavaScriptBuildToolLocal(repoPath)
	info.TestFramework = detectJavaScriptTestFrameworkLocal(repoPath)
	info.Version = detectJavaScriptVersionLocal(repoPath)
	info.Dependencies = detectJavaScriptDependenciesLocal(repoPath)
	info.HasTests = detectJavaScriptTestsLocal(repoPath)
	info.Modules = detectJavaScriptModulesLocal(repoPath)
	return nil
}

// Вспомогательные функции для анализа JavaScript/Node.js
func detectJavaScriptBuildTool(remoteInfo *git.RemoteRepoInfo) string {
	if remoteInfo.HasFile("yarn.lock") {
		return "yarn"
	}
	if remoteInfo.HasFile("pnpm-lock.yaml") {
		return "pnpm"
	}
	if remoteInfo.HasFile("package-lock.json") {
		return "npm"
	}
	if remoteInfo.HasFile("package.json") {
		return "npm"
	}
	return "unknown"
}

func detectJavaScriptBuildToolLocal(repoPath string) string {
	if exists(filepath.Join(repoPath, "yarn.lock")) {
		return "yarn"
	}
	if exists(filepath.Join(repoPath, "pnpm-lock.yaml")) {
		return "pnpm"
	}
	if exists(filepath.Join(repoPath, "package-lock.json")) {
		return "npm"
	}
	if exists(filepath.Join(repoPath, "package.json")) {
		return "npm"
	}
	return "unknown"
}

func detectJavaScriptTestFramework(remoteInfo *git.RemoteRepoInfo) string {
	// Анализируем package.json на наличие тестовых фреймворков
	if content, exists := remoteInfo.GetFileContent("package.json"); exists {
		if strings.Contains(content, "jest") {
			return "jest"
		}
		if strings.Contains(content, "mocha") {
			return "mocha"
		}
		if strings.Contains(content, "vitest") {
			return "vitest"
		}
		if strings.Contains(content, "jasmine") {
			return "jasmine"
		}
		if strings.Contains(content, "cypress") {
			return "cypress"
		}
		if strings.Contains(content, "playwright") {
			return "playwright"
		}
	}

	// Проверяем конфигурационные файлы
	if remoteInfo.HasFile("jest.config.js") || remoteInfo.HasFile("jest.config.ts") {
		return "jest"
	}
	if remoteInfo.HasFile("vitest.config.js") || remoteInfo.HasFile("vitest.config.ts") {
		return "vitest"
	}
	if remoteInfo.HasFile("cypress.config.js") || remoteInfo.HasFile("cypress.config.ts") {
		return "cypress"
	}
	if remoteInfo.HasFile("playwright.config.js") || remoteInfo.HasFile("playwright.config.ts") {
		return "playwright"
	}

	return "jest" // по умолчанию
}

func detectJavaScriptTestFrameworkLocal(repoPath string) string {
	// Анализ для локального репозитория
	packagePath := filepath.Join(repoPath, "package.json")
	if exists(packagePath) {
		content, err := os.ReadFile(packagePath)
		if err == nil {
			text := string(content)
			if strings.Contains(text, "jest") {
				return "jest"
			}
			if strings.Contains(text, "mocha") {
				return "mocha"
			}
			if strings.Contains(text, "vitest") {
				return "vitest"
			}
			if strings.Contains(text, "cypress") {
				return "cypress"
			}
			if strings.Contains(text, "playwright") {
				return "playwright"
			}
		}
	}

	// Проверяем конфигурационные файлы
	configFiles := []string{
		"jest.config.js", "jest.config.ts",
		"vitest.config.js", "vitest.config.ts",
		"cypress.config.js", "cypress.config.ts",
		"playwright.config.js", "playwright.config.ts",
	}

	for _, configFile := range configFiles {
		if exists(filepath.Join(repoPath, configFile)) {
			if strings.Contains(configFile, "jest") {
				return "jest"
			}
			if strings.Contains(configFile, "vitest") {
				return "vitest"
			}
			if strings.Contains(configFile, "cypress") {
				return "cypress"
			}
			if strings.Contains(configFile, "playwright") {
				return "playwright"
			}
		}
	}

	return "jest"
}

func detectJavaScriptVersion(remoteInfo *git.RemoteRepoInfo) string {
	// Анализируем package.json для получения версии Node.js
	if content, exists := remoteInfo.GetFileContent("package.json"); exists {
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.Contains(line, "engines") {
				// Ищем что-то вроде: "engines": { "node": ">=16.0.0" }
				if strings.Contains(line, "node") {
					// Упрощенный парсинг версии Node.js
					if strings.Contains(line, "16") {
						return "16"
					}
					if strings.Contains(line, "18") {
						return "18"
					}
					if strings.Contains(line, "20") {
						return "20"
					}
				}
			}
		}
	}

	// Проверяем .nvmrc файл
	if content, exists := remoteInfo.GetFileContent(".nvmrc"); exists {
		return strings.TrimSpace(content)
	}

	// Проверяем .node-version файл
	if content, exists := remoteInfo.GetFileContent(".node-version"); exists {
		return strings.TrimSpace(content)
	}

	return "18" // версия по умолчанию
}

func detectJavaScriptVersionLocal(repoPath string) string {
	// Аналогично для локального анализа
	packagePath := filepath.Join(repoPath, "package.json")
	if exists(packagePath) {
		content, err := os.ReadFile(packagePath)
		if err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.Contains(line, "engines") && strings.Contains(line, "node") {
					if strings.Contains(line, "16") {
						return "16"
					}
					if strings.Contains(line, "18") {
						return "18"
					}
					if strings.Contains(line, "20") {
						return "20"
					}
				}
			}
		}
	}

	// Проверяем .nvmrc
	if exists(filepath.Join(repoPath, ".nvmrc")) {
		content, err := os.ReadFile(filepath.Join(repoPath, ".nvmrc"))
		if err == nil {
			return strings.TrimSpace(string(content))
		}
	}

	// Проверяем .node-version
	if exists(filepath.Join(repoPath, ".node-version")) {
		content, err := os.ReadFile(filepath.Join(repoPath, ".node-version"))
		if err == nil {
			return strings.TrimSpace(string(content))
		}
	}

	return "18"
}

func detectJavaScriptDependencies(remoteInfo *git.RemoteRepoInfo) []string {
	deps := []string{}

	// Анализируем package.json на наличие популярных зависимостей
	if content, exists := remoteInfo.GetFileContent("package.json"); exists {
		// Frontend фреймворки
		if strings.Contains(content, "react") {
			deps = append(deps, "frontend-framework:react")
		}
		if strings.Contains(content, "vue") {
			deps = append(deps, "frontend-framework:vue")
		}
		if strings.Contains(content, "angular") {
			deps = append(deps, "frontend-framework:angular")
		}
		if strings.Contains(content, "svelte") {
			deps = append(deps, "frontend-framework:svelte")
		}

		// Backend фреймворки
		if strings.Contains(content, "express") {
			deps = append(deps, "backend-framework:express")
		}
		if strings.Contains(content, "koa") {
			deps = append(deps, "backend-framework:koa")
		}
		if strings.Contains(content, "fastify") {
			deps = append(deps, "backend-framework:fastify")
		}
		if strings.Contains(content, "nest") {
			deps = append(deps, "backend-framework:nestjs")
		}
	}

	return deps
}

func detectJavaScriptDependenciesLocal(repoPath string) []string {
	deps := []string{}

	packagePath := filepath.Join(repoPath, "package.json")
	if exists(packagePath) {
		content, err := os.ReadFile(packagePath)
		if err == nil {
			text := string(content)
			// Frontend фреймворки
			if strings.Contains(text, "react") {
				deps = append(deps, "frontend-framework:react")
			}
			if strings.Contains(text, "vue") {
				deps = append(deps, "frontend-framework:vue")
			}
			if strings.Contains(text, "angular") {
				deps = append(deps, "frontend-framework:angular")
			}
			if strings.Contains(text, "svelte") {
				deps = append(deps, "frontend-framework:svelte")
			}

			// Backend фреймворки
			if strings.Contains(text, "express") {
				deps = append(deps, "backend-framework:express")
			}
			if strings.Contains(text, "koa") {
				deps = append(deps, "backend-framework:koa")
			}
			if strings.Contains(text, "fastify") {
				deps = append(deps, "backend-framework:fastify")
			}
			if strings.Contains(text, "nest") {
				deps = append(deps, "backend-framework:nestjs")
			}
		}
	}

	return deps
}

func detectJavaScriptTestsFromMemory(remoteInfo *git.RemoteRepoInfo) bool {
	// Ищем файлы с тестами в JavaScript проектах
	for _, file := range remoteInfo.Structure {
		fileName := filepath.Base(file)
		if strings.HasPrefix(fileName, "test") ||
			strings.HasSuffix(fileName, ".test.js") ||
			strings.HasSuffix(fileName, ".test.ts") ||
			strings.HasSuffix(fileName, ".test.jsx") ||
			strings.HasSuffix(fileName, ".test.tsx") ||
			strings.HasSuffix(fileName, ".spec.js") ||
			strings.HasSuffix(fileName, ".spec.ts") ||
			strings.HasSuffix(fileName, ".spec.jsx") ||
			strings.HasSuffix(fileName, ".spec.tsx") ||
			strings.Contains(file, "/tests/") ||
			strings.Contains(file, "/__tests__/") ||
			strings.Contains(file, "/test/") {
			return true
		}
	}
	return false
}

func detectJavaScriptTestsLocal(repoPath string) bool {
	// Ищем тесты в локальной директории
	patterns := []string{
		filepath.Join(repoPath, "**/*.test.js"),
		filepath.Join(repoPath, "**/*.test.ts"),
		filepath.Join(repoPath, "**/*.test.jsx"),
		filepath.Join(repoPath, "**/*.test.tsx"),
		filepath.Join(repoPath, "**/*.spec.js"),
		filepath.Join(repoPath, "**/*.spec.ts"),
		filepath.Join(repoPath, "**/*.spec.jsx"),
		filepath.Join(repoPath, "**/*.spec.tsx"),
		filepath.Join(repoPath, "**/test/**/*.js"),
		filepath.Join(repoPath, "**/test/**/*.ts"),
		filepath.Join(repoPath, "**/tests/**/*.js"),
		filepath.Join(repoPath, "**/tests/**/*.ts"),
		filepath.Join(repoPath, "**/__tests__/**/*.js"),
		filepath.Join(repoPath, "**/__tests__/**/*.ts"),
	}

	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		if len(matches) > 0 {
			return true
		}
	}
	return false
}

// Функции для обнаружения JavaScript модулей (workspaces)
func detectJavaScriptModulesFromMemory(remoteInfo *git.RemoteRepoInfo) []string {
	modules := []string{}

	// Проверяем package.json на наличие workspaces
	if content, exists := remoteInfo.GetFileContent("package.json"); exists {
		lines := strings.Split(content, "\n")
		inWorkspaces := false
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.Contains(line, "workspaces") {
				inWorkspaces = true
				continue
			}
			if inWorkspaces && strings.Contains(line, "packages") {
				// Ищем массив пакетов
				if strings.Contains(line, "[") {
					// Упрощенный парсинг для примера
					parts := strings.Split(line, "\"")
					for i := 1; i < len(parts); i += 2 {
						if i < len(parts) && parts[i] != "" {
							modules = append(modules, parts[i])
						}
					}
				}
			}
		}
	}

	// Также ищем поддиректории с package.json
	for _, file := range remoteInfo.Structure {
		if strings.Contains(file, "/package.json") && file != "package.json" {
			dir := filepath.Dir(file)
			modules = append(modules, dir)
		}
	}

	return modules
}

func detectJavaScriptModulesLocal(repoPath string) []string {
	modules := []string{}

	// Проверяем package.json на наличие workspaces
	packagePath := filepath.Join(repoPath, "package.json")
	if exists(packagePath) {
		content, err := os.ReadFile(packagePath)
		if err == nil {
			lines := strings.Split(string(content), "\n")
			inWorkspaces := false
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.Contains(line, "workspaces") {
					inWorkspaces = true
					continue
				}
				if inWorkspaces && strings.Contains(line, "packages") {
					if strings.Contains(line, "[") {
						parts := strings.Split(line, "\"")
						for i := 1; i < len(parts); i += 2 {
							if i < len(parts) && parts[i] != "" {
								modules = append(modules, parts[i])
							}
						}
					}
				}
			}
		}
	}

	// Ищем поддиректории с package.json
	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.Name() == "package.json" && path != filepath.Join(repoPath, "package.json") {
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