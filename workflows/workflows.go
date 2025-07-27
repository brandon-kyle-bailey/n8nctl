// Package workflows provides functions to generate, preview, and diff n8n workflow YAML files.
package workflows

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/brandon-kyle-bailey/n8nctl/utils"
)

func GenerateStarterWorkflowYAML() error {
	yamlContent := `
name: Sample Workflow
nodes:
  - id: "1"
    name: Start
    type: n8n-nodes-base.manualTrigger
    typeVersion: 1
    position: [250, 300]
  - id: "2"
    name: HTTP Request
    type: n8n-nodes-base.httpRequest
    typeVersion: 1
    position: [450, 300]
    credentials:
      httpBasicAuth:
        id: "credential-id"
        name: "My HTTP Basic Auth"
    parameters:
      url: "https://jsonplaceholder.typicode.com/posts/1"
connections:
  Start:
    main:
      - - node: HTTP Request
          type: main
          index: 0
settings: {}
`

	fileName := "workflow.yaml"
	if _, err := os.Stat(fileName); err == nil {
		return fmt.Errorf("%s already exists", fileName)
	}
	return os.WriteFile(fileName, []byte(yamlContent), 0644)
}

func PreviewWorkflowJSONWithPrompt() (bool, error) {
	// Read workflow.yaml first
	yamlBytes, err := os.ReadFile("workflow.yaml")
	if err != nil {
		return false, fmt.Errorf("workflow.yaml not found")
	}

	yamlStr := string(yamlBytes)

	// Only inject JS code if the marker exists
	if strings.Contains(yamlStr, "jsCode: file(index.js)") {
		yamlWithJSBytes, err := injectJSCode("workflow.yaml")
		if err != nil {
			return false, fmt.Errorf("failed to inject index.js code: %w", err)
		}
		yamlStr = string(yamlWithJSBytes)
	}

	// Load env and inject variables as before
	envMap, err := utils.LoadDotEnv(".env")
	if err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("failed to load .env file: %w", err)
	}

	if envMap != nil {
		yamlStr = injectEnvVariables(yamlStr, envMap)
	}

	// Proceed with yq, diff, prompt, etc...
	cmd := exec.Command("yq", ".", "-")
	cmd.Stdin = strings.NewReader(yamlStr)
	newJSON, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("yq failed: %w", err)
	}

	oldJSONBytes, err := os.ReadFile(".out/workflow.json")
	oldExists := err == nil

	fmt.Println("Workflow JSON preview:")
	fmt.Println(string(newJSON))

	if oldExists {
		fmt.Println("\nShowing diff between existing and new workflow JSON:")
		if err := utils.RunDiff(oldJSONBytes, newJSON); err != nil {
			return false, err
		}
	} else {
		fmt.Println("\nNo existing .out/workflow.json found, skipping diff.")
	}

	fmt.Print("\nWrite this JSON to .out/workflow.json? (y/N): ")
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer == "y" || answer == "yes" {
		if _, err := os.Stat(".out"); os.IsNotExist(err) {
			if err := os.Mkdir(".out", 0755); err != nil {
				return false, fmt.Errorf("failed to create .out directory: %w", err)
			}
		}

		err = os.WriteFile(".out/workflow.json", newJSON, 0644)
		if err != nil {
			return false, fmt.Errorf("failed to write .out/workflow.json: %w", err)
		}
		fmt.Println("\nSaved to .out/workflow.json")
		return true, nil
	} else {
		fmt.Println("Aborted, no changes written.")
		return false, nil
	}
}

func DiffWorkflowJSON() error {
	if _, err := os.Stat("workflow.yaml"); os.IsNotExist(err) {
		return fmt.Errorf("workflow.yaml not found")
	}
	if _, err := os.Stat(".out/workflow.json"); os.IsNotExist(err) {
		return fmt.Errorf(".out/workflow.json does not exist, please run preview and save the JSON first")
	}

	cmd := exec.Command("yq", ".", "workflow.yaml")
	newJSON, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("yq failed: %w", err)
	}

	oldJSONBytes, err := os.ReadFile(".out/workflow.json")
	if err != nil {
		return fmt.Errorf("failed to read .out/workflow.json: %w", err)
	}

	return utils.RunDiff(oldJSONBytes, newJSON)
}

// injectEnvVariables replaces ${{VAR_NAME}} with values from env map
func injectEnvVariables(yaml string, env map[string]string) string {
	re := regexp.MustCompile(`\${{\s*([A-Za-z_][A-Za-z0-9_]*)\s*}}`)
	return re.ReplaceAllStringFunc(yaml, func(match string) string {
		matches := re.FindStringSubmatch(match)
		if len(matches) < 2 {
			return match
		}
		if val, ok := env[matches[1]]; ok {
			return val
		}
		return match // leave unresolved if missing
	})
}

// injectJSCode replaces lines like `jsCode: file(index.js)` in the YAML
// with the actual contents of the file using a YAML block scalar.
func injectJSCode(yamlPath string) ([]byte, error) {
	yamlBytes, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML: %w", err)
	}
	lines := strings.Split(string(yamlBytes), "\n")

	var outputLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Match pattern: jsCode: file(index.js)
		if strings.HasPrefix(trimmed, "jsCode: file(") && strings.HasSuffix(trimmed, ")") {
			fileName := trimmed[len("jsCode: file(") : len(trimmed)-1]
			jsPath := filepath.Join(filepath.Dir(yamlPath), fileName)

			jsBytes, err := os.ReadFile(jsPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read %s: %w", jsPath, err)
			}
			jsCode := string(jsBytes)

			// Find indentation of original line (spaces before 'jsCode')
			indentation := line[:len(line)-len(strings.TrimLeft(line, " "))]

			// Write the block scalar line with the same indentation
			outputLines = append(outputLines, indentation+"jsCode: |")

			// Indent JS code lines one level further (e.g. 2 spaces more)
			jsIndent := indentation + "  "
			for jsLine := range strings.SplitSeq(jsCode, "\n") {
				outputLines = append(outputLines, jsIndent+jsLine)
			}
		} else {
			outputLines = append(outputLines, line)
		}
	}

	return []byte(strings.Join(outputLines, "\n")), nil
}
