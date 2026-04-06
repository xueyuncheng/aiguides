package tools

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"

	"aiguide/internal/app/aiguide/table"

	gossh "golang.org/x/crypto/ssh"
)

// TestNewSSHExecuteTool verifies the constructor returns a non-nil tool.
func TestNewSSHExecuteTool(t *testing.T) {
	tool, err := NewSSHExecuteTool()
	if err != nil {
		t.Fatalf("NewSSHExecuteTool() error = %v", err)
	}
	if tool == nil {
		t.Fatal("NewSSHExecuteTool() returned nil")
	}
}

// TestSSHExecute_EmptyCommand verifies that an empty command returns Success=false.
func TestSSHExecute_EmptyCommand(t *testing.T) {
	output, err := executeSSHCommand(context.Background(), SSHExecuteInput{Command: ""})
	if err != nil {
		t.Fatalf("executeSSHCommand() unexpected Go error: %v", err)
	}
	if output.Success {
		t.Error("expected Success=false for empty command")
	}
	if output.Error == "" {
		t.Error("expected non-empty Error for empty command")
	}
}

// TestSSHExecute_MissingDBContext verifies graceful failure when no DB is in context.
func TestSSHExecute_MissingDBContext(t *testing.T) {
	output, err := executeSSHCommand(context.Background(), SSHExecuteInput{Command: "echo hi"})
	if err != nil {
		t.Fatalf("executeSSHCommand() unexpected Go error: %v", err)
	}
	if output.Success {
		t.Error("expected Success=false when DB context is missing")
	}
	if !strings.Contains(output.Error, "database context") {
		t.Errorf("unexpected error message: %q", output.Error)
	}
}

// TestSSHExecute_CancelledContext verifies the context cancellation path.
func TestSSHExecute_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	output, err := executeSSHCommand(ctx, SSHExecuteInput{Command: "echo hi"})
	// Either a structured output or a context error is acceptable.
	if err == nil && output != nil && output.Success {
		t.Error("expected failure for already-cancelled context")
	}
}

// TestLimitedWriter_CapsBytesWritten verifies that limitedWriter stops at limit.
func TestLimitedWriter_CapsBytesWritten(t *testing.T) {
	var buf bytes.Buffer
	lw := &limitedWriter{buf: &buf, limit: 10}

	n, err := lw.Write([]byte("hello world! this is way more than ten bytes"))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	// n should equal len(input) — limitedWriter silently discards the excess.
	if n == 0 {
		t.Fatal("Write() returned 0 bytes written")
	}
	if buf.Len() > 10 {
		t.Errorf("buffer exceeds limit: got %d bytes, want <= 10", buf.Len())
	}
}

// TestLimitedWriter_MultipleWrites verifies accumulation across writes.
func TestLimitedWriter_MultipleWrites(t *testing.T) {
	var buf bytes.Buffer
	lw := &limitedWriter{buf: &buf, limit: 5}

	lw.Write([]byte("abc")) //nolint:errcheck
	lw.Write([]byte("def")) //nolint:errcheck

	if buf.Len() > 5 {
		t.Errorf("buffer exceeds limit after two writes: got %d bytes", buf.Len())
	}
	if !strings.HasPrefix(buf.String(), "abcde") {
		t.Errorf("unexpected buffer content: %q", buf.String())
	}
}

// ---------------------------------------------------------------------------
// In-process mock SSH server helpers
// ---------------------------------------------------------------------------

// startMockSSHServer starts a minimal in-process SSH server that accepts
// password authentication and executes the requested command via a simple
// built-in dispatcher (no real shell). It returns the listener address and a
// stop function.
func startMockSSHServer(t *testing.T, username, password string, handler func(cmd string) (stdout, stderr string, exitCode int)) (addr string, stop func()) {
	t.Helper()

	// Generate host key.
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey() error = %v", err)
	}
	hostSigner, err := gossh.NewSignerFromKey(privateKey)
	if err != nil {
		t.Fatalf("gossh.NewSignerFromKey() error = %v", err)
	}

	config := &gossh.ServerConfig{
		PasswordCallback: func(conn gossh.ConnMetadata, pass []byte) (*gossh.Permissions, error) {
			if conn.User() == username && string(pass) == password {
				return nil, nil
			}
			return nil, fmt.Errorf("invalid credentials")
		},
	}
	config.AddHostKey(hostSigner)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return // listener closed
			}
			go handleMockSSHConn(conn, config, handler)
		}
	}()

	return ln.Addr().String(), func() { ln.Close() }
}

