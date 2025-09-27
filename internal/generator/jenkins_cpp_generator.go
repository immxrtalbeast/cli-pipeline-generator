package generator

import (
	"strings"

	"github.com/immxrtalbeast/pipeline-gen/internal/analyzer"
)

func generateJenkinsCppPipeline(info *analyzer.ProjectInfo) string {
	var pipeline strings.Builder

	buildTool := info.BuildTool
	if buildTool == "unknown" {
		buildTool = "make" // значение по умолчанию
	}

	pipeline.WriteString(`pipeline {
    agent any
    
    environment {
        CXX_STANDARD = '`)

	if info.Version != "" {
		pipeline.WriteString(info.Version)
	} else {
		pipeline.WriteString("17")
	}

	pipeline.WriteString(`'
        BUILD_TYPE = 'Release'
    }
    
    tools {
        cmake 'CMake'
        gcc 'GCC'
    }
    
    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }
        
        stage('Setup') {
            steps {
                sh 'g++ --version'
                sh 'cmake --version'
                sh 'make --version'
            }
        }
        
        stage('Dependencies') {
            steps {
                sh 'apt-get update'
                sh 'apt-get install -y build-essential cmake g++ make`)

	// Добавляем зависимости для тестов
	if info.HasTests {
		pipeline.WriteString(getTestDependencies(info.TestFramework))
	}

	pipeline.WriteString(`'
            }
        }
        
        stage('Configure') {
            steps {`)

	// Конфигурация в зависимости от build tool
	switch buildTool {
	case "cmake":
		pipeline.WriteString(`
                sh 'cmake -B build -DCMAKE_BUILD_TYPE=$BUILD_TYPE -DCMAKE_CXX_STANDARD=$CXX_STANDARD'`)
	case "autotools":
		pipeline.WriteString(`
                sh 'test -f autogen.sh && ./autogen.sh || autoreconf -i'
                sh './configure'`)
	case "meson":
		pipeline.WriteString(`
                sh 'pip3 install meson'
                sh 'meson setup build'`)
	default:
		pipeline.WriteString(`
                sh './configure'`)
	}

	pipeline.WriteString(`
            }
        }
        
        stage('Build') {
            steps {`)

	// Сборка в зависимости от build tool
	switch buildTool {
	case "cmake":
		pipeline.WriteString(`
                sh 'cmake --build build --config $BUILD_TYPE'`)
	case "meson":
		pipeline.WriteString(`
                sh 'meson compile -C build'`)
	default:
		pipeline.WriteString(`
                sh 'make -j$(nproc)'`)
	}

	pipeline.WriteString(`
            }
        }
`)

	if info.HasTests {
		pipeline.WriteString(`
        stage('Test') {
            steps {`)

		switch buildTool {
		case "cmake":
			pipeline.WriteString(`
                sh 'ctest --test-dir build --output-on-failure'`)
		default:
			pipeline.WriteString(`
                sh 'make test'`)
		}

		pipeline.WriteString(`
            }
            post {
                always {
                    publishTestResults testResultsPattern: '**/test-results.xml'
                }
            }
        }
`)
	}

	pipeline.WriteString(`
        stage('Static Analysis') {
            steps {
                sh 'apt-get install -y clang-tidy cppcheck'
                sh 'cppcheck --enable=all --inconclusive --std=c++$CXX_STANDARD src/ include/ || true'
                sh 'find src/ include/ -name "*.cpp" -o -name "*.h" | xargs clang-tidy -p build/ || true'
            }
        }
        
        stage('Code Format') {
            steps {
                sh 'apt-get install -y clang-format'
                sh 'find src/ include/ -name "*.cpp" -o -name "*.h" | xargs clang-format -i --dry-run -Werror || true'
            }
        }
`)

	if info.HasTests {
		pipeline.WriteString(`
        stage('Coverage') {
            steps {
                sh 'apt-get install -y gcovr lcov'
                sh 'cmake -B build-coverage -DCMAKE_BUILD_TYPE=Debug -DCMAKE_CXX_FLAGS="--coverage"'
                sh 'cmake --build build-coverage'
                sh './build-coverage/tests/unit_tests'
                sh 'gcovr --root . --xml-pretty --output coverage.xml'
            }
            post {
                always {
                    publishCoverage adapters: [coberturaAdapter('coverage.xml')], sourceFileResolver: sourceFiles('STORE_LAST_BUILD')
                }
            }
        }
`)
	}

	pipeline.WriteString(`
        stage('Package') {
            steps {`)

	switch buildTool {
	case "cmake":
		pipeline.WriteString(`
                sh 'cmake --build build --target package'`)
	default:
		pipeline.WriteString(`
                sh 'make package'`)
	}

	pipeline.WriteString(`
                archiveArtifacts artifacts: '*.deb,*.rpm,*.tar.gz,*.zip,build/**,dist/**', fingerprint: true
            }
        }
`)

	// Documentation stage
	pipeline.WriteString(`
        stage('Documentation') {
            steps {
                sh 'apt-get install -y doxygen graphviz'
                sh 'doxygen Doxyfile || true'
                archiveArtifacts artifacts: 'docs/**,html/**,latex/**', fingerprint: true
            }
        }
`)

	// Cross-compilation if needed
	if containsDependency(info.Dependencies, "embedded") || containsDependency(info.Dependencies, "cross-platform") {
		pipeline.WriteString(`
        stage('Cross-Compile') {
            steps {
                sh 'apt-get install -y gcc-arm-linux-gnueabihf g++-arm-linux-gnueabihf'
                sh 'cmake -B build-arm -DCMAKE_TOOLCHAIN_FILE=toolchain-arm.cmake'
                sh 'cmake --build build-arm'
                archiveArtifacts artifacts: 'build-arm/**', fingerprint: true
            }
        }
`)
	}

	pipeline.WriteString(`    }
    
    post {
        always {
            cleanWs()
        }
        success {
            echo 'C++ Pipeline completed successfully!'
        }
        failure {
            echo 'C++ Pipeline failed!'
        }
    }
}`)

	return pipeline.String()
}
