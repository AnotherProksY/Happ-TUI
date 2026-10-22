// Package logbuf keeps the last lines of the core's log in memory, so that
// `happ-tui logs` works without access to journald or the log file.
package logbuf

import (
	"bytes"
	"sync"
)

const capacity = 2000

type Buffer struct {
	mu        sync.Mutex
	lines     []string
	next      int
	full      bool
	partial   []byte
	followers map[int]func(string) bool
	nextID    int
}

var Default = &Buffer{followers: map[int]func(string) bool{}}

// Write takes log output in any chunks and stores it line by line.
func (b *Buffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	b.partial = append(b.partial, p...)
	var complete []string
	for {
		i := bytes.IndexByte(b.partial, '\n')
		if i < 0 {
			break
		}
		line := string(b.partial[:i])
		b.partial = b.partial[i+1:]
		b.add(line)
		complete = append(complete, line)
	}
	followers := make(map[int]func(string) bool, len(b.followers))
	for id, f := range b.followers {
		followers[id] = f
	}
	b.mu.Unlock()

	for id, follow := range followers {
		for _, line := range complete {
			if !follow(line) {
				b.Unfollow(id)
				break
			}
		}
	}
	return len(p), nil
}

func (b *Buffer) add(line string) {
	if len(b.lines) < capacity {
		b.lines = append(b.lines, line)
		return
	}
	b.lines[b.next] = line
	b.next = (b.next + 1) % capacity
	b.full = true
}

// Last returns up to n of the newest lines, oldest first.
func (b *Buffer) Last(n int) []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	ordered := b.lines
	if b.full {
		ordered = append(append([]string{}, b.lines[b.next:]...), b.lines[:b.next]...)
	}
	if n <= 0 || n > len(ordered) {
		n = len(ordered)
	}
	return append([]string{}, ordered[len(ordered)-n:]...)
}

// Follow calls send for every new line until it returns false.
func (b *Buffer) Follow(send func(string) bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.nextID++
	b.followers[b.nextID] = send
}

func (b *Buffer) Unfollow(id int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.followers, id)
}
