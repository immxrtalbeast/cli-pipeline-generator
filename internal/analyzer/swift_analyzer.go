package analyzer

import (
    "os"
    "path/filepath"
    "strings"

    "github.com/immxrtalbeast/pipeline-gen/internal/git"
)

func analyzeSwiftProjectFromMemory(remoteInfo *git.RemoteRepoInfo, info *ProjectInfo) {
    info.BuildTool = detectSwiftBuildTool(remoteInfo)
    info.TestFramework = detectSwiftTestFramework(remoteInfo)
    info.Version = detectSwiftVersion(remoteInfo)
    info.Dependencies = detectSwiftDependencies(remoteInfo)
    info.HasTests = detectSwiftTestsFromMemory(remoteInfo)
    info.PackageManager = "spm" // Swift Package Manager по умолчанию
}

func analyzeSwiftProject(repoPath string, info *ProjectInfo) error {
    info.BuildTool = detectSwiftBuildToolLocal(repoPath)
    info.TestFramework = detectSwiftTestFrameworkLocal(repoPath)
    info.Version = detectSwiftVersionLocal(repoPath)
    info.Dependencies = detectSwiftDependenciesLocal(repoPath)
    info.HasTests = detectSwiftTestsLocal(repoPath)
    info.PackageManager = "spm"
    return nil
}

// Вспомогательные функции для анализа Swift
func detectSwiftBuildTool(remoteInfo *git.RemoteRepoInfo) string {
    if remoteInfo.HasFile("Package.swift") {
        return "swift-package-manager"
    }
    if remoteInfo.HasFile(".xcodeproj") || remoteInfo.HasFile(".xcworkspace") {
        return "xcodebuild"
    }
    return "unknown"
}

func detectSwiftBuildToolLocal(repoPath string) string {
    if exists(filepath.Join(repoPath, "Package.swift")) {
        return "swift-package-manager"
    }
    
    // Проверяем наличие Xcode проектов
    matches, _ := filepath.Glob(filepath.Join(repoPath, "*.xcodeproj"))
    if len(matches) > 0 {
        return "xcodebuild"
    }
    
    matches, _ = filepath.Glob(filepath.Join(repoPath, "*.xcworkspace"))
    if len(matches) > 0 {
        return "xcodebuild"
    }
    
    return "unknown"
}

func detectSwiftTestFramework(remoteInfo *git.RemoteRepoInfo) string {
    // Swift использует XCTest по умолчанию
    if content, exists := remoteInfo.GetFileContent("Package.swift"); exists {
        if strings.Contains(content, "XCTest") || strings.Contains(content, "testTarget") {
            return "xctest"
        }
    }
    
    // Проверяем наличие тестовых файлов
    for _, file := range remoteInfo.Structure {
        if strings.Contains(file, "Test.swift") || 
           strings.Contains(file, "Tests.swift") || 
           strings.Contains(file, "/Tests/") {
            return "xctest"
        }
    }
    
    return "xctest" // по умолчанию
}

func detectSwiftTestFrameworkLocal(repoPath string) string {
    // Проверяем Package.swift
    packagePath := filepath.Join(repoPath, "Package.swift")
    if exists(packagePath) {
        content, err := os.ReadFile(packagePath)
        if err == nil {
            text := string(content)
            if strings.Contains(text, "XCTest") || strings.Contains(text, "testTarget") {
                return "xctest"
            }
        }
    }
    
    // Ищем тестовые файлы
    patterns := []string{
        filepath.Join(repoPath, "**/*Test.swift"),
        filepath.Join(repoPath, "**/*Tests.swift"),
        filepath.Join(repoPath, "**/Tests/**/*.swift"),
    }
    
    for _, pattern := range patterns {
        matches, _ := filepath.Glob(pattern)
        if len(matches) > 0 {
            return "xctest"
        }
    }
    
    return "xctest"
}

func detectSwiftVersion(remoteInfo *git.RemoteRepoInfo) string {
    // Анализируем версию Swift из Package.swift
    if content, exists := remoteInfo.GetFileContent("Package.swift"); exists {
        lines := strings.Split(content, "\n")
        for _, line := range lines {
            line = strings.TrimSpace(line)
            if strings.Contains(line, "swift-tools-version:") {
                // Пример: // swift-tools-version:5.7
                parts := strings.Split(line, "swift-tools-version:")
                if len(parts) > 1 {
                    version := strings.TrimSpace(parts[1])
                    // Убираем возможные комментарии после версии
                    if commentIndex := strings.Index(version, "//"); commentIndex != -1 {
                        version = strings.TrimSpace(version[:commentIndex])
                    }
                    return version
                }
            }
        }
    }
    
    return "5.7" // версия по умолчанию
}

