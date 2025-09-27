package analyzer

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/immxrtalbeast/pipeline-gen/internal/git"
)

func analyzeCppProjectFromMemory(remoteInfo *git.RemoteRepoInfo, info *ProjectInfo) {
	info.BuildTool = detectCppBuildTool(remoteInfo)
	info.TestFramework = detectCppTestFramework(remoteInfo)
	info.Version = detectCppVersion(remoteInfo)
	info.Dependencies = detectCppDependencies(remoteInfo)
	info.HasTests = detectCppTestsFromMemory(remoteInfo)
	info.Modules = detectCppModulesFromMemory(remoteInfo)
}

func analyzeCppProject(repoPath string, info *ProjectInfo) error {
	info.BuildTool = detectCppBuildToolLocal(repoPath)
	info.TestFramework = detectCppTestFrameworkLocal(repoPath)
	info.Version = detectCppVersionLocal(repoPath)
	info.Dependencies = detectCppDependenciesLocal(repoPath)
	info.HasTests = detectCppTestsLocal(repoPath)
	info.Modules = detectCppModulesLocal(repoPath)
	return nil
}

// Вспомогательные функции для анализа C++
func detectCppBuildTool(remoteInfo *git.RemoteRepoInfo) string {
	// Проверяем различные системы сборки C++
	if remoteInfo.HasFile("CMakeLists.txt") {
		return "cmake"
	}
	if remoteInfo.HasFile("Makefile") || remoteInfo.HasFile("makefile") {
		return "make"
	}
	if remoteInfo.HasFile("configure") || remoteInfo.HasFile("configure.ac") {
		return "autotools"
	}
	if remoteInfo.HasFile("meson.build") {
		return "meson"
	}
	if remoteInfo.HasFile("bazel.BUILD") || remoteInfo.HasFile("BUILD") {
		return "bazel"
	}
	if remoteInfo.HasFile("conanfile.txt") || remoteInfo.HasFile("conanfile.py") {
		return "conan"
	}
	if remoteInfo.HasFile("premake5.lua") {
		return "premake"
	}
	
	// Проверяем наличие файлов проектов IDE
	for _, file := range remoteInfo.Structure {
		if strings.HasSuffix(file, ".sln") || strings.HasSuffix(file, ".vcxproj") {
			return "msbuild"
		}
		if strings.HasSuffix(file, ".pro") {
			return "qmake"
		}
	}
	
	return "make" // по умолчанию
}

func detectCppBuildToolLocal(repoPath string) string {
	// Проверяем локальные файлы
	if exists(filepath.Join(repoPath, "CMakeLists.txt")) {
		return "cmake"
	}
	if exists(filepath.Join(repoPath, "Makefile")) || exists(filepath.Join(repoPath, "makefile")) {
		return "make"
	}
	if exists(filepath.Join(repoPath, "configure")) || exists(filepath.Join(repoPath, "configure.ac")) {
		return "autotools"
	}
	if exists(filepath.Join(repoPath, "meson.build")) {
		return "meson"
	}
	
	// Ищем файлы проектов
	matches, _ := filepath.Glob(filepath.Join(repoPath, "*.sln"))
	if len(matches) > 0 {
		return "msbuild"
	}
	
	matches, _ = filepath.Glob(filepath.Join(repoPath, "*.pro"))
	if len(matches) > 0 {
		return "qmake"
	}
	
	return "make"
}

