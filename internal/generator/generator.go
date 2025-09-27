package generator

import (
	"fmt"
	"os"

	"github.com/immxrtalbeast/pipeline-gen/internal/analyzer"
)

func GeneratePipeline(info *analyzer.ProjectInfo, outputFile string, format string) error {
	var pipelineContent string

	// Выбираем генератор в зависимости от формата
	switch format {
	case "gitlab":
		switch info.Language {
		case "go":
			pipelineContent = generateGitLabGoPipeline(info)
		case "python":
			pipelineContent = generateGitLabPythonPipeline(info)
		case "java_gradle", "java_maven":
			pipelineContent = generateGitLabJavaPipeline(info)
		case "javascript":
			pipelineContent = generateGitLabJavaScriptPipeline(info)
		case "csharp":
			pipelineContent = generateGitLabCSharpPipeline(info)
		case "ruby":
			pipelineContent = generateGitLabRubyPipeline(info)
		case "rust":
			pipelineContent = generateGitLabRustPipeline(info)
		case "cpp":
			pipelineContent = generateGitLabCppPipeline(info)
		default:
			return fmt.Errorf("unsupported language: %s", info.Language)
		}
	case "jenkins":
		switch info.Language {
		case "go":
			pipelineContent = generateJenkinsGoPipeline(info)
		case "python":
			pipelineContent = generateJenkinsPythonPipeline(info)
		case "java_gradle", "java_maven":
			pipelineContent = generateJenkinsJavaPipeline(info)
		case "javascript":
			pipelineContent = generateJenkinsJavaScriptPipeline(info)
		case "csharp":
			pipelineContent = generateJenkinsCSharpPipeline(info)
		case "ruby":
			pipelineContent = generateJenkinsRubyPipeline(info)
		case "rust":
			pipelineContent = generateJenkinsRustPipeline(info)
		case "cpp":
			pipelineContent = generateJenkinsCppPipeline(info)
		default:
			return fmt.Errorf("unsupported language: %s", info.Language)
		}
	default:
		// GitHub Actions (по умолчанию)
		switch info.Language {
		case "go":
			pipelineContent = generateGoPipelineActions(info)
		case "python":
			pipelineContent = generatePythonPipeline(info)
		case "java_gradle", "java_maven":
			pipelineContent = generateJavaPipeline(info)
		case "javascript":
			pipelineContent = generateJavaScriptPipeline(info)
		case "csharp":
			pipelineContent = generateCSharpPipeline(info)
		case "ruby":
			pipelineContent = generateRubyPipeline(info)
		case "rust":
			pipelineContent = generateRustPipeline(info)
		case "cpp":
			pipelineContent = generateCppPipeline(info)
		default:
			return fmt.Errorf("unsupported language: %s", info.Language)
		}
	}
	fmt.Println(info)
	return os.WriteFile(outputFile, []byte(pipelineContent), 0644)
}
