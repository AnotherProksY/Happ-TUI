package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/AnotherProksY/Happ-TUI/core/db"
	"github.com/AnotherProksY/Happ-TUI/core/lib"
	"github.com/AnotherProksY/Happ-TUI/core/lib/TCPServer"
	"github.com/AnotherProksY/Happ-TUI/core/lib/config"
	proxy "github.com/AnotherProksY/Happ-TUI/core/lib/proxy/mainproxy"
	"github.com/AnotherProksY/Happ-TUI/core/structs"
	"github.com/AnotherProksY/Happ-TUI/core/utils"

	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

func main() {
	log_file := utils.LogFile("core.log")
	log.Println("logs:", log_file)
	log.SetOutput(&lumberjack.Logger{
		Filename:   log_file,
		MaxSize:    20,
		MaxBackups: 1,
		MaxAge:     0,
		Compress:   false,
	})

	log.SetPrefix("debug: ")
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	stop_sig := make(chan bool, 1)

	if err := utils.MigrateLegacyDirs(); err != nil {
		log.Println("failed to migrate bushuray directories:", err)
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
