// Package utils provides utility functions for reading from stdin, printing JSON responses, and running diffs on JSON files.
package utils

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

func ReadStdin() string {
	reader := bufio.NewReader(os.Stdin)
	var sb strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			fmt.Println("Error reading stdin:", err)
			os.Exit(1)
		}
		sb.WriteString(line)
		if err == io.EOF {
			break
		}
	}
	return strings.TrimSpace(sb.String())
}

func PrintJSONResponse(data []byte) {
	var prettyJSON bytes.Buffer
	err := json.Indent(&prettyJSON, data, "", "  ")
	if err != nil {
		fmt.Println(string(data))
		return
	}
	fmt.Println(prettyJSON.String())
}

func RunDiff(oldJSON, newJSON []byte) error {
	oldTmpFile, err := os.CreateTemp("", "oldworkflow-*.json")
	if err != nil {
		return fmt.Errorf("failed to create temp file for old JSON: %w", err)
	}
	defer os.Remove(oldTmpFile.Name())
	if _, err := oldTmpFile.Write(oldJSON); err != nil {
		return fmt.Errorf("failed to write old JSON to temp file: %w", err)
	}
	oldTmpFile.Close()

	newTmpFile, err := os.CreateTemp("", "newworkflow-*.json")
	if err != nil {
		return fmt.Errorf("failed to create temp file for new JSON: %w", err)
	}
	defer os.Remove(newTmpFile.Name())
	if _, err := newTmpFile.Write(newJSON); err != nil {
		return fmt.Errorf("failed to write new JSON to temp file: %w", err)
	}
	newTmpFile.Close()

	diffCmdName := "diff"
	if _, err := exec.LookPath("colordiff"); err == nil {
		diffCmdName = "colordiff"
	}

	diffCmd := exec.Command(diffCmdName, "-u", oldTmpFile.Name(), newTmpFile.Name())
	diffCmd.Stdout = os.Stdout
	diffCmd.Stderr = os.Stderr

	err = diffCmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// differences found, not an error here
			return nil
		}
		return fmt.Errorf("diff command failed: %w", err)
	}

	fmt.Println("No differences detected.")
	return nil
}

func LoadDotEnv(filename string) (map[string]string, error) {
	env := make(map[string]string)

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.Trim(strings.TrimSpace(parts[1]), `"`)
			env[key] = val
		}
	}
	return env, scanner.Err()
}
