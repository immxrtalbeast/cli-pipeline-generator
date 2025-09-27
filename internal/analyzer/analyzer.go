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
	PackageManager string   `json:"package_manager"`
	Structure      []string `json:"structure"`
	RepositoryURL  string   `json:"repository_url"`
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

	if info.Language == "go" {
		analyzeGoProjectFromMemory(remoteInfo, info)
		info.Architecture = detectGoArchitectureFromMemory(remoteInfo)
	}

	switch info.Language {
	case "go":
		analyzeGoProjectFromMemory(remoteInfo, info)
		info.Architecture = detectGoArchitectureFromMemory(remoteInfo)
	case "python":
		analyzePythonProjectFromMemory(remoteInfo, info)
	case "java_gradle", "java_maven":
		analyzeJavaProjectFromMemory(remoteInfo, info)
	case "rust":
		analyzeRustProjectFromMemory(remoteInfo, info)
	case "cpp":
		analyzeCppProjectFromMemory(remoteInfo, info)
	case "javascript":
		analyzeJavaScriptProjectFromMemory(remoteInfo, info)
	case "ruby":
		analyzeRubyProjectFromMemory(remoteInfo, info)
	case "csharp":
		analyzeCSharpProjectFromMemory(remoteInfo, info)
	case "swift":
		analyzeSwiftProjectFromMemory(remoteInfo, info)
	case "php":
		analyzePHPProjectFromMemory(remoteInfo, info)
	}
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

	if remoteInfo.HasFile("build.gradle") || remoteInfo.HasFile("build.gradle.kts") {
		return "java_gradle"
	}
	if remoteInfo.HasFile("pom.xml") {
		return "java_maven"
	}
	if remoteInfo.HasFile("CMakeLists.txt") {
		return "cpp"
	}
	if remoteInfo.HasFile("Makefile") {
		return "cpp"
	}
	if remoteInfo.HasFile("Gemfile") {
		return "ruby"
	}
	if remoteInfo.HasFile(".csproj") || remoteInfo.HasFile(".sln"){
		return "csharp"
	}
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
				extCount["cpp"]++
			case ".rb", ".rake", ".gemspec":
				extCount["ruby"]++
			case ".csproj",".sln":
				extCount["csharp"]++
			case ".swift":
				extCount["swift"]++
			case ".php":
				extCount["php"]++
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

	switch lang {
	case "python":
		err = analyzePythonProject(repoPath, info)
		if err != nil {
			return nil, err
		}
	case "go":
		err = analyzeGoProject(repoPath, info)
		if err != nil {
			return nil, err
		}
	case "rust":
		err = analyzeRustProject(repoPath, info)
		if err != nil {
			return nil, err
		}
	case "cpp":
		err = analyzeCppProject(repoPath, info)
		if err != nil {
			return nil, err
		}
	case "javascript":
		err = analyzeJavaScriptProject(repoPath, info)
		if err != nil {
			return nil, err
		}
	case "ruby":
		err = analyzeRubyProject(repoPath, info)
		if err != nil {
			return nil, err
		}
	case "csharp":
		err = analyzeCSharpProject(repoPath, info)
		if err != nil {
			return nil, err
		}
	case "swift":
		err = analyzeSwiftProject(repoPath, info)
		if err != nil {
			return nil, err
		}
	case "php":
		err = analyzePHPProject(repoPath, info)
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
		"setup.py",
		"pyproject.toml",
		"build.gradle",
		"build.gradle.kts",
		"pom.xml",
		"Cargo.toml",
		"CMakeLists.txt",
		"Makefile",
		"Gemfile",
		".csproj",
		".sln",	
		"Package.swift",
		"composer.json",
		"artisan",
		"symfony",
		"index.php",
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
			case "pom.xml":
				return "java_maven", nil
			case "build.gradle", "build.gradle.kts":
				return "java_gradle", nil
			case "CMakeLists.txt", "Makefile":
				return "cpp", nil
			case "Gemfile":
				return "ruby", nil
			case ".csproj", ".sln":
				return "csharp", nil
			case "Package.swift":
				return "swift", nil
			case "composer.json", "arisan", "symfony", "index.php":
				return "php", nil
			}
		}
	}

	return "unknown", nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
