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
		fmt.Sprintf("grep -A 5 '<artifactId>%s</artifactId>' pom.xml | grep '<version>' | sed 's/<[^>]*>//g' && grep -A 5 '<parent>' pom.xml | grep '<version>' | sed 's/<[^>]*>//g'", repo.Name),
	)

	cmd.Dir = repoPath
	output, err = cmd.Output()
	if err == nil {
		repo.Maven.Version = strings.TrimSpace(string(output))
	}

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return err
	}
	versions := strings.Split(string(output), "\n")

	if len(versions) > 1 {
		repo.Maven.Version = strings.TrimSpace(versions[0])
		repo.Maven.ParentVersion = strings.TrimSpace(versions[1])
	}
	return nil
}
