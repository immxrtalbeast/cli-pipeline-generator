package analyzer

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/immxrtalbeast/pipeline-gen/internal/git"
)

func analyzeRubyProjectFromMemory(remoteInfo *git.RemoteRepoInfo, info *ProjectInfo) {
	info.BuildTool = detectRubyBuildTool(remoteInfo)
	info.TestFramework = detectRubyTestFramework(remoteInfo)
	info.Version = detectRubyVersion(remoteInfo)
	info.Dependencies = detectRubyDependencies(remoteInfo)
	info.HasTests = detectRubyTestsFromMemory(remoteInfo)
	info.Modules = detectRubyModulesFromMemory(remoteInfo)
}

func analyzeRubyProject(repoPath string, info *ProjectInfo) error {
	info.BuildTool = detectRubyBuildToolLocal(repoPath)
	info.TestFramework = detectRubyTestFrameworkLocal(repoPath)
	info.Version = detectRubyVersionLocal(repoPath)
	info.Dependencies = detectRubyDependenciesLocal(repoPath)
	info.HasTests = detectRubyTestsLocal(repoPath)
	info.Modules = detectRubyModulesLocal(repoPath)
	return nil
}

// Вспомогательные функции для анализа Ruby
func detectRubyBuildTool(remoteInfo *git.RemoteRepoInfo) string {
	if remoteInfo.HasFile("Gemfile") {
		return "bundler"
	}
	if remoteInfo.HasFile("Rakefile") {
		return "rake"
	}
	if remoteInfo.HasFile("gemspec") {
		return "gem"
	}
	return "ruby"
}

func detectRubyBuildToolLocal(repoPath string) string {
	if exists(filepath.Join(repoPath, "Gemfile")) {
		return "bundler"
	}
	if exists(filepath.Join(repoPath, "Rakefile")) {
		return "rake"
	}
	// Ищем .gemspec файлы
	matches, _ := filepath.Glob(filepath.Join(repoPath, "*.gemspec"))
	if len(matches) > 0 {
		return "gem"
	}
	return "ruby"
}

func detectRubyTestFramework(remoteInfo *git.RemoteRepoInfo) string {
	// Проверяем Gemfile на наличие тестовых фреймворков
	if content, exists := remoteInfo.GetFileContent("Gemfile"); exists {
		if strings.Contains(content, "rspec") {
			return "rspec"
		}
		if strings.Contains(content, "minitest") {
			return "minitest"
		}
		if strings.Contains(content, "test-unit") {
			return "test-unit"
		}
		if strings.Contains(content, "cucumber") {
			return "cucumber"
		}
	}

	// Проверяем конфигурационные файлы
	if remoteInfo.HasFile("spec/spec_helper.rb") || remoteInfo.HasFile("spec/rails_helper.rb") {
		return "rspec"
	}
	if remoteInfo.HasFile("test/test_helper.rb") {
		return "minitest"
	}

	return "minitest" // по умолчанию
}

func detectRubyTestFrameworkLocal(repoPath string) string {
	// Анализ для локального репозитория
	gemfilePath := filepath.Join(repoPath, "Gemfile")
	if exists(gemfilePath) {
		content, err := os.ReadFile(gemfilePath)
		if err == nil {
			text := string(content)
			if strings.Contains(text, "rspec") {
				return "rspec"
			}
			if strings.Contains(text, "minitest") {
				return "minitest"
			}
			if strings.Contains(text, "test-unit") {
				return "test-unit"
			}
			if strings.Contains(text, "cucumber") {
				return "cucumber"
			}
		}
	}

	// Проверяем конфигурационные файлы
	if exists(filepath.Join(repoPath, "spec/spec_helper.rb")) ||
		exists(filepath.Join(repoPath, "spec/rails_helper.rb")) {
		return "rspec"
	}
	if exists(filepath.Join(repoPath, "test/test_helper.rb")) {
		return "minitest"
	}

	return "minitest"
}

func detectRubyVersion(remoteInfo *git.RemoteRepoInfo) string {
	// Проверяем .ruby-version файл
	if content, exists := remoteInfo.GetFileContent(".ruby-version"); exists {
		return strings.TrimSpace(content)
	}

	// Проверяем Gemfile на наличие версии Ruby
	if content, exists := remoteInfo.GetFileContent("Gemfile"); exists {
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "ruby ") {
				// Ищем что-то вроде: ruby '2.7.0'
				parts := strings.Split(line, "'")
				if len(parts) > 1 {
					return parts[1]
				}
				// Или: ruby "2.7.0"
				parts = strings.Split(line, "\"")
				if len(parts) > 1 {
					return parts[1]
				}
			}
		}
	}

	// Проверяем .gemspec файл
	if content, exists := remoteInfo.GetFileContent("*.gemspec"); exists {
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.Contains(line, "required_ruby_version") {
				// Упрощенный парсинг версии
				if strings.Contains(line, ">=") {
					parts := strings.Split(line, ">=")
					if len(parts) > 1 {
						version := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
						return version
					}
				}
			}
		}
	}

	return "2.7" // версия по умолчанию
}