// startMockSSHServerWithKey starts a minimal in-process SSH server that only
// accepts public-key authentication for the given authorisedKey. It returns the
// listener address and a stop function.
func startMockSSHServerWithKey(t *testing.T, username string, authorisedKey gossh.PublicKey, handler func(cmd string) (stdout, stderr string, exitCode int)) (addr string, stop func()) {
	t.Helper()

	hostKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey() for host key: %v", err)
	}
	hostSigner, err := gossh.NewSignerFromKey(hostKey)
	if err != nil {
		t.Fatalf("gossh.NewSignerFromKey() for host key: %v", err)
	}

	authorisedKeyBytes := authorisedKey.Marshal()

	config := &gossh.ServerConfig{
		PublicKeyCallback: func(conn gossh.ConnMetadata, key gossh.PublicKey) (*gossh.Permissions, error) {
			if conn.User() == username && bytes.Equal(key.Marshal(), authorisedKeyBytes) {
				return nil, nil
			}
			return nil, fmt.Errorf("unauthorised key")
		},
	}
	config.AddHostKey(hostSigner)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go handleMockSSHConn(conn, config, handler)
		}
	}()

	return ln.Addr().String(), func() { ln.Close() }
}

func handleMockSSHConn(conn net.Conn, config *gossh.ServerConfig, handler func(cmd string) (string, string, int)) {
	sshConn, chans, reqs, err := gossh.NewServerConn(conn, config)
	if err != nil {
		return
	}
	defer sshConn.Close()

	go gossh.DiscardRequests(reqs)

	for newChan := range chans {
		if newChan.ChannelType() != "session" {
			newChan.Reject(gossh.UnknownChannelType, "unsupported channel type") //nolint:errcheck
			continue
		}
		ch, requests, err := newChan.Accept()
		if err != nil {
			continue
		}
		go func(ch gossh.Channel, requests <-chan *gossh.Request) {
			defer ch.Close()
			for req := range requests {
				if req.Type != "exec" {
					if req.WantReply {
						req.Reply(false, nil) //nolint:errcheck
					}
					continue
				}
				// Parse the command length-prefixed payload.
				if len(req.Payload) < 4 {
					if req.WantReply {
						req.Reply(false, nil) //nolint:errcheck
					}
					continue
				}
				cmdLen := int(req.Payload[0])<<24 | int(req.Payload[1])<<16 | int(req.Payload[2])<<8 | int(req.Payload[3])
				if len(req.Payload) < 4+cmdLen {
					if req.WantReply {
						req.Reply(false, nil) //nolint:errcheck
					}
					continue
				}
				cmd := string(req.Payload[4 : 4+cmdLen])
				if req.WantReply {
					req.Reply(true, nil) //nolint:errcheck
				}

				stdout, stderr, exitCode := handler(cmd)
				io.WriteString(ch, stdout)          //nolint:errcheck
				io.WriteString(ch.Stderr(), stderr) //nolint:errcheck

				exitPayload := gossh.Marshal(struct{ Status uint32 }{uint32(exitCode)})
				ch.SendRequest("exit-status", false, exitPayload) //nolint:errcheck
				return
			}
		}(ch, requests)
	}
}

// ---------------------------------------------------------------------------
// Integration tests using the mock SSH server — password auth
// ---------------------------------------------------------------------------

// TestRunSSHCommand_Success verifies stdout capture and exit code 0.
func TestRunSSHCommand_Success(t *testing.T) {
	addr, stop := startMockSSHServer(t, "user", "pass", func(cmd string) (string, string, int) {
		return "hello\n", "", 0
	})
	defer stop()

	authMethods := []gossh.AuthMethod{gossh.Password("pass")}
	output, err := runSSHCommand(context.Background(), addr, "user", authMethods, "echo hello")
	if err != nil {
		t.Fatalf("runSSHCommand() error = %v", err)
	}
	if !output.Success {
		t.Errorf("expected Success=true, got error: %s", output.Error)
	}
	if output.ExitCode != 0 {
		t.Errorf("expected ExitCode=0, got %d", output.ExitCode)
	}
	if !strings.Contains(output.Stdout, "hello") {
		t.Errorf("expected stdout to contain 'hello', got: %q", output.Stdout)
	}
}

