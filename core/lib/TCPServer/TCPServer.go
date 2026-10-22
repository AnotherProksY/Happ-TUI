package TCPServer

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	cmd "github.com/AnotherProksY/Happ-TUI/core/commands"
	"github.com/AnotherProksY/Happ-TUI/core/db"
	"github.com/AnotherProksY/Happ-TUI/core/lib"
	"github.com/AnotherProksY/Happ-TUI/core/lib/config"
	proxy "github.com/AnotherProksY/Happ-TUI/core/lib/proxy/mainproxy"
	"github.com/AnotherProksY/Happ-TUI/core/structs"
	"github.com/AnotherProksY/Happ-TUI/core/utils"
	"io"
	"log"
	"net"
	"sync"
)

type Server struct {
	clients              map[*client]struct{}
	DB                   *db.DB
	mutex                sync.Mutex
	proxy_manager        *proxy.ProxyManager
	stop_sig             chan<- bool
	removeFailedProfiles bool
	appConfig            config.AppConfig
}

// client is one connection to the control socket. Broadcasts and direct
// replies both go through write, so frames never interleave.
type client struct {
	conn    *net.UnixConn
	uid     int
	writeMu sync.Mutex
}

func (c *client) write(msg []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	packet := make([]byte, 4+len(msg))
	binary.BigEndian.PutUint32(packet[:4], uint32(len(msg)))
	copy(packet[4:], msg)
	return writeFull(c.conn, packet)
}

func NewServer(database *db.DB, proxy_manager *proxy.ProxyManager, stop_sig chan<- bool, appConfig config.AppConfig) *Server {
	return &Server{
		DB:                   database,
		clients:              make(map[*client]struct{}),
		proxy_manager:        proxy_manager,
		stop_sig:             stop_sig,
		removeFailedProfiles: appConfig.RemoveFailedProfiles,
		appConfig:            appConfig,
	}
}

func (s *Server) Start() {
	socket_path := utils.SocketPath()
	listen, err := utils.ListenSocket(socket_path)
	if err != nil {
		log.Fatal("failed to listen on ", socket_path, ": ", err)
	}

	log.Println("listening on", socket_path, "for uid", utils.AllowedUID(), "and root")

	go s.handleStatusChange()
	go s.handleTestResults()
	go s.autoUpdateSubscriptions()

	go func() {
		for {
			conn, err := listen.AcceptUnix()
			if err != nil {
				log.Printf("failed to accept connection: %v", err)
				continue
			}

			uid, err := utils.PeerUID(conn)
			c := &client{conn: conn, uid: uid}
			if err != nil || (uid != 0 && uid != utils.AllowedUID()) {
				log.Println("rejected connection from uid", uid, err)
				c.write(lib.CreateJsonReply("error", "", structs.ErrorReply{Key: "access-denied", Message: "access denied"}))
				conn.Close()
				continue
			}

			s.mutex.Lock()
			s.clients[c] = struct{}{}
			s.mutex.Unlock()

			go s.handleConnection(c)
		}
	}()
}

func writeFull(conn net.Conn, data []byte) error {
	for len(data) > 0 {
		n, err := conn.Write(data)
		if err != nil {
			return err
		}
		data = data[n:]
	}
	return nil
}

func (s *Server) BroadCast(msg []byte) {
	s.mutex.Lock()
	clients := make([]*client, 0, len(s.clients))
	for c := range s.clients {
		clients = append(clients, c)
	}
	s.mutex.Unlock()

	for _, c := range clients {
		if err := c.write(msg); err != nil {
			log.Printf("Error sending to uid %d: %v", c.uid, err)
			c.conn.Close()
			s.mutex.Lock()
			delete(s.clients, c)
			s.mutex.Unlock()
		}
	}
}

