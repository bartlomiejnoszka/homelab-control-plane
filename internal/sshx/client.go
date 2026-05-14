package sshx

import (
	"context"
	"fmt"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

// Runner is the small interface needed by code that executes remote commands.
// Keeping it tiny makes tests easy because a fake can implement one method.
type Runner interface {
	Run(ctx context.Context, command string) (string, error)
}

// SSHRunner executes commands over SSH using private-key authentication.
type SSHRunner struct {
	host       string
	port       int
	user       string
	keyPath    string
	timeout    time.Duration
	dialerFunc func(network, addr string, config *ssh.ClientConfig) (*ssh.Client, error)
}

// NewSSHRunner constructs an SSH-backed Runner.
// In PHP terms, this is a factory function instead of a service container.
func NewSSHRunner(host string, port int, user, keyPath string, timeout time.Duration) *SSHRunner {
	return &SSHRunner{host: host, port: port, user: user, keyPath: keyPath, timeout: timeout, dialerFunc: ssh.Dial}
}

// Run connects to the remote host, executes command, and returns stdout/stderr
// combined as a string when the command succeeds.
func (r *SSHRunner) Run(ctx context.Context, command string) (string, error) {
	signer, err := signerFromPrivateKey(r.keyPath)
	if err != nil {
		return "", err
	}
	cfg := &ssh.ClientConfig{User: r.user, Auth: []ssh.AuthMethod{ssh.PublicKeys(signer)}, HostKeyCallback: ssh.InsecureIgnoreHostKey(), Timeout: r.timeout} //nolint:gosec
	client, err := r.dialerFunc("tcp", fmt.Sprintf("%s:%d", r.host, r.port), cfg)
	if err != nil {
		return "", fmt.Errorf("ssh connection failed: %w", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("create ssh session: %w", err)
	}
	defer session.Close()

	type result struct {
		out []byte
		err error
	}
	ch := make(chan result, 1)
	go func() {
		out, err := session.CombinedOutput(command)
		ch <- result{out: out, err: err}
	}()

	select {
	case <-ctx.Done():
		_ = client.Close()
		return "", fmt.Errorf("command timeout or canceled: %w", ctx.Err())
	case res := <-ch:
		if res.err != nil {
			return "", fmt.Errorf("command failed: %v: %s", res.err, string(res.out))
		}
		return string(res.out), nil
	}
}

func signerFromPrivateKey(path string) (ssh.Signer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("private key cannot be read: %w", err)
	}
	signer, err := ssh.ParsePrivateKey(data)
	if err != nil {
		return nil, fmt.Errorf("private key parse failed: %w", err)
	}
	return signer, nil
}
