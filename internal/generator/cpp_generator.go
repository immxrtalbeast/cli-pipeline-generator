package generator

import (
	"fmt"
	"strings"

	"github.com/immxrtalbeast/pipeline-gen/internal/analyzer"
)

func generateCppPipeline(info *analyzer.ProjectInfo) string {
	var pipeline strings.Builder

	buildTool := info.BuildTool
	if buildTool == "unknown" {
		buildTool = "make" // значение по умолчанию
	}

	pipeline.WriteString(fmt.Sprintf(`name: C++ CI/CD Pipeline (%s)

on:
  push:
    branches: [ main, master, develop ]
  pull_request:
    branches: [ main, master, develop ]

jobs:
`, strings.Title(buildTool)))

	// Job для установки зависимостей и конфигурации
	pipeline.WriteString(`  setup:
    runs-on: ${{ matrix.os }}
    strategy:
      matrix:
        os: [ubuntu-latest, windows-latest, macos-latest]
        build_type: [Debug, Release]
    steps:
    - uses: actions/checkout@v3
    - name: Install dependencies
      run: |
        sudo apt-get update
        sudo apt-get install -y cmake g++ make
      if: matrix.os == 'ubuntu-latest'
`)

	// Добавляем специфичные для build tool шаги
	switch buildTool {
	case "cmake":
		pipeline.WriteString(`    - name: Configure CMake
      run: cmake -B ${{github.workspace}}/build -DCMAKE_BUILD_TYPE=${{ matrix.build_type }}

    - name: Build
      run: cmake --build ${{github.workspace}}/build --config ${{ matrix.build_type }}
`)
	case "make":
		pipeline.WriteString(`    - name: Configure
      run: ./configure
      if: matrix.os == 'ubuntu-latest'

    - name: Build
      run: make -j$(nproc)
`)
	case "autotools":
		pipeline.WriteString(`    - name: Autogen
      run: ./autogen.sh
      if: exists('autogen.sh')

    - name: Configure
      run: ./configure

    - name: Build
      run: make -j$(nproc)
`)
	case "meson":
		pipeline.WriteString(`    - name: Setup Meson
      run: pip install meson ninja

    - name: Configure
      run: meson setup build

    - name: Build
      run: meson compile -C build
`)
	}

	// Job для тестирования
	if info.HasTests {
		pipeline.WriteString(fmt.Sprintf(`  test:
    runs-on: ${{ matrix.os }}
    needs: setup
    strategy:
      matrix:
        os: [ubuntu-latest, windows-latest]
        build_type: [Debug]
    steps:
    - uses: actions/checkout@v3
    - name: Install dependencies
      run: |
        sudo apt-get update
        sudo apt-get install -y cmake g++ make %s
      if: matrix.os == 'ubuntu-latest'
`, getTestDependencies(info.TestFramework)))

		// Конфигурация для тестов
		switch buildTool {
		case "cmake":
			pipeline.WriteString(`    - name: Configure with tests
      run: cmake -B build -DCMAKE_BUILD_TYPE=${{ matrix.build_type }} -DBUILD_TESTING=ON

    - name: Build tests
      run: cmake --build build --target all test

    - name: Run tests
      run: ctest --test-dir build --output-on-failure
`)
		default:
			pipeline.WriteString(`    - name: Build and run tests
      run: |
        make -j$(nproc)
        make test
`)
		}

		// Для конкретных фреймворков тестирования
		switch info.TestFramework {
		case "gtest":
			pipeline.WriteString(`    - name: Install Google Test
      run: sudo apt-get install -y libgtest-dev libgmock-dev
      if: matrix.os == 'ubuntu-latest'
`)
		case "catch2":
			pipeline.WriteString(`    - name: Install Catch2
      run: sudo apt-get install -y catch2
      if: matrix.os == 'ubuntu-latest'
`)
		}
	}

	// Job для статического анализа
	pipeline.WriteString(`  static-analysis:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - name: Install analysis tools
      run: |
        sudo apt-get update
        sudo apt-get install -y clang-tidy cppcheck
    - name: Run cppcheck
      run: cppcheck --enable=all --inconclusive --std=c++${{ env.CPP_STANDARD }} src/ include/
    - name: Run clang-tidy
      run: |
        find src/ include/ -name '*.cpp' -o -name '*.h' | xargs clang-tidy -p build/
      if: env.BUILD_TOOL == 'cmake'
`)

	// Job для сборки релиза
	previousJob := "setup"
	if info.HasTests {
		previousJob = "test"
	}

	pipeline.WriteString(fmt.Sprintf(`  build-release:
    runs-on: ${{ matrix.os }}
    needs: %s
    strategy:
      matrix:
        os: [ubuntu-latest, windows-latest, macos-latest]
    steps:
    - uses: actions/checkout@v3
    - name: Install compiler
      run: |
        sudo apt-get update
        sudo apt-get install -y cmake g++-11
      if: matrix.os == 'ubuntu-latest'
`, previousJob))

	// Сборка релиза в зависимости от build tool
	switch buildTool {
	case "cmake":
		pipeline.WriteString(`    - name: Configure Release
      run: cmake -B build -DCMAKE_BUILD_TYPE=Release -DCMAKE_CXX_COMPILER=g++-11

    - name: Build Release
      run: cmake --build build --config Release --parallel

    - name: Run benchmarks
      run: ./build/benchmarks/benchmark_runner
      if: exists('benchmarks')
`)
	case "make":
		pipeline.WriteString(`    - name: Build Release
      run: make -j$(nproc) CXX=g++-11 CXXFLAGS="-O3 -DNDEBUG"
`)
	}

	pipeline.WriteString(`    - name: Upload artifacts
      uses: actions/upload-artifact@v3
      with:
        name: cpp-${{ matrix.os }}-release
        path: |
          build/**
          *.exe
          *.dll
          *.so
          *.dylib
`)

	// Job для проверки формата кода
	pipeline.WriteString(`  format:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - name: Install clang-format
      run: sudo apt-get install -y clang-format
    - name: Check code format
      run: |
        find src/ include/ -name '*.cpp' -o -name '*.h' | xargs clang-format -i --dry-run -Werror
`)

	// Job для сборки документации (Doxygen)
	pipeline.WriteString(`  docs:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - name: Install Doxygen
      run: sudo apt-get install -y doxygen graphviz
    - name: Generate documentation
      run: doxygen Doxyfile
      if: exists('Doxyfile')
    - name: Upload documentation
      uses: actions/upload-artifact@v3
      with:
        name: cpp-docs
        path: docs/
`)

	// Job для проверки покрытия (если включено)
	if containsDependency(info.Dependencies, "coverage") || info.HasTests {
		pipeline.WriteString(`  coverage:
    runs-on: ubuntu-latest
    needs: test
    steps:
    - uses: actions/checkout@v3
    - name: Install coverage tools
      run: sudo apt-get install -y gcovr lcov
    - name: Build with coverage
      run: |
        cmake -B build -DCMAKE_BUILD_TYPE=Debug -DCMAKE_CXX_FLAGS="--coverage"
        cmake --build build
    - name: Run tests with coverage
      run: |
        ./build/tests/unit_tests
        gcovr --root . --xml-pretty --output coverage.xml
    - name: Upload coverage
      uses: codecov/codecov-action@v3
      with:
        file: coverage.xml
`)
	}

	// Job для кросс-компиляции (если нужно)
	if containsDependency(info.Dependencies, "embedded") || containsDependency(info.Dependencies, "cross-platform") {
		pipeline.WriteString(`  cross-compile:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - name: Install cross-compiler
      run: |
        sudo apt-get update
        sudo apt-get install -y gcc-arm-linux-gnueabihf g++-arm-linux-gnueabihf
    - name: Cross-compile for ARM
      run: |
        cmake -B build-arm -DCMAKE_TOOLCHAIN_FILE=toolchain-arm.cmake
        cmake --build build-arm
`)
	}

	return pipeline.String()
}

// Вспомогательная функция для получения зависимостей тестов
func getTestDependencies(testFramework string) string {
	switch testFramework {
	case "gtest":
		return "libgtest-dev libgmock-dev"
	case "catch2":
		return "catch2"
	case "boost-test":
		return "libboost-test-dev"
	default:
		return ""
	}
}

// Вспомогательная функция для определения стандарта C++
func getCppStandard(version string) string {
	if version == "" {
		return "17"
	}
	return version
}