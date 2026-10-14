package transfer

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/sftp"
)

type Kind int

const (
	Upload Kind = iota
	Download
)

type State int

const (
	Queued State = iota
	Running
	Done
	Failed
	Cancelled
)

const parallel = 2

var ErrCancelled = errors.New("cancelled")

type Job struct {
	ID     int
	Kind   Kind
	Name   string
	Local  string
	Remote string
	Writer io.WriteCloser

	client *sftp.Client
	total  atomic.Int64
	done   atomic.Int64
	files  atomic.Int64
	state  atomic.Int32
	err    error
	cancel context.CancelFunc
	ctx    context.Context
	onDone func(*Job)
}

func (j *Job) Progress() (done, total int64) {
	return j.done.Load(), j.total.Load()
}

func (j *Job) Files() int64 {
	return j.files.Load()
}

func (j *Job) State() State {
	return State(j.state.Load())
}

func (j *Job) Err() error {
	return j.err
}

func (j *Job) Finished() bool {
	s := j.State()
	return s == Done || s == Failed || s == Cancelled
}

type Queue struct {
	mu      sync.Mutex
	jobs    []*Job
	next    int
	slots   chan struct{}
	changed func()
}

func New(changed func()) *Queue {
	if changed == nil {
		changed = func() {}
	}

	return &Queue{slots: make(chan struct{}, parallel), changed: changed}
}

func (q *Queue) Jobs() []*Job {
	q.mu.Lock()
	defer q.mu.Unlock()

	return append([]*Job(nil), q.jobs...)
}

func (q *Queue) Active() int {
	n := 0

	for _, j := range q.Jobs() {
		if !j.Finished() {
			n++
		}
	}

	return n
}

func (q *Queue) Add(client *sftp.Client, j *Job, onDone func(*Job)) *Job {
	q.mu.Lock()
	q.next++
	j.ID = q.next
	j.client = client
	j.onDone = onDone
	j.ctx, j.cancel = context.WithCancel(context.Background())
	q.jobs = append(q.jobs, j)
	q.mu.Unlock()

	go q.run(j)
	q.changed()
	return j
}

func (q *Queue) Cancel(id int) {
	for _, j := range q.Jobs() {
		if j.ID == id && !j.Finished() {
			j.cancel()
		}
	}
}

func (q *Queue) ClearFinished() {
	q.mu.Lock()
	live := q.jobs[:0]

	for _, j := range q.jobs {
		if !j.Finished() {
			live = append(live, j)
		}
	}

	q.jobs = live
	q.mu.Unlock()
	q.changed()
}

func (q *Queue) run(j *Job) {
	select {
	case q.slots <- struct{}{}:
	case <-j.ctx.Done():
		j.finish(Cancelled, ErrCancelled)
		q.changed()
		return
	}

	defer func() { <-q.slots }()

	j.state.Store(int32(Running))
	q.changed()

	stop := make(chan struct{})

	go func() {
		t := time.NewTicker(200 * time.Millisecond)
		defer t.Stop()

		for {
			select {
			case <-stop:
				return
			case <-t.C:
				q.changed()
			}
		}
	}()

	var err error

	if j.Kind == Upload {
		err = j.upload()
	} else {
		err = j.download()
	}

	close(stop)

	switch {
	case err == nil:
		j.finish(Done, nil)
	case j.ctx.Err() != nil:
		j.finish(Cancelled, ErrCancelled)
	default:
		j.finish(Failed, err)
	}

	q.changed()
}

func (j *Job) finish(s State, err error) {
	j.err = err
	j.state.Store(int32(s))

	if j.onDone != nil {
		j.onDone(j)
	}
}

type counter struct {
	ctx context.Context
	n   *atomic.Int64
	r   io.Reader
}

func (c counter) Read(p []byte) (int, error) {
	if err := c.ctx.Err(); err != nil {
		return 0, err
	}

	n, err := c.r.Read(p)
	c.n.Add(int64(n))
	return n, err
}

func (j *Job) copy(dst io.Writer, src io.Reader) error {
	_, err := io.Copy(dst, counter{ctx: j.ctx, n: &j.done, r: src})
	return err
}

func (j *Job) upload() error {
	info, err := os.Stat(j.Local)

	if err != nil {
		return err
	}

	if !info.IsDir() {
		j.total.Store(info.Size())
		return j.uploadFile(j.Local, j.Remote)
	}

	var size int64

	filepath.WalkDir(j.Local, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			if fi, err := d.Info(); err == nil {
				size += fi.Size()
			}
		}

		return nil
	})

	j.total.Store(size)

	return filepath.WalkDir(j.Local, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if err := j.ctx.Err(); err != nil {
			return err
		}

		rel, _ := filepath.Rel(j.Local, p)
		dst := path.Join(j.Remote, filepath.ToSlash(rel))

		if d.IsDir() {
			return j.client.MkdirAll(dst)
		}

		return j.uploadFile(p, dst)
	})
}

func (j *Job) uploadFile(src, dst string) error {
	in, err := os.Open(src)

	if err != nil {
		return err
	}

	defer in.Close()

	out, err := j.client.Create(dst)

	if err != nil {
		return err
	}

	err = j.copy(out, in)

	if cerr := out.Close(); err == nil {
		err = cerr
	}

	if err == nil {
		j.files.Add(1)
	}

	return err
}

func (j *Job) download() error {
	info, err := j.client.Stat(j.Remote)

	if err != nil {
		return err
	}

	if !info.IsDir() {
		j.total.Store(info.Size())

		if j.Writer != nil {
			defer j.Writer.Close()
			return j.downloadTo(j.Remote, j.Writer)
		}

		return j.downloadFile(j.Remote, j.Local)
	}

	var size int64
	walk := j.client.Walk(j.Remote)

	for walk.Step() {
		if walk.Err() == nil && !walk.Stat().IsDir() {
			size += walk.Stat().Size()
		}
	}

	j.total.Store(size)
	walk = j.client.Walk(j.Remote)

	for walk.Step() {
		if err := walk.Err(); err != nil {
			return err
		}

		if err := j.ctx.Err(); err != nil {
			return err
		}

		rel := walk.Path()[len(j.Remote):]
		dst := filepath.Join(j.Local, filepath.FromSlash(rel))

		if walk.Stat().IsDir() {
			if err := os.MkdirAll(dst, 0o755); err != nil {
				return err
			}

			continue
		}

		if err := j.downloadFile(walk.Path(), dst); err != nil {
			return err
		}
	}

	return nil
}

func (j *Job) downloadFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	out, err := os.Create(dst)

	if err != nil {
		return err
	}

	err = j.downloadTo(src, out)

	if cerr := out.Close(); err == nil {
		err = cerr
	}

	if err != nil {
		os.Remove(dst)
	}

	return err
}

func (j *Job) downloadTo(src string, w io.Writer) error {
	in, err := j.client.Open(src)

	if err != nil {
		return err
	}

	defer in.Close()

	if err := j.copy(w, in); err != nil {
		return err
	}

	j.files.Add(1)
	return nil
}