// TestRunSSHCommand_NonZeroExit verifies that a non-zero exit is reflected in
// ExitCode and Success=false, but not returned as a Go error.
func TestRunSSHCommand_NonZeroExit(t *testing.T) {
	addr, stop := startMockSSHServer(t, "user", "pass", func(cmd string) (string, string, int) {
		return "", "command not found\n", 127
	})
	defer stop()

	authMethods := []gossh.AuthMethod{gossh.Password("pass")}
	output, err := runSSHCommand(context.Background(), addr, "user", authMethods, "nonexistent")
	if err != nil {
		t.Fatalf("runSSHCommand() unexpected Go error for non-zero exit: %v", err)
	}
	if output.Success {
		t.Error("expected Success=false for exit code 127")
	}
	if output.ExitCode != 127 {
		t.Errorf("expected ExitCode=127, got %d", output.ExitCode)
	}
	if !strings.Contains(output.Stderr, "command not found") {
		t.Errorf("expected stderr to contain 'command not found', got: %q", output.Stderr)
	}
}

// TestRunSSHCommand_WrongPassword verifies that invalid credentials return an error.
func TestRunSSHCommand_WrongPassword(t *testing.T) {
	addr, stop := startMockSSHServer(t, "user", "correctpass", func(cmd string) (string, string, int) {
		return "ok", "", 0
	})
	defer stop()

	authMethods := []gossh.AuthMethod{gossh.Password("wrongpass")}
	_, err := runSSHCommand(context.Background(), addr, "user", authMethods, "echo hi")
	if err == nil {
		t.Fatal("expected error for wrong password, got nil")
	}
}

// TestRunSSHCommand_StderrCapture verifies stderr is captured separately.
func TestRunSSHCommand_StderrCapture(t *testing.T) {
	addr, stop := startMockSSHServer(t, "user", "pass", func(cmd string) (string, string, int) {
		return "out\n", "err\n", 0
	})
	defer stop()

	authMethods := []gossh.AuthMethod{gossh.Password("pass")}
	output, err := runSSHCommand(context.Background(), addr, "user", authMethods, "cmd")
	if err != nil {
		t.Fatalf("runSSHCommand() error = %v", err)
	}
	if !strings.Contains(output.Stdout, "out") {
		t.Errorf("expected stdout 'out', got %q", output.Stdout)
	}
	if !strings.Contains(output.Stderr, "err") {
		t.Errorf("expected stderr 'err', got %q", output.Stderr)
	}
}

// ---------------------------------------------------------------------------
// Integration tests using the mock SSH server — public-key auth
// ---------------------------------------------------------------------------

// generateRSAKeyPairForTest generates an RSA key pair and returns the private
// key as a PEM-encoded byte slice plus the corresponding gossh.PublicKey.
func generateRSAKeyPairForTest(t *testing.T) (privatePEM []byte, pub gossh.PublicKey) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey(): %v", err)
	}
	der := x509.MarshalPKCS1PrivateKey(privateKey)
	privatePEM = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: der})

	signer, err := gossh.NewSignerFromKey(privateKey)
	if err != nil {
		t.Fatalf("gossh.NewSignerFromKey(): %v", err)
	}
	return privatePEM, signer.PublicKey()
}

// TestRunSSHCommand_PublicKeyAuth verifies end-to-end public-key authentication
// against a mock SSH server that only accepts public-key auth.
func TestRunSSHCommand_PublicKeyAuth(t *testing.T) {
	privatePEM, pubKey := generateRSAKeyPairForTest(t)

	addr, stop := startMockSSHServerWithKey(t, "user", pubKey, func(cmd string) (string, string, int) {
		return "key-auth-ok\n", "", 0
	})
	defer stop()

	signer, err := gossh.ParsePrivateKey(privatePEM)
	if err != nil {
		t.Fatalf("gossh.ParsePrivateKey(): %v", err)
	}
	authMethods := []gossh.AuthMethod{gossh.PublicKeys(signer)}

	output, err := runSSHCommand(context.Background(), addr, "user", authMethods, "echo key-auth-ok")
	if err != nil {
		t.Fatalf("runSSHCommand() error = %v", err)
	}
	if !output.Success {
		t.Errorf("expected Success=true, got error: %s", output.Error)
	}
	if !strings.Contains(output.Stdout, "key-auth-ok") {
		t.Errorf("expected stdout to contain 'key-auth-ok', got: %q", output.Stdout)
	}
}

