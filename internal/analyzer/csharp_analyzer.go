package analyzer

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/immxrtalbeast/pipeline-gen/internal/git"
)

func analyzeCSharpProjectFromMemory(remoteInfo *git.RemoteRepoInfo, info *ProjectInfo) {
	info.BuildTool = detectCSharpBuildToolFromMemory(remoteInfo)
	info.TestFramework = detectCSharpTestFrameworkFromMemory(remoteInfo)
	info.Version = detectCSharpVersionFromMemory(remoteInfo)
	info.Dependencies = detectCSharpDependenciesFromMemory(remoteInfo)
	info.HasTests = detectCSharpTestsFromMemory(remoteInfo)
	info.Modules = detectCSharpModulesFromMemory(remoteInfo)
}

func analyzeCSharpProject(repoPath string, info *ProjectInfo) error {
	info.BuildTool = detectCSharpBuildToolLocal(repoPath)
	info.TestFramework = detectCSharpTestFrameworkLocal(repoPath)
	info.Version = detectCSharpVersionLocal(repoPath)
	info.Dependencies = detectCSharpDependenciesLocal(repoPath)
	info.HasTests = detectCSharpTestsLocal(repoPath)
	info.Modules = detectCSharpModulesLocal(repoPath)
	return nil
}

func detectCSharpBuildToolFromMemory(remoteInfo *git.RemoteRepoInfo) string {
	if hasAnyWithSuffix(remoteInfo.Structure, ".sln") {
		return ".NET"
	}
	if hasAnyWithSuffix(remoteInfo.Structure, ".csproj") {
		return ".NET"
	}
	return "dotnet"
}

func detectCSharpBuildToolLocal(repoPath string) string {
	if anyGlob(repoPath, "*.sln") || anyGlob(repoPath, "**/*.sln") {
		return ".NET"
	}
	if anyGlob(repoPath, "*.csproj") || anyGlob(repoPath, "**/*.csproj") {
		return ".NET"
	}
	return "dotnet"
}

func detectCSharpTestFrameworkFromMemory(remoteInfo *git.RemoteRepoInfo) string {
	// Look for references to test frameworks in project files
	for _, file := range remoteInfo.Structure {
		if strings.HasSuffix(strings.ToLower(file), ".csproj") {
			if content, ok := remoteInfo.GetFileContent(file); ok {
				text := strings.ToLower(content)
				if strings.Contains(text, "mstest.testframework") {
					return "mstest"
				}
				if strings.Contains(text, "nunit") {
					return "nunit"
				}
				if strings.Contains(text, "xunit") {
					return "xunit"
				}
			}
		}
	}
	return "xunit"
}

func detectCSharpTestFrameworkLocal(repoPath string) string {
	paths := globAll(repoPath, []string{"**/*.csproj"})
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		text := strings.ToLower(string(b))
		if strings.Contains(text, "mstest.testframework") {
			return "mstest"
		}
		if strings.Contains(text, "nunit") {
			return "nunit"
		}
		if strings.Contains(text, "xunit") {
			return "xunit"
		}
	}
	return "xunit"
}

func detectCSharpVersionFromMemory(remoteInfo *git.RemoteRepoInfo) string {
	// Try global.json SDK version
	if content, ok := remoteInfo.GetFileContent("global.json"); ok {
		// naive parse
		if strings.Contains(content, "\"version\"") {
			lines := strings.Split(content, "\n")
			for _, l := range lines {
				l = strings.TrimSpace(l)
				if strings.Contains(l, "\"version\"") {
					parts := strings.Split(l, ":")
					if len(parts) > 1 {
						return strings.Trim(strings.Trim(parts[1], ", "), "\"")
					}
				}
			}
		}
	}
	// fallback by TargetFramework
	for _, file := range remoteInfo.Structure {
		if strings.HasSuffix(strings.ToLower(file), ".csproj") {
			if content, ok := remoteInfo.GetFileContent(file); ok {
				if v := parseTargetFrameworkVersion(content); v != "" {
					return v
				}
			}
		}
	}
	return "8.0"
}

func detectCSharpVersionLocal(repoPath string) string {
	if exists(filepath.Join(repoPath, "global.json")) {
		b, err := os.ReadFile(filepath.Join(repoPath, "global.json"))
		if err == nil {
			text := string(b)
			if strings.Contains(text, "\"version\"") {
				lines := strings.Split(text, "\n")
				for _, l := range lines {
					l = strings.TrimSpace(l)
					if strings.Contains(l, "\"version\"") {
						parts := strings.Split(l, ":")
						if len(parts) > 1 {
							return strings.Trim(strings.Trim(parts[1], ", "), "\"")
						}
					}
				}
			}
		}
	}
	paths := globAll(repoPath, []string{"**/*.csproj"})
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		if v := parseTargetFrameworkVersion(string(b)); v != "" {
			return v
		}
	}
	return "8.0"
}

