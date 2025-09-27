/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/immxrtalbeast/pipeline-gen/internal/analyzer"
	"github.com/immxrtalbeast/pipeline-gen/internal/generator"
	"github.com/spf13/cobra"
)

var (
	repoPath   string
	remoteRepo string
	branch     string
	outputFile string
	format     string
	listFile   string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "pipeline-gen",
	Short: "Generate CI/CD pipelines for projects",
	Long:  `A CLI tool to generate CI/CD pipelines based on project language and architecture`,
	Run: func(cmd *cobra.Command, args []string) {
		var projectInfo *analyzer.ProjectInfo
		var err error
		if repoPath != "" {
			projectInfo, err = analyzer.AnalyzeLocalRepo(repoPath)
			if err != nil {
				fmt.Printf("Error analyzing local repository: %v\n", err)
				os.Exit(1)
			}
		} else if remoteRepo != "" {
			fmt.Printf("Analyzing remote repository: %s (branch: %s)\n", remoteRepo, branch)
			projectInfo, err = analyzer.AnalyzeRemoteRepo(remoteRepo, branch)
			if err != nil {
				fmt.Printf("Error analyzing remote repository: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("✓ Repository analyzed successfully in memory")
		} else if listFile != "" {
			err := generator.ProcessRepositoryList(listFile, branch)
			if err != nil {
				fmt.Printf("Error processing repository list: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("✓ Pipelines generated successfully for all repositories in %s\n", listFile)
			return
		} else {
			fmt.Println("Please specify either --repo, --remote or --list")
			cmd.Help()
			os.Exit(1)
		}

		err = generator.GeneratePipeline(projectInfo, outputFile, format)
		if err != nil {
			fmt.Printf("Error generating pipeline: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✓ Pipeline generated successfully: %s\n", outputFile)
		fmt.Printf("✓ Format: %s\n", format)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&repoPath, "repo", "r", "", "Path to local repository")
	rootCmd.Flags().StringVarP(&listFile, "list", "l", "", "Path to txt file with links to repositories")
	rootCmd.Flags().StringVarP(&remoteRepo, "remote", "R", "", "URL of remote git repository")
	rootCmd.Flags().StringVarP(&branch, "branch", "b", "main", "Branch to analyze")
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "pipeline.yml", "Output pipeline file")
	rootCmd.Flags().StringVarP(&format, "format", "f", "github", "CI/CD format (github, gitlab, jenkins)")
}
