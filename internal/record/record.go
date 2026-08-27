package record

import (
	"os"
	"path/filepath"
	"sync"

	"platformscreendoor/internal/event"
)

type Handle interface {
	Close() error
}

type HandleFactory interface {
	Open(path string) (Handle, error)
}

type FileFactory struct {
	Base string
}

func NewFileFactory(base string) *FileFactory {
	return &FileFactory{Base: base}
}

func (f *FileFactory) Open(path string) (Handle, error) {
	full := filepath.Join(f.Base, filepath.Base(path))
	return os.OpenFile(full, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
}

type Recorder struct {
	mu      sync.Mutex
	factory HandleFactory
	streams map[string]Handle
	store   *Store
	base    string
}

func NewRecorder(factory HandleFactory, base string, store *Store) *Recorder {
	return &Recorder{
		factory: factory,
		streams: make(map[string]Handle),
		store:   store,
		base:    base,
	}
}

func (r *Recorder) Begin(session string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.streams[session]; ok {
		return nil
	}
	if r.factory == nil {
		r.streams[session] = nil
		return nil
	}
	h, err := r.factory.Open(r.base + "/" + session)
	if err != nil {
		return err
	}
	r.streams[session] = h
	return nil
}

func (r *Recorder) Append(e event.Event) error {
	if r.store != nil {
		r.store.Append(e)
	}
	return nil
}

func (r *Recorder) Done(session string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	h, ok := r.streams[session]
	if !ok {
		return nil
	}
	delete(r.streams, session)
	if h != nil {
		return h.Close()
	}
	return nil
}

func (r *Recorder) OpenCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.streams)
}

func (r *Recorder) Streams() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, 0, len(r.streams))
	for session := range r.streams {
		out = append(out, session)
	}
	return out
}