func parseTargetFrameworkVersion(csproj string) string {
	// Look for <TargetFramework>net8.0</TargetFramework>
	low := strings.ToLower(csproj)
	if strings.Contains(low, "<targetframework>") {
		lines := strings.Split(low, "\n")
		for _, l := range lines {
			l = strings.TrimSpace(l)
			if strings.Contains(l, "<targetframework>") {
				v := l
				v = strings.TrimPrefix(v, "<targetframework>")
				v = strings.Split(v, "</targetframework>")[0]
				v = strings.TrimSpace(v)
				if strings.HasPrefix(v, "net") {
					return strings.TrimPrefix(v, "net")
				}
			}
		}
	}
	return ""
}

func detectCSharpDependenciesFromMemory(remoteInfo *git.RemoteRepoInfo) []string {
	deps := []string{}
	for _, file := range remoteInfo.Structure {
		if strings.HasSuffix(strings.ToLower(file), ".csproj") {
			if content, ok := remoteInfo.GetFileContent(file); ok {
				text := strings.ToLower(content)
				if strings.Contains(text, "microsoft.aspnetcore.app") || strings.Contains(text, "aspnetcore") {
					deps = append(deps, "web-framework:aspnetcore")
				}
				if strings.Contains(text, "entityframeworkcore") {
					deps = append(deps, "database:ef-core")
				}
				if strings.Contains(text, "serilog") {
					deps = append(deps, "logging:serilog")
				}
			}
		}
	}
	return deps
}

func detectCSharpDependenciesLocal(repoPath string) []string {
	deps := []string{}
	paths := globAll(repoPath, []string{"**/*.csproj"})
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		text := strings.ToLower(string(b))
		if strings.Contains(text, "microsoft.aspnetcore.app") || strings.Contains(text, "aspnetcore") {
			deps = append(deps, "web-framework:aspnetcore")
		}
		if strings.Contains(text, "entityframeworkcore") {
			deps = append(deps, "database:ef-core")
		}
		if strings.Contains(text, "serilog") {
			deps = append(deps, "logging:serilog")
		}
	}
	return deps
}

func detectCSharpTestsFromMemory(remoteInfo *git.RemoteRepoInfo) bool {
	for _, file := range remoteInfo.Structure {
		name := strings.ToLower(filepath.Base(file))
		if strings.HasSuffix(name, ".cs") && (strings.Contains(file, "/tests/") || strings.Contains(file, "/test/") || strings.Contains(name, "tests")) {
			return true
		}
		if strings.HasSuffix(name, ".csproj") && strings.Contains(strings.ToLower(file), "test") {
			return true
		}
	}
	return false
}

func detectCSharpTestsLocal(repoPath string) bool {
	patterns := []string{"**/*Tests.cs", "**/*Test.cs", "**/*.Tests.cs"}
	for _, g := range patterns {
		if anyGlob(repoPath, g) {
			return true
		}
	}
	// if any test project exists
	if anyGlob(repoPath, "**/*Test*.csproj") || anyGlob(repoPath, "**/*.Tests.csproj") {
		return true
	}
	return false
}

func detectCSharpModulesFromMemory(remoteInfo *git.RemoteRepoInfo) []string {
	mods := []string{}
	for _, f := range remoteInfo.Structure {
		if strings.HasSuffix(strings.ToLower(f), ".csproj") {
			mods = append(mods, filepath.Dir(f))
		}
	}
	return mods
}

func detectCSharpModulesLocal(repoPath string) []string {
	mods := []string{}
	_ = filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info != nil && !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".csproj") {
			rel, _ := filepath.Rel(repoPath, filepath.Dir(path))
			mods = append(mods, rel)
		}
		return nil
	})
	return mods
}

// helpers
func anyGlob(repoPath, pattern string) bool {
	matches, _ := filepath.Glob(filepath.Join(repoPath, pattern))
	if len(matches) > 0 {
		return true
	}
	return false
}

func globAll(repoPath string, patterns []string) []string {
	out := []string{}
	for _, p := range patterns {
		m, _ := filepath.Glob(filepath.Join(repoPath, p))
		out = append(out, m...)
	}
	return out
}

func hasAnyWithSuffix(files []string, suffix string) bool {
	for _, f := range files {
		if strings.HasSuffix(strings.ToLower(f), strings.ToLower(suffix)) {
			return true
		}
	}
	return false
}