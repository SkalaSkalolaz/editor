package main

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// GitHubProject представляет проект с GitHub
type GitHubProject struct {
	URL         string
	Owner       string
	Repo        string
	Branch      string
	LocalPath   string
	AccessToken string
	Files       map[string]string
}

// parseGitHubURL разбирает URL GitHub и извлекает владельца, репозиторий и ветку
func parseGitHubURL(githubURL string) (*GitHubProject, error) {
	parsed, err := url.Parse(githubURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Проверяем, что это GitHub URL
	if parsed.Host != "github.com" {
		return nil, fmt.Errorf("not a GitHub URL")
	}

	pathParts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(pathParts) < 2 {
		return nil, fmt.Errorf("invalid GitHub URL format")
	}

	project := &GitHubProject{
		URL:    githubURL,
		Owner:  pathParts[0],
		Repo:   pathParts[1],
		Branch: "main",
	}

	if len(pathParts) > 3 && pathParts[2] == "tree" {
		project.Branch = pathParts[3]
	}

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	project.LocalPath = filepath.Join(cwd, project.Repo)

	return project, nil
}

// downloadGitHubRepo загружает репозиторий с GitHub с возможностью перезаписи
func (gp *GitHubProject) downloadGitHubRepo(overwrite bool) error {
	if _, err := os.Stat(gp.LocalPath); err == nil {
		if !overwrite {
			return fmt.Errorf("directory %s already exists", gp.Repo)
		}
		if err := os.RemoveAll(gp.LocalPath); err != nil {
			return fmt.Errorf("failed to remove existing directory: %w", err)
		}
	}

	if err := os.MkdirAll(gp.LocalPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	zipURL := fmt.Sprintf("https://github.com/%s/%s/archive/refs/heads/%s.zip",
		gp.Owner, gp.Repo, gp.Branch)

	req, err := http.NewRequest("GET", zipURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if gp.AccessToken != "" {
		if strings.HasPrefix(gp.AccessToken, "ghp_") || strings.HasPrefix(gp.AccessToken, "github_pat_") {
			req.Header.Set("Authorization", "token "+gp.AccessToken)
		} else {
			if decoded, err := base64.StdEncoding.DecodeString(gp.AccessToken); err == nil {
				req.Header.Set("Authorization", "token "+string(decoded))
			} else {
				req.Header.Set("Authorization", "token "+gp.AccessToken)
			}
		}
	}

	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download repository: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to download repository: HTTP %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp("", "github-*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return fmt.Errorf("failed to save zip file: %w", err)
	}

	if err := gp.extractZip(tmpFile.Name()); err != nil {
		return fmt.Errorf("failed to extract zip: %w", err)
	}

	return nil
}

// extractZip распаковывает ZIP архив
func (gp *GitHubProject) extractZip(zipPath string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer reader.Close()

	prefix := fmt.Sprintf("%s-%s/", gp.Repo, gp.Branch)

	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		relativePath := strings.TrimPrefix(file.Name, prefix)
		if relativePath == file.Name {
			prefix = fmt.Sprintf("%s-%s/", gp.Repo, strings.ReplaceAll(gp.Branch, "/", "-"))
			relativePath = strings.TrimPrefix(file.Name, prefix)
			if relativePath == file.Name {
				continue
			}
		}

		fullPath := filepath.Join(gp.LocalPath, relativePath)

		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		outFile, err := os.Create(fullPath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", fullPath, err)
		}

		zipFile, err := file.Open()
		if err != nil {
			outFile.Close()
			return fmt.Errorf("failed to open zip file %s: %w", file.Name, err)
		}

		if _, err := io.Copy(outFile, zipFile); err != nil {
			outFile.Close()
			zipFile.Close()
			return fmt.Errorf("failed to extract file %s: %w", fullPath, err)
		}

		outFile.Close()
		zipFile.Close()
	}

	return nil
}

// isGitHubURL проверяет, является ли строка URL GitHub
func isGitHubURL(path string) bool {
	return strings.HasPrefix(path, "https://github.com/") ||
		strings.HasPrefix(path, "http://github.com/")
}

// loadGitHubProject загружает проект с GitHub и создает редактор
func loadGitHubProject(githubURL, provider, model, accessToken string) (*Editor, error) {
	project, err := parseGitHubURL(githubURL)
	if err != nil {
		return nil, err
	}

	project.AccessToken = accessToken

	if err := project.downloadGitHubRepo(false); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			fmt.Printf("Project directory '%s' already exists. Overwrite? (y/n): ", project.Repo)
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(response) == "y" || strings.ToLower(response) == "yes" {
				if err := project.downloadGitHubRepo(true); err != nil {
					return nil, fmt.Errorf("failed to overwrite project: %w", err)
				}
			} else {
				fmt.Printf("Using existing directory: %s\n", project.LocalPath)
			}
		} else {
			return nil, err
		}
	}

	editor := NewEditorWithProject(project.LocalPath, provider, model)
	if editor == nil {
		return nil, fmt.Errorf("failed to create editor for project")
	}

	editor.githubProject = project

	return editor, nil
}

