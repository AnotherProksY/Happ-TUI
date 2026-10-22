package connection

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	sharedtypes "github.com/AnotherProksY/Happ-TUI/shared_types"
)

// SystemSocketPath is where a root core (system service or started with
// sudo) listens.
const SystemSocketPath = "/run/happ-tui/core.sock"

// ErrNoCore means no core is listening on any of the sockets.
var ErrNoCore = errors.New("core is not running")

// ErrClosed means the connection to the core was lost.
var ErrClosed = errors.New("connection to the core was closed")

// ServerError is an "error" reply of the core. Key is stable for scripts.
type ServerError struct {
	Key     string `json:"key"`
	Message string `json:"message"`
}

func (e *ServerError) Error() string {
	return e.Message
}

// Client talks to the core over its control socket. Replies are matched to
// requests by request_id; everything else is a notification.
type Client struct {
	conn    net.Conn
	writeMu sync.Mutex
	mu      sync.Mutex
	pending map[string]chan sharedtypes.TcpMessage
	nextID  atomic.Uint64
	done    chan struct{}
	err     error
}

// SocketPaths are the sockets a core may listen on, in lookup order: a root
// core first, then the core of the invoking user.
func SocketPaths() []string {
	uid := os.Getuid()
	if uid == 0 {
		if sudo_uid, err := strconv.Atoi(os.Getenv("SUDO_UID")); err == nil {
			uid = sudo_uid
		}
	}
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" || uid != os.Getuid() {
		dir = filepath.Join("/run/user", strconv.Itoa(uid))
	}
	return []string{SystemSocketPath, filepath.Join(dir, "happ-tui", "core.sock")}
}

// Dial connects to the first core that answers.
func Dial() (*Client, error) {
	for _, path := range SocketPaths() {
		conn, err := net.DialTimeout("unix", path, 2*time.Second)
		if err != nil {
			continue
		}
		return &Client{
			conn:    conn,
			pending: make(map[string]chan sharedtypes.TcpMessage),
			done:    make(chan struct{}),
		}, nil
	}
	return nil, ErrNoCore
}

// Start reads from the core until the connection closes, passing
// notifications to notify. It must be called before Request.
func (c *Client) Start(notify func(sharedtypes.TcpMessage)) {
	go func() {
		err := c.readLoop(notify)
		c.mu.Lock()
		c.err = err
		for _, ch := range c.pending {
			close(ch)
		}
		c.pending = nil
		c.mu.Unlock()
		close(c.done)
	}()
}

func (c *Client) readLoop(notify func(sharedtypes.TcpMessage)) error {
	reader := bufio.NewReader(c.conn)
	for {
		lengthBuf := make([]byte, 4)
		if _, err := io.ReadFull(reader, lengthBuf); err != nil {
			return ErrClosed
		}
		length := binary.BigEndian.Uint32(lengthBuf)
		if length == 0 || length > 100*1024*1024 {
			return fmt.Errorf("invalid message length %d", length)
		}
		payload := make([]byte, length)
		if _, err := io.ReadFull(reader, payload); err != nil {
			return ErrClosed
		}

		var msg sharedtypes.TcpMessage
		if err := json.Unmarshal(payload, &msg); err != nil {
			log.Printf("Invalid JSON: %v %v", string(payload), err)
			continue
		}

		if msg.RequestId == "" {
			// the core rejects a connection with an error that answers nothing
			if msg.Msg == "error" {
				return decodeServerError(msg)
			}
			if notify != nil {
				notify(msg)
			}
			continue
		}

		c.mu.Lock()
		ch, ok := c.pending[msg.RequestId]
		delete(c.pending, msg.RequestId)
		c.mu.Unlock()
		if ok {
			ch <- msg
		}
	}
}

// Done is closed when the connection to the core is lost.
func (c *Client) Done() <-chan struct{} {
	return c.done
}

// Err is why the connection was lost, valid after Done is closed.
func (c *Client) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}

// Send writes a command without waiting for an answer.
func (c *Client) Send(msg []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	packet := make([]byte, 4+len(msg))
	binary.BigEndian.PutUint32(packet[:4], uint32(len(msg)))
	copy(packet[4:], msg)

	if err := writeFull(c.conn, packet); err != nil {
		return fmt.Errorf("error sending %s: %w", msg, err)
	}
	return nil
}

// Request sends a command and waits for the core's answer, decoding its data
// into out (which may be nil). An "error" answer becomes a *ServerError.
func (c *Client) Request(timeout time.Duration, msg string, data any, out any) error {
	id := strconv.FormatUint(c.nextID.Add(1), 10)
	ch := make(chan sharedtypes.TcpMessage, 1)

	c.mu.Lock()
	if c.pending == nil {
		c.mu.Unlock()
		return c.closedErr()
	}
	c.pending[id] = ch
	c.mu.Unlock()

	payload, err := json.Marshal(sharedtypes.Message[any]{Msg: msg, RequestId: id, Data: data})
	if err != nil {
		return err
	}
	if err := c.Send(payload); err != nil {
		return err
	}

	select {
	case reply, ok := <-ch:
		if !ok {
			return c.closedErr()
		}
		if reply.Msg == "error" {
			return decodeServerError(reply)
		}
		if out != nil {
			return json.Unmarshal(reply.Data, out)
		}
		return nil
	case <-time.After(timeout):
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return &ServerError{Key: "timeout", Message: fmt.Sprintf("no answer from the core within %s", timeout)}
	}
}

func (c *Client) closedErr() error {
	<-c.done
	if err := c.Err(); err != nil {
		return err
	}
	return ErrClosed
}

func decodeServerError(msg sharedtypes.TcpMessage) error {
	server_err := &ServerError{}
	if err := json.Unmarshal(msg.Data, server_err); err != nil || server_err.Message == "" {
		return &ServerError{Key: "error", Message: string(msg.Data)}
	}
	return server_err
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

func (c *Client) Close() error {
	return c.conn.Close()
}

// ConnectOrSpawn connects to a running core, or starts one with spawn and
// waits for it to listen.
func ConnectOrSpawn(spawn func() error) (*Client, error) {
	c, err := Dial()
	if err == nil {
		return c, nil
	}
	if err := spawn(); err != nil {
		return nil, fmt.Errorf("failed to start the core: %w", err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
		if c, err := Dial(); err == nil {
			return c, nil
		}
	}
	return nil, fmt.Errorf("the core was started but doesn't answer; see ~/.local/state/happ-tui/core.log")
}
