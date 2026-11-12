package main

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeTone writes a sine tone as a WAV file. format 1 is integer PCM, 3 is float.
func writeTone(t *testing.T, path string, rate, ch, bits, format int, freq, secs float64) {
	t.Helper()
	n := int(float64(rate) * secs)
	bps := bits / 8
	data := make([]byte, n*ch*bps)
	for i := 0; i < n; i++ {
		v := 0.5 * math.Sin(2*math.Pi*freq*float64(i)/float64(rate))
		for c := 0; c < ch; c++ {
			b := data[(i*ch+c)*bps:]
			switch {
			case format == 3 && bits == 32:
				binary.LittleEndian.PutUint32(b, math.Float32bits(float32(v)))
			case format == 3:
				binary.LittleEndian.PutUint64(b, math.Float64bits(v))
			case bits == 8:
				b[0] = byte(int(v*127) + 128)
			case bits == 16:
				binary.LittleEndian.PutUint16(b, uint16(int16(v*32767)))
			case bits == 24:
				x := int32(v * 8388607)
				b[0], b[1], b[2] = byte(x), byte(x>>8), byte(x>>16)
			default:
				binary.LittleEndian.PutUint32(b, uint32(int32(v*2147483647)))
			}
		}
	}
	h := make([]byte, 44)
	copy(h, "RIFF")
	binary.LittleEndian.PutUint32(h[4:], uint32(36+len(data)))
	copy(h[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(h[16:], 16)
	binary.LittleEndian.PutUint16(h[20:], uint16(format))
	binary.LittleEndian.PutUint16(h[22:], uint16(ch))
	binary.LittleEndian.PutUint32(h[24:], uint32(rate))
	binary.LittleEndian.PutUint32(h[28:], uint32(rate*ch*bps))
	binary.LittleEndian.PutUint16(h[32:], uint16(ch*bps))
	binary.LittleEndian.PutUint16(h[34:], uint16(bits))
	copy(h[36:], "data")
	binary.LittleEndian.PutUint32(h[40:], uint32(len(data)))
	if err := os.WriteFile(path, append(h, data...), 0o644); err != nil {
		t.Fatal(err)
	}
}

// decodeFile runs a file through the decoder and resampler, as the player does.
func decodeFile(t *testing.T, path string) []float32 {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	src, rate, err := openTrack(f, filepath.Ext(path))
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	rs := &resampler{src: src, step: float64(rate) / sampleRate, buf: make([]float32, 4096*2)}
	var out []float32
	for {
		chunk := make([]float32, musicChunkFrames*2)
		n := rs.read(chunk)
		if n == 0 {
			return out
		}
		out = append(out, chunk[:n*2]...)
	}
}

// pitch estimates a tone's frequency from the left channel's zero crossings.
func pitch(pcm []float32) float64 {
	cross := 0
	for i := 2; i < len(pcm); i += 2 {
		if (pcm[i-2] < 0) != (pcm[i] < 0) {
			cross++
		}
	}
	return float64(cross) / 2 / (float64(len(pcm)/2) / sampleRate)
}

func TestWAVDecodeAndResample(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name                   string
		rate, ch, bits, format int
		freq                   float64
	}{
		{"s16_44k_stereo", 44100, 2, 16, 1, 440},
		{"s24_48k_mono", 48000, 1, 24, 1, 330},
		{"u8_22k", 22050, 1, 8, 1, 220},
		{"s32_96k", 96000, 2, 32, 1, 1000},
		{"f32_32k", 32000, 2, 32, 3, 550},
		{"f64_44k", 44100, 1, 64, 3, 660},
	}
	for _, c := range cases {
		p := filepath.Join(dir, c.name+".wav")
		writeTone(t, p, c.rate, c.ch, c.bits, c.format, c.freq, 1.5)
		pcm := decodeFile(t, p)
		frames := len(pcm) / 2
		if want := int(1.5 * sampleRate); frames < want-3 || frames > want+3 {
			t.Errorf("%s: %d frames at %d Hz, want about %d", c.name, frames, sampleRate, want)
		}
		if f := pitch(pcm); math.Abs(f-c.freq) > c.freq*0.01 {
			t.Errorf("%s: pitch %.1f Hz, want %.0f", c.name, f, c.freq)
		}
		var peak float32
		for _, v := range pcm {
			peak = max(peak, v, -v)
		}
		if peak < 0.45 || peak > 0.55 {
			t.Errorf("%s: peak %.3f, want about 0.5", c.name, peak)
		}
	}
}

func TestTrackTitle(t *testing.T) {
	for in, want := range map[string]string{
		"/m/my_song (live).mp3":  "MY SONG LIVE",
		"/m/01 - Intro.ogg":      "01 - INTRO",
		"/m/Ünïcode__name!!.wav": "N CODE NAME!!",
		"/m/x.mp3":               "X",
	} {
		if got := trackTitle(in); got != want {
			t.Errorf("trackTitle(%q) = %q, want %q", in, got, want)
		}
	}
}

// pull reads frames from the player the way the mixer does.
func pull(t *testing.T, m *MusicPlayer, frames int) {
	t.Helper()
	for i := 0; i < frames; i++ {
		if _, _, ok := m.next(); !ok {
			t.Fatalf("music stalled after %d frames", i)
		}
	}
}

func TestPlayerFolder(t *testing.T) {
	empty := NewMusicPlayer(t.TempDir())
	if empty.Active() {
		t.Fatal("an empty folder should leave the built-in soundtrack on")
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "broken.mp3"), []byte("not audio"), 0o644); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("ignored"), 0o644)
	writeTone(t, filepath.Join(dir, "first.wav"), 48000, 2, 16, 1, 440, 3) // longer than the decode-ahead buffer
	m := NewMusicPlayer(dir)
	m.Block = true
	if !m.Active() {
		t.Fatal("folder with a song should be active")
	}
	pull(t, m, sampleRate/2)
	if m.Title != "FIRST" {
		t.Fatalf("playing %q, want FIRST (broken.mp3 must be skipped)", m.Title)
	}

	// a song copied in while the game runs plays next
	writeTone(t, filepath.Join(dir, "added.wav"), 44100, 1, 16, 1, 330, 1)
	deadline := time.Now().Add(10 * time.Second)
	for m.Title != "ADDED" {
		if time.Now().After(deadline) {
			t.Fatalf("new song never played; still on %q", m.Title)
		}
		pull(t, m, 4096)
	}

	// skipping moves straight on
	seq := m.TitleSeq
	m.Skip()
	pull(t, m, 1)
	if m.TitleSeq != seq+1 {
		t.Fatalf("skip did not start another track (seq %d -> %d)", seq, m.TitleSeq)
	}
}
