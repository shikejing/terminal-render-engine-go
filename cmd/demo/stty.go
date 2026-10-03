package main

import (
	"os"
	"os/exec"
	"strings"
)

// execCommand 调用 stty size 返回 "rows cols"。
func execCommand() string {
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
