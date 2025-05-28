package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func mvnVersion(repoPath string, repo *Repo) error {
	var (
		output []byte
		err    error
	)
	cmd := exec.Command(
		shell,
		"-c",
		fmt.Sprintf("grep -A 5 -n '<artifactId>%s</artifactId>' pom.xml | grep '<version>' | sed 's/<[^>]*>//g' && grep -A 5 -n '<parent>' pom.xml | grep '<version>' | sed 's/<[^>]*>//g'", repo.Name),
	)

	cmd.Dir = repoPath
	output, err = cmd.Output()

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return err
	}
	versions := strings.Split(string(output), "\n")

	if len(versions) > 1 {
		ln, ver, found := strings.Cut(versions[0], "-")
		if found {
			repo.Maven.Version = strings.TrimSpace(ver)
			repo.Maven.Vln = strings.TrimSpace(ln)
		}

		ln, ver, found = strings.Cut(versions[1], "-")
		if found {
			repo.Maven.ParentVersion = strings.TrimSpace(ver)
			repo.Maven.Pvln = strings.TrimSpace(ln)
		}
	}
	return nil
}

// mvn versions:set -DnewVersion=<version>
func updateMvnVersion(repoPath string, version string, lineNum string, repo *Repo) error {
	var (
		output []byte
		err    error
	)

	cmd := exec.Command(
		// "sed",
		// "-i",
		// "",
		// fmt.Sprintf("%ss#^\\([ \\t]*\\).*#\\1<version>%s</version>#", lineNum, version),
		// "pom.xml",
		"sed",
		updateMvn(version, lineNum)...,
	)

	fmt.Println(cmd)
	cmd.Dir = repoPath
	output, err = cmd.Output()

	if err != nil {
		fmt.Printf("Output: %s, Error: %v\n", output, err)
		return err
	}
	repo.Maven.Version = strings.TrimSpace(string(version))
	return nil
}

// mvn versions:update-parent -DparentVersion=<version>
func updateMvnParentVersion(repoPath string, version string, lineNum string, repo *Repo) error {
	var (
		output []byte
		err    error
	)
	cmd := exec.Command(
		"sed",
		updateMvn(version, lineNum)...,
	)

	cmd.Dir = repoPath
	output, err = cmd.Output()

	if err != nil {
		fmt.Printf("Output: %s, Error: %v\n", output, err)
		return err
	}
	repo.Maven.ParentVersion = strings.TrimSpace(string(version))

	return nil
}
