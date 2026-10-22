package utils

import "net"

// FreeLocalPort asks the kernel for a free TCP port on 127.0.0.1.
func FreeLocalPort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}
