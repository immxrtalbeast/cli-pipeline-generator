package analyzer

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/immxrtalbeast/pipeline-gen/internal/git"
)

func analyzeRustProjectFromMemory(remoteInfo *git.RemoteRepoInfo, info *ProjectInfo) {
	info.BuildTool = detectRustBuildTool(remoteInfo)
	info.TestFramework = detectRustTestFramework(remoteInfo)
	info.Version = detectRustVersion(remoteInfo)
	info.Dependencies = detectRustDependencies(remoteInfo)
	info.HasTests = detectRustTestsFromMemory(remoteInfo)
	info.Modules = detectRustCratesFromMemory(remoteInfo)
}

func analyzeRustProject(repoPath string, info *ProjectInfo) error {
	info.BuildTool = detectRustBuildToolLocal(repoPath)
	info.TestFramework = detectRustTestFrameworkLocal(repoPath)
	info.Version = detectRustVersionLocal(repoPath)
	info.Dependencies = detectRustDependenciesLocal(repoPath)
	info.HasTests = detectRustTestsLocal(repoPath)
	info.Modules = detectRustCratesLocal(repoPath)
	return nil
}

// Вспомогательные функции для анализа Rust/Cargo
func detectRustBuildTool(remoteInfo *git.RemoteRepoInfo) string {
	if remoteInfo.HasFile("Cargo.toml") {
		return "cargo"
	}
	return "unknown"
}

func detectRustBuildToolLocal(repoPath string) string {
	if exists(filepath.Join(repoPath, "Cargo.toml")) {
		return "cargo"
	}
	return "unknown"
}

func detectRustTestFramework(remoteInfo *git.RemoteRepoInfo) string {
	// Rust использует встроенную систему тестирования, но могут быть дополнительные фреймворки
	if content, exists := remoteInfo.GetFileContent("Cargo.toml"); exists {
		if strings.Contains(content, "proptest") {
			return "proptest"
		}
		if strings.Contains(content, "rstest") {
			return "rstest"
		}
		if strings.Contains(content, "cucumber") {
			return "cucumber"
		}
	}
	return "builtin" // встроенная система тестов Rust
}

func detectRustTestFrameworkLocal(repoPath string) string {
	cargoPath := filepath.Join(repoPath, "Cargo.toml")
	if exists(cargoPath) {
		content, err := os.ReadFile(cargoPath)
		if err == nil {
			text := string(content)
			if strings.Contains(text, "proptest") {
				return "proptest"
			}
			if strings.Contains(text, "rstest") {
				return "rstest"
			}
		}
	}
	return "builtin"
}

func detectRustVersion(remoteInfo *git.RemoteRepoInfo) string {
	// Проверяем rust-toolchain.toml или rust-toolchain
	if content, exists := remoteInfo.GetFileContent("rust-toolchain.toml"); exists {
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.Contains(line, "channel") {
				// channel = "stable"
				// channel = "1.67.0"
				if strings.Contains(line, "=") {
					parts := strings.Split(line, "=")
					if len(parts) > 1 {
						version := strings.Trim(parts[1], " \"'")
						return version
					}
				}
			}
		}
	}

	// Проверяем файл rust-toolchain (без расширения)
	if content, exists := remoteInfo.GetFileContent("rust-toolchain"); exists {
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				return line
			}
		}
	}

	// Проверяем Cargo.toml на наличие ограничений версии
	if content, exists := remoteInfo.GetFileContent("Cargo.toml"); exists {
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.Contains(line, "rust-version") {
				// rust-version = "1.60"
				if strings.Contains(line, "=") {
					parts := strings.Split(line, "=")
					if len(parts) > 1 {
						version := strings.Trim(parts[1], " \"'")
						return version
					}
				}
			}
		}
	}

	return "stable" // версия по умолчанию
}

func detectRustVersionLocal(repoPath string) string {
	// Проверяем rust-toolchain.toml
	toolchainPath := filepath.Join(repoPath, "rust-toolchain.toml")
	if exists(toolchainPath) {
		content, err := os.ReadFile(toolchainPath)
		if err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.Contains(line, "channel") {
					if strings.Contains(line, "=") {
						parts := strings.Split(line, "=")
						if len(parts) > 1 {
							version := strings.Trim(parts[1], " \"'")
							return version
						}
					}
				}
			}
		}
	}

	// Проверяем rust-toolchain
	toolchainPath = filepath.Join(repoPath, "rust-toolchain")
	if exists(toolchainPath) {
		content, err := os.ReadFile(toolchainPath)
		if err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" && !strings.HasPrefix(line, "#") {
					return line
				}
			}
		}
	}

	return "stable"
}

