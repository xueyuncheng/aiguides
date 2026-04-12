package tools

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"time"

	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/middleware"

	gossh "golang.org/x/crypto/ssh"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

const (
	sshDialTimeout    = 15 * time.Second
	sshCommandTimeout = 60 * time.Second
	sshMaxOutputBytes = 1 << 20 // 1 MiB cap per stream
)

// SSHExecuteInput defines the input parameters for the ssh_execute tool.
type SSHExecuteInput struct {
	// ServerName is the Name field of the SSH server config to use.
	// If empty and the user has a default server configured, that one is used.
	ServerName string `json:"server_name,omitempty" jsonschema:"Name of the SSH server config to use. Leave empty to use the default server."`
	// Command is the shell command to run on the remote machine.
	Command string `json:"command" jsonschema:"Shell command to execute on the remote server."`
}

// SSHExecuteOutput holds the result of running a remote SSH command.
type SSHExecuteOutput struct {
	Success     bool   `json:"success"`
	Stdout      string `json:"stdout,omitempty"`
	Stderr      string `json:"stderr,omitempty"`
	ExitCode    int    `json:"exit_code"`
	Host        string `json:"host,omitempty"`
	Error       string `json:"error,omitempty"`
	NeedsConfig bool   `json:"needs_config,omitempty"`
}

// NewSSHExecuteTool creates a tool that executes shell commands on a remote
// machine over SSH. Credentials are loaded from the user's stored SSH server
// configs (set up via the /api/ssh_server_configs endpoints).
// Both password and public-key authentication are supported.
func NewSSHExecuteTool() (tool.Tool, error) {
	cfg := functiontool.Config{
		Name: "ssh_execute",
		Description: "Execute a shell command on a remote machine via SSH. " +
			"The user must have at least one SSH server configured at /settings/ssh-servers. " +
			"Specify server_name to target a particular server, or leave it empty to use the default. " +
			"Returns stdout, stderr, and the exit code of the command.",
	}

	handler := func(ctx tool.Context, input SSHExecuteInput) (*SSHExecuteOutput, error) {
		return executeSSHCommand(ctx, input)
	}

	return functiontool.New(cfg, handler)
}

// executeSSHCommand loads credentials from the DB and runs the requested command.
func executeSSHCommand(ctx context.Context, input SSHExecuteInput) (*SSHExecuteOutput, error) {
	select {
	case <-ctx.Done():
		return &SSHExecuteOutput{Success: false, Error: "operation cancelled"}, ctx.Err()
	default:
	}

	if input.Command == "" {
		return &SSHExecuteOutput{Success: false, Error: "command must not be empty"}, nil
	}

	tx, ok := middleware.GetTx(ctx)
	if !ok {
		slog.Error("middleware.GetTx() failed in executeSSHCommand")
		return &SSHExecuteOutput{Success: false, Error: "internal error: database context unavailable"}, nil
	}

	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		slog.Error("middleware.GetUserID() failed in executeSSHCommand")
		return &SSHExecuteOutput{Success: false, Error: "internal error: user context unavailable"}, nil
	}

	// Resolve the SSH server config.
	var serverCfg table.SSHServerConfig
	if input.ServerName != "" {
		if err := tx.Where("user_id = ? AND name = ?", userID, input.ServerName).
			First(&serverCfg).Error; err != nil {
			slog.Error("db.First() error in executeSSHCommand", "user_id", userID, "server_name", input.ServerName, "err", err)
			return &SSHExecuteOutput{
				Success:     false,
				Error:       fmt.Sprintf("SSH server config %q not found. Please add it at /settings/ssh-servers.", input.ServerName),
				NeedsConfig: true,
			}, nil
		}
	} else {
		// Try the default server first; fall back to the most recently created one.
		if err := tx.Where("user_id = ?", userID).
			Order("is_default DESC, created_at DESC").
			First(&serverCfg).Error; err != nil {
			slog.Error("db.First() error finding default SSH config", "user_id", userID, "err", err)
			return &SSHExecuteOutput{
				Success:     false,
				Error:       "No SSH server config found. Please add one at /settings/ssh-servers.",
				NeedsConfig: true,
			}, nil
		}
	}

	authMethods, err := buildAuthMethods(serverCfg)
	if err != nil {
		slog.Error("buildAuthMethods() error", "host", serverCfg.Host, "err", err)
		return &SSHExecuteOutput{
			Success: false,
			Host:    fmt.Sprintf("%s:%d", serverCfg.Host, serverCfg.Port),
			Error:   err.Error(),
		}, nil
	}

	host := fmt.Sprintf("%s:%d", serverCfg.Host, serverCfg.Port)
	slog.Info("ssh_execute: connecting", "host", host, "user", serverCfg.Username,
		"auth_method", serverCfg.AuthMethod, "command", input.Command)

	output, err := runSSHCommand(ctx, host, serverCfg.Username, authMethods, input.Command)
	if err != nil {
		slog.Error("runSSHCommand() error", "host", host, "err", err)
		return &SSHExecuteOutput{
			Success: false,
			Host:    host,
			Error:   err.Error(),
		}, nil
	}

	return output, nil
}