func detectCppTestFramework(remoteInfo *git.RemoteRepoInfo) string {
	// Анализируем исходные файлы на наличие тестовых фреймворков
	for _, file := range remoteInfo.Structure {
		if strings.HasSuffix(file, ".cpp") || strings.HasSuffix(file, ".h") || strings.HasSuffix(file, ".hpp") {
			if content, exists := remoteInfo.GetFileContent(file); exists {
				if strings.Contains(content, "#include <gtest/gtest.h>") || 
				   strings.Contains(content, "#include \"gtest/gtest.h\"") {
					return "gtest"
				}
				if strings.Contains(content, "#include <catch2/catch.hpp>") || 
				   strings.Contains(content, "#include \"catch2/catch.hpp\"") {
					return "catch2"
				}
				if strings.Contains(content, "#include <boost/test/unit_test.hpp>") {
					return "boost-test"
				}
				if strings.Contains(content, "#include <doctest/doctest.h>") {
					return "doctest"
				}
			}
		}
	}
	
	// Проверяем файлы конфигурации
	if content, exists := remoteInfo.GetFileContent("CMakeLists.txt"); exists {
		if strings.Contains(content, "find_package(GTest") || strings.Contains(content, "gtest") {
			return "gtest"
		}
		if strings.Contains(content, "Catch2") {
			return "catch2"
		}
	}
	
	return "custom" // пользовательская система тестов
}

func detectCppTestFrameworkLocal(repoPath string) string {
	// Упрощенная проверка - сканируем файлы по шаблону
	patterns := []string{
		filepath.Join(repoPath, "**/*.cpp"),
		filepath.Join(repoPath, "**/*.h"),
		filepath.Join(repoPath, "**/*.hpp"),
	}
	
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		for _, match := range matches {
			content, err := os.ReadFile(match)
			if err == nil {
				text := string(content)
				if strings.Contains(text, "gtest/gtest.h") {
					return "gtest"
				}
				if strings.Contains(text, "catch2/catch.hpp") {
					return "catch2"
				}
				if strings.Contains(text, "boost/test/unit_test.hpp") {
					return "boost-test"
				}
				if strings.Contains(text, "doctest/doctest.h") {
					return "doctest"
				}
			}
		}
	}
	
	return "custom"
}

func detectCppVersion(remoteInfo *git.RemoteRepoInfo) string {
	// Пытаемся определить стандарт C++ из файлов
	stdRegex := regexp.MustCompile(`-std=c\+\+(\d+)(\w*)`)
	
	// Проверяем CMakeLists.txt
	if content, exists := remoteInfo.GetFileContent("CMakeLists.txt"); exists {
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			if strings.Contains(line, "CXX_STANDARD") {
				// set(CMAKE_CXX_STANDARD 17)
				re := regexp.MustCompile(`CXX_STANDARD\s+(\d+)`)
				matches := re.FindStringSubmatch(line)
				if len(matches) > 1 {
					return matches[1]
				}
			}
			if strings.Contains(line, "-std=c++") {
				matches := stdRegex.FindStringSubmatch(line)
				if len(matches) > 1 {
					return matches[1]
				}
			}
		}
	}
	
	// Проверяем Makefile
	if content, exists := remoteInfo.GetFileContent("Makefile"); exists {
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			if strings.Contains(line, "-std=c++") {
				matches := stdRegex.FindStringSubmatch(line)
				if len(matches) > 1 {
					return matches[1]
				}
			}
		}
	}
	
	// Проверяем исходные файлы на наличие макросов
	for _, file := range remoteInfo.Structure {
		if strings.HasSuffix(file, ".cpp") || strings.HasSuffix(file, ".h") {
			if content, exists := remoteInfo.GetFileContent(file); exists {
				if strings.Contains(content, "__cplusplus") {
					// Можно попытаться определить по значению макроса
					if strings.Contains(content, "199711L") {
						return "98"
					}
					if strings.Contains(content, "201103L") {
						return "11"
					}
					if strings.Contains(content, "201402L") {
						return "14"
					}
					if strings.Contains(content, "201703L") {
						return "17"
					}
					if strings.Contains(content, "202002L") {
						return "20"
					}
				}
			}
		}
	}
	
	return "17" // по умолчанию C++17
}

