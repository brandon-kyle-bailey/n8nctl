package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const apiKeyHeader = "X-N8N-API-KEY"

type Action struct {
	Description string
	NeedsID     bool
}

var entities = map[string]map[string]Action{
	"users": {
		"list":   {"List all users", false},
		"create": {"Create a new user", false},
		"get":    {"Get a user by ID", true},
		"update": {"Update a user by ID", true},
		"delete": {"Delete a user by ID", true},
	},
	"audit": {
		"list": {"List audit logs", false},
		"get":  {"Get an audit log by ID", true},
	},
	"executions": {
		"list":   {"List executions", false},
		"get":    {"Get an execution by ID", true},
		"delete": {"Delete an execution by ID", true},
	},
	"workflows": {
		"list":       {"List workflows", false},
		"create":     {"Create a workflow", false},
		"preview":    {"Preview workflow JSON from YAML (with confirmation to save and show diff)", false},
		"diff":       {"Show diff between existing and new workflow JSON", false},
		"deploy":     {"Deploy a workflow", false},
		"get":        {"Get a workflow by ID", true},
		"update":     {"Update a workflow by ID", true},
		"delete":     {"Delete a workflow by ID", true},
		"activate":   {"Activate a workflow by ID", true},
		"deactivate": {"Deactivate a workflow by ID", true},
	},
	"credentials": {
		"list":   {"List credentials", false},
		"create": {"Create a credential", false},
		"get":    {"Get a credential by ID", true},
		"update": {"Update a credential by ID", true},
		"delete": {"Delete a credential by ID", true},
	},
	"tags": {
		"list":   {"List tags", false},
		"create": {"Create a tag", false},
		"get":    {"Get a tag by ID", true},
		"update": {"Update a tag by ID", true},
		"delete": {"Delete a tag by ID", true},
	},
	"source-control": {
		"list":   {"List source control configs", false},
		"get":    {"Get a source control config by ID", true},
		"update": {"Update a source control config by ID", true},
	},
	"variables": {
		"list":   {"List variables", false},
		"create": {"Create a variable", false},
		"get":    {"Get a variable by ID", true},
		"update": {"Update a variable by ID", true},
		"delete": {"Delete a variable by ID", true},
	},
	"projects": {
		"list":   {"List projects", false},
		"create": {"Create a project", false},
		"get":    {"Get a project by ID", true},
		"update": {"Update a project by ID", true},
		"delete": {"Delete a project by ID", true},
	},
}

type Config struct {
	APIToken string `json:"api_token"`
	BaseURL  string `json:"base_url"`
}

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	entity := os.Args[1]

	if entity == "login" {
		handleLogin(os.Args[2:])
		return
	}

	if entity == "--help" || entity == "-h" || entity == "help" {
		printHelp()
		return
	}

	actions, ok := entities[entity]
	if !ok {
		fmt.Printf("Unknown entity: %s\n\n", entity)
		printHelp()
		os.Exit(1)
	}

	handleEntityCommand(entity, os.Args[2:], actions)
}

func printHelp() {
	fmt.Println(`n8nctl CLI - usage:
  n8nctl <entity> <action> [parameters] [flags]

Entities:`)
	for e := range entities {
		fmt.Printf("  %s\n", e)
	}
	fmt.Println(`
Special commands:
  login        Login and store your API token and base URL

Config:
  Config is stored in ~/.n8nctl/config.json

Dependencies:
		- yq: sudo apt install yq or brew install yq
		- colordiff: sudo apt install colordiff or brew install colordiff

Use "n8nctl <entity> --help" for available actions and usage details.`)
}

func handleEntityCommand(entity string, args []string, actions map[string]Action) {
	if len(args) == 0 {
		fmt.Printf("%s requires an action. Use --help for available actions.\n", entity)
		os.Exit(1)
	}
	action := args[0]
	if action == "--help" || action == "-h" {
		printEntityHelp(entity, actions)
		return
	}
	a, ok := actions[action]
	if !ok {
		fmt.Printf("Unknown action for %s: %s\n", entity, action)
		os.Exit(1)
	}
	if a.NeedsID && len(args) < 2 {
		fmt.Printf("Action '%s' requires an ID parameter\n", action)
		os.Exit(1)
	}
	params := args[1:]
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\nPlease run `n8nctl login` first.\n", err)
		os.Exit(1)
	}
	err = handleGenericEntityAction(entity, action, params, cfg)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func printEntityHelp(entity string, actions map[string]Action) {
	fmt.Printf("Available actions for %s:\n", entity)
	for a, desc := range actions {
		fmt.Printf("  %-10s %s\n", a, desc.Description)
	}
	fmt.Printf("\nUsage:\n  n8nctl %s <action> [id] [flags]\n", entity)
}

