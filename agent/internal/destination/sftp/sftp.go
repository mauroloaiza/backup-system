// Package sftp implements a backup destination over SFTP (SSH File Transfer Protocol).
// Supports both password and private-key authentication.
package sftp

import (
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"strings"

	gossh "golang.org/x/crypto/ssh"

	"github.com/pkg/sftp"
)

// Client is a destination.Writer backed by an SFTP server.
type Client struct {
	ssh    *gossh.Client
	sftp   *sftp.Client
	root   string // remote base path
}

// New connects to an SFTP server and returns a Client.
// Authentication priority: keyFile > password > none.
func New(host string, port int, user, password, keyFile, remotePath string) (*Client, error) {
	var authMethods []gossh.AuthMethod

	if keyFile != "" {
		am, err := keyFileAuth(keyFile, password)
		if err != nil {
			return nil, fmt.Errorf("sftp: load key %q: %w", keyFile, err)
		}
		authMethods = append(authMethods, am)
	}

	if password != "" {
		authMethods = append(authMethods, gossh.Password(password))
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("sftp: no authentication method configured (need password or key_file)")
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	sshCfg := &gossh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: gossh.InsecureIgnoreHostKey(), // TODO: load known_hosts
		Timeout:         0,
	}

	sshConn, err := gossh.Dial("tcp", addr, sshCfg)
	if err != nil {
		return nil, fmt.Errorf("sftp: ssh dial %s: %w", addr, err)
	}

	sftpClient, err := sftp.NewClient(sshConn)
	if err != nil {
		sshConn.Close()
		return nil, fmt.Errorf("sftp: open sftp session: %w", err)
	}

	// Ensure root path exists on the server.
	if remotePath == "" {
		remotePath = "."
	}
	if err := sftpClient.MkdirAll(remotePath); err != nil {
		sftpClient.Close()
		sshConn.Close()
		return nil, fmt.Errorf("sftp: mkdir %q: %w", remotePath, err)
	}

	return &Client{ssh: sshConn, sftp: sftpClient, root: remotePath}, nil
}

// Write creates (or overwrites) an object at <root>/<name>.
// Parent directories are created on demand.
func (c *Client) Write(name string) (io.WriteCloser, error) {
	full := c.abs(name)
	dir := path.Dir(full)
	if err := c.sftp.MkdirAll(dir); err != nil {
		return nil, fmt.Errorf("sftp write mkdir %q: %w", dir, err)
	}
	f, err := c.sftp.OpenFile(full, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return nil, fmt.Errorf("sftp write open %q: %w", full, err)
	}
	return f, nil
}

// Read opens an object for reading.
func (c *Client) Read(name string) (io.ReadCloser, error) {
	f, err := c.sftp.Open(c.abs(name))
	if err != nil {
		return nil, fmt.Errorf("sftp read %q: %w", name, err)
	}
	return f, nil
}

// Delete removes a named object.
func (c *Client) Delete(name string) error {
	if err := c.sftp.Remove(c.abs(name)); err != nil {
		return fmt.Errorf("sftp delete %q: %w", name, err)
	}
	return nil
}

// List returns all object names under the given prefix, relative to root.
func (c *Client) List(prefix string) ([]string, error) {
	base := c.abs(prefix)
	// Strip trailing slash for Walk.
	base = strings.TrimRight(base, "/")

	walker := c.sftp.Walk(base)
	var out []string
	for walker.Step() {
		if err := walker.Err(); err != nil {
			continue // skip unreadable entries
		}
		if walker.Stat().IsDir() {
			continue
		}
		// Make path relative to root.
		rel := strings.TrimPrefix(walker.Path(), c.root+"/")
		out = append(out, rel)
	}
	return out, nil
}

// Close closes the SFTP session and the underlying SSH connection.
func (c *Client) Close() error {
	_ = c.sftp.Close()
	return c.ssh.Close()
}

// abs returns the absolute remote path for a relative object name.
func (c *Client) abs(name string) string {
	return path.Join(c.root, name)
}

// keyFileAuth loads a PEM private key, optionally decrypting with passphrase.
func keyFileAuth(keyFile, passphrase string) (gossh.AuthMethod, error) {
	keyBytes, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("read key file: %w", err)
	}

	var signer gossh.Signer
	if passphrase != "" {
		signer, err = gossh.ParsePrivateKeyWithPassphrase(keyBytes, []byte(passphrase))
	} else {
		signer, err = gossh.ParsePrivateKey(keyBytes)
	}
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	return gossh.PublicKeys(signer), nil
}

// Ensure net is imported (used indirectly via ssh.Dial).
var _ = net.Dial
