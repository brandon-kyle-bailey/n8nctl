// Package entities provides a mapping of entity names to their actions and descriptions.
package entities

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/brandon-kyle-bailey/n8nctl/config"
	"github.com/brandon-kyle-bailey/n8nctl/utils"
	"github.com/brandon-kyle-bailey/n8nctl/workflows"
)

func PrintHelp() {
	fmt.Println(`
N8NCtl ⚡ A lightweight CLI for managing n8n workflows declaratively with YAML. 

Usage:
	n8nctl <entity> <action> [parameters] [flags]

Entities:`)
	for entity := range Entities {
		fmt.Printf("	%s\n", entity)
	}
	fmt.Println(`
Special commands:
	login:	Login and store your API token and base URL

Config:
	Config is stored in ~/.n8nctl/config.json

Environment:
	.env file can be used for environment variable injection. (use workflows preview to verify values)

Flags:
	--schema  Show JSON schema for an entity's action when used with --help or an action command

Dependencies:
	- yq: sudo apt install yq or brew install yq
	- colordiff: sudo apt install colordiff or brew install colordiff

Use "n8nctl <entity> --help" for available actions and usage details.
Use "n8nctl <entity> <action> --schema" to see the JSON schema for that action.`)
}

func HandleEntityCommand(entity string, args []string, actions map[string]Action, cfg config.Config) {
	if len(args) == 0 {
		fmt.Printf("%s requires an action. Use --help for available actions.\n", entity)
		os.Exit(1)
	}

	action := args[0]

	// Detect --help and optional --schema flag (anywhere in args)
	showSchema := false
	for _, arg := range args {
		if arg == "--schema" {
			showSchema = true
		}
	}

	if action == "--help" || action == "-h" {
		// Show help with or without schema based on flag
		PrintEntityHelp(entity, actions, showSchema)
		return
	}

	// Also allow --schema with actions, e.g. n8nctl credentials create --schema
	if showSchema {
		a, ok := actions[action]
		if !ok {
			fmt.Printf("Unknown action for %s: %s\n", entity, action)
			os.Exit(1)
		}
		if a.Schema == "" {
			fmt.Printf("No schema available for action %s on entity %s\n", action, entity)
		} else {
			fmt.Printf("Schema for %s %s:\n", entity, action)
			fmt.Println(a.Schema)
		}
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

	err := handleGenericEntityAction(entity, action, params, cfg)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func PrintEntityHelp(entity string, actions map[string]Action, showSchema bool) {
	fmt.Printf("Available actions for %s:\n", entity)
	for actionName, action := range actions {
		fmt.Printf("  %-10s %s\n", actionName, action.Description)
		if showSchema && action.Schema != "" {
			fmt.Println("    Example schema:")
			for line := range strings.SplitSeq(action.Schema, "\n") {
				fmt.Printf("      %s\n", line)
			}
		}
	}
	fmt.Printf("\nUsage:\n  n8nctl %s <action> [id] [flags]\n", entity)
	fmt.Println("\nFlags:")
	fmt.Println("  --schema  Show JSON schema for the action (use with --help or an action)")
}

func handleGenericEntityAction(entity, action string, params []string, cfg config.Config) error {
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
			return workflows.GenerateStarterWorkflowYAML()
		}
		method = "POST"
		if len(params) >= 2 && params[0] == "--data" {
			body = params[1]
		} else {
			fmt.Println("Enter JSON data for creation:")
			body = utils.ReadStdin()
		}
		url = basePath

	case "update":
		method = "PATCH"
		if len(params) < 1 {
			return fmt.Errorf("missing ID for update")
		}
		id := params[0]
		url = fmt.Sprintf("%s/%s", basePath, id)
		if len(params) >= 3 && params[1] == "--data" {
			body = params[2]
		} else {
			fmt.Println("Enter JSON data for update:")
			body = utils.ReadStdin()
		}
	case "delete":
		method = "DELETE"
		url = fmt.Sprintf("%s/%s", basePath, params[0])
	case "preview":
		if entity == "workflows" {
			confirmed, err := workflows.PreviewWorkflowJSONWithPrompt()
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
			return workflows.DiffWorkflowJSON()
		}
		return fmt.Errorf("diff not supported for %s", entity)
	case "deploy":
		if entity == "workflows" {
			confirmed, err := workflows.PreviewWorkflowJSONWithPrompt()
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
	case "activate", "deactivate":
		url = fmt.Sprintf("%s/%s/%s", basePath, params[0], action)
		method = "POST"
	default:
		return fmt.Errorf("action %s not implemented for entity %s", action, entity)
	}

	resp, err := n8nAPIRequest(client, method, url, body, cfg.APIToken)
	if err != nil {
		return err
	}

	if action == "delete" {
		fmt.Printf("%s %s successful\n", entity, action)
		return nil
	}

	utils.PrintJSONResponse(resp)
	return nil
}

func n8nAPIRequest(client *http.Client, method, url, body, apiKey string) ([]byte, error) {
	var reqBody io.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-N8N-API-KEY", apiKey)
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

func HandleLogin(args []string) {
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
	cfg := config.Config{APIToken: *token, BaseURL: strings.TrimRight(*baseURL, "/")}
	err := config.SaveConfig(cfg)
	if err != nil {
		fmt.Printf("Failed to save config: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Login successful, credentials saved.")
}