// pushToGitHub отправляет изменения обратно на GitHub
func (gp *GitHubProject) pushToGitHub(commitMessage string) error {
	if gp.AccessToken == "" {
		return fmt.Errorf("access token is required to push changes")
	}

	if err := execCommand("git", "--version"); err != nil {
		return fmt.Errorf("git is not available: %w", err)
	}

	if _, err := os.Stat(filepath.Join(gp.LocalPath, ".git")); os.IsNotExist(err) {
		if err := execCommandInDir(gp.LocalPath, "git", "init"); err != nil {
			return fmt.Errorf("failed to init git repo: %w", err)
		}

		if err := execCommandInDir(gp.LocalPath, "git", "config", "user.email", "editor@localhost"); err != nil {
			return fmt.Errorf("failed to set git user email: %w", err)
		}
		if err := execCommandInDir(gp.LocalPath, "git", "config", "user.name", "Editor User"); err != nil {
			return fmt.Errorf("failed to set git user name: %w", err)
		}

		remoteURL := fmt.Sprintf("https://%s@github.com/%s/%s.git", gp.AccessToken, gp.Owner, gp.Repo)
		if err := execCommandInDir(gp.LocalPath, "git", "remote", "add", "origin", remoteURL); err != nil {
			return fmt.Errorf("failed to add remote: %w", err)
		}
	}

	if err := execCommandInDir(gp.LocalPath, "git", "add", "."); err != nil {
		return fmt.Errorf("failed to add files: %w", err)
	}

	statusCmd := exec.Command("git", "status", "--porcelain")
	statusCmd.Dir = gp.LocalPath
	output, err := statusCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	}

	if len(output) == 0 {
		return fmt.Errorf("no changes to commit")
	}

	commitCmd := exec.Command("git", "commit", "-m", commitMessage)
	commitCmd.Dir = gp.LocalPath

	var commitOut bytes.Buffer
	var commitErr bytes.Buffer
	commitCmd.Stdout = &commitOut
	commitCmd.Stderr = &commitErr

	if err := commitCmd.Run(); err != nil {
		errorMsg := commitErr.String()
		if errorMsg == "" {
			errorMsg = err.Error()
		}
		return fmt.Errorf("failed to commit: %s", errorMsg)
	}

	pushCmd := exec.Command("git", "push", "-u", "origin", "HEAD:"+gp.Branch, "--force")
	pushCmd.Dir = gp.LocalPath

	var pushOut bytes.Buffer
	var pushErr bytes.Buffer
	pushCmd.Stdout = &pushOut
	pushCmd.Stderr = &pushErr

	pushErrMsg := ""
	if err := pushCmd.Run(); err != nil {
		pushErrMsg = pushErr.String()
		if pushErrMsg == "" {
			pushErrMsg = err.Error()
		}

		pushCmd = exec.Command("git", "push", "-u", "origin", "HEAD:"+gp.Branch)
		pushCmd.Dir = gp.LocalPath

		pushOut.Reset()
		pushErr.Reset()
		pushCmd.Stdout = &pushOut
		pushCmd.Stderr = &pushErr

		if err := pushCmd.Run(); err != nil {
			errorMsg := pushErr.String()
			if errorMsg == "" {
				errorMsg = err.Error()
			}
			return fmt.Errorf("failed to push: %s", errorMsg)
		}
	}

	return nil
}

// execCommand выполняет команду в текущей директории
func execCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	return cmd.Run()
}

// execCommandInDir выполняет команду в указанной директории
func execCommandInDir(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	return cmd.Run()
}

// saveToGitHub сохраняет изменения в GitHub репозиторий
func (e *Editor) saveToGitHub() {
	if e.githubProject == nil {
		e.showError("Not a GitHub project")
		return
	}
	e.saveCurrentFileToGitHub()
}

