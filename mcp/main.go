// Command kubedrive-mcp is the stdio bridge to KubeDrive's MCP server, for
// clients that speak only stdio (Claude Desktop). It ships beside the app in
// KubeDrive.app/Contents/MacOS and in the MCP bundle; the AI Agents window
// shows the config that runs it, with the client token in the environment:
//
//	{"mcpServers":{"kubedrive":{"command":"/Applications/KubeDrive.app/Contents/MacOS/kubedrive-mcp",
//	  "env":{"KUBEDRIVE_MCP_TOKEN":"kd_…"}}}}
//
// The token comes only from the environment: a flag would show it to every
// user of the machine in the process list. KUBEDRIVE_MCP_URL replaces the
// server's address when the port was changed. Without the token, or with
// KubeDrive not running, the bridge still answers initialize and tools/list
// (see internal/mcpbridge) and fails tool calls with the reason.
//
// Started by hand in a terminal (from the Start menu in the Microsoft Store
// package, which has to list it there), it explains what it is and waits for
// Enter instead of reading MCP from the keyboard.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/kubedrive/kubedrive/mcp/mcpbridge"
)

// version is set at build time (scripts/mcpb.sh, the app's build).
var version = "dev"

func main() { os.Exit(run()) }

// run returns the exit code: 2 for a bad invocation, 1 for a failure.
func run() int {
	url := flag.String("url", envOr("KUBEDRIVE_MCP_URL", mcpbridge.DefaultURL), "KubeDrive's MCP endpoint")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "kubedrive-mcp: forwards MCP over stdio to KubeDrive's local server; the client token comes from KUBEDRIVE_MCP_TOKEN.")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() > 0 {
		flag.Usage()
		return 2
	}
	if interactive(os.Stdin) {
		fmt.Println("KubeDrive MCP bridge: AI agents such as Claude Desktop start it to reach KubeDrive.")
		fmt.Println("To connect an agent, open KubeDrive → AI Agents and copy the configuration shown there.")
		fmt.Print("\nPress Enter to close.")
		_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
		return 0
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	err := mcpbridge.Run(ctx, os.Stdin, os.Stdout, mcpbridge.Options{
		URL: *url, Token: os.Getenv("KUBEDRIVE_MCP_TOKEN"), Version: version,
		Log: func(format string, args ...any) { fmt.Fprintf(os.Stderr, "kubedrive-mcp: "+format+"\n", args...) },
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "kubedrive-mcp:", err)
		return 1
	}
	return 0
}

// interactive reports whether f is a terminal a person types into; MCP
// clients connect the bridge through pipes.
func interactive(f *os.File) bool {
	st, err := f.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}

func envOr(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}
