package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

type flowsResponse struct {
	Data []apiFlow `json:"data"`
}

type apiFlow struct {
	ID          int             `json:"id"`
	Name        string          `json:"name"`
	Nodes       []apiNode       `json:"nodes"`
	InsertedAt  string          `json:"inserted_at"`
	UpdatedAt   string          `json:"updated_at"`
	Connections []apiConnection `json:"connections"`
}

type apiNode struct {
	Controls map[string]any `json:"controls"`
}

type apiConnection struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

func (m model) startConnection() (model, tea.Cmd) {
	server := strings.TrimSpace(m.hostInput.Value())
	if server == "" {
		server = defaultServer
		m.hostInput.SetValue(server)
	}

	m.connecting = true
	m.connected = false
	m.connectError = ""
	m.connectStatus = "Connecting..."
	m.hostInput.Blur()

	return m, tea.Batch(m.spinner.Tick, connectCmd(server))
}

func connectCmd(server string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
		defer cancel()

		address, err := normalizeAddress(server)
		if err != nil {
			return connectionResultMsg{err: err}
		}

		if err := tryConnect(ctx, address); err != nil {
			return connectionResultMsg{err: err}
		}

		return connectionResultMsg{host: address}
	}
}

func loadWorkflowsCmd(host string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
		defer cancel()

		workflows, err := loadWorkflows(ctx, host)
		return workflowsLoadedMsg{
			workflows: workflows,
			err:       err,
		}
	}
}

func tryConnect(ctx context.Context, address string) error {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", address)
	if err != nil {
		return err
	}

	return conn.Close()
}

func normalizeAddress(server string) (string, error) {
	server = strings.TrimSpace(server)
	if server == "" {
		return "", fmt.Errorf("empty server")
	}

	if strings.Contains(server, "://") {
		u, err := url.Parse(server)
		if err != nil {
			return "", fmt.Errorf("invalid server: %w", err)
		}
		if u.Host == "" {
			return "", fmt.Errorf("invalid server: missing host")
		}
		return u.Host, nil
	}

	if _, _, err := net.SplitHostPort(server); err == nil {
		return server, nil
	}

	if strings.Contains(server, ":") {
		return server, nil
	}

	return net.JoinHostPort(server, "4000"), nil
}

func loadWorkflows(ctx context.Context, host string) ([]workflow, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+host+"/api/flows/", nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: connectTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("flows api returned %s", resp.Status)
	}

	var payload flowsResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	workflows := make([]workflow, 0, len(payload.Data))
	for _, item := range payload.Data {
		workflows = append(workflows, mapFlow(item))
	}
	return workflows, nil
}

func mapFlow(item apiFlow) workflow {
	updatedAt := parseAPITime(item.UpdatedAt)
	insertedAt := parseAPITime(item.InsertedAt)
	return workflow{
		id:      item.ID,
		name:    fallbackString(item.Name, fmt.Sprintf("Flow %d", item.ID)),
		status:  deriveWorkflowStatus(item),
		lastRun: relativeTime(updatedAt),
		nodes:   len(item.Nodes),
		recentRuns: []string{
			"Updated " + relativeTime(updatedAt),
			"Created " + relativeTime(insertedAt),
			fmt.Sprintf("%d connections", len(item.Connections)),
		},
	}
}

func deriveWorkflowStatus(item apiFlow) string {
	status := ""
	for _, node := range item.Nodes {
		if raw, ok := node.Controls["status"]; ok {
			if value, ok := raw.(string); ok && strings.TrimSpace(value) != "" {
				status = strings.TrimSpace(value)
			}
		}
	}

	switch strings.ToLower(status) {
	case "failed", "error":
		return "Failed"
	case "running":
		return "Running"
	case "success":
		return "Active"
	}

	if len(item.Nodes) > 0 {
		return "Active"
	}
	return "Idle"
}

func parseAPITime(value string) time.Time {
	if strings.TrimSpace(value) == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return t
}