func detectCppVersionLocal(repoPath string) string {
	stdRegex := regexp.MustCompile(`-std=c\+\+(\d+)(\w*)`)
	
	// Проверяем CMakeLists.txt
	cmakePath := filepath.Join(repoPath, "CMakeLists.txt")
	if exists(cmakePath) {
		content, err := os.ReadFile(cmakePath)
		if err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				if strings.Contains(line, "CXX_STANDARD") {
					re := regexp.MustCompile(`CXX_STANDARD\s+(\d+)`)
					matches := re.FindStringSubmatch(line)
					if len(matches) > 1 {
						return matches[1]
					}
				}
			}
		}
	}
	
	// Проверяем Makefile
	makefilePath := filepath.Join(repoPath, "Makefile")
	if exists(makefilePath) {
		content, err := os.ReadFile(makefilePath)
		if err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				if strings.Contains(line, "-std=c++") {
					matches := stdRegex.FindStringSubmatch(line)
					if len(matches) > 1 {
						return matches[1]
					}
				}
			}
		}
	}
	
	return "17"
}

func detectCppDependencies(remoteInfo *git.RemoteRepoInfo) []string {
	deps := []string{}

	// Анализируем CMakeLists.txt на зависимости
	if content, exists := remoteInfo.GetFileContent("CMakeLists.txt"); exists {
		if strings.Contains(content, "find_package(OpenGL") || strings.Contains(content, "OpenGL::") {
			deps = append(deps, "graphics:opengl")
		}
		if strings.Contains(content, "find_package(Qt") || strings.Contains(content, "Qt5") || strings.Contains(content, "Qt6") {
			deps = append(deps, "gui:qt")
		}
		if strings.Contains(content, "Boost") || strings.Contains(content, "find_package(Boost") {
			deps = append(deps, "framework:boost")
		}
		if strings.Contains(content, "OpenCV") || strings.Contains(content, "find_package(OpenCV") {
			deps = append(deps, "vision:opencv")
		}
		if strings.Contains(content, "Threads") {
			deps = append(deps, "threading")
		}
		if strings.Contains(content, "OpenMP") {
			deps = append(deps, "parallel:openmp")
		}
		if strings.Contains(content, "MPI") {
			deps = append(deps, "parallel:mpi")
		}
		if strings.Contains(content, "CUDA") {
			deps = append(deps, "gpu:cuda")
		}
		if strings.Contains(content, "OpenCL") {
			deps = append(deps, "gpu:opencl")
		}
		if strings.Contains(content, "SFML") {
			deps = append(deps, "multimedia:sfml")
		}
		if strings.Contains(content, "SDL2") {
			deps = append(deps, "multimedia:sdl2")
		}
	}

	// Анализируем conanfile.txt/py
	if content, exists := remoteInfo.GetFileContent("conanfile.txt"); exists {
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "#") && strings.Contains(line, "/") {
				deps = append(deps, "conan:"+line)
			}
		}
	}

	// Анализируем исходные файлы на включения
	for _, file := range remoteInfo.Structure {
		if strings.HasSuffix(file, ".cpp") || strings.HasSuffix(file, ".h") {
			if content, exists := remoteInfo.GetFileContent(file); exists {
				if strings.Contains(content, "#include <mysql.h>") || strings.Contains(content, "#include <pqxx/pqxx>") {
					deps = append(deps, "database")
				}
				if strings.Contains(content, "#include <curl/curl.h>") {
					deps = append(deps, "network:curl")
				}
				if strings.Contains(content, "#include <openssl/") {
					deps = append(deps, "crypto:openssl")
				}
				if strings.Contains(content, "#include <json/json.h>") || strings.Contains(content, "#include <nlohmann/json.hpp>") {
					deps = append(deps, "json")
				}
			}
		}
	}

	return deps
}

