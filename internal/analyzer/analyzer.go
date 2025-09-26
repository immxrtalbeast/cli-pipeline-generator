package analyzer

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/immxrtalbeast/pipeline-gen/internal/git"
)

type ProjectInfo struct {
	Language       string   `json:"language"`
	Version        string   `json:"version"`
	Architecture   string   `json:"architecture"`
	Dependencies   []string `json:"dependencies"`
	BuildTool      string   `json:"build_tool"`
	TestFramework  string   `json:"test_framework"`
	HasDockerfile  bool     `json:"has_dockerfile"`
	HasMakefile    bool     `json:"has_makefile"`
	Modules        []string `json:"modules"`
	RepositoryType string   `json:"repository_type"` // "local" или "remote"
	RemoteURL      string   `json:"remote_url,omitempty"`
	HasTests       bool     `json:"has_tests"` // ← Добавляем это поле
}

func AnalyzeRemoteRepo(repoURL, branch string) (*ProjectInfo, error) {
	remoteInfo, err := git.AnalyzeRemoteRepo(repoURL, branch)
	if err != nil {
		return nil, err
	}

	info := &ProjectInfo{
		RepositoryType: "remote",
		RemoteURL:      repoURL,
	}

	// Определяем язык проекта
	info.Language = detectLanguageFromMemory(remoteInfo)

	// Для Go проектов
	if info.Language == "go" {
		analyzeGoProjectFromMemory(remoteInfo, info)
		info.Architecture = detectGoArchitectureFromMemory(remoteInfo)

	}

	// Проверяем дополнительные файлы
	info.HasDockerfile = remoteInfo.HasFile("Dockerfile")
	info.HasMakefile = remoteInfo.HasFile("Makefile")

	return info, nil
}
func detectLanguageFromMemory(remoteInfo *git.RemoteRepoInfo) string {
	// Проверяем наличие конфигурационных файлов
	if remoteInfo.HasFile("go.mod") {
		return "go"
	}
	if remoteInfo.HasFile("package.json") {
		return "javascript"
	}
	if remoteInfo.HasFile("requirements.txt") || remoteInfo.HasFile("setup.py") {
		return "python"
	}
	if remoteInfo.HasFile("Cargo.toml") {
		return "rust"
	}
	if remoteInfo.HasFile("pom.xml") || remoteInfo.HasFile("build.gradle") {
		return "java"
	}

	// Анализируем расширения файлов
	return detectLanguageByExtensions(remoteInfo.Structure)
}
func detectLanguageByExtensions(fileList []string) string {
	extCount := make(map[string]int)

	for _, file := range fileList {
		if strings.Contains(file, ".") {
			ext := strings.ToLower(filepath.Ext(file))
			switch ext {
			case ".go":
				extCount["go"]++
			case ".js", ".ts", ".jsx", ".tsx":
				extCount["javascript"]++
			case ".py":
				extCount["python"]++
			case ".rs":
				extCount["rust"]++
			case ".java":
				extCount["java"]++
			case ".cpp", ".c", ".h", ".hpp":
				extCount["c++"]++
			}
		}
	}

	// Возвращаем язык с наибольшим количеством файлов
	var maxLang string
	maxCount := 0
	for lang, count := range extCount {
		if count > maxCount {
			maxCount = count
			maxLang = lang
		}
	}

	if maxLang == "" {
		return "unknown"
	}
	return maxLang
}

func AnalyzeLocalRepo(repoPath string) (*ProjectInfo, error) {
	info := &ProjectInfo{}

	// Определяем язык проекта
	lang, err := detectLanguage(repoPath)
	if err != nil {
		return nil, err
	}
	info.Language = lang

	if lang == "go" {
		err = analyzeGoProject(repoPath, info)
		if err != nil {
			return nil, err
		}
	}
	switch lang {
	case "python":
		err = analyzePythonProject(repoPath, info)
		if err != nil {
			return nil, err
		}
	}

	info.Architecture = detectArchitecture(repoPath)

	return info, nil
}

func detectLanguage(repoPath string) (string, error) {
	files := []string{
		"go.mod",
		"package.json",
		"requirements.txt",
		"Cargo.toml",
		"pom.xml",
		"build.gradle",
	}

	for _, file := range files {
		if exists(filepath.Join(repoPath, file)) {
			switch file {
			case "go.mod":
				return "go", nil
			case "package.json":
				return "javascript", nil
			case "requirements.txt":
				return "python", nil
			case "Cargo.toml":
				return "rust", nil
			case "pom.xml", "build.gradle":
				return "java", nil
			}
		}
	}

	return "unknown", nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
