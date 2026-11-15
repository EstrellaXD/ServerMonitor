package collectors

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHConnection holds a lazily-opened, lock-guarded SSH client used by the
// Linux and UNAS collectors. Connections are recreated automatically on
// any error.
type SSHConnection struct {
	host     string
	port     int
	username string
	password string
	keyPath  string

	mu     sync.Mutex
	client *ssh.Client
}

func NewSSHConnection(host string, port int, username, password, keyPath string) *SSHConnection {
	return &SSHConnection{
		host:     host,
		port:     port,
		username: username,
		password: password,
		keyPath:  keyPath,
	}
}

func (c *SSHConnection) buildAuth() ([]ssh.AuthMethod, error) {
	if c.keyPath != "" {
		data, err := os.ReadFile(c.keyPath)
		if err != nil {
			return nil, fmt.Errorf("read key: %w", err)
		}
		signer, err := ssh.ParsePrivateKey(data)
		if err != nil {
			return nil, fmt.Errorf("parse key: %w", err)
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
	}
	if c.password != "" {
		// Some sshd builds (e.g. UniFi UNAS) disable the "password" method
		// and ask for the password via keyboard-interactive instead.
		answer := func(_, _ string, questions []string, _ []bool) ([]string, error) {
			answers := make([]string, len(questions))
			for i := range answers {
				answers[i] = c.password
			}
			return answers, nil
		}
		return []ssh.AuthMethod{
			ssh.Password(c.password),
			ssh.KeyboardInteractive(answer),
		}, nil
	}
	return nil, errors.New("no authentication method provided")
}

func (c *SSHConnection) ensure() error {
	if c.client != nil {
		// Reuse if still healthy; ssh.Client doesn't expose a public ping,
		// so rely on session-open failure to detect a dead conn and reset.
		return nil
	}
	auth, err := c.buildAuth()
	if err != nil {
		return err
	}
	cfg := &ssh.ClientConfig{
		User:            c.username,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}
	addr := net.JoinHostPort(c.host, fmt.Sprintf("%d", c.port))
	client, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		return fmt.Errorf("connect %s: %w", addr, err)
	}
	c.client = client
	return nil
}

// Run executes cmd on the remote host and returns stdout. Honors ctx
// cancellation. On any error, the underlying connection is closed so the
// next call reconnects.
func (c *SSHConnection) Run(ctx context.Context, cmd string) (string, error) {
	out, _, err := c.exec(ctx, cmd)
	var ee *execError
	if errors.As(err, &ee) {
		return "", fmt.Errorf("exec %q: %w", cmd, ee.err)
	}
	return out, err
}

// RunAction executes a control command and returns stdout. Unlike Run, a
// failed command reports its stderr instead of the (long) command text.
func (c *SSHConnection) RunAction(ctx context.Context, cmd string) (string, error) {
	out, stderr, err := c.exec(ctx, cmd)
	var ee *execError
	if errors.As(err, &ee) {
		if msg := strings.TrimSpace(stderr); msg != "" {
			return "", errors.New(msg)
		}
		return "", ee.err
	}
	return out, err
}

// execError marks a failure of the remote command itself, as opposed to a
// connection or session error.
type execError struct{ err error }

func (e *execError) Error() string { return e.err.Error() }
func (e *execError) Unwrap() error { return e.err }

// exec runs cmd and returns stdout and stderr.
func (c *SSHConnection) exec(ctx context.Context, cmd string) (string, string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensure(); err != nil {
		return "", "", err
	}

	sess, err := c.client.NewSession()
	if err != nil {
		c.dropLocked()
		return "", "", fmt.Errorf("session open: %w", err)
	}
	defer sess.Close()

	var out, stderr bytes.Buffer
	sess.Stdout = &out
	sess.Stderr = &stderr

	done := make(chan error, 1)
	go func() { done <- sess.Run(cmd) }()

	select {
	case <-ctx.Done():
		_ = sess.Signal(ssh.SIGKILL)
		c.dropLocked()
		return "", "", ctx.Err()
	case err := <-done:
		if err != nil {
			c.dropLocked()
			return "", stderr.String(), &execError{err}
		}
	}

	return out.String(), stderr.String(), nil
}

func (c *SSHConnection) dropLocked() {
	if c.client != nil {
		_ = c.client.Close()
		c.client = nil
	}
}

func (c *SSHConnection) Close() {
	c.mu.Lock()
	c.dropLocked()
	c.mu.Unlock()
}
