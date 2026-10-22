package cmd

import (
	"github.com/AnotherProksY/Happ-TUI/core/lib"
	"github.com/AnotherProksY/Happ-TUI/core/lib/logbuf"
	"github.com/AnotherProksY/Happ-TUI/core/structs"
)

// Logs sends the newest log lines, and keeps sending new ones with Follow
// until the client goes away.
func (cmd *Cmd) Logs(data structs.LogsData) {
	cmd.reply("logs", structs.LogsReply{Lines: logbuf.Default.Last(data.N)})
	if data.Follow {
		reply := cmd.Reply
		logbuf.Default.Follow(func(line string) bool {
			return reply(lib.CreateJsonNotification("log", line)) == nil
		})
	}
}
