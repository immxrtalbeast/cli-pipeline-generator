package generator

import (
	"strings"

	"github.com/immxrtalbeast/pipeline-gen/internal/analyzer"
)

func generateGitLabCppPipeline(info *analyzer.ProjectInfo) string {
	var pipeline strings.Builder

	buildTool := info.BuildTool
	if buildTool == "unknown" {
		buildTool = "make" // значение по умолчанию
	}

	pipeline.WriteString(`stages:
  - setup
  - test
  - analysis
  - build
  - docs
  - deploy

variables:
  BUILD_TYPE: "Release"
  CXX_STANDARD: '`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("17")
	}

	pipeline.WriteString(`'

setup:
  stage: setup
  image: ubuntu:latest
  before_script:
    - apt-get update
    - apt-get install -y cmake g++ make build-essential
  script:`)

	// Добавляем специфичные для build tool шаги
	switch buildTool {
	case "cmake":
		pipeline.WriteString(`
    - cmake -B build -DCMAKE_BUILD_TYPE=$BUILD_TYPE -DCMAKE_CXX_STANDARD=$CXX_STANDARD
    - cmake --build build --config $BUILD_TYPE`)
	case "make":
		pipeline.WriteString(`
    - ./configure
    - make -j$(nproc)`)
	case "autotools":
		pipeline.WriteString(`
    - test -f autogen.sh && ./autogen.sh || autoreconf -i
    - ./configure
    - make -j$(nproc)`)
	case "meson":
		pipeline.WriteString(`
    - apt-get install -y python3-pip ninja-build
    - pip3 install meson
    - meson setup build
    - meson compile -C build`)
	}

	pipeline.WriteString(`
  artifacts:
    paths:
      - build/
    expire_in: 1 hour

`)

	if info.HasTests {
		pipeline.WriteString(`test:
  stage: test
  image: ubuntu:latest
  before_script:
    - apt-get update
    - apt-get install -y cmake g++ make build-essential `)

		pipeline.WriteString(getTestDependencies(info.TestFramework))

		pipeline.WriteString(`
  script:`)

		// Конфигурация для тестов
		switch buildTool {
		case "cmake":
			pipeline.WriteString(`
    - cmake -B build -DCMAKE_BUILD_TYPE=Debug -DBUILD_TESTING=ON
    - cmake --build build --target all test
    - ctest --test-dir build --output-on-failure`)
		default:
			pipeline.WriteString(`
    - make -j$(nproc) test
    - make test`)
		}

		pipeline.WriteString(`
  dependencies:
    - setup
  artifacts:
    paths:
      - test_results/
    expire_in: 1 week

`)

		// Coverage job
		pipeline.WriteString(`coverage:
  stage: test
  image: ubuntu:latest
  before_script:
    - apt-get update
    - apt-get install -y cmake g++ make build-essential gcovr lcov
  script:
    - cmake -B build -DCMAKE_BUILD_TYPE=Debug -DCMAKE_CXX_FLAGS="--coverage"
    - cmake --build build
    - ./build/tests/unit_tests
    - gcovr --root . --xml-pretty --output coverage.xml
  artifacts:
    reports:
      coverage_report:
        coverage_format: cobertura
        path: coverage.xml
    paths:
      - coverage.xml
    expire_in: 1 week
  coverage: '/lines: \d+\.\d+/'

`)
	}

	pipeline.WriteString(`static_analysis:
  stage: analysis
  image: ubuntu:latest
  before_script:
    - apt-get update
    - apt-get install -y clang-tidy cppcheck
  script:
    - cppcheck --enable=all --inconclusive --std=c++$CXX_STANDARD src/ include/
    - find src/ include/ -name '*.cpp' -o -name '*.h' | xargs clang-tidy -p build/
  dependencies:
    - setup
  artifacts:
    paths:
      - analysis_report/
    expire_in: 1 week

format_check:
  stage: analysis
  image: ubuntu:latest
  before_script:
    - apt-get update
    - apt-get install -y clang-format
  script:
    - find src/ include/ -name '*.cpp' -o -name '*.h' | xargs clang-format -i --dry-run -Werror
  dependencies:
    - setup

`)

	pipeline.WriteString(`build_release:
  stage: build
  image: ubuntu:latest
  before_script:
    - apt-get update
    - apt-get install -y cmake g++-11 make build-essential
  script:`)

	// Сборка релиза в зависимости от build tool
	switch buildTool {
	case "cmake":
		pipeline.WriteString(`
    - cmake -B build -DCMAKE_BUILD_TYPE=Release -DCMAKE_CXX_COMPILER=g++-11
    - cmake --build build --config Release --parallel`)
	case "make":
		pipeline.WriteString(`
    - make -j$(nproc) CXX=g++-11 CXXFLAGS="-O3 -DNDEBUG"`)
	}

	pipeline.WriteString(`
  dependencies:
    - setup
  artifacts:
    paths:
      - build/
      - *.exe
      - *.so
      - *.a
    expire_in: 1 week

`)

	// Multi-platform builds
	pipeline.WriteString(`build_linux:
  stage: build
  image: ubuntu:latest
  before_script:
    - apt-get update
    - apt-get install -y cmake g++ make build-essential
  script:
    - cmake -B build-linux -DCMAKE_BUILD_TYPE=Release
    - cmake --build build-linux
  artifacts:
    paths:
      - build-linux/
    expire_in: 1 week

build_windows:
  stage: build
  image: mcr.microsoft.com/windows:latest
  before_script:
    - choco install visualstudio2019buildtools -y
    - choco install cmake -y
  script:
    - cmake -B build-windows -DCMAKE_BUILD_TYPE=Release
    - cmake --build build-windows
  artifacts:
    paths:
      - build-windows/
    expire_in: 1 week

`)

	// Documentation job
	pipeline.WriteString(`docs:
  stage: docs
  image: ubuntu:latest
  before_script:
    - apt-get update
    - apt-get install -y doxygen graphviz
  script:
    - doxygen Doxyfile
  dependencies:
    - build_release
  artifacts:
    paths:
      - docs/
    expire_in: 1 week
  only:
    - main
    - master

`)

	// Cross-compilation if needed
	if containsDependency(info.Dependencies, "embedded") || containsDependency(info.Dependencies, "cross-platform") {
		pipeline.WriteString(`cross_compile_arm:
  stage: build
  image: ubuntu:latest
  before_script:
    - apt-get update
    - apt-get install -y gcc-arm-linux-gnueabihf g++-arm-linux-gnueabihf
  script:
    - cmake -B build-arm -DCMAKE_TOOLCHAIN_FILE=toolchain-arm.cmake
    - cmake --build build-arm
  artifacts:
    paths:
      - build-arm/
    expire_in: 1 week

`)
	}

	pipeline.WriteString(`deploy:
  stage: deploy
  image: alpine:latest
  script:
    - echo "Deploying C++ application"
  only:
    - main
    - master
`)

	return pipeline.String()
}
