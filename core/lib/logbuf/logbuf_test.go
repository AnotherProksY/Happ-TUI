package logbuf

import (
	"fmt"
	"testing"
)

func TestBuffer(t *testing.T) {
	b := &Buffer{followers: map[int]func(string) bool{}}
	var followed []string
	b.Follow(func(line string) bool {
		followed = append(followed, line)
		return len(followed) < 2
	})
	b.Write([]byte("one\ntw"))
	b.Write([]byte("o\nthree\nfour\n"))
	if got := b.Last(2); len(got) != 2 || got[0] != "three" || got[1] != "four" {
		t.Errorf("Last(2) = %v", got)
	}
	if len(followed) != 2 || followed[1] != "two" {
		t.Errorf("follower got %v and must stop after refusing", followed)
	}
	for i := 0; i < capacity+5; i++ {
		fmt.Fprintf(b, "line %d\n", i)
	}
	all := b.Last(0)
	if len(all) != capacity || all[len(all)-1] != fmt.Sprintf("line %d", capacity+4) || all[0] != "line 5" {
		t.Errorf("ring buffer order wrong: first %q last %q len %d", all[0], all[len(all)-1], len(all))
	}
}
