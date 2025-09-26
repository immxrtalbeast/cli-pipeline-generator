package analyzer

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/immxrtalbeast/pipeline-gen/internal/git"
)

func analyzeGoProjectFromMemory(remoteInfo *git.RemoteRepoInfo, info *ProjectInfo) {
	info.BuildTool = "go"
	info.TestFramework = "testing"

	// Читаем go.mod для получения версии
	if content, exists := remoteInfo.GetFileContent("go.mod"); exists {
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "go ") {
				info.Version = strings.TrimSpace(strings.TrimPrefix(line, "go "))
				break
			}
		}
	}

	// Анализируем зависимости
	info.Dependencies = detectGoDependenciesFromMemory(remoteInfo)
	info.Modules = detectGoModulesFromMemory(remoteInfo)

	for _, file := range remoteInfo.Structure {
		if strings.HasSuffix(file, "_test.go") {
			info.HasTests = true
		}
	}
	info.MainFilePath = findMainFilePathFromMemory(remoteInfo)
}

func findMainFilePathFromMemory(remoteInfo *git.RemoteRepoInfo) string {
	// Сначала ищем main.go
	for _, file := range remoteInfo.Structure {
		if strings.HasSuffix(file, "main.go") {
			return file
		}
	}

	// Если main.go не найден, ищем другие .go файлы с функцией main()
	for _, file := range remoteInfo.Structure {
		if strings.HasSuffix(file, ".go") && !strings.HasSuffix(file, "_test.go") {
			if content, exists := remoteInfo.GetFileContent(file); exists {
				if containsMainFunction(content) {
					return file
				}
			}
		}
	}

	return ""
}

func containsMainFunction(content string) bool {
	lines := strings.Split(content, "\n")
	inBlockComment := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "/*") {
			inBlockComment = true
		}
		if strings.Contains(line, "*/") {
			inBlockComment = false
			continue
		}
		if inBlockComment {
			continue
		}
		if strings.HasPrefix(line, "//") {
			continue
		}
		if strings.Contains(line, "func main()") {
			return true
		}
	}
	return false
}

func detectGoDependenciesFromMemory(remoteInfo *git.RemoteRepoInfo) []string {
	deps := []string{}

	// Анализируем go.mod на наличие популярных зависимостей
	if content, exists := remoteInfo.GetFileContent("go.mod"); exists {
		if strings.Contains(content, "github.com/gin-gonic/gin") {
			deps = append(deps, "web-framework:gin")
		}
		if strings.Contains(content, "github.com/gorilla/mux") {
			deps = append(deps, "web-framework:gorilla")
		}
		if strings.Contains(content, "database/sql") || strings.Contains(content, "gorm.io/gorm") {
			deps = append(deps, "database")
		}
	}

	return deps
}

func detectGoModulesFromMemory(remoteInfo *git.RemoteRepoInfo) []string {
	modules := []string{}

	// Ищем вложенные go.mod файлы
	for _, file := range remoteInfo.Structure {
		if strings.HasSuffix(file, "go.mod") && file != "go.mod" {
			dir := filepath.Dir(file)
			modules = append(modules, dir)
		}
	}

	return modules
}
func detectGoArchitectureFromMemory(remoteInfo *git.RemoteRepoInfo) string {
	// Определяем архитектуру по структуре каталогов
	hasCmd := remoteInfo.HasDirectory("cmd")
	hasPkg := remoteInfo.HasDirectory("pkg")
	hasInternal := remoteInfo.HasDirectory("internal")

	if hasCmd && hasPkg {
		return "standard-go-layout"
	}
	if hasInternal {
		return "with-internal-packages"
	}

	// Проверяем наличие типичных структур
	for _, file := range remoteInfo.Structure {
		if strings.Contains(file, "/cmd/") || strings.Contains(file, "/pkg/") {
			return "standard-go-layout"
		}
	}

	return "simple"
}

//local

func analyzeGoProject(repoPath string, info *ProjectInfo) error {
	// Читаем go.mod для получения версии и зависимостей
	goModPath := filepath.Join(repoPath, "go.mod")
	if exists(goModPath) {
		content, err := os.ReadFile(goModPath)
		if err != nil {
			return err
		}

		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "go ") {
				info.Version = strings.TrimSpace(strings.TrimPrefix(line, "go "))
			}
		}
	}

	info.BuildTool = "go"
	info.TestFramework = "testing"

	// Простой анализ зависимостей
	info.Dependencies = detectGoDependencies(repoPath)
	info.MainFilePath = findMainFilePath(repoPath)
	return nil
}

func findMainFilePath(repoPath string) string {
	mainGoPath := filepath.Join(repoPath, "main.go")
	if exists(mainGoPath) {
		return "main.go"
	}
	var mainFilePath string
	filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if mainFilePath != "" {
			return filepath.SkipAll
		}
		if info.IsDir() || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if strings.HasSuffix(path, ".go") {
			if strings.HasSuffix(path, "main.go") {
				relPath, _ := filepath.Rel(repoPath, path)
				mainFilePath = relPath
				return filepath.SkipAll
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			if containsMainFunction(string(content)) {
				relPath, _ := filepath.Rel(repoPath, path)
				mainFilePath = relPath
			}
		}

		return nil
	})

	return mainFilePath
}
func detectGoDependencies(repoPath string) []string {
	deps := []string{}

	// Проверяем наличие common зависимостей через анализ импортов
	goFiles, _ := filepath.Glob(filepath.Join(repoPath, "*.go"))
	for _, file := range goFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "import") {
				if strings.Contains(line, "github.com/gin-gonic/gin") {
					deps = append(deps, "web-framework:gin")
				}
				if strings.Contains(line, "database/sql") {
					deps = append(deps, "database:sql")
				}
			}
		}
	}

	return deps
}

func detectArchitecture(repoPath string) string {
	if exists(filepath.Join(repoPath, "cmd")) && exists(filepath.Join(repoPath, "pkg")) {
		return "standard-go-layout"
	}
	if exists(filepath.Join(repoPath, "internal")) {
		return "with-internal-packages"
	}
	return "simple"
}