func (s *Server) handleConnection(c *client) {
	defer func() {
		c.conn.Close()
		s.mutex.Lock()
		delete(s.clients, c)
		s.mutex.Unlock()
	}()

	conn_handler := cmd.Cmd{DB: s.DB, BroadCast: s.BroadCast, Reply: c.write, PeerUID: c.uid, AppConfig: s.appConfig}
	reader := bufio.NewReader(c.conn)

	for {
		lengthBuf := make([]byte, 4)
		_, err := io.ReadFull(reader, lengthBuf)

		if err != nil {
			if err != io.EOF {
				log.Printf("Failed to read length , %v", err)
			}
			return
		}

		length := binary.BigEndian.Uint32(lengthBuf)
		if length == 0 || length > 100*1024*1024 {
			log.Printf("Invalid length %d", length)
			return
		}

		payload := make([]byte, length)

		_, err = io.ReadFull(reader, payload)

		if err != nil {
			log.Printf("Failed to read the payload %v", err)
			return
		}

		var raw_tcp_message structs.TCPMessage

		if err := json.Unmarshal(payload, &raw_tcp_message); err != nil {
			log.Printf("Invalid JSON: %v", err)
			return
		}

		// a copy per message, so handlers running in goroutines keep their request id
		command_handler := conn_handler
		command_handler.RequestId = raw_tcp_message.RequestId

		switch raw_tcp_message.Msg {
		case "die":
			if utils.IsServiceMode() && c.uid != 0 {
				command_handler.ReplyError("service-mode", "the core runs as a system service, stop it with: sudo systemctl stop happ-tui-core")
				continue
			}
			s.stop_sig <- true
		case "add-profiles":
			var data structs.AddProfilesData
			if err := json.Unmarshal(raw_tcp_message.Data, &data); err != nil {
				log.Printf("Invalid body for add-profiles %v", err)
				return
			}
			command_handler.AddProfiles(data)

		case "delete-profiles":
			var data structs.DeleteProfilesData
			if err := json.Unmarshal(raw_tcp_message.Data, &data); err != nil {
				log.Printf("Invalid body for delete-profiles %v", err)
				return
			}
			command_handler.DeleteProfiles(data)

		case "add-group":
			var data structs.AddGroupData
			if err := json.Unmarshal(raw_tcp_message.Data, &data); err != nil {
				log.Printf("Invalid body for add-group %v", err)
				return
			}
			group_added, err := command_handler.AddGroup(data)
			// fetch right away like Happ does, so the group gets its profiles and title
			if err == nil && group_added.SubscriptionUrl != "" {
				go command_handler.UpdateSubscription(structs.UpdateSubscriptionData{GroupId: group_added.Id}, s.proxy_manager)
			}

		case "delete-group":
			var data structs.DeleteGroupData
			if err := json.Unmarshal(raw_tcp_message.Data, &data); err != nil {
				log.Printf("Invalid body for delete-group %v", err)
				return
			}
			command_handler.DeleteGroup(data)

		case "connect":
			var data structs.ConnectData
			if err := json.Unmarshal(raw_tcp_message.Data, &data); err != nil {
				log.Printf("Invalid body for connect %v", err)
				return
			}
			command_handler.Connect(data, s.proxy_manager)

		case "disconnect":
			var data structs.DisconnectData
			if err := json.Unmarshal(raw_tcp_message.Data, &data); err != nil {
				log.Printf("Invalid body for disconnect %v", err)
				return
			}
			command_handler.Disconnect(data, s.proxy_manager)

		case "test-profile":
			var data structs.TestProfileData
			if err := json.Unmarshal(raw_tcp_message.Data, &data); err != nil {
				log.Printf("Invalid body for test-profile%v", err)
				return
			}
			go command_handler.TestProfile(data, s.proxy_manager)

		case "stop-tests":
			command_handler.StopTests(s.proxy_manager)

		case "get-application-state":
			var data structs.GetApplicationStateData
			if err := json.Unmarshal(raw_tcp_message.Data, &data); err != nil {
				log.Printf("Invalid body for get-application-state%v", err)
				return
			}
			command_handler.GetApplicationState(data, s.proxy_manager)

		case "update-subscription":
			var data structs.UpdateSubscriptionData
			if err := json.Unmarshal(raw_tcp_message.Data, &data); err != nil {
				log.Printf("Invalid body for update-subscription%v", err)
				return
			}
			go command_handler.UpdateSubscription(data, s.proxy_manager)

		case "enable-tun":
			var data structs.EnableTunData
			if err := json.Unmarshal(raw_tcp_message.Data, &data); err != nil {
				log.Printf("Invalid body for enable-tun%v", err)
				return
			}
			command_handler.EnableTun(data, s.proxy_manager)

		case "disable-tun":
			var data structs.DisableTunData
			if err := json.Unmarshal(raw_tcp_message.Data, &data); err != nil {
				log.Printf("Invalid body for disable-tun%v", err)
				return
			}
			command_handler.DisableTun(data, s.proxy_manager)

		case "get-status":
			command_handler.GetStatus(s.proxy_manager)

		case "up":
			var data structs.UpData
			if err := json.Unmarshal(raw_tcp_message.Data, &data); err != nil {
				command_handler.ReplyError("bad-request", "invalid body for up")
				continue
			}
			go command_handler.Up(data, s.proxy_manager)

		case "down":
			go command_handler.Down(s.proxy_manager)

		case "toggle":
			var data structs.ToggleData
			if err := json.Unmarshal(raw_tcp_message.Data, &data); err != nil {
				command_handler.ReplyError("bad-request", "invalid body for toggle")
				continue
			}
			go command_handler.Toggle(data, s.proxy_manager)

		case "logs":
			var data structs.LogsData
			if err := json.Unmarshal(raw_tcp_message.Data, &data); err != nil {
				command_handler.ReplyError("bad-request", "invalid body for logs")
				continue
			}
			command_handler.Logs(data)

		case "get-config":
			command_handler.GetConfig()

		case "is-root":
			var data structs.IsRootData
			if err := json.Unmarshal(raw_tcp_message.Data, &data); err != nil {
				log.Printf("Invalid body for is-root%v", err)
				return
			}
			command_handler.IsRoot(data)

		case "update-profile":
			var data structs.UpdateProfileData
			if err := json.Unmarshal(raw_tcp_message.Data, &data); err != nil {
				log.Printf("Invalid body for update-profile%v", err)
				return
			}
			command_handler.UpdateProfile(data)

		default:
			log.Println("Message not supported")
			if raw_tcp_message.RequestId != "" {
				command_handler.ReplyError("unknown-command", "unknown command "+raw_tcp_message.Msg)
			}
		}
	}
}
