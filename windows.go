//go:build windows || linux

package main

import "fmt"

func init() {
	shell = "bash"
	grep = "C:/Program Files/Git/usr/bin/grep.exe"
	sed = "C:/Program Files/Git/usr/bin/sed.exe"
	updateMvn = func(version string, lineNum string) []string {
		return []string{
			"-i",
			fmt.Sprintf("%ss#^\\([ \\t]*\\).*#\\1<version>%s</version>#", lineNum, version),
			"pom.xml",
		}
	}
}
