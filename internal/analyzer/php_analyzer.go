package analyzer

import (
    "os"
    "path/filepath"
    "strings"
    "regexp"

    "github.com/immxrtalbeast/pipeline-gen/internal/git"
)

func analyzePHPProjectFromMemory(remoteInfo *git.RemoteRepoInfo, info *ProjectInfo) {
    info.BuildTool = detectPHPBuildTool(remoteInfo)
    info.TestFramework = detectPHPTestFramework(remoteInfo)
    info.Version = detectPHPVersion(remoteInfo)
    info.Dependencies = detectPHPDependencies(remoteInfo)
    info.HasTests = detectPHPTestsFromMemory(remoteInfo)
    info.PackageManager = detectPHPPackageManager(remoteInfo)
}

func analyzePHPProject(repoPath string, info *ProjectInfo) error {
    info.BuildTool = detectPHPBuildToolLocal(repoPath)
    info.TestFramework = detectPHPTestFrameworkLocal(repoPath)
    info.Version = detectPHPVersionLocal(repoPath)
    info.Dependencies = detectPHPDependenciesLocal(repoPath)
    info.HasTests = detectPHPTestsLocal(repoPath)
    info.PackageManager = detectPHPPackageManagerLocal(repoPath)
    return nil
}

// Вспомогательные функции для анализа PHP
func detectPHPBuildTool(remoteInfo *git.RemoteRepoInfo) string {
    if remoteInfo.HasFile("composer.json") {
        return "composer"
    }
    if remoteInfo.HasFile("package.json") && remoteInfo.HasFile("webpack.mix.js") {
        return "laravel-mix"
    }
    if remoteInfo.HasFile("artisan") {
        return "laravel"
    }
    if remoteInfo.HasFile("symfony") {
        return "symfony"
    }
    return "php" // простой PHP проект
}

func detectPHPBuildToolLocal(repoPath string) string {
    if exists(filepath.Join(repoPath, "composer.json")) {
        return "composer"
    }
    if exists(filepath.Join(repoPath, "artisan")) {
        return "laravel"
    }
    if exists(filepath.Join(repoPath, "symfony")) {
        return "symfony"
    }
    if exists(filepath.Join(repoPath, "package.json")) && exists(filepath.Join(repoPath, "webpack.mix.js")) {
        return "laravel-mix"
    }
    return "php"
}

func detectPHPPackageManager(remoteInfo *git.RemoteRepoInfo) string {
    if remoteInfo.HasFile("composer.json") {
        return "composer"
    }
    return "none"
}

func detectPHPPackageManagerLocal(repoPath string) string {
    if exists(filepath.Join(repoPath, "composer.json")) {
        return "composer"
    }
    return "none"
}

func detectPHPTestFramework(remoteInfo *git.RemoteRepoInfo) string {
    // Проверяем composer.json на наличие тестовых фреймворков
    if content, exists := remoteInfo.GetFileContent("composer.json"); exists {
        text := strings.ToLower(content)
        if strings.Contains(text, "phpunit/phpunit") {
            return "phpunit"
        }
        if strings.Contains(text, "codeception/codeception") {
            return "codeception"
        }
        if strings.Contains(text, "behat/behat") {
            return "behat"
        }
        if strings.Contains(text, "phpstan/phpstan") {
            return "phpstan" // статический анализ
        }
        if strings.Contains(text, "pestphp/pest") {
            return "pest"
        }
    }

    // Проверяем наличие конфигурационных файлов тестов
    if remoteInfo.HasFile("phpunit.xml") || remoteInfo.HasFile("phpunit.xml.dist") {
        return "phpunit"
    }
    if remoteInfo.HasFile("codeception.yml") {
        return "codeception"
    }
    if remoteInfo.HasFile("behat.yml") {
        return "behat"
    }
    if remoteInfo.HasFile("pest.yml") {
        return "pest"
    }

    // Ищем тестовые файлы
    for _, file := range remoteInfo.Structure {
        if strings.Contains(file, "Test.php") && 
           (strings.Contains(file, "/tests/") || strings.Contains(file, "/Tests/")) {
            return "phpunit" // предположительно
        }
    }

    return "phpunit" // по умолчанию
}

