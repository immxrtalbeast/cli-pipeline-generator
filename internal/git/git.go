package git

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
)

// RemoteRepoInfo представляет информацию об удаленном репозитории
type RemoteRepoInfo struct {
	URL           string
	DefaultBranch string
	FileTree      map[string]string // путь -> содержимое файла
	Structure     []string          // список файлов и директорий
}

// AnalyzeRemoteRepo анализирует удаленный репозиторий в памяти
func AnalyzeRemoteRepo(repoURL, branch string) (*RemoteRepoInfo, error) {
	fmt.Printf("Cloning repository: %s (branch: %s)\n", repoURL, branch)

	info := &RemoteRepoInfo{
		URL:       repoURL,
		FileTree:  make(map[string]string),
		Structure: []string{},
	}

	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Пробуем клонировать с указанной веткой
	repo, err := git.CloneContext(ctx, memory.NewStorage(), nil, &git.CloneOptions{
		URL:           repoURL,
		Depth:         1,
		SingleBranch:  true,
		ReferenceName: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", branch)),
	})

	if err != nil {
		fmt.Printf("Error cloning with branch %s: %v\n", branch, err)

		// Если ошибка связана с веткой, пробуем основные ветки по порядку
		if strings.Contains(err.Error(), "couldn't find remote ref") {
			return tryDefaultBranches(repoURL)
		}
		return nil, fmt.Errorf("error cloning repository: %v", err)
	}

	// Получаем ссылку на HEAD
	ref, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("error getting HEAD: %v", err)
	}

	info.DefaultBranch = ref.Name().Short()
	fmt.Printf("Successfully cloned repository, branch: %s\n", info.DefaultBranch)

	// Получаем дерево файлов
	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return nil, fmt.Errorf("error getting commit: %v", err)
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("error getting tree: %v", err)
	}

	// Обходим дерево файлов
	err = buildFileTreeSimple(tree, info)
	if err != nil {
		return nil, fmt.Errorf("error building file tree: %v", err)
	}

	fmt.Printf("Repository analyzed successfully. Found %d files\n", len(info.Structure))
	return info, nil
}

func tryDefaultBranches(repoURL string) (*RemoteRepoInfo, error) {
	fmt.Println("Trying default branches...")

	// Пробуем основные ветки по порядку
	branches := []string{"main", "master", "develop"}

	for _, branch := range branches {
		fmt.Printf("Trying branch: %s\n", branch)

		// Создаем контекст с таймаутом для каждой попытки
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		repo, err := git.CloneContext(ctx, memory.NewStorage(), nil, &git.CloneOptions{
			URL:           repoURL,
			Depth:         1,
			SingleBranch:  true,
			ReferenceName: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", branch)),
		})
		cancel() // Освобождаем контекст сразу после использования

		if err == nil {
			fmt.Printf("Successfully cloned with branch: %s\n", branch)

			info := &RemoteRepoInfo{
				URL:       repoURL,
				FileTree:  make(map[string]string),
				Structure: []string{},
			}

			// Получаем дерево файлов
			ref, err := repo.Head()
			if err != nil {
				continue // Пробуем следующую ветку
			}

			info.DefaultBranch = branch

			commit, err := repo.CommitObject(ref.Hash())
			if err != nil {
				continue
			}

			tree, err := commit.Tree()
			if err != nil {
				continue
			}

			err = buildFileTreeSimple(tree, info)
			if err != nil {
				continue
			}

			return info, nil
		}

		fmt.Printf("Failed with branch %s: %v\n", branch, err)
	}

	return nil, fmt.Errorf("could not find repository on any default branch (main, master, develop)")
}

// Упрощенная версия buildFileTree для большей надежности
func buildFileTreeSimple(tree *object.Tree, info *RemoteRepoInfo) error {
	return tree.Files().ForEach(func(f *object.File) error {
		info.Structure = append(info.Structure, f.Name)

		// Читаем только конфигурационные файлы
		if isConfigFile(f.Name) {
			content, err := f.Contents()
			if err == nil && len(content) < 100000 {
				info.FileTree[f.Name] = content
			}
		}

		return nil
	})
}

func isConfigFile(filename string) bool {
	configFiles := []string{
		"go.mod", "package.json", "requirements.txt", "Cargo.toml",
		"pom.xml", "build.gradle", "Dockerfile", "Makefile", "Makefile.",
		".gitignore", "README.md", "readme.md", "docker-compose.yml",
		"Jenkinsfile", ".travis.yml", ".github/", ".gitlab-ci.yml",
	}

	base := filepath.Base(filename)
	for _, pattern := range configFiles {
		if strings.HasSuffix(pattern, "/") {
			if strings.Contains(filename, pattern) {
				return true
			}
		} else if base == pattern {
			return true
		}
	}
	return false
}

// GetFileContent возвращает содержимое файла из репозитория
func (r *RemoteRepoInfo) GetFileContent(path string) (string, bool) {
	content, exists := r.FileTree[path]
	return content, exists
}

// HasFile проверяет наличие файла в репозитории
func (r *RemoteRepoInfo) HasFile(path string) bool {
	for _, file := range r.Structure {
		if file == path {
			return true
		}
	}
	return false
}

// HasDirectory проверяет наличие директории в репозитории
func (r *RemoteRepoInfo) HasDirectory(dirName string) bool {
	searchPath := dirName + "/"
	for _, path := range r.Structure {
		if strings.HasPrefix(path, searchPath) {
			return true
		}
	}
	return false
}

// HasFileWithExtension проверяет наличие файлов с определенным расширением
func (r *RemoteRepoInfo) HasFileWithExtension(ext string) bool {
	for _, path := range r.Structure {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}