func handleGenericEntityAction(entity, action string, params []string, cfg Config) error {
	client := &http.Client{}
	basePath := fmt.Sprintf("%s/api/v1/%s", strings.ToLower(cfg.BaseURL), entity)
	var url, method, body string
	switch action {
	case "list":
		method = "GET"
		url = basePath
	case "get":
		method = "GET"
		url = fmt.Sprintf("%s/%s", basePath, params[0])
	case "create":
		if entity == "workflows" {
			return generateStarterWorkflowYAML()
		}
		method = "POST"
		fmt.Println("Enter JSON data for creation:")
		body = readStdin()
		url = basePath
	case "preview":
		if entity == "workflows" {
			confirmed, err := previewWorkflowJSONWithPrompt()
			if err != nil {
				return err
			}
			if !confirmed {
				fmt.Println("Preview aborted by user.")
			}
			return nil
		}
		return fmt.Errorf("preview not supported for %s", entity)
	case "diff":
		if entity == "workflows" {
			return diffWorkflowJSON()
		}
		return fmt.Errorf("diff not supported for %s", entity)
	case "deploy":
		if entity == "workflows" {
			confirmed, err := previewWorkflowJSONWithPrompt()
			if err != nil {
				return err
			}
			if !confirmed {
				fmt.Println("Deploy aborted by user.")
				return nil
			}

			jsonPath := ".out/workflow.json"
			jsonBytes, err := os.ReadFile(jsonPath)
			if err != nil {
				return fmt.Errorf("could not read %s: %w", jsonPath, err)
			}
			url = basePath
			method = "POST"
			body = string(jsonBytes)
		} else {
			return fmt.Errorf("deploy not supported for %s", entity)
		}
	case "update":
		method = "PATCH"
		fmt.Println("Enter JSON data for update:")
		body = readStdin()
		url = fmt.Sprintf("%s/%s", basePath, params[0])
	case "delete":
		method = "DELETE"
		url = fmt.Sprintf("%s/%s", basePath, params[0])
	case "activate", "deactivate":
		url = fmt.Sprintf("%s/%s/%s", basePath, params[0], action)
		method = "POST"
	default:
		return fmt.Errorf("action %s not implemented for entity %s", action, entity)
	}

	resp, err := apiRequest(client, method, url, body, cfg.APIToken)
	if err != nil {
		return err
	}
	if action == "delete" {
		fmt.Printf("%s %s successful\n", entity, action)
		return nil
	}
	printJSONResponse(resp)
	return nil
}

func apiRequest(client *http.Client, method, url, body, apiKey string) ([]byte, error) {
	var reqBody io.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set(apiKeyHeader, apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error: %s\n%s", resp.Status, string(data))
	}
	return data, nil
}

func printJSONResponse(data []byte) {
	var prettyJSON bytes.Buffer
	err := json.Indent(&prettyJSON, data, "", "  ")
	if err != nil {
		fmt.Println(string(data))
		return
	}
	fmt.Println(prettyJSON.String())
}

func readStdin() string {
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

func generateStarterWorkflowYAML() error {
	yamlContent := `name: Sample Workflow
nodes:
  - id: "1"
    name: Start
    type: n8n-nodes-base.start
    typeVersion: 1
    position: [250, 300]
  - id: "2"
    name: HTTP Request
    type: n8n-nodes-base.httpRequest
    typeVersion: 1
    position: [450, 300]
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

// helper to run a diff (colordiff if available) on two JSON byte slices
func runDiff(oldJSON, newJSON []byte) error {
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

func previewWorkflowJSONWithPrompt() (bool, error) {
	if _, err := os.Stat("workflow.yaml"); os.IsNotExist(err) {
		return false, fmt.Errorf("workflow.yaml not found")
	}

	cmd := exec.Command("yq", ".", "workflow.yaml")
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
		if err := runDiff(oldJSONBytes, newJSON); err != nil {
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

func diffWorkflowJSON() error {
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

	return runDiff(oldJSONBytes, newJSON)
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	configDir := filepath.Join(home, ".n8nctl")
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		if err := os.Mkdir(configDir, 0700); err != nil {
			return "", err
		}
	}
	return filepath.Join(configDir, "config.json"), nil
}

func loadConfig() (Config, error) {
	path, err := configPath()
	if err != nil {
		return Config{}, err
	}
	f, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer f.Close()
	var cfg Config
	err = json.NewDecoder(f).Decode(&cfg)
	return cfg, err
}

func saveConfig(cfg Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	return encoder.Encode(cfg)
}

func handleLogin(args []string) {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	baseURL := fs.String("base-url", "", "API base URL")
	token := fs.String("token", "", "API access token (see <base-url>/settings/api)")
	fs.Parse(args)
	reader := bufio.NewReader(os.Stdin)
	if *baseURL == "" {
		fmt.Print("Enter API base URL: ")
		input, _ := reader.ReadString('\n')
		*baseURL = strings.TrimSpace(input)
	}
	if *token == "" {
		fmt.Printf("Enter API token (visit %s/settings/api to generate one): ", *baseURL)
		input, _ := reader.ReadString('\n')
		*token = strings.TrimSpace(input)
	}
	if *token == "" || *baseURL == "" {
		fmt.Println("Error: both token and base-url are required")
		os.Exit(1)
	}
	cfg := Config{APIToken: *token, BaseURL: strings.TrimRight(*baseURL, "/")}
	err := saveConfig(cfg)
	if err != nil {
		fmt.Printf("Failed to save config: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Login successful, credentials saved.")
}