func detectPHPTestFrameworkLocal(repoPath string) string {
    // Проверяем composer.json
    composerPath := filepath.Join(repoPath, "composer.json")
    if exists(composerPath) {
        content, err := os.ReadFile(composerPath)
        if err == nil {
            text := strings.ToLower(string(content))
            if strings.Contains(text, "phpunit/phpunit") {
                return "phpunit"
            }
            if strings.Contains(text, "codeception/codeception") {
                return "codeception"
            }
            if strings.Contains(text, "pestphp/pest") {
                return "pest"
            }
        }
    }

    // Проверяем конфигурационные файлы
    if exists(filepath.Join(repoPath, "phpunit.xml")) || 
       exists(filepath.Join(repoPath, "phpunit.xml.dist")) {
        return "phpunit"
    }
    if exists(filepath.Join(repoPath, "codeception.yml")) {
        return "codeception"
    }
    if exists(filepath.Join(repoPath, "pest.yml")) {
        return "pest"
    }

    // Ищем тестовые файлы
    patterns := []string{
        filepath.Join(repoPath, "**/tests/**/*Test.php"),
        filepath.Join(repoPath, "**/Tests/**/*Test.php"),
        filepath.Join(repoPath, "**/*Test.php"),
    }

    for _, pattern := range patterns {
        matches, _ := filepath.Glob(pattern)
        if len(matches) > 0 {
            return "phpunit"
        }
    }

    return "phpunit"
}

func detectPHPVersion(remoteInfo *git.RemoteRepoInfo) string {
    // Анализируем версию PHP из composer.json
    if content, exists := remoteInfo.GetFileContent("composer.json"); exists {
        // Ищем "php": "^7.4|^8.0" и т.д.
        re := regexp.MustCompile(`"php"\s*:\s*"([^"]+)"`)
        matches := re.FindStringSubmatch(content)
        if len(matches) > 1 {
            versionConstraint := matches[1]
            // Упрощенное извлечение версии (берем первую цифру после ^)
            if strings.Contains(versionConstraint, "^") {
                parts := strings.Split(versionConstraint, "^")
                if len(parts) > 1 {
                    return strings.Split(parts[1], "|")[0] // берем первую версию
                }
            }
            return "8.1" // fallback
        }
    }

    // Проверяем наличие файла .php-version
    if content, exists := remoteInfo.GetFileContent(".php-version"); exists {
        return strings.TrimSpace(content)
    }

    // Проверяем наличие .tool-versions (asdf)
    if content, exists := remoteInfo.GetFileContent(".tool-versions"); exists {
        lines := strings.Split(content, "\n")
        for _, line := range lines {
            if strings.Contains(line, "php") {
                parts := strings.Fields(line)
                if len(parts) > 1 {
                    return parts[1]
                }
            }
        }
    }

    return "8.1" // версия по умолчанию
}

func detectPHPVersionLocal(repoPath string) string {
    // Проверяем composer.json
    composerPath := filepath.Join(repoPath, "composer.json")
    if exists(composerPath) {
        content, err := os.ReadFile(composerPath)
        if err == nil {
            re := regexp.MustCompile(`"php"\s*:\s*"([^"]+)"`)
            matches := re.FindStringSubmatch(string(content))
            if len(matches) > 1 {
                versionConstraint := matches[1]
                if strings.Contains(versionConstraint, "^") {
                    parts := strings.Split(versionConstraint, "^")
                    if len(parts) > 1 {
                        return strings.Split(parts[1], "|")[0]
                    }
                }
            }
        }
    }

    // Проверяем .php-version
    phpVersionPath := filepath.Join(repoPath, ".php-version")
    if exists(phpVersionPath) {
        content, err := os.ReadFile(phpVersionPath)
        if err == nil {
            return strings.TrimSpace(string(content))
        }
    }

    // Проверяем .tool-versions
    toolVersionsPath := filepath.Join(repoPath, ".tool-versions")
    if exists(toolVersionsPath) {
        content, err := os.ReadFile(toolVersionsPath)
        if err == nil {
            lines := strings.Split(string(content), "\n")
            for _, line := range lines {
                if strings.Contains(line, "php") {
                    parts := strings.Fields(line)
                    if len(parts) > 1 {
                        return parts[1]
                    }
                }
            }
        }
    }

    return "8.1"
}

