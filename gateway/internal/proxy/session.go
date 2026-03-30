package proxy

import (
	"context"
	"fmt"
	"os/exec"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/bobbyhouse/plugins/gateway/internal/profile"
)

// Session holds the running state for one loaded profile.
type Session struct {
	profile *profile.Profile
	clients map[string]*mcp.ClientSession // server name → MCP stdio client session
	procs   []*exec.Cmd                   // docker run processes (for cleanup)
}

// NewSession starts a docker container for each server in the profile and
// completes the MCP handshake.
func NewSession(ctx context.Context, p *profile.Profile) (*Session, error) {
	s := &Session{
		profile: p,
		clients: make(map[string]*mcp.ClientSession),
	}

	for _, srv := range p.Servers {
		args := []string{"run", "--rm", "-i"}
		for k, v := range srv.Config {
			args = append(args, "-e", k+"="+v)
		}
		for _, mount := range srv.Mounts {
			args = append(args, "-v", mount)
		}
		args = append(args, srv.Identifier)

		cmd := exec.CommandContext(context.Background(), "docker", args...)
		stdin, err := cmd.StdinPipe()
		if err != nil {
			s.Close()
			return nil, fmt.Errorf("server %s: stdin pipe: %w", srv.Name, err)
		}
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			s.Close()
			return nil, fmt.Errorf("server %s: stdout pipe: %w", srv.Name, err)
		}
		if err := cmd.Start(); err != nil {
			s.Close()
			return nil, fmt.Errorf("server %s: start docker: %w", srv.Name, err)
		}
		s.procs = append(s.procs, cmd)

		transport := &mcp.IOTransport{Reader: stdout, Writer: stdin}
		c := mcp.NewClient(&mcp.Implementation{Name: "profile-gateway", Version: "1.0.0"}, nil)
		cs, err := c.Connect(ctx, transport, nil)
		if err != nil {
			s.Close()
			return nil, fmt.Errorf("server %s: MCP connect: %w", srv.Name, err)
		}
		s.clients[srv.Name] = cs
	}

	return s, nil
}

// Clients returns the map of server name to MCP client session.
func (s *Session) Clients() map[string]*mcp.ClientSession {
	return s.clients
}

// Close sends SIGTERM to all running docker processes.
func (s *Session) Close() {
	for _, cs := range s.clients {
		_ = cs.Close()
	}
	for _, cmd := range s.procs {
		if cmd.Process != nil {
			_ = cmd.Process.Signal(syscall.SIGTERM)
			_ = cmd.Wait()
		}
	}
}
