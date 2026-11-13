package main

// Custom soundtrack. Any MP3, OGG or WAV file in the game's "music" folder
// (or a folder inside it) replaces the built-in synth music; with the folder empty the synth plays.
// Tracks are decoded and resampled on a background goroutine and streamed to
// the mixer in small chunks, so a long track costs little memory. The folder
// is rescanned at every track change, and every couple of seconds while it
// holds nothing playable, so songs copied in while the game runs are picked
// up: a newly added song plays next.

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/hajimehoshi/go-mp3"
	"github.com/jfreymuth/oggvorbis"
)

const (
	musicGain        = 0.55 // songs are mastered loud; this sits them under the effects
	musicChunkFrames = 2048
	musicBufChunks   = 24 // decoded ahead: about 1.1 s
)

var musicExts = map[string]bool{".mp3": true, ".ogg": true, ".wav": true}

// musicDir is where the player's songs are: the "music" folder next to the
// game on desktops, the phone's Music folder on Android (see platform_*.go).
func musicDir() string { return defaultMusicDir() }

type musicChunk struct {
	gen   int32
	pcm   []float32 // interleaved stereo at sampleRate
	title string    // set on the first chunk of each track
}

type MusicPlayer struct {
	dir    string
	ch     chan musicChunk
	gen    atomic.Int32 // bumped to skip the current track
	active atomic.Bool  // the folder holds at least one playable track
	bad    map[string]string

	// game-thread side
	cur      []float32
	pos      int
	Title    string // track now playing
	TitleSeq int    // bumps every time a track starts
	Block    bool   // offline rendering: wait for the decoder rather than drop out
}

func NewMusicPlayer(dir string) *MusicPlayer {
	m := &MusicPlayer{dir: dir, ch: make(chan musicChunk, musicBufChunks), bad: map[string]string{}}
	m.active.Store(len(m.scan()) > 0)
	go m.run()
	return m
}

// SetTrack records the song now playing and announces a new one in flight.
func (g *Game) SetTrack(t string) {
	if t != "" && t != g.Track {
		g.Note, g.NoteT, g.NoteBad = "NOW PLAYING  "+t, 3, false
	}
	g.Track = t
}

// Active reports whether custom music is in use (so the synth music stays quiet).
func (m *MusicPlayer) Active() bool { return m != nil && m.active.Load() }

// Skip ends the current track; the next one starts as soon as it is decoded.
func (m *MusicPlayer) Skip() {
	if m == nil {
		return
	}
	m.gen.Add(1)
	m.cur, m.pos = nil, 0
}

// next returns the next stereo frame; ok is false when nothing is decoded yet.
func (m *MusicPlayer) next() (l, r float32, ok bool) {
	for m.pos >= len(m.cur) {
		var c musicChunk
		if m.Block {
			select {
			case c = <-m.ch:
			case <-time.After(3 * time.Second):
				return 0, 0, false
			}
		} else {
			select {
			case c = <-m.ch:
			default:
				return 0, 0, false
			}
		}
		if c.gen != m.gen.Load() {
			continue // left over from a skipped track
		}
		if c.title != "" {
			m.Title, m.TitleSeq = c.title, m.TitleSeq+1
		}
		m.cur, m.pos = c.pcm, 0
	}
	l, r = m.cur[m.pos], m.cur[m.pos+1]
	m.pos += 2
	return l, r, true
}

// scan lists the playable files in the folder and the folders inside it
// (a phone's Music folder is usually sorted into artist and album folders),
// skipping files that failed to decode.
func (m *MusicPlayer) scan() []string {
	var files []string
	var walk func(dir string, depth int)
	walk = func(dir string, depth int) {
		ents, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range ents {
			name := e.Name()
			if strings.HasPrefix(name, ".") {
				continue
			}
			p := filepath.Join(dir, name)
			st, err := os.Stat(p) // follows links
			if err != nil {
				continue
			}
			if st.IsDir() {
				if depth < 3 {
					walk(p, depth+1)
				}
				continue
			}
			if !st.Mode().IsRegular() || !musicExts[strings.ToLower(filepath.Ext(name))] {
				continue
			}
			if m.bad[p] == stamp(st) {
				continue
			}
			files = append(files, p)
		}
	}
	walk(m.dir, 0)
	sort.Strings(files)
	return files
}

// stamp identifies one version of a file, so a replaced bad file is retried.
func stamp(st os.FileInfo) string { return fmt.Sprint(st.Size(), st.ModTime().UnixNano()) }

