package prefetch

import "testing"

func TestOpenEmpty(t *testing.T) {
	_, err := Open(nil)
	if err == nil {
		t.Fatal("expected error for nil input")
	}
}

func TestOpenTooShort(t *testing.T) {
	_, err := Open([]byte{0x00, 0x00, 0x00})
	if err == nil {
		t.Fatal("expected error for short input")
	}
}