func detectSwiftVersionLocal(repoPath string) string {
    packagePath := filepath.Join(repoPath, "Package.swift")
    if exists(packagePath) {
        content, err := os.ReadFile(packagePath)
        if err == nil {
            lines := strings.Split(string(content), "\n")
            for _, line := range lines {
                line = strings.TrimSpace(line)
                if strings.Contains(line, "swift-tools-version:") {
                    parts := strings.Split(line, "swift-tools-version:")
                    if len(parts) > 1 {
                        version := strings.TrimSpace(parts[1])
                        if commentIndex := strings.Index(version, "//"); commentIndex != -1 {
                            version = strings.TrimSpace(version[:commentIndex])
                        }
                        return version
                    }
                }
            }
        }
    }
    
    return "5.7"
}

func detectSwiftDependencies(remoteInfo *git.RemoteRepoInfo) []string {
    deps := []string{}
    
    // Анализируем Package.swift на наличие популярных зависимостей
    if content, exists := remoteInfo.GetFileContent("Package.swift"); exists {
        text := strings.ToLower(content)
        
        // Фреймворки
        if strings.Contains(text, "vapor") {
            deps = append(deps, "framework:vapor")
        }
        if strings.Contains(text, "perfect") {
            deps = append(deps, "framework:perfect")
        }
        if strings.Contains(text, "kitura") {
            deps = append(deps, "framework:kitura")
        }
        
        // Базы данных
        if strings.Contains(text, "fluent") || strings.Contains(text, "sqlite") || 
           strings.Contains(text, "postgres") || strings.Contains(text, "mongodb") {
            deps = append(deps, "database")
        }
        
        // UI фреймворки (для iOS/macOS)
        if strings.Contains(text, "swiftui") || strings.Contains(text, "uikit") || 
           strings.Contains(text, "appkit") {
            deps = append(deps, "ui-framework")
        }
        
        // Сетевые библиотеки
        if strings.Contains(text, "alamofire") || strings.Contains(text, "urlsession") {
            deps = append(deps, "networking")
        }
    }
    
    return deps
}

func detectSwiftDependenciesLocal(repoPath string) []string {
    deps := []string{}
    
    packagePath := filepath.Join(repoPath, "Package.swift")
    if exists(packagePath) {
        content, err := os.ReadFile(packagePath)
        if err == nil {
            text := strings.ToLower(string(content))
            
            if strings.Contains(text, "vapor") {
                deps = append(deps, "framework:vapor")
            }
            if strings.Contains(text, "perfect") {
                deps = append(deps, "framework:perfect")
            }
            if strings.Contains(text, "fluent") || strings.Contains(text, "sqlite") {
                deps = append(deps, "database")
            }
            if strings.Contains(text, "swiftui") || strings.Contains(text, "uikit") {
                deps = append(deps, "ui-framework")
            }
        }
    }
    
    return deps
}

func detectSwiftTestsFromMemory(remoteInfo *git.RemoteRepoInfo) bool {
    // Ищем тестовые файлы в Swift проектах
    for _, file := range remoteInfo.Structure {
        if strings.HasSuffix(file, "Test.swift") ||
           strings.HasSuffix(file, "Tests.swift") ||
           strings.Contains(file, "/Tests/") ||
           strings.Contains(file, "/test/") && strings.HasSuffix(file, ".swift") {
            return true
        }
    }
    
    // Проверяем наличие тестовой цели в Package.swift
    if content, exists := remoteInfo.GetFileContent("Package.swift"); exists {
        if strings.Contains(content, "testTarget") || strings.Contains(content, ".testTarget") {
            return true
        }
    }
    
    return false
}

func detectSwiftTestsLocal(repoPath string) bool {
    // Ищем тесты в локальной директории
    patterns := []string{
        filepath.Join(repoPath, "**/*Test.swift"),
        filepath.Join(repoPath, "**/*Tests.swift"),
        filepath.Join(repoPath, "**/Tests/**/*.swift"),
        filepath.Join(repoPath, "**/test/**/*.swift"),
    }
    
    for _, pattern := range patterns {
        matches, _ := filepath.Glob(pattern)
        if len(matches) > 0 {
            return true
        }
    }
    
    // Проверяем Package.swift на наличие тестовой цели
    packagePath := filepath.Join(repoPath, "Package.swift")
    if exists(packagePath) {
        content, err := os.ReadFile(packagePath)
        if err == nil {
            if strings.Contains(string(content), "testTarget") {
                return true
            }
        }
    }
    
    return false
}