// ---------------------------------------------------------------------------
// Unit tests for buildAuthMethods
// ---------------------------------------------------------------------------

// newTestSSHConfig builds a minimal SSHServerConfig for buildAuthMethods tests.
func newTestSSHConfig(authMethod table.SSHAuthMethod, password, privateKey, passphrase string) table.SSHServerConfig {
	return table.SSHServerConfig{
		AuthMethod: authMethod,
		Password:   password,
		PrivateKey: privateKey,
		Passphrase: passphrase,
	}
}

// TestBuildAuthMethods_Password verifies that a password config returns exactly
// one auth method without error.
func TestBuildAuthMethods_Password(t *testing.T) {
	cfg := newTestSSHConfig(table.SSHAuthMethodPassword, "secret", "", "")
	methods, err := buildAuthMethods(cfg)
	if err != nil {
		t.Fatalf("buildAuthMethods() error = %v", err)
	}
	if len(methods) != 1 {
		t.Fatalf("expected 1 auth method, got %d", len(methods))
	}
}

// TestBuildAuthMethods_EmptyMethodDefaultsToPassword verifies that an empty
// AuthMethod field is treated as "password" for backwards compatibility.
func TestBuildAuthMethods_EmptyMethodDefaultsToPassword(t *testing.T) {
	cfg := newTestSSHConfig("", "secret", "", "")
	methods, err := buildAuthMethods(cfg)
	if err != nil {
		t.Fatalf("buildAuthMethods() error = %v", err)
	}
	if len(methods) != 1 {
		t.Fatalf("expected 1 auth method, got %d", len(methods))
	}
}

// TestBuildAuthMethods_KeyMissingKey verifies that key auth with no private key
// stored returns an error.
func TestBuildAuthMethods_KeyMissingKey(t *testing.T) {
	cfg := newTestSSHConfig(table.SSHAuthMethodKey, "", "", "")
	_, err := buildAuthMethods(cfg)
	if err == nil {
		t.Fatal("expected error for key auth with empty private key")
	}
}

// TestBuildAuthMethods_KeyValid verifies that a valid PEM private key produces
// one auth method without error.
func TestBuildAuthMethods_KeyValid(t *testing.T) {
	privatePEM, _ := generateRSAKeyPairForTest(t)
	cfg := newTestSSHConfig(table.SSHAuthMethodKey, "", string(privatePEM), "")
	methods, err := buildAuthMethods(cfg)
	if err != nil {
		t.Fatalf("buildAuthMethods() error = %v", err)
	}
	if len(methods) != 1 {
		t.Fatalf("expected 1 auth method, got %d", len(methods))
	}
}

// TestBuildAuthMethods_UnknownMethod verifies that an unknown auth method
// returns an error.
func TestBuildAuthMethods_UnknownMethod(t *testing.T) {
	cfg := newTestSSHConfig("agent", "", "", "")
	_, err := buildAuthMethods(cfg)
	if err == nil {
		t.Fatal("expected error for unknown auth method")
	}
}

// ---------------------------------------------------------------------------
// Unit tests for parsePrivateKey
// ---------------------------------------------------------------------------

// TestParsePrivateKey_Valid verifies parsing a valid unencrypted PEM key.
func TestParsePrivateKey_Valid(t *testing.T) {
	privatePEM, _ := generateRSAKeyPairForTest(t)
	_, err := parsePrivateKey(privatePEM, nil)
	if err != nil {
		t.Fatalf("parsePrivateKey() error = %v", err)
	}
}

// TestParsePrivateKey_Invalid verifies that garbage input returns an error.
func TestParsePrivateKey_Invalid(t *testing.T) {
	_, err := parsePrivateKey([]byte("not-a-pem-key"), nil)
	if err == nil {
		t.Fatal("expected error for invalid PEM, got nil")
	}
}
