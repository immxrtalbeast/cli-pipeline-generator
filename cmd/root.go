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
	tempDir    string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "pipeline-gen",
	Short: "Generate CI/CD pipelines for projects",
	Long:  `A  CLI tool to generate CI/CD pipelines based on project language and architecture`,
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
			// Анализ удаленного репозитория в памяти
			fmt.Printf("Analyzing remote repository: %s (branch: %s)\n", remoteRepo, branch)
			projectInfo, err = analyzer.AnalyzeRemoteRepo(remoteRepo, branch)
			if err != nil {
				fmt.Printf("Error analyzing remote repository: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("✓ Repository analyzed successfully in memory")
		} else {
			fmt.Println("Please specify either --repo or --webhook")
			cmd.Help()
			os.Exit(1)
		}

		err = generator.GeneratePipeline(projectInfo, outputFile)
		if err != nil {
			fmt.Printf("Error generating pipeline: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✓ Pipeline generated successfully: %s\n", outputFile)
		fmt.Printf("✓ Detected: %s project, architecture: %s",
			projectInfo.Language, projectInfo.Architecture)
		if projectInfo.Version != "" {
			fmt.Printf(", version: %s", projectInfo.Version)
		}
		fmt.Println()
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
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cli-pipeline-generator.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	rootCmd.Flags().StringVarP(&repoPath, "repo", "r", "", "Path to local repository")
	rootCmd.Flags().StringVarP(&remoteRepo, "remote", "R", "", "URL of remote git repository")
	rootCmd.Flags().StringVarP(&branch, "branch", "b", "main", "Branch to analyze")
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "pipeline.yml", "Output pipeline file")
}
