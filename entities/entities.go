// Package entities provides a mapping of entity names to their actions, descriptions, and optional example schemas.
package entities

type Action struct {
	Description string
	NeedsID     bool
	Schema      string // Optional JSON schema or example payload
}

var Entities = map[string]map[string]Action{
	"users": {
		"list":   {Description: "List all users", NeedsID: false},
		"create": {Description: "Create a new user", NeedsID: false},
		"get":    {Description: "Get a user by ID", NeedsID: true},
		"update": {Description: "Update a user by ID", NeedsID: true},
		"delete": {Description: "Delete a user by ID", NeedsID: true},
	},
	"audit": {
		"create": {Description: "Create an audit log", NeedsID: false},
	},
	"executions": {
		"list":   {Description: "List executions", NeedsID: false},
		"get":    {Description: "Get an execution by ID", NeedsID: true},
		"delete": {Description: "Delete an execution by ID", NeedsID: true},
	},
	"workflows": {
		"list": {Description: "List workflow instances", NeedsID: false},
		"get":  {Description: "Get a workflow instance by ID", NeedsID: true},
		"create": {
			Description: "Create a workflow instance",
			NeedsID:     false,
			Schema: `{
  "name": "My Workflow",
  "nodes": [
    {
      "id": "1",
      "name": "Start",
      "type": "n8n-nodes-base.manualTrigger",
      "typeVersion": 1,
      "position": [250, 300]
    }
  ],
  "connections": {},
  "active": false
}`,
		},
		"update":     {Description: "Update a workflow instance by ID", NeedsID: true},
		"delete":     {Description: "Delete a workflow instance by ID", NeedsID: true},
		"activate":   {Description: "Activate a workflow instance by ID", NeedsID: true},
		"deactivate": {Description: "Deactivate a workflow instance by ID", NeedsID: true},
		"preview":    {Description: "Preview a workflow template (with confirmation to save and show diff)", NeedsID: false},
		"diff":       {Description: "Show diff between existing and new workflow templates", NeedsID: false},
		"deploy":     {Description: "Deploy a workflow instance", NeedsID: false, Schema: "(No schema — uses .out/workflow.json from preview)"},
		"rollback":   {Description: "Rollback a workflow instance", NeedsID: false},
	},
	"credentials": {
		"list": {Description: "List credentials", NeedsID: false},
		"create": {
			Description: "Create a credential",
			NeedsID:     false,
			Schema: `{
  "name": "Joe's GitHub Credentials",
  "type": "httpHeaderAuth",
  "data": {
    "name": "Authorization",
    "value": "Bearer ghp_xxxxyyyyyyyyyy"
  },
  "nodesAccess": [
    {
      "nodeType": "n8n-nodes-base.httpRequest"
    }
  ]
}`,
		},
		"get":    {Description: "Get a credential by ID", NeedsID: true},
		"update": {Description: "Update a credential by ID", NeedsID: true},
		"delete": {Description: "Delete a credential by ID", NeedsID: true},
	},
	"tags": {
		"list":   {Description: "List tags", NeedsID: false},
		"create": {Description: "Create a tag", NeedsID: false},
		"get":    {Description: "Get a tag by ID", NeedsID: true},
		"update": {Description: "Update a tag by ID", NeedsID: true},
		"delete": {Description: "Delete a tag by ID", NeedsID: true},
	},
	"source-control": {
		"list":   {Description: "List source control configs", NeedsID: false},
		"get":    {Description: "Get a source control config by ID", NeedsID: true},
		"update": {Description: "Update a source control config by ID", NeedsID: true},
	},
	"variables": {
		"list":   {Description: "List variables", NeedsID: false},
		"create": {Description: "Create a variable", NeedsID: false},
		"get":    {Description: "Get a variable by ID", NeedsID: true},
		"update": {Description: "Update a variable by ID", NeedsID: true},
		"delete": {Description: "Delete a variable by ID", NeedsID: true},
	},
	"projects": {
		"list":   {Description: "List projects", NeedsID: false},
		"create": {Description: "Create a project", NeedsID: false},
		"get":    {Description: "Get a project by ID", NeedsID: true},
		"update": {Description: "Update a project by ID", NeedsID: true},
		"delete": {Description: "Delete a project by ID", NeedsID: true},
	},
}