func (m *MusicPlayer) run() {
	seen := map[string]bool{}
	var queue []string
	last := ""
	for {
		files := m.scan()
		m.active.Store(len(files) > 0)
		if len(files) == 0 {
			time.Sleep(2 * time.Second)
			continue
		}
		present := map[string]bool{}
		for _, f := range files {
			present[f] = true
		}
		for f := range seen {
			if !present[f] {
				delete(seen, f)
			}
		}
		q := queue[:0]
		for _, f := range queue {
			if present[f] {
				q = append(q, f)
			}
		}
		// songs added since the last look play next
		var fresh []string
		for _, f := range files {
			if !seen[f] {
				seen[f] = true
				fresh = append(fresh, f)
			}
		}
		rand.Shuffle(len(fresh), func(i, j int) { fresh[i], fresh[j] = fresh[j], fresh[i] })
		queue = append(fresh, q...)
		if len(queue) == 0 {
			queue = append(queue, files...)
			rand.Shuffle(len(queue), func(i, j int) { queue[i], queue[j] = queue[j], queue[i] })
			if len(queue) > 1 && queue[0] == last {
				queue[0], queue[len(queue)-1] = queue[len(queue)-1], queue[0]
			}
		}
		f := queue[0]
		queue = queue[1:]
		last = f
		if !m.stream(f, m.gen.Load()) {
			if st, err := os.Stat(f); err == nil {
				m.bad[f] = stamp(st)
			}
		}
	}
}

// stream decodes one track into the chunk channel. It returns false when the
// file could not be decoded at all.
func (m *MusicPlayer) stream(path string, gen int32) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	src, rate, err := openTrack(bufio.NewReaderSize(f, 64<<10), strings.ToLower(filepath.Ext(path)))
	if err != nil || rate < 4000 || rate > 384000 {
		return false
	}
	rs := &resampler{src: src, step: float64(rate) / sampleRate, buf: make([]float32, 4096*2)}
	title := trackTitle(path)
	played := false
	for m.gen.Load() == gen {
		pcm := make([]float32, musicChunkFrames*2)
		n := rs.read(pcm)
		if n == 0 {
			break
		}
		c := musicChunk{gen: gen, pcm: pcm[:n*2]}
		if !played {
			c.title, played = title, true
		}
		m.ch <- c
	}
	return played
}

// trackTitle turns a file name into something the HUD font can draw.
func trackTitle(path string) string {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	var b strings.Builder
	space := false
	for _, ch := range strings.ToUpper(name) {
		if _, ok := glyphs[ch]; !ok || ch == '_' {
			space = b.Len() > 0
			continue
		}
		if space {
			b.WriteByte(' ')
			space = false
		}
		b.WriteRune(ch)
	}
	t := b.String()
	if len(t) > 40 {
		t = strings.TrimSpace(t[:40])
	}
	return t
}

// --- decoding ---

// frameSource yields interleaved stereo frames at its own sample rate. Read
// may return frames together with an error; the error ends the track.
type frameSource interface {
	Read(dst []float32) (frames int, err error)
}

func openTrack(r io.Reader, ext string) (frameSource, int, error) {
	switch ext {
	case ".mp3":
		d, err := mp3.NewDecoder(r)
		if err != nil {
			return nil, 0, err
		}
		return &mp3Source{d: d}, d.SampleRate(), nil
	case ".ogg":
		d, err := oggvorbis.NewReader(r)
		if err != nil {
			return nil, 0, err
		}
		if d.Channels() < 1 {
			return nil, 0, errors.New("ogg: no channels")
		}
		return &oggSource{d: d, ch: d.Channels()}, d.SampleRate(), nil
	case ".wav":
		return openWAV(r)
	}
	return nil, 0, errors.New("unsupported file type")
}

type mp3Source struct {
	d   *mp3.Decoder
	buf []byte
}

func (s *mp3Source) Read(dst []float32) (int, error) {
	need := len(dst) / 2 * 4 // always 16-bit stereo
	if cap(s.buf) < need {
		s.buf = make([]byte, need)
	}
	b := s.buf[:need]
	n, err := io.ReadFull(s.d, b)
	frames := n / 4
	for i := 0; i < frames*2; i++ {
		dst[i] = float32(int16(binary.LittleEndian.Uint16(b[i*2:]))) / 32768
	}
	if err == io.ErrUnexpectedEOF {
		err = io.EOF
	}
	return frames, err
}

type oggSource struct {
	d   *oggvorbis.Reader
	ch  int
	buf []float32
}

func (s *oggSource) Read(dst []float32) (int, error) {
	need := len(dst) / 2 * s.ch
	if cap(s.buf) < need {
		s.buf = make([]float32, need)
	}
	b := s.buf[:need]
	n, err := s.d.Read(b)
	frames := n / s.ch
	for i := 0; i < frames; i++ {
		l := b[i*s.ch]
		r := l
		if s.ch > 1 {
			r = b[i*s.ch+1]
		}
		dst[2*i], dst[2*i+1] = l, r
	}
	return frames, err
}

type wavSource struct {
	r         io.Reader
	ch, bytes int // channels, bytes per sample
	float     bool
	left      int64 // bytes of sample data still to read
	buf       []byte
}