func detectRubyVersionLocal(repoPath string) string {
	// Аналогично для локального анализа
	if exists(filepath.Join(repoPath, ".ruby-version")) {
		content, err := os.ReadFile(filepath.Join(repoPath, ".ruby-version"))
		if err == nil {
			return strings.TrimSpace(string(content))
		}
	}

	gemfilePath := filepath.Join(repoPath, "Gemfile")
	if exists(gemfilePath) {
		content, err := os.ReadFile(gemfilePath)
		if err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "ruby ") {
					parts := strings.Split(line, "'")
					if len(parts) > 1 {
						return parts[1]
					}
					parts = strings.Split(line, "\"")
					if len(parts) > 1 {
						return parts[1]
					}
				}
			}
		}
	}

	// Ищем .gemspec файлы
	matches, _ := filepath.Glob(filepath.Join(repoPath, "*.gemspec"))
	if len(matches) > 0 {
		content, err := os.ReadFile(matches[0])
		if err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.Contains(line, "required_ruby_version") {
					if strings.Contains(line, ">=") {
						parts := strings.Split(line, ">=")
						if len(parts) > 1 {
							version := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
							return version
						}
					}
				}
			}
		}
	}

	return "2.7"
}

func detectRubyDependencies(remoteInfo *git.RemoteRepoInfo) []string {
	deps := []string{}

	// Анализируем Gemfile на наличие популярных зависимостей
	if content, exists := remoteInfo.GetFileContent("Gemfile"); exists {
		// Web фреймворки
		if strings.Contains(content, "rails") {
			deps = append(deps, "web-framework:rails")
		}
		if strings.Contains(content, "sinatra") {
			deps = append(deps, "web-framework:sinatra")
		}
		if strings.Contains(content, "hanami") {
			deps = append(deps, "web-framework:hanami")
		}
		if strings.Contains(content, "grape") {
			deps = append(deps, "web-framework:grape")
		}

		// Базы данных
		if strings.Contains(content, "activerecord") {
			deps = append(deps, "database:activerecord")
		}
		if strings.Contains(content, "sequel") {
			deps = append(deps, "database:sequel")
		}
		if strings.Contains(content, "mongoid") {
			deps = append(deps, "database:mongoid")
		}
		if strings.Contains(content, "redis") {
			deps = append(deps, "database:redis")
		}

		// Тестирование
		if strings.Contains(content, "rspec") {
			deps = append(deps, "testing:rspec")
		}
		if strings.Contains(content, "cucumber") {
			deps = append(deps, "testing:cucumber")
		}
		if strings.Contains(content, "capybara") {
			deps = append(deps, "testing:capybara")
		}

		// Background jobs
		if strings.Contains(content, "sidekiq") {
			deps = append(deps, "background-jobs:sidekiq")
		}
		if strings.Contains(content, "resque") {
			deps = append(deps, "background-jobs:resque")
		}
		if strings.Contains(content, "delayed_job") {
			deps = append(deps, "background-jobs:delayed_job")
		}

		// API
		if strings.Contains(content, "rack") {
			deps = append(deps, "rack")
		}
		if strings.Contains(content, "puma") {
			deps = append(deps, "server:puma")
		}
		if strings.Contains(content, "unicorn") {
			deps = append(deps, "server:unicorn")
		}

		// Authentication
		if strings.Contains(content, "devise") {
			deps = append(deps, "auth:devise")
		}
		if strings.Contains(content, "omniauth") {
			deps = append(deps, "auth:omniauth")
		}

		// Serialization
		if strings.Contains(content, "jbuilder") {
			deps = append(deps, "serialization:jbuilder")
		}
		if strings.Contains(content, "json") {
			deps = append(deps, "serialization:json")
		}
	}

	return deps
}

