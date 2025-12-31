package csafe

import "testing"

func TestLRC(t *testing.T) {
	// For payload [1,2,3] and length 3, lrc should make sum == 0
	ln := byte(3)
	chk := lrc(ln, []byte{1, 2, 3})
	total := uint16(ln) + uint16(sum8([]byte{1, 2, 3})) + uint16(chk)
	if byte(total&0xFF) != 0 {
		t.Fatalf("lrc incorrect")
	}
}

func TestBuildFrame(t *testing.T) {
	p := []byte{0x91}
	f := Build(p)
	if len(f) != len(p)+4 {
		t.Fatalf("unexpected frame length: %d", len(f))
	}
	if f[0] != StartByte || f[len(f)-1] != StopByte {
		t.Fatalf("start/stop bytes incorrect")
	}
}

func TestParseRoundTrip(t *testing.T) {
	payload := []byte{0x91, 0x01, 0x02}
	f := Build(payload)
	out, err := Parse(f)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if string(out) != string(payload) {
		t.Fatalf("payload mismatch: %v != %v", out, payload)
	}
}

func TestParseBadChecksum(t *testing.T) {
	f := Build([]byte{0x91})
	f[len(f)-2] ^= 0xFF // corrupt checksum
	if _, err := Parse(f); err == nil {
		t.Fatalf("expected checksum error")
	}
}
