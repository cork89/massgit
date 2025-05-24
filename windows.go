//go:build windows || linux

package main

func init() {
	shell = "bash"
	grep = "C:/Program Files/Git/usr/bin/grep.exe"
	sed = "C:/Program Files/Git/usr/bin/sed.exe"
}