// buildAuthMethods converts a stored SSHServerConfig into the slice of
// gossh.AuthMethod values that the SSH client will try in order.
func buildAuthMethods(cfg table.SSHServerConfig) ([]gossh.AuthMethod, error) {
	// Treat an empty AuthMethod as "password" for backwards compatibility.
	method := cfg.AuthMethod
	if method == "" {
		method = table.SSHAuthMethodPassword
	}

	switch method {
	case table.SSHAuthMethodPassword:
		return []gossh.AuthMethod{gossh.Password(cfg.Password)}, nil

	case table.SSHAuthMethodKey:
		if cfg.PrivateKey == "" {
			return nil, fmt.Errorf("auth_method is %q but no private key is stored", method)
		}
		signer, err := parsePrivateKey([]byte(cfg.PrivateKey), []byte(cfg.Passphrase))
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		return []gossh.AuthMethod{gossh.PublicKeys(signer)}, nil

	default:
		return nil, fmt.Errorf("unknown auth_method %q; expected %q or %q",
			method, table.SSHAuthMethodPassword, table.SSHAuthMethodKey)
	}
}

// parsePrivateKey parses a PEM-encoded private key, decrypting it with
// passphrase if it is non-empty.
func parsePrivateKey(pemBytes, passphrase []byte) (gossh.Signer, error) {
	if len(passphrase) > 0 {
		return gossh.ParsePrivateKeyWithPassphrase(pemBytes, passphrase)
	}
	return gossh.ParsePrivateKey(pemBytes)
}

// runSSHCommand dials the SSH server, runs cmd, and returns the captured output.
// A non-zero exit code is reflected in output.ExitCode but is NOT treated as an
// error at the Go level — the caller decides what to do with it.
func runSSHCommand(ctx context.Context, host, username string, authMethods []gossh.AuthMethod, cmd string) (*SSHExecuteOutput, error) {
	sshCfg := &gossh.ClientConfig{
		User: username,
		Auth: authMethods,
		// InsecureIgnoreHostKey is acceptable for an AI-assistant use-case where
		// the user is deliberately connecting to a host they control.
		// For production hardening, replace with a known-hosts verifier.
		HostKeyCallback: gossh.InsecureIgnoreHostKey(), //nolint:gosec
		Timeout:         sshDialTimeout,
	}

	// Respect context cancellation during dial.
	dialDone := make(chan struct {
		client *gossh.Client
		err    error
	}, 1)
	go func() {
		client, err := gossh.Dial("tcp", host, sshCfg)
		dialDone <- struct {
			client *gossh.Client
			err    error
		}{client, err}
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled while dialling SSH host %s", host)
	case result := <-dialDone:
		if result.err != nil {
			return nil, fmt.Errorf("SSH dial error for host %s: %w", host, result.err)
		}
		defer result.client.Close()
		return runOnClient(ctx, result.client, host, cmd)
	}
}

