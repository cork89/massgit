package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// gets all dirs with a ".git" dir inside of the parentDir
func findGitRepos(parentDir string) ([]string, error) {
	var repos []string
	entries, err := os.ReadDir(parentDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			gitPath := filepath.Join(parentDir, entry.Name(), ".git")
			if info, err := os.Stat(gitPath); err == nil && info.IsDir() {
				repos = append(repos, filepath.Join(parentDir, entry.Name()))
			}
		}
	}
	return repos, nil
}

// run git rev-parse --verify <branch>
func checkGitBranch(repoPath string, branch string) (bool, error) {
	cmd := exec.Command("git", "rev-parse", "--verify", branch)
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	if strings.Contains(string(out), "fatal") {
		return false, nil
	}
	return true, nil
}

// run git switch <branch>
func switchGitBranch(repoPath string, branch string) (bool, error) {
	cmd := exec.Command("git", "switch", branch)
	cmd.Dir = repoPath
	_, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return true, nil
}

// run git switch -c <branch>
func createGitBranch(repoPath string, branch string) (bool, error) {
	cmd := exec.Command("git", "switch", "-c", branch)
	cmd.Dir = repoPath
	_, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return true, nil
}

// run git rev-parse --abbrev-ref HEAD
func gitBranch(repoPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// run git status -s
func gitStatus(repoPath string) (string, error) {
	cmd := exec.Command("git", "status", "-s")
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// run git add <file>
func gitAdd(repoPath string, fileName string) (string, error) {
	cmd := exec.Command("git", "add", fileName)
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// run git commit -m <commit msg>
func gitCommit(repoPath string, commitMsg string) (string, error) {
	cmd := exec.Command("git", "commit", "-m", commitMsg)
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