func detectRustDependencies(remoteInfo *git.RemoteRepoInfo) []string {
	deps := []string{}

	if content, exists := remoteInfo.GetFileContent("Cargo.toml"); exists {
		// Определяем тип проекта
		if strings.Contains(content, "[lib]") {
			deps = append(deps, "type:library")
		} else {
			deps = append(deps, "type:binary")
		}

		// Популярные крейты и фреймворки
		if strings.Contains(content, "tokio") {
			deps = append(deps, "runtime:tokio")
		}
		if strings.Contains(content, "async-std") {
			deps = append(deps, "runtime:async-std")
		}
		if strings.Contains(content, "serde") {
			deps = append(deps, "serialization:serde")
		}
		if strings.Contains(content, "actix-web") {
			deps = append(deps, "framework:actix-web")
		}
		if strings.Contains(content, "rocket") {
			deps = append(deps, "framework:rocket")
		}
		if strings.Contains(content, "warp") {
			deps = append(deps, "framework:warp")
		}
		if strings.Contains(content, "diesel") {
			deps = append(deps, "orm:diesel")
		}
		if strings.Contains(content, "sqlx") {
			deps = append(deps, "database:sqlx")
		}
		if strings.Contains(content, "clap") || strings.Contains(content, "structopt") {
			deps = append(deps, "cli")
		}
		if strings.Contains(content, "reqwest") {
			deps = append(deps, "http-client")
		}
		if strings.Contains(content, "hyper") {
			deps = append(deps, "http")
		}
	}

	return deps
}

func detectRustDependenciesLocal(repoPath string) []string {
	deps := []string{}

	cargoPath := filepath.Join(repoPath, "Cargo.toml")
	if exists(cargoPath) {
		content, err := os.ReadFile(cargoPath)
		if err == nil {
			text := string(content)
			if strings.Contains(text, "[lib]") {
				deps = append(deps, "type:library")
			} else {
				deps = append(deps, "type:binary")
			}

			if strings.Contains(text, "tokio") {
				deps = append(deps, "runtime:tokio")
			}
			if strings.Contains(text, "actix-web") {
				deps = append(deps, "framework:actix-web")
			}
			if strings.Contains(text, "diesel") {
				deps = append(deps, "orm:diesel")
			}
		}
	}

	return deps
}

func detectRustTestsFromMemory(remoteInfo *git.RemoteRepoInfo) bool {
	// Ищем тестовые файлы в Rust проектах
	for _, file := range remoteInfo.Structure {
		if strings.HasSuffix(file, ".rs") {
			// Проверяем наличие тестов в исходных файлах
			if content, exists := remoteInfo.GetFileContent(file); exists {
				if strings.Contains(content, "#[test]") ||
					strings.Contains(content, "#[cfg(test)]") ||
					strings.Contains(content, "#[tokio::test]") {
					return true
				}
			}
		}
	}

	// Проверяем наличие тестов в Cargo.toml
	if content, exists := remoteInfo.GetFileContent("Cargo.toml"); exists {
		if strings.Contains(content, "dev-dependencies") {
			return true
		}
	}

	return false
}

func detectRustTestsLocal(repoPath string) bool {
	// Ищем тесты в исходных файлах
	patterns := []string{
		filepath.Join(repoPath, "**/*.rs"),
	}

	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		for _, match := range matches {
			content, err := os.ReadFile(match)
			if err == nil {
				text := string(content)
				if strings.Contains(text, "#[test]") ||
					strings.Contains(text, "#[cfg(test)]") {
					return true
				}
			}
		}
	}

	// Проверяем dev-dependencies в Cargo.toml
	cargoPath := filepath.Join(repoPath, "Cargo.toml")
	if exists(cargoPath) {
		content, err := os.ReadFile(cargoPath)
		if err == nil {
			if strings.Contains(string(content), "dev-dependencies") {
				return true
			}
		}
	}

	return false
}

// Функции для обнаружения крейтов (workspace members)
func detectRustCratesFromMemory(remoteInfo *git.RemoteRepoInfo) []string {
	crates := []string{}

	// Проверяем Cargo.toml на наличие workspace
	if content, exists := remoteInfo.GetFileContent("Cargo.toml"); exists {
		if strings.Contains(content, "[workspace]") {
			lines := strings.Split(content, "\n")
			inMembersSection := false
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.Contains(line, "members") {
					inMembersSection = true
					continue
				}
				if inMembersSection {
					if strings.Contains(line, "[") {
						break // Конец секции members
					}
					if strings.Contains(line, "\"") {
						// Извлекаем имена членов workspace
						parts := strings.Split(line, "\"")
						for i := 1; i < len(parts); i += 2 {
							if i < len(parts) && parts[i] != "" {
								crates = append(crates, parts[i])
							}
						}
					}
				}
			}
		}
	}

	// Если это не workspace, добавляем корневой крейт
	if len(crates) == 0 && remoteInfo.HasFile("Cargo.toml") {
		crates = append(crates, ".")
	}

	return crates
}

func detectRustCratesLocal(repoPath string) []string {
	crates := []string{}

	cargoPath := filepath.Join(repoPath, "Cargo.toml")
	if exists(cargoPath) {
		content, err := os.ReadFile(cargoPath)
		if err == nil {
			text := string(content)
			if strings.Contains(text, "[workspace]") {
				lines := strings.Split(text, "\n")
				inMembersSection := false
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if strings.Contains(line, "members") {
						inMembersSection = true
						continue
					}
					if inMembersSection {
						if strings.Contains(line, "[") {
							break
						}
						if strings.Contains(line, "\"") {
							parts := strings.Split(line, "\"")
							for i := 1; i < len(parts); i += 2 {
								if i < len(parts) && parts[i] != "" {
									crates = append(crates, parts[i])
								}
							}
						}
					}
				}
			}
		}
	}

	if len(crates) == 0 && exists(cargoPath) {
		crates = append(crates, ".")
	}

	return crates
}