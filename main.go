package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	mainmodel "github.com/AnotherProksY/Happ-TUI/components/MainModel"
	"github.com/AnotherProksY/Happ-TUI/global"
	appconfig "github.com/AnotherProksY/Happ-TUI/lib/AppConfig"
	connection "github.com/AnotherProksY/Happ-TUI/lib/Connection"
	notif_publisher "github.com/AnotherProksY/Happ-TUI/lib/NotifPublisher"
	servercmds "github.com/AnotherProksY/Happ-TUI/lib/ServerCommands"
	sharedtypes "github.com/AnotherProksY/Happ-TUI/shared_types"
	"github.com/AnotherProksY/Happ-TUI/utils"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"

	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

func main() {
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

	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println("happ-tui", global.Version)
		return
	}
	if err := utils.MigrateLegacyDirs(); err != nil {
		log.Println("failed to migrate bushuray directories:", err)
	}
	appconfig.LoadConfig()
	port := appconfig.GetConfig().CoreTCPPort

	if len(os.Args) > 1 && os.Args[1] == "die" {
		die(port)
		return
	}

	C := connection.ConnectionHandler{}
	C.Init("127.0.0.1", port)

	err := C.GetConnection()
	if err != nil {
		fmt.Println("core was not found at", port, "trying to spawn")
		err := utils.SpawnCore()
		if err != nil {
			fmt.Println("failed to spawn core:", err)
			return
		}
		time.Sleep(1000 * time.Millisecond)
		err = C.GetConnection()
		if err != nil {
			fmt.Println("failed to connect to core:", err)
			return
		}
	} else {
		fmt.Println("connection established")
	}

	zone.NewGlobal()
	p := tea.NewProgram(mainmodel.InitModel(), tea.WithAltScreen(), tea.WithMouseCellMotion())

	go C.HandleConnection(p)
	servercmds.Init(&C)
	servercmds.GetApplicationState()
	notif_publisher.Init(p)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error %v\n", err)
		os.Exit(1)
	}
}

func die(port int) {
	C := connection.ConnectionHandler{}
	C.Init("127.0.0.1", port)

	if err := C.GetConnection(); err != nil {
		fmt.Println("Happ-TUI is not running")
		return
	}
	defer C.Close()

	if err := C.Send(servercmds.CreateJsonCommand("die", sharedtypes.DieData{})); err != nil {
		fmt.Println("failed to stop Happ-TUI:", err)
		return
	}

	if err := C.WaitForClose(); err != nil {
		fmt.Println("failed while waiting for Happ-TUI to stop:", err)
		return
	}

	fmt.Println("Happ-TUI has stopped successfully")
}