// pushCurrentFileToGitHub отправляет изменения текущего файла на GitHub
func (gp *GitHubProject) pushCurrentFileToGitHub(filename, commitMessage string) error {
	if gp.AccessToken == "" {
		return fmt.Errorf("access token is required to push changes")
	}

	if err := execCommand("git", "--version"); err != nil {
		return fmt.Errorf("git is not available: %w", err)
	}

	gitDir := filepath.Join(gp.LocalPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		if err := execCommandInDir(gp.LocalPath, "git", "init"); err != nil {
			return fmt.Errorf("failed to init git repo: %w", err)
		}

		if err := execCommandInDir(gp.LocalPath, "git", "config", "user.email", "editor@localhost"); err != nil {
			return fmt.Errorf("failed to set git user email: %w", err)
		}
		if err := execCommandInDir(gp.LocalPath, "git", "config", "user.name", "Editor User"); err != nil {
			return fmt.Errorf("failed to set git user name: %w", err)
		}

		remoteURL := fmt.Sprintf("https://%s@github.com/%s/%s.git", gp.AccessToken, gp.Owner, gp.Repo)
		if err := execCommandInDir(gp.LocalPath, "git", "remote", "add", "origin", remoteURL); err != nil {
			return fmt.Errorf("failed to add remote: %w", err)
		}
	}

	if filename != "" {
		relPath, err := filepath.Rel(gp.LocalPath, filename)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		if err := execCommandInDir(gp.LocalPath, "git", "add", relPath); err != nil {
			return fmt.Errorf("failed to add file %s: %w", relPath, err)
		}
	} else {
		if err := execCommandInDir(gp.LocalPath, "git", "add", "."); err != nil {
			return fmt.Errorf("failed to add files: %w", err)
		}
	}

	statusCmd := exec.Command("git", "status", "--porcelain")
	statusCmd.Dir = gp.LocalPath
	output, err := statusCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	}

	if len(output) == 0 {
		return fmt.Errorf("no changes to commit for file %s", filename)
	}

	commitCmd := exec.Command("git", "commit", "-m", commitMessage)
	commitCmd.Dir = gp.LocalPath

	var commitOut bytes.Buffer
	var commitErr bytes.Buffer
	commitCmd.Stdout = &commitOut
	commitCmd.Stderr = &commitErr

	if err := commitCmd.Run(); err != nil {
		errorMsg := commitErr.String()
		if errorMsg == "" {
			errorMsg = err.Error()
		}
		return fmt.Errorf("failed to commit: %s", errorMsg)
	}

	pushCmd := exec.Command("git", "push", "-u", "origin", "HEAD:"+gp.Branch)
	pushCmd.Dir = gp.LocalPath

	var pushOut bytes.Buffer
	var pushErr bytes.Buffer
	pushCmd.Stdout = &pushOut
	pushCmd.Stderr = &pushErr

	if err := pushCmd.Run(); err != nil {
		errorMsg := pushErr.String()
		if errorMsg == "" {
			errorMsg = err.Error()
		}

		pushCmd = exec.Command("git", "push", "-u", "origin", "HEAD:"+gp.Branch, "--force")
		pushCmd.Dir = gp.LocalPath

		pushOut.Reset()
		pushErr.Reset()
		pushCmd.Stdout = &pushOut
		pushCmd.Stderr = &pushErr

		if err := pushCmd.Run(); err != nil {
			errorMsg := pushErr.String()
			if errorMsg == "" {
				errorMsg = err.Error()
			}
			return fmt.Errorf("failed to push: %s", errorMsg)
		}
	}

	return nil
}

// saveCurrentFileToGitHub сохраняет изменения текущего файла в GitHub репозиторий
func (e *Editor) saveCurrentFileToGitHub() {
	if e.githubProject == nil {
		e.showError("Not a GitHub project")
		return
	}

	if !e.dirty && e.filename == "" {
		e.showError("No changes to save in current file")
		return
	}

	currentCanvas := e.currentCanvas
	e.syncEditorToCanvas()

	if e.dirty {
		if err := e.save(); err != nil {
			e.showError(fmt.Sprintf("Failed to save file locally: %v", err))
			e.currentCanvas = currentCanvas
			e.syncCanvasToEditor()
			e.render()
			return
		}
	}

	filePath := e.filename
	if filePath == "" {
		e.showError("Cannot save unnamed file to GitHub")
		return
	}

	if !strings.HasPrefix(filePath, e.githubProject.LocalPath) {
		e.showError("File is not within GitHub project directory")
		return
	}

	e.promptShow("Commit message for "+filepath.Base(filePath), func(message string) {
		message = strings.TrimSpace(message)
		if message == "" {
			message = "Update " + filepath.Base(filePath)
		}

		e.statusMessage("Pushing changes to GitHub...")
		e.render()

		if err := e.githubProject.pushCurrentFileToGitHub(filePath, message); err != nil {
			e.showError("Failed to push to GitHub: " + err.Error())
		} else {
			e.statusMessage("Successfully pushed " + filepath.Base(filePath) + " to GitHub")
		}

		e.currentCanvas = currentCanvas
		e.syncCanvasToEditor()
		e.render()
	})
}
