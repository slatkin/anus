package miniflux

import "testing"

func TestReadStatusToggle(t *testing.T) {
	if ReadStatusRead.Toggle() != ReadStatusUnread {
		t.Error("read.Toggle() should return unread")
	}
	if ReadStatusUnread.Toggle() != ReadStatusRead {
		t.Error("unread.Toggle() should return read")
	}
}

func TestReadStatusToggleIdempotent(t *testing.T) {
	s := ReadStatusUnread
	if s.Toggle().Toggle() != s {
		t.Error("double toggle should return original status")
	}
}
