package TCPServer

import (
	cmd "github.com/AnotherProksY/Happ-TUI/core/commands"
	"github.com/AnotherProksY/Happ-TUI/core/lib"
	"log"
)

func (s *Server) handleStatusChange() {
	log.Println("listening to connection state change")
	for status := range s.proxy_manager.StatusChanged {
		s.BroadCast(lib.CreateJsonNotification("status-changed", status))
		// the full report, for `happ-tui status --watch`
		s.BroadCast(lib.CreateJsonNotification("status", cmd.BuildStatus(s.DB, s.proxy_manager, status)))
	}
}
