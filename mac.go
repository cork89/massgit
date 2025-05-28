//go:build darwin

package main

import "fmt"

func init() {
	shell = "zsh"
	grep = "grep"
	sed = "sed"
	updateMvn = func(version string, lineNum string) []string {
		return []string{
			"-i",
			"",
			fmt.Sprintf("%ss#^\\([ \\t]*\\).*#\\1<version>%s</version>#", lineNum, version),
			"pom.xml",
		}
	}
}