// runOnClient opens a session on an already-established SSH client and runs cmd.
func runOnClient(ctx context.Context, client *gossh.Client, host, cmd string) (*SSHExecuteOutput, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to open SSH session on %s: %w", host, err)
	}
	defer session.Close()

	var stdoutBuf, stderrBuf bytes.Buffer
	session.Stdout = &limitedWriter{buf: &stdoutBuf, limit: sshMaxOutputBytes}
	session.Stderr = &limitedWriter{buf: &stderrBuf, limit: sshMaxOutputBytes}

	// Run with a timeout derived from the context or our hard cap.
	cmdCtx, cancel := context.WithTimeout(ctx, sshCommandTimeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- session.Run(cmd)
	}()

	var runErr error
	select {
	case <-cmdCtx.Done():
		_ = session.Signal(gossh.SIGKILL)
		return nil, fmt.Errorf("SSH command timed out on %s", host)
	case runErr = <-done:
	}

	exitCode := 0
	if runErr != nil {
		if exitErr, ok := runErr.(*gossh.ExitError); ok {
			exitCode = exitErr.ExitStatus()
			runErr = nil // non-zero exit is not a Go-level error
		} else {
			return nil, fmt.Errorf("SSH run error on %s: %w", host, runErr)
		}
	}

	return &SSHExecuteOutput{
		Success:  exitCode == 0,
		Stdout:   stdoutBuf.String(),
		Stderr:   stderrBuf.String(),
		ExitCode: exitCode,
		Host:     host,
	}, nil
}

// SSHListServersInput defines the (empty) input for the ssh_list_servers tool.
type SSHListServersInput struct{}

// SSHListServersOutput holds the list of configured SSH servers.
type SSHListServersOutput struct {
	Servers []SSHServerInfo `json:"servers"`
	Count   int             `json:"count"`
}

// SSHServerInfo is a single SSH server entry returned by ssh_list_servers.
// Credentials (password, private key, passphrase) are intentionally omitted.
type SSHServerInfo struct {
	ID         int                 `json:"id"`
	Name       string              `json:"name"`
	Host       string              `json:"host"`
	Port       int                 `json:"port"`
	Username   string              `json:"username"`
	AuthMethod table.SSHAuthMethod `json:"auth_method"`
	IsDefault  bool                `json:"is_default"`
}

// NewSSHListServersTool creates a tool that lists the user's configured SSH servers.
// Credentials are never included in the output.
func NewSSHListServersTool() (tool.Tool, error) {
	cfg := functiontool.Config{
		Name: "ssh_list_servers",
		Description: "List all SSH servers configured by the user. " +
			"Returns server names, hosts, ports, usernames, and auth methods. " +
			"Credentials (passwords, private keys) are never included in the output. " +
			"Use this before ssh_execute to discover available server names.",
	}

	handler := func(ctx tool.Context, _ SSHListServersInput) (*SSHListServersOutput, error) {
		tx, ok := middleware.GetTx(ctx)
		if !ok {
			slog.Error("middleware.GetTx() failed in ssh_list_servers")
			return &SSHListServersOutput{}, nil
		}

		userID, ok := middleware.GetUserID(ctx)
		if !ok {
			slog.Error("middleware.GetUserID() failed in ssh_list_servers")
			return &SSHListServersOutput{}, nil
		}

		var configs []table.SSHServerConfig
		if err := tx.Where("user_id = ?", userID).
			Order("is_default DESC, created_at DESC").
			Find(&configs).Error; err != nil {
			slog.Error("db.Find() error in ssh_list_servers", "user_id", userID, "err", err)
			return &SSHListServersOutput{}, nil
		}

		servers := make([]SSHServerInfo, len(configs))
		for i, c := range configs {
			servers[i] = SSHServerInfo{
				ID:         c.ID,
				Name:       c.Name,
				Host:       c.Host,
				Port:       c.Port,
				Username:   c.Username,
				AuthMethod: c.AuthMethod,
				IsDefault:  c.IsDefault,
			}
		}
		return &SSHListServersOutput{Servers: servers, Count: len(servers)}, nil
	}

	return functiontool.New(cfg, handler)
}

// limitedWriter caps writes to a bytes.Buffer at a maximum byte count.
type limitedWriter struct {
	buf     *bytes.Buffer
	limit   int
	written int
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	remaining := w.limit - w.written
	if remaining <= 0 {
		return len(p), nil // silently discard overflow
	}
	if len(p) > remaining {
		p = p[:remaining]
	}
	n, err := w.buf.Write(p)
	w.written += n
	return n, err
}
