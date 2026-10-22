package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/AnotherProksY/Happ-TUI/core/db"
	"github.com/AnotherProksY/Happ-TUI/core/lib"
	"github.com/AnotherProksY/Happ-TUI/core/lib/TCPServer"
	"github.com/AnotherProksY/Happ-TUI/core/lib/config"
	"github.com/AnotherProksY/Happ-TUI/core/lib/logbuf"
	proxy "github.com/AnotherProksY/Happ-TUI/core/lib/proxy/mainproxy"
	"github.com/AnotherProksY/Happ-TUI/core/structs"
	"github.com/AnotherProksY/Happ-TUI/core/utils"

	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

func usage() {
	fmt.Fprint(os.Stderr, `usage:
  happ-tui-core                      run for the current user (started by happ-tui)
  happ-tui-core --service --user NAME
                                     run as a root system service for NAME
  happ-tui-core --cleanup            remove TUN rules left by a stopped core
  happ-tui-core migrate --user NAME  copy NAME's config and groups to the
                                     system service directories, once
`)
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		os.Exit(migrate(os.Args[2:]))
	}

	flags := flag.NewFlagSet("happ-tui-core", flag.ContinueOnError)
	flags.Usage = usage
	service := flags.Bool("service", false, "")
	service_user := flags.String("user", "", "")
	cleanup := flags.Bool("cleanup", false, "")
	if err := flags.Parse(os.Args[1:]); err != nil || flags.NArg() > 0 {
		usage()
		os.Exit(2)
	}

	if *cleanup {
		if err := proxy.CleanupTun(); err != nil {
			fmt.Fprintln(os.Stderr, "cleanup:", err)
			os.Exit(1)
		}
		return
	}

	if *service {
		if !utils.IsRoot() || *service_user == "" {
			fmt.Fprintln(os.Stderr, "--service must run as root and needs --user NAME")
			os.Exit(2)
		}
		if err := utils.SetServiceUser(*service_user); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		// journald keeps the log; the ring buffer serves `happ-tui logs`
		log.SetOutput(io.MultiWriter(os.Stderr, logbuf.Default))
	} else {
		log_file := utils.LogFile("core.log")
		log.Println("logs:", log_file)
		log.SetOutput(io.MultiWriter(&lumberjack.Logger{
			Filename:   log_file,
			MaxSize:    20,
			MaxBackups: 1,
			MaxAge:     0,
			Compress:   false,
		}, logbuf.Default))
	}

	log.SetPrefix("debug: ")
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	stop_sig := make(chan bool, 1)

	if !*service {
		if err := utils.MigrateLegacyDirs(); err != nil {
			log.Println("failed to migrate bushuray directories:", err)
		}
	}
	appConfig, err := config.LoadAppConfig()
	if err != nil {
		log.Println("failed to load application config:", err, "using defaults")
	}

	database := db.DB{}
	database.Initialize()
	proxy_manager := proxy.ProxyManager{}
	proxy_manager.Init(appConfig)

	server := TCPServer.NewServer(&database, &proxy_manager, stop_sig, appConfig)
	server.Start()

	connectOnStartup(&database, &proxy_manager, appConfig.AutoConnectOnStart)

	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		reason := ""
		select {
		case sig := <-sigs:
			reason = fmt.Sprintf("Received signal %v , cleaning up...", sig)
		case <-stop_sig:
			reason = "Received stop request , cleaning up..."
		}
		log.Println(reason)
		if err := proxy_manager.Stop(); err != nil {
			log.Println("failed to clean up proxy state:", err)
		}
		server.BroadCast(lib.CreateJsonNotification("warn", structs.Warning{Key: "died", Content: reason}))
		os.Exit(0)
	}()
	select {}
}

func migrate(args []string) int {
	flags := flag.NewFlagSet("migrate", flag.ContinueOnError)
	flags.Usage = usage
	name := flags.String("user", "", "")
	if err := flags.Parse(args); err != nil || *name == "" {
		usage()
		return 2
	}
	if !utils.IsRoot() {
		fmt.Fprintln(os.Stderr, "migrate must run as root")
		return 2
	}
	if err := utils.MigrateUserData(*name); err != nil {
		fmt.Fprintln(os.Stderr, "migrate:", err)
		return 1
	}
	return 0
}

func connectOnStartup(database *db.DB, proxy_manager *proxy.ProxyManager, autoConnect bool) {
	if autoConnect {
		profile, err := database.GetLatestConnectedProfile()
		if err != nil {
			return
		}
		err = proxy_manager.Connect(profile, false)
		if err != nil {
			log.Fatal("failed to connect to profile on startup", err)
		}
	}
}
