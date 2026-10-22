// Package cli implements the non-interactive commands of happ-tui, for
// scripts and status bar widgets. They never need a terminal and never write
// files.
package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	connection "github.com/AnotherProksY/Happ-TUI/lib/Connection"
	sharedtypes "github.com/AnotherProksY/Happ-TUI/shared_types"
	"github.com/AnotherProksY/Happ-TUI/utils"
)

const (
	exitOK    = 0
	exitError = 1
	exitUsage = 2
)

func Usage(w io.Writer) {
	fmt.Fprint(w, `usage: happ-tui [command]

Without a command the interactive TUI starts.

commands:
  status [--json] [--watch] [--interval 2s]
                     show the connection state; --watch prints a line on every
                     change and every interval while connected
  up [--tun] [--profile GROUP:ID] [--timeout 15s] [--json]
                     connect, by default to the last used profile
  down [--json]      disconnect (and turn TUN mode off)
  toggle [--tun] [--json]
                     connect or disconnect; with --tun a connection without
                     TUN gets TUN turned on
  logs [-n 200] [-f] show the core's log
  die                stop the core
  version            print the version
`)
}

// Run executes a command and returns the exit code.
func Run(name string, args []string) int {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	as_json := flags.Bool("json", false, "")
	tun := flags.Bool("tun", false, "")
	profile_arg := flags.String("profile", "", "")
	timeout := flags.Duration("timeout", 15*time.Second, "")
	watch := flags.Bool("watch", false, "")
	interval := flags.Duration("interval", 2*time.Second, "")
	lines := flags.Int("n", 200, "")
	follow := flags.Bool("f", false, "")
	if err := flags.Parse(args); err != nil || flags.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "happ-tui %s: invalid arguments\n\n", name)
		Usage(os.Stderr)
		return exitUsage
	}

	var profile *sharedtypes.ProfileID
	if *profile_arg != "" {
		id, err := parseProfileID(*profile_arg)
		if err != nil {
			fmt.Fprintln(os.Stderr, "happ-tui:", err)
			return exitUsage
		}
		profile = &id
	}

	out := output{json: *as_json}
	switch name {
	case "status":
		if *watch {
			return watchStatus(*interval)
		}
		return status(out)
	case "up":
		return change(out, *timeout, true, "up", sharedtypes.UpData{Profile: profile, Tun: *tun})
	case "down":
		return change(out, *timeout, false, "down", struct{}{})
	case "toggle":
		return change(out, *timeout, true, "toggle", sharedtypes.ToggleData{Profile: profile, Tun: *tun})
	case "logs":
		return logs(*lines, *follow)
	}
	Usage(os.Stderr)
	return exitUsage
}

func parseProfileID(value string) (sharedtypes.ProfileID, error) {
	group, id, ok := strings.Cut(value, ":")
	group_id, err1 := strconv.Atoi(group)
	profile_id, err2 := strconv.Atoi(id)
	if !ok || err1 != nil || err2 != nil {
		return sharedtypes.ProfileID{}, fmt.Errorf("--profile must be GROUP:ID, e.g. 1:12")
	}
	return sharedtypes.ProfileID{GroupId: group_id, Id: profile_id}, nil
}

type output struct {
	json bool
}

// report prints a status report, as JSON or for people.
func (o output) report(report sharedtypes.StatusReport) {
	if o.json {
		printJSON(report)
		return
	}
	printHuman(os.Stdout, report)
}

// fail prints err to stderr and, with --json, as {"error": ...} to stdout.
func (o output) fail(err error) int {
	server_err := &connection.ServerError{}
	if !errors.As(err, &server_err) {
		server_err = &connection.ServerError{Key: "error", Message: err.Error()}
		if errors.Is(err, connection.ErrNoCore) {
			server_err.Key = "core-stopped"
		}
	}
	fmt.Fprintln(os.Stderr, "happ-tui:", server_err.Message)
	if o.json {
		printJSON(map[string]any{"error": server_err})
	}
	return exitError
}

func printJSON(value any) {
	data, _ := json.Marshal(value)
	fmt.Println(string(data))
}

// marshalling a stopped report would print every field as null, while
// scripts expect just {"core":"stopped"}
func (o output) stopped() {
	if o.json {
		fmt.Println(`{"core":"stopped"}`)
		return
	}
	fmt.Println("core        stopped")
}

func status(out output) int {
	c, err := connection.Dial()
	if err != nil {
		out.stopped()
		return exitOK
	}
	defer c.Close()
	c.Start(nil)

	var report sharedtypes.StatusReport
	if err := c.Request(5*time.Second, "get-status", struct{}{}, &report); err != nil {
		return out.fail(err)
	}
	out.report(report)
	return exitOK
}

// change sends up, down or toggle and prints the resulting status.
func change(out output, timeout time.Duration, spawn bool, msg string, data any) int {
	var c *connection.Client
	var err error
	if spawn {
		c, err = connection.ConnectOrSpawn(utils.SpawnCore)
	} else {
		c, err = connection.Dial()
		if err != nil {
			// nothing to disconnect
			out.stopped()
			return exitOK
		}
	}
	if err != nil {
		return out.fail(err)
	}
	defer c.Close()
	c.Start(nil)

	var report sharedtypes.StatusReport
	if err := c.Request(timeout, msg, data, &report); err != nil {
		return out.fail(err)
	}
	out.report(report)
	return exitOK
}

// watchStatus prints a JSON line on every change, and every interval while
// connected. It never ends on its own: when the core goes away it prints
// {"core":"stopped"} and waits for it to come back.
func watchStatus(interval time.Duration) int {
	out := output{json: true}
	last_was_stopped := false
	for {
		c, err := connection.Dial()
		if err != nil {
			if !last_was_stopped {
				out.stopped()
				last_was_stopped = true
			}
			time.Sleep(time.Second)
			continue
		}
		last_was_stopped = false

		reports := make(chan sharedtypes.StatusReport, 16)
		c.Start(func(msg sharedtypes.TcpMessage) {
			if msg.Msg != "status" {
				return
			}
			var report sharedtypes.StatusReport
			if json.Unmarshal(msg.Data, &report) == nil {
				reports <- report
			}
		})

		var report sharedtypes.StatusReport
		if err := c.Request(5*time.Second, "get-status", struct{}{}, &report); err == nil {
			out.report(report)
		}

		ticker := time.NewTicker(interval)
	loop:
		for {
			select {
			case report = <-reports:
				out.report(report)
			case <-ticker.C:
				if report.Connection != "connected" {
					continue
				}
				if err := c.Request(5*time.Second, "get-status", struct{}{}, &report); err == nil {
					out.report(report)
				}
			case <-c.Done():
				break loop
			}
		}
		ticker.Stop()
		c.Close()
	}
}

// logs prints the core's log from its in-memory buffer, which works for a
// system service without access to journald.
func logs(lines int, follow bool) int {
	c, err := connection.Dial()
	if err != nil {
		return output{}.fail(err)
	}
	defer c.Close()
	c.Start(func(msg sharedtypes.TcpMessage) {
		var line string
		if msg.Msg == "log" && json.Unmarshal(msg.Data, &line) == nil {
			fmt.Println(line)
		}
	})

	var reply struct {
		Lines []string `json:"lines"`
	}
	if err := c.Request(5*time.Second, "logs", map[string]any{"n": lines, "follow": follow}, &reply); err != nil {
		return output{}.fail(err)
	}
	for _, line := range reply.Lines {
		fmt.Println(line)
	}
	if follow {
		<-c.Done()
	}
	return exitOK
}
