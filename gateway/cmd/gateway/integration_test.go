package main

import (
	"bufio"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
	"time"
)

const gatewayBin = "/tmp/gateway-test"

func TestMain(m *testing.M) {
	cmd := exec.Command("go", "build", "-o", gatewayBin, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		panic("build failed: " + string(out))
	}
	m.Run()
}

func TestGatewayIntegration(t *testing.T) {
	cmd := exec.Command(gatewayBin)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start gateway: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	scanner := bufio.NewScanner(stdout)

	send := func(msg string) {
		t.Helper()
		if _, err := stdin.Write([]byte(msg + "\n")); err != nil {
			t.Fatalf("write to gateway: %v", err)
		}
	}

	// readResponse reads lines until it finds a JSON-RPC response (has "id" field),
	// skipping any notifications (which have no "id").
	readResponse := func() string {
		t.Helper()
		deadline := time.After(10 * time.Second)
		for {
			done := make(chan bool, 1)
			var line string
			go func() {
				if scanner.Scan() {
					line = scanner.Text()
				}
				done <- true
			}()
			select {
			case <-done:
			case <-deadline:
				t.Fatal("timeout waiting for gateway response")
			}
			if line == "" {
				continue
			}
			var msg map[string]any
			if err := json.Unmarshal([]byte(line), &msg); err != nil {
				t.Fatalf("invalid JSON from gateway: %v\nraw: %s", err, line)
			}
			if _, hasID := msg["id"]; hasID {
				return line
			}
			// It's a notification — skip and read next line.
		}
	}

	// MCP handshake
	send(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0.0.1"}}}`)
	initResp := readResponse()
	if !strings.Contains(initResp, "profile-gateway") {
		t.Errorf("initialize response missing 'profile-gateway', got: %s", initResp)
	}

	// Send initialized notification
	send(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)

	// load call with bogus profile — expect a valid error result, not a crash
	send(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"load","arguments":{"profile":"docker.io/bogus/does-not-exist:latest","config":{}}}}`)
	callResp := readResponse()
	if callResp == "" {
		t.Fatal("got empty response to tools/call")
	}

	var resp map[string]any
	if err := json.Unmarshal([]byte(callResp), &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v\nraw: %s", err, callResp)
	}
	if _, ok := resp["error"]; ok {
		// A JSON-RPC level error is OK only if it's not a panic exit.
		t.Logf("got JSON-RPC error (acceptable): %s", callResp)
	} else if result, ok := resp["result"]; ok {
		resultMap, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("result is not an object: %T", result)
		}
		// isError should be true since the profile doesn't exist.
		t.Logf("got CallToolResult: isError=%v", resultMap["isError"])
	} else {
		t.Fatalf("response has neither 'error' nor 'result': %s", callResp)
	}

	// Verify gateway hasn't panicked/exited.
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	select {
	case err := <-exited:
		t.Fatalf("gateway exited unexpectedly: %v", err)
	case <-time.After(100 * time.Millisecond):
		// still running — good
	}
}
