package generator

import (
	"fmt"
	"os"
	"strings"

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

	if info.HasDockerfile {
		pipelineContent = addDeployStage(pipelineContent, info, format)
	}
	fmt.Printf("%s, %s, %s, %s, %s, %s \n", info.Language, info.Version, info.Architecture, info.BuildTool, info.TestFramework, info.PackageManager)
	return os.WriteFile(outputFile, []byte(pipelineContent), 0644)
}
func addDeployStage(pipelineContent string, info *analyzer.ProjectInfo, format string) string {
	switch format {
	case "gitlab":
		return addGitLabDeployStage(pipelineContent, info)
	case "jenkins":
		return addJenkinsDeployStage(pipelineContent, info)
	default:
		return addGitHubDeployStage(pipelineContent, info)
	}
}

func addGitLabDeployStage(pipelineContent string, info *analyzer.ProjectInfo) string {
	deployStage := `
deploy:
  stage: deploy
  image: alpine:latest
  before_script:
    - apk add --no-cache openssh-client
    - eval $(ssh-agent -s)
    - echo "$SSH_PRIVATE_KEY" | ssh-add -
    - mkdir -p ~/.ssh
    - chmod 700 ~/.ssh
  script:
    - ssh -o StrictHostKeyChecking=no $DEPLOY_USER@$DEPLOY_SERVER "docker pull $CI_REGISTRY_IMAGE:latest"
    - ssh -o StrictHostKeyChecking=no $DEPLOY_USER@$DEPLOY_SERVER "docker stop ${CI_PROJECT_NAME} || true"
    - ssh -o StrictHostKeyChecking=no $DEPLOY_USER@$DEPLOY_SERVER "docker rm ${CI_PROJECT_NAME} || true"
    - ssh -o StrictHostKeyChecking=no $DEPLOY_USER@$DEPLOY_SERVER "docker run -d --name ${CI_PROJECT_NAME} -p 8080:8080 $CI_REGISTRY_IMAGE:latest"
  environment:
    name: production
    url: https://$DEPLOY_SERVER
  only:
    - main
    - master
  when: manual
`

	if strings.Contains(pipelineContent, "\ndeploy:") {
		deployStart := strings.Index(pipelineContent, "\ndeploy:")
		if deployStart != -1 {
			nextStageStart := strings.Index(pipelineContent[deployStart+1:], "\n  ")
			if nextStageStart == -1 {
				nextStageStart = strings.Index(pipelineContent[deployStart+1:], "\n")
			}

			if nextStageStart != -1 {
				endPos := deployStart + 1 + nextStageStart
				return pipelineContent[:deployStart+1] + deployStage + pipelineContent[endPos:]
			} else {
				return pipelineContent[:deployStart+1] + deployStage
			}
		}
	}

	return pipelineContent + deployStage
}

func addGitHubDeployStage(pipelineContent string, info *analyzer.ProjectInfo) string {
	deployStage := `
  deploy:
    runs-on: ubuntu-latest
    needs: build
    if: github.ref == 'refs/heads/main' || github.ref == 'refs/heads/master'
    environment: production
    steps:
    - name: Checkout code
      uses: actions/checkout@v3
    
    - name: Deploy to server
      uses: appleboy/ssh-action@master
      with:
        host: ${{ secrets.DEPLOY_HOST }}
        username: ${{ secrets.DEPLOY_USER }}
        key: ${{ secrets.DEPLOY_SSH_KEY }}
        script: |
          docker pull ${{ secrets.REGISTRY_URL }}/${{ github.repository }}:latest
          docker stop ${{ github.event.repository.name }} || true
          docker rm ${{ github.event.repository.name }} || true
          docker run -d --name ${{ github.event.repository.name }} -p 8080:8080 ${{ secrets.REGISTRY_URL }}/${{ github.repository }}:latest
`

	if strings.Contains(pipelineContent, "\n  deploy:") {
		deployStart := strings.Index(pipelineContent, "\n  deploy:")
		if deployStart != -1 {
			nextJobStart := strings.Index(pipelineContent[deployStart+1:], "\n  ")
			if nextJobStart != -1 {
				endPos := deployStart + 1 + nextJobStart
				return pipelineContent[:deployStart+1] + deployStage + pipelineContent[endPos:]
			} else {
				return pipelineContent[:deployStart+1] + deployStage
			}
		}
	}

	return pipelineContent + deployStage
}

func addJenkinsDeployStage(pipelineContent string, info *analyzer.ProjectInfo) string {
	deployStage := `
        stage('Deploy to Production') {
            when {
                branch 'main'
            }
            steps {
                script {
                    sshagent(['deploy-ssh-key']) {
                        sh """
                            ssh -o StrictHostKeyChecking=no ${DEPLOY_USER}@${DEPLOY_SERVER} "
                                docker pull your-registry.com/your-project:latest
                                docker stop your-project || true
                                docker rm your-project || true  
                                docker run -d --name your-project -p 8080:8080 your-registry.com/your-project:latest
                            "
                        """
                    }
                }
            }
        }
`

	if strings.Contains(pipelineContent, "stage('Deploy") {
		deployStart := strings.Index(pipelineContent, "stage('Deploy")
		if deployStart != -1 {
			braceCount := 0
			inStage := false
			endPos := deployStart

			for i := deployStart; i < len(pipelineContent); i++ {
				if pipelineContent[i] == '{' {
					braceCount++
					inStage = true
				} else if pipelineContent[i] == '}' {
					braceCount--
					if inStage && braceCount == 0 {
						endPos = i + 1
						break
					}
				}
			}

			if endPos > deployStart {
				return pipelineContent[:deployStart] + deployStage + pipelineContent[endPos:]
			}
		}
	}
	lastBraceIndex := strings.LastIndex(pipelineContent, "    }")
	if lastBraceIndex != -1 {
		before := pipelineContent[:lastBraceIndex]
		after := pipelineContent[lastBraceIndex:]
		return before + deployStage + after
	}

	return pipelineContent + deployStage
}
