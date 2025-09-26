package generator

import (
	"fmt"
	"os"

	"github.com/immxrtalbeast/pipeline-gen/internal/analyzer"
)

func GeneratePipeline(info *analyzer.ProjectInfo, outputFile string) error {
	var pipelineContent string

	switch info.Language {
	case "go":
		pipelineContent = generateGoPipelineActions(info)
	case "python":
		pipelineContent = generatePythonPipeline(info)
	case "java_gradle", "java_maven":
		pipelineContent = generateJavaPipeline(info)
	default:
		return fmt.Errorf("unsupported language: %s", info.Language)
	}
	fmt.Println(info)
	return os.WriteFile(outputFile, []byte(pipelineContent), 0644)
}
