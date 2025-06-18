package utility

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func GetProcessId(processName string) (int, error) {
	cmd := exec.Command("tasklist")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, err
	}
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	scanner := bufio.NewScanner(stdout)
	skipped := 0
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println("line:", line)
		if skipped < 2 {
			skipped++
			continue
		}
		parts := strings.Fields(line)
		if parts[0] == processName {
			if len(parts) >= 2 {
				pid, err := strconv.Atoi(parts[1])
				if err != nil {
					return 0, err
				}
				return pid, err
			}
		}
	}
	return 0, fmt.Errorf("process not found")
}