// openWAV reads a RIFF/WAVE header: integer PCM (8/16/24/32-bit) or float (32/64-bit).
func openWAV(r io.Reader) (frameSource, int, error) {
	var hdr [12]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, 0, err
	}
	if string(hdr[:4]) != "RIFF" || string(hdr[8:]) != "WAVE" {
		return nil, 0, errors.New("wav: not a RIFF/WAVE file")
	}
	format, ch, rate, bits := 0, 0, 0, 0
	for {
		var ck [8]byte
		if _, err := io.ReadFull(r, ck[:]); err != nil {
			return nil, 0, err
		}
		size := int64(binary.LittleEndian.Uint32(ck[4:]))
		switch string(ck[:4]) {
		case "fmt ":
			if size < 16 || size > 1024 {
				return nil, 0, errors.New("wav: bad fmt chunk")
			}
			b := make([]byte, size+size&1)
			if _, err := io.ReadFull(r, b); err != nil {
				return nil, 0, err
			}
			format = int(binary.LittleEndian.Uint16(b[0:]))
			ch = int(binary.LittleEndian.Uint16(b[2:]))
			rate = int(binary.LittleEndian.Uint32(b[4:]))
			bits = int(binary.LittleEndian.Uint16(b[14:]))
			if format == 0xFFFE && size >= 26 { // WAVE_FORMAT_EXTENSIBLE: the real format leads the sub-format GUID
				format = int(binary.LittleEndian.Uint16(b[24:]))
			}
		case "data":
			isFloat := format == 3 && (bits == 32 || bits == 64)
			isInt := format == 1 && (bits == 8 || bits == 16 || bits == 24 || bits == 32)
			if ch < 1 || !(isFloat || isInt) {
				return nil, 0, fmt.Errorf("wav: unsupported format %d, %d-bit", format, bits)
			}
			if size == 0 || size == 0xFFFFFFFF { // streamed files leave the size open
				size = math.MaxInt64
			}
			return &wavSource{r: r, ch: ch, bytes: bits / 8, float: isFloat, left: size}, rate, nil
		default:
			if _, err := io.CopyN(io.Discard, r, size+size&1); err != nil {
				return nil, 0, err
			}
		}
	}
}

func (s *wavSource) Read(dst []float32) (int, error) {
	fb := s.ch * s.bytes
	need := int64(len(dst) / 2 * fb)
	if need > s.left {
		need = s.left - s.left%int64(fb)
	}
	if need <= 0 {
		return 0, io.EOF
	}
	if int64(cap(s.buf)) < need {
		s.buf = make([]byte, need)
	}
	b := s.buf[:need]
	n, err := io.ReadFull(s.r, b)
	s.left -= int64(n)
	frames := n / fb
	for i := 0; i < frames; i++ {
		f := b[i*fb:]
		l := s.sample(f)
		r := l
		if s.ch > 1 {
			r = s.sample(f[s.bytes:])
		}
		dst[2*i], dst[2*i+1] = l, r
	}
	if err == io.ErrUnexpectedEOF {
		err = io.EOF
	}
	return frames, err
}

func (s *wavSource) sample(b []byte) float32 {
	switch {
	case s.float && s.bytes == 4:
		return math.Float32frombits(binary.LittleEndian.Uint32(b))
	case s.float:
		return float32(math.Float64frombits(binary.LittleEndian.Uint64(b)))
	case s.bytes == 1:
		return (float32(b[0]) - 128) / 128
	case s.bytes == 2:
		return float32(int16(binary.LittleEndian.Uint16(b))) / 32768
	case s.bytes == 3:
		return float32(int32(uint32(b[0])<<8|uint32(b[1])<<16|uint32(b[2])<<24)>>8) / 8388608
	default:
		return float32(int32(binary.LittleEndian.Uint32(b))) / 2147483648
	}
}

// resampler converts a frameSource to sampleRate by linear interpolation.
type resampler struct {
	src         frameSource
	step        float64 // source frames per output frame
	t           float64
	pl, pr      float32 // the two source frames t falls between
	cl, cr      float32
	buf         []float32
	n, i        int
	eof, primed bool
}

func (r *resampler) frame() (l, rr float32, ok bool) {
	for r.i >= r.n {
		if r.eof {
			return 0, 0, false
		}
		empty := 0
		for {
			n, err := r.src.Read(r.buf)
			r.n, r.i = n, 0
			if err != nil {
				r.eof = true
			}
			if n > 0 || r.eof {
				break
			}
			if empty++; empty > 64 { // a decoder that makes no progress
				r.eof = true
			}
		}
	}
	l, rr = r.buf[2*r.i], r.buf[2*r.i+1]
	r.i++
	return l, rr, true
}

// read fills out with stereo frames and returns how many; 0 means the track ended.
func (r *resampler) read(out []float32) int {
	if !r.primed {
		var ok bool
		if r.pl, r.pr, ok = r.frame(); !ok {
			return 0
		}
		if r.cl, r.cr, ok = r.frame(); !ok {
			r.cl, r.cr = r.pl, r.pr
		}
		r.primed = true
	}
	frames := 0
	for frames*2 < len(out) {
		for r.t >= 1 {
			l, rr, ok := r.frame()
			if !ok {
				return frames
			}
			r.pl, r.pr, r.cl, r.cr = r.cl, r.cr, l, rr
			r.t--
		}
		k := float32(r.t)
		out[2*frames] = r.pl + (r.cl-r.pl)*k
		out[2*frames+1] = r.pr + (r.cr-r.pr)*k
		frames++
		r.t += r.step
	}
	return frames
}