func detectRubyDependenciesLocal(repoPath string) []string {
	deps := []string{}

	gemfilePath := filepath.Join(repoPath, "Gemfile")
	if exists(gemfilePath) {
		content, err := os.ReadFile(gemfilePath)
		if err == nil {
			text := string(content)
			// Web фреймворки
			if strings.Contains(text, "rails") {
				deps = append(deps, "web-framework:rails")
			}
			if strings.Contains(text, "sinatra") {
				deps = append(deps, "web-framework:sinatra")
			}
			if strings.Contains(text, "hanami") {
				deps = append(deps, "web-framework:hanami")
			}
			if strings.Contains(text, "grape") {
				deps = append(deps, "web-framework:grape")
			}

			// Базы данных
			if strings.Contains(text, "activerecord") {
				deps = append(deps, "database:activerecord")
			}
			if strings.Contains(text, "sequel") {
				deps = append(deps, "database:sequel")
			}
			if strings.Contains(text, "mongoid") {
				deps = append(deps, "database:mongoid")
			}
			if strings.Contains(text, "redis") {
				deps = append(deps, "database:redis")
			}

			// Тестирование
			if strings.Contains(text, "rspec") {
				deps = append(deps, "testing:rspec")
			}
			if strings.Contains(text, "cucumber") {
				deps = append(deps, "testing:cucumber")
			}
			if strings.Contains(text, "capybara") {
				deps = append(deps, "testing:capybara")
			}

			// Background jobs
			if strings.Contains(text, "sidekiq") {
				deps = append(deps, "background-jobs:sidekiq")
			}
			if strings.Contains(text, "resque") {
				deps = append(deps, "background-jobs:resque")
			}
			if strings.Contains(text, "delayed_job") {
				deps = append(deps, "background-jobs:delayed_job")
			}

			// API
			if strings.Contains(text, "rack") {
				deps = append(deps, "rack")
			}
			if strings.Contains(text, "puma") {
				deps = append(deps, "server:puma")
			}
			if strings.Contains(text, "unicorn") {
				deps = append(deps, "server:unicorn")
			}

			// Authentication
			if strings.Contains(text, "devise") {
				deps = append(deps, "auth:devise")
			}
			if strings.Contains(text, "omniauth") {
				deps = append(deps, "auth:omniauth")
			}

			// Serialization
			if strings.Contains(text, "jbuilder") {
				deps = append(deps, "serialization:jbuilder")
			}
			if strings.Contains(text, "json") {
				deps = append(deps, "serialization:json")
			}
		}
	}

	return deps
}

func detectRubyTestsFromMemory(remoteInfo *git.RemoteRepoInfo) bool {
	// Ищем файлы с тестами в Ruby проектах
	for _, file := range remoteInfo.Structure {
		fileName := filepath.Base(file)
		if strings.HasPrefix(fileName, "test_") ||
			strings.HasSuffix(fileName, "_test.rb") ||
			strings.HasSuffix(fileName, "_spec.rb") ||
			strings.Contains(file, "/spec/") ||
			strings.Contains(file, "/test/") ||
			strings.Contains(file, "/features/") {
			return true
		}
	}
	return false
}

func detectRubyTestsLocal(repoPath string) bool {
	// Ищем тесты в локальной директории
	patterns := []string{
		filepath.Join(repoPath, "test_*.rb"),
		filepath.Join(repoPath, "*_test.rb"),
		filepath.Join(repoPath, "*_spec.rb"),
		filepath.Join(repoPath, "spec/**/*.rb"),
		filepath.Join(repoPath, "test/**/*.rb"),
		filepath.Join(repoPath, "features/**/*.rb"),
		filepath.Join(repoPath, "**/test_*.rb"),
		filepath.Join(repoPath, "**/*_test.rb"),
		filepath.Join(repoPath, "**/*_spec.rb"),
	}

	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		if len(matches) > 0 {
			return true
		}
	}
	return false
}

// Функции для обнаружения Ruby модулей (gems)
func detectRubyModulesFromMemory(remoteInfo *git.RemoteRepoInfo) []string {
	modules := []string{}

	// Ищем .gemspec файлы в поддиректориях
	for _, file := range remoteInfo.Structure {
		if strings.HasSuffix(file, ".gemspec") && file != "*.gemspec" {
			dir := filepath.Dir(file)
			modules = append(modules, dir)
		}
	}

	// Ищем поддиректории с Gemfile
	for _, file := range remoteInfo.Structure {
		if strings.Contains(file, "/Gemfile") && file != "Gemfile" {
			dir := filepath.Dir(file)
			modules = append(modules, dir)
		}
	}

	return modules
}

func detectRubyModulesLocal(repoPath string) []string {
	modules := []string{}

	// Ищем .gemspec файлы в поддиректориях
	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if strings.HasSuffix(info.Name(), ".gemspec") && path != filepath.Join(repoPath, "*.gemspec") {
			relPath, _ := filepath.Rel(repoPath, filepath.Dir(path))
			modules = append(modules, relPath)
		}
		return nil
	})

	if err != nil {
		// Игнорируем ошибки обхода
	}

	// Ищем поддиректории с Gemfile
	err = filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.Name() == "Gemfile" && path != filepath.Join(repoPath, "Gemfile") {
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