func detectCppDependenciesLocal(repoPath string) []string {
	deps := []string{}

	cmakePath := filepath.Join(repoPath, "CMakeLists.txt")
	if exists(cmakePath) {
		content, err := os.ReadFile(cmakePath)
		if err == nil {
			text := string(content)
			if strings.Contains(text, "Qt5") || strings.Contains(text, "Qt6") {
				deps = append(deps, "gui:qt")
			}
			if strings.Contains(text, "Boost") {
				deps = append(deps, "framework:boost")
			}
			if strings.Contains(text, "OpenCV") {
				deps = append(deps, "vision:opencv")
			}
			if strings.Contains(text, "CUDA") {
				deps = append(deps, "gpu:cuda")
			}
		}
	}

	return deps
}

func detectCppTestsFromMemory(remoteInfo *git.RemoteRepoInfo) bool {
	// Ищем тестовые файлы
	for _, file := range remoteInfo.Structure {
		if strings.Contains(strings.ToLower(file), "test") && 
		   (strings.HasSuffix(file, ".cpp") || strings.HasSuffix(file, ".h")) {
			return true
		}
		if strings.Contains(file, "/test/") || strings.Contains(file, "/tests/") {
			return true
		}
	}
	
	// Проверяем конфигурационные файлы тестов
	if remoteInfo.HasFile("CTestTestfile.cmake") {
		return true
	}
	if content, exists := remoteInfo.GetFileContent("CMakeLists.txt"); exists {
		if strings.Contains(content, "enable_testing()") || strings.Contains(content, "add_test") {
			return true
		}
	}
	
	return false
}

func detectCppTestsLocal(repoPath string) bool {
	// Ищем тестовые директории и файлы
	patterns := []string{
		filepath.Join(repoPath, "test"),
		filepath.Join(repoPath, "tests"),
		filepath.Join(repoPath, "**/test/**/*.cpp"),
		filepath.Join(repoPath, "**/*test*.cpp"),
		filepath.Join(repoPath, "CTestTestfile.cmake"),
	}

	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		if len(matches) > 0 {
			return true
		}
	}

	// Проверяем CMakeLists.txt на наличие тестов
	cmakePath := filepath.Join(repoPath, "CMakeLists.txt")
	if exists(cmakePath) {
		content, err := os.ReadFile(cmakePath)
		if err == nil {
			if strings.Contains(string(content), "enable_testing()") {
				return true
			}
		}
	}

	return false
}

// Функции для обнаружения модулей/подпроектов
func detectCppModulesFromMemory(remoteInfo *git.RemoteRepoInfo) []string {
	modules := []string{}

	// Для CMake ищем add_subdirectory
	if content, exists := remoteInfo.GetFileContent("CMakeLists.txt"); exists {
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "add_subdirectory(") {
				// add_subdirectory(src)
				parts := strings.Split(line, "(")
				if len(parts) > 1 {
					module := strings.TrimRight(parts[1], " )")
					modules = append(modules, module)
				}
			}
		}
	}

	// Ищем поддиректории с собственными CMakeLists.txt
	for _, file := range remoteInfo.Structure {
		if strings.Contains(file, "/CMakeLists.txt") && file != "CMakeLists.txt" {
			dir := filepath.Dir(file)
			modules = append(modules, dir)
		}
	}

	return modules
}

func detectCppModulesLocal(repoPath string) []string {
	modules := []string{}

	// Анализируем главный CMakeLists.txt
	cmakePath := filepath.Join(repoPath, "CMakeLists.txt")
	if exists(cmakePath) {
		content, err := os.ReadFile(cmakePath)
		if err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "add_subdirectory(") {
					parts := strings.Split(line, "(")
					if len(parts) > 1 {
						module := strings.TrimRight(parts[1], " )")
						modules = append(modules, module)
					}
				}
			}
		}
	}

	// Ищем поддиректории с CMakeLists.txt
	filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.Name() == "CMakeLists.txt" && path != cmakePath {
			relPath, _ := filepath.Rel(repoPath, filepath.Dir(path))
			modules = append(modules, relPath)
		}
		return nil
	})

	return modules
}