package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sync/atomic"
	"time"

	"github.com/AnotherProksY/Happ-TUI/cli"
	mainmodel "github.com/AnotherProksY/Happ-TUI/components/MainModel"
	"github.com/AnotherProksY/Happ-TUI/global"
	appconfig "github.com/AnotherProksY/Happ-TUI/lib/AppConfig"
	connection "github.com/AnotherProksY/Happ-TUI/lib/Connection"
	notif_publisher "github.com/AnotherProksY/Happ-TUI/lib/NotifPublisher"
	servercmds "github.com/AnotherProksY/Happ-TUI/lib/ServerCommands"
	servernotifs "github.com/AnotherProksY/Happ-TUI/lib/ServerNotifs"
	sharedtypes "github.com/AnotherProksY/Happ-TUI/shared_types"
	"github.com/AnotherProksY/Happ-TUI/utils"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"

	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version":
			fmt.Println("happ-tui", global.Version)
			return
		case "die":
			os.Exit(die())
		case "status", "up", "down", "toggle", "logs":
			os.Exit(cli.Run(os.Args[1], os.Args[2:]))
		case "help", "-h", "--help":
			cli.Usage(os.Stdout)
			return
		default:
			fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
			cli.Usage(os.Stderr)
			os.Exit(2)
		}
	}
	runTUI()
}

func runTUI() {
	// keep early messages (home directory lookup) off the terminal
	log.SetOutput(io.Discard)
	log.SetOutput(&lumberjack.Logger{
		Filename:   utils.LogFile("happ-tui.log"),
		MaxSize:    20,
		MaxBackups: 1,
		MaxAge:     0,
		Compress:   false,
	})
	log.SetPrefix("debug: ")
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)

	C, err := connection.ConnectOrSpawn(utils.SpawnCore)
	if err != nil {
		fmt.Fprintln(os.Stderr, "happ-tui:", err)
		os.Exit(1)
	}

	// notifications are dropped until the UI exists; it asks for the full
	// state right after starting
	var program atomic.Pointer[tea.Program]
	C.Start(func(msg sharedtypes.TcpMessage) {
		if program.Load() != nil {
			servernotifs.HandleNotification(msg)
		}
	})

	var config appconfig.AppConfig
	if err := C.Request(5*time.Second, "get-config", struct{}{}, &config); err != nil {
		fmt.Fprintln(os.Stderr, "happ-tui: failed to get settings from the core:", err)
		os.Exit(1)
	}
	appconfig.SetConfig(config)

	zone.NewGlobal()
	p := tea.NewProgram(mainmodel.InitModel(), tea.WithAltScreen(), tea.WithMouseCellMotion())
	notif_publisher.Init(p)
	program.Store(p)
	go func() {
		<-C.Done()
		p.Quit()
	}()

	servercmds.Init(C)
	servercmds.GetApplicationState()

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error %v\n", err)
		os.Exit(1)
	}
}

func die() int {
	C, err := connection.Dial()
	if err != nil {
		fmt.Println("Happ-TUI is not running")
		return 0
	}
	defer C.Close()
	C.Start(nil)

	// the core exits without answering, so a closed connection means success
	err = C.Request(10*time.Second, "die", sharedtypes.DieData{}, nil)
	if err != nil && !errors.Is(err, connection.ErrClosed) {
		fmt.Fprintln(os.Stderr, "failed to stop Happ-TUI:", err)
		return 1
	}

	fmt.Println("Happ-TUI has stopped successfully")
	return 0
}