func detectPHPDependencies(remoteInfo *git.RemoteRepoInfo) []string {
    deps := []string{}

    // Анализируем composer.json на наличие популярных фреймворков и библиотек
    if content, exists := remoteInfo.GetFileContent("composer.json"); exists {
        text := strings.ToLower(content)

        // Фреймворки
        if strings.Contains(text, `"laravel/framework"`) || strings.Contains(text, `"illuminate/`) {
            deps = append(deps, "framework:laravel")
        }
        if strings.Contains(text, `"symfony/`) {
            deps = append(deps, "framework:symfony")
        }
        if strings.Contains(text, `"codeigniter4/framework"`) {
            deps = append(deps, "framework:codeigniter")
        }

        // Базы данных
        if strings.Contains(text, `"doctrine/orm"`) || strings.Contains(text, `"illuminate/database"`) {
            deps = append(deps, "database:orm")
        }
        if strings.Contains(text, `"mongodb/mongodb"`) {
            deps = append(deps, "database:mongodb")
        }

        // API
        if strings.Contains(text, `"laravel/sanctum"`) || strings.Contains(text, `"tymon/jwt-auth"`) {
            deps = append(deps, "api:authentication")
        }

        // Frontend
        if strings.Contains(text, `"laravel/ui"`) || remoteInfo.HasFile("webpack.mix.js") {
            deps = append(deps, "frontend:build-tools")
        }

        // Тестирование
        if strings.Contains(text, `"phpunit/phpunit"`) {
            deps = append(deps, "testing:phpunit")
        }
        if strings.Contains(text, `"mockery/mockery"`) {
            deps = append(deps, "testing:mockery")
        }

        // Анализ кода
        if strings.Contains(text, `"phpstan/phpstan"`) {
            deps = append(deps, "quality:static-analysis")
        }
        if strings.Contains(text, `"squizlabs/php_codesniffer"`) {
            deps = append(deps, "quality:code-style")
        }
    }

    // Проверяем наличие специфичных файлов фреймворков
    if remoteInfo.HasFile("artisan") {
        deps = append(deps, "framework:laravel")
    }
    if remoteInfo.HasFile("symfony") {
        deps = append(deps, "framework:symfony")
    }

    return deps
}

func detectPHPDependenciesLocal(repoPath string) []string {
    deps := []string{}

    composerPath := filepath.Join(repoPath, "composer.json")
    if exists(composerPath) {
        content, err := os.ReadFile(composerPath)
        if err == nil {
            text := strings.ToLower(string(content))

            if strings.Contains(text, `"laravel/framework"`) {
                deps = append(deps, "framework:laravel")
            }
            if strings.Contains(text, `"symfony/`) {
                deps = append(deps, "framework:symfony")
            }
            if strings.Contains(text, `"doctrine/orm"`) {
                deps = append(deps, "database:orm")
            }
            if strings.Contains(text, `"phpunit/phpunit"`) {
                deps = append(deps, "testing:phpunit")
            }
        }
    }

    // Проверяем файлы фреймворков
    if exists(filepath.Join(repoPath, "artisan")) {
        deps = append(deps, "framework:laravel")
    }
    if exists(filepath.Join(repoPath, "symfony")) {
        deps = append(deps, "framework:symfony")
    }

    return deps
}

func detectPHPTestsFromMemory(remoteInfo *git.RemoteRepoInfo) bool {
    // Ищем тестовые файлы в PHP проектах
    for _, file := range remoteInfo.Structure {
        if (strings.HasSuffix(file, "Test.php") || strings.HasSuffix(file, "Test.php")) &&
           (strings.Contains(file, "/tests/") || strings.Contains(file, "/Tests/")) {
            return true
        }
    }

    // Проверяем наличие конфигурационных файлов тестов
    if remoteInfo.HasFile("phpunit.xml") || remoteInfo.HasFile("phpunit.xml.dist") ||
       remoteInfo.HasFile("codeception.yml") || remoteInfo.HasFile("behat.yml") ||
       remoteInfo.HasFile("pest.yml") {
        return true
    }

    // Проверяем composer.json на наличие тестовых зависимостей
    if content, exists := remoteInfo.GetFileContent("composer.json"); exists {
        if strings.Contains(content, "phpunit") || strings.Contains(content, "codeception") ||
           strings.Contains(content, "behat") || strings.Contains(content, "pest") {
            return true
        }
    }

    return false
}

func detectPHPTestsLocal(repoPath string) bool {
    // Ищем тесты в локальной директории
    patterns := []string{
        filepath.Join(repoPath, "**/tests/**/*Test.php"),
        filepath.Join(repoPath, "**/Tests/**/*Test.php"),
        filepath.Join(repoPath, "**/*Test.php"),
    }

    for _, pattern := range patterns {
        matches, _ := filepath.Glob(pattern)
        if len(matches) > 0 {
            return true
        }
    }

    // Проверяем конфигурационные файлы
    testConfigs := []string{
        "phpunit.xml",
        "phpunit.xml.dist", 
        "codeception.yml",
        "behat.yml",
        "pest.yml",
    }

    for _, config := range testConfigs {
        if exists(filepath.Join(repoPath, config)) {
            return true
        }
    }

    return false
}