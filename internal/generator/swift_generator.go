package generator

import (
    "fmt"
    "strings"

    "github.com/immxrtalbeast/pipeline-gen/internal/analyzer"
)

func generateSwiftPipeline(info *analyzer.ProjectInfo) string {
    var pipeline strings.Builder

    buildTool := "spm" // Swift Package Manager по умолчанию
    if info.BuildTool == "xcodebuild" {
        buildTool = "xcodebuild"
    }

    pipeline.WriteString(fmt.Sprintf(`name: Swift CI/CD Pipeline (%s)

on:
  push:
    branches: [ main, master, develop ]
  pull_request:
    branches: [ main, master, develop ]

jobs:
`, buildTool))

    // Job для тестов
    if info.HasTests {
        pipeline.WriteString(`  test:
    runs-on: macos-latest
    strategy:
      matrix:
        xcode: ['15.0', '14.3']
        swift-version: [`)

        // Добавляем версии Swift
        if info.Version != "" && info.Version != "5.7" {
            pipeline.WriteString(fmt.Sprintf(" '%s', '5.7', '5.8' ", info.Version))
        } else {
            pipeline.WriteString(" '5.7', '5.8', '5.9' ")
        }

        pipeline.WriteString(`]
    steps:
    - uses: actions/checkout@v3
    - name: Select Xcode ${{ matrix.xcode }}
      run: sudo xcode-select -s /Applications/Xcode_${{ matrix.xcode }}.app
`)

        if buildTool == "spm" {
            pipeline.WriteString(`    - name: Resolve dependencies
      run: swift package resolve
      
    - name: Build with SwiftPM
      run: swift build
      
    - name: Run tests with SwiftPM
      run: swift test
`)

            // Для проектов с Vapor
            if containsDependency(info.Dependencies, "vapor") {
                pipeline.WriteString(`    - name: Run Vapor tests
      run: swift test --enable-test-discovery
`)
            }

        } else {
            // Для Xcode проектов
            pipeline.WriteString(`    - name: Build with xcodebuild
      run: xcodebuild -scheme MyApp -destination 'platform=iOS Simulator,name=iPhone 14' build
      
    - name: Run tests with xcodebuild
      run: xcodebuild test -scheme MyApp -destination 'platform=iOS Simulator,name=iPhone 14'
`)
        }

        // Добавляем отчет о покрытии если есть соответствующие зависимости
        if containsDependency(info.Dependencies, "coverage") {
            pipeline.WriteString(`    - name: Generate code coverage
      run: xcrun llvm-cov export -format="lcov" .build/debug/MyAppPackageTests.xctest/Contents/MacOS/MyAppPackageTests -instr-profile .build/debug/codecov/default.profdata > lcov.info
      
    - name: Upload coverage to Codecov
      uses: codecov/codecov-action@v3
      with:
        file: lcov.info
`)
        }

    } else {
        // Если тестов нет - простая проверка сборки
        pipeline.WriteString(`  verify:
    runs-on: macos-latest
    steps:
    - uses: actions/checkout@v3
    - name: Select Xcode
      run: sudo xcode-select -s /Applications/Xcode_15.0.app
`)

        if buildTool == "spm" {
            pipeline.WriteString(`    - name: Verify Swift package
      run: swift package resolve && swift build
`)
        } else {
            pipeline.WriteString(`    - name: Verify Xcode project
      run: xcodebuild -scheme MyApp -destination 'generic/platform=iOS' build
`)
        }
    }

    // Job для сборки
    previousJob := "test"
    if !info.HasTests {
        previousJob = "verify"
    }

    pipeline.WriteString(fmt.Sprintf(`  build:
    runs-on: macos-latest
    needs: %s
    steps:
    - uses: actions/checkout@v3
    - name: Select Xcode 15.0
      run: sudo xcode-select -s /Applications/Xcode_15.0.app
`, previousJob))

    if buildTool == "spm" {
        pipeline.WriteString(`    - name: Build release with SwiftPM
      run: swift build -c release
      
    - name: Create artifacts directory
      run: mkdir -p artifacts
      
    - name: Archive build products
      run: |
        cp -R .build/release artifacts/
        find .build/release -name "*.dylib" -o -name "*.a" | xargs -I {} cp {} artifacts/
      
    - name: Upload artifacts
      uses: actions/upload-artifact@v3
      with:
        name: swift-artifacts
        path: artifacts/
`)

        // Для исполняемых продуктов
        pipeline.WriteString(`    - name: Detect executable products
      run: |
        EXECUTABLES=$(swift package describe --type json | grep '"type":"executable"' | wc -l)
        echo "EXECUTABLES=$EXECUTABLES" >> $GITHUB_ENV
        
    - name: Package executables
      if: env.EXECUTABLES != '0'
      run: |
        swift package archive-source
        mv *.zip artifacts/ 2>/dev/null || true
`)

    } else {
        pipeline.WriteString(`    - name: Archive with xcodebuild
      run: xcodebuild -scheme MyApp -configuration Release -archivePath ./build/MyApp.xcarchive archive
      
    - name: Export IPA
      run: xcodebuild -exportArchive -archivePath ./build/MyApp.xcarchive -exportOptionsPlist ExportOptions.plist -exportPath ./build
      
    - name: Upload artifacts
      uses: actions/upload-artifact@v3
      with:
        name: xcode-artifacts
        path: build/
`)
    }

    // Job для линтинга (SwiftLint)
    if containsDependency(info.Dependencies, "swiftlint") || fileExistsInStructure(info.Structure, ".swiftlint.yml") {
        pipeline.WriteString(`  lint:
    runs-on: macos-latest
    steps:
    - uses: actions/checkout@v3
    - name: Install SwiftLint
      run: brew install swiftlint
    - name: Run SwiftLint
      run: swiftlint
`)
    }

    // Job для документации (Swift-DocC)
    if containsDependency(info.Dependencies, "documentation") {
        pipeline.WriteString(`  documentation:
    runs-on: macos-latest
    needs: build
    steps:
    - uses: actions/checkout@v3
    - name: Generate documentation
      run: swift package generate-documentation
    - name: Upload documentation
      uses: actions/upload-artifact@v3
      with:
        name: documentation
        path: .build/documentation/
`)
    }

    // Job для публикации в Swift Package Index
    if containsDependency(info.Dependencies, "spi") || strings.Contains(info.RepositoryURL, "github.com") {
        pipeline.WriteString(`  publish-spi:
    runs-on: macos-latest
    needs: build
    if: github.event_name == 'push' && contains(github.ref, 'refs/tags/')
    steps:
    - uses: actions/checkout@v3
    - name: Validate for Swift Package Index
      run: |
        swift package diagnose-api-breaking-changes $(git describe --abbrev=0 --tags)
      env:
        SPI_TOKEN: ${{ secrets.SPI_TOKEN }}
`)
    }

    return pipeline.String()
}

// Вспомогательная функция для проверки наличия файла в структуре
func fileExistsInStructure(structure []string, filename string) bool {
    for _, file := range structure {
        if strings.Contains(file, filename) {
            return true
        }
    }
    return false
}