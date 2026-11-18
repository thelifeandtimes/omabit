//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"sync"
	"syscall"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
)

var (
	ErrRecordingJobStorage  = errors.New("private recording job storage changed or is incomplete; preserve its original files")
	ErrRecordingJobActive   = errors.New("recording job already has an active worker; inspect its saved readiness")
	ErrRecordingJobClaimed  = errors.New("recording job was already claimed; never replay it, inspect its original coverage")
	ErrRecordingJobLimit    = errors.New("recording job storage admission limit reached; preserve existing jobs")
	ErrRecordingJobNotReady = errors.New("recording job has no durable readiness acknowledgement")
)

// RecordingJobs pins an already authorized private directory. It does not select
// a runner, authorize a launch, create its root or install a service. Job/claim/
// readiness originals are never overwritten, adopted or automatically removed.
type RecordingJobs struct {
	root    *os.Root
	maxJobs int
	reserve uint64
	free    func(*os.Root, uint64) error
}

func OpenRecordingJobs(path string, maxJobs int, reserve uint64) (*RecordingJobs, error) {
	if !recordingJobPath(path) || maxJobs < 1 || maxJobs > 4096 || reserve == 0 {
		return nil, ErrRecordingJobStorage
	}
	before, err := os.Lstat(path)
	if err != nil || !recordingJobDirectory(before) {
		return nil, ErrRecordingJobStorage
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, ErrRecordingJobStorage
	}
	file, err := root.Open(".")
	if err != nil {
		root.Close()
		return nil, ErrRecordingJobStorage
	}
	after, err := file.Stat()
	file.Close()
	if err != nil || !recordingJobDirectory(after) || !os.SameFile(before, after) {
		root.Close()
		return nil, ErrRecordingJobStorage
	}
	return &RecordingJobs{root: root, maxJobs: maxJobs, reserve: reserve, free: recordingJobSpace}, nil
}

func (s *RecordingJobs) Close() error { return s.root.Close() }

func recordingJobDirectory(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && info.Mode() == os.ModeDir|0700 && stat.Uid == uint32(os.Geteuid())
}

func recordingJobRootValid(root *os.Root) bool {
	file, err := root.Open(".")
	if err != nil {
		return false
	}
	defer file.Close()
	info, err := file.Stat()
	return err == nil && recordingJobDirectory(info)
}

func recordingJobChildBound(parent *os.Root, id string, child *os.Root) bool {
	path, err := parent.Lstat(id)
	if err != nil || !recordingJobDirectory(path) {
		return false
	}
	file, err := child.Open(".")
	if err != nil {
		return false
	}
	defer file.Close()
	info, err := file.Stat()
	return err == nil && recordingJobDirectory(info) && os.SameFile(path, info)
}

func recordingJobFile(root *os.Root, name string, flags int, max int64) (*os.File, error) {
	file, err := root.OpenFile(name, flags|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0600)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil || !recordingJobFileInfo(info, max) {
		file.Close()
		return nil, ErrRecordingJobStorage
	}
	return file, nil
}

func recordingJobFileInfo(info os.FileInfo, max int64) bool {
	if info == nil {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && info.Mode() == 0600 && stat.Uid == uint32(os.Geteuid()) && stat.Nlink == 1 && info.Size() <= max
}

func recordingJobBound(root *os.Root, name string, file *os.File, size int64) bool {
	info, e1 := file.Stat()
	path, e2 := root.Lstat(name)
	return e1 == nil && e2 == nil && recordingJobFileInfo(info, size) && info.Size() == size && os.SameFile(info, path)
}

func recordingJobChild(root *os.Root, id string) (*os.Root, error) {
	if !recordingJobHex(id, 32) {
		return nil, ErrRecordingJobStorage
	}
	before, err := root.Lstat(id)
	if err != nil || !recordingJobDirectory(before) {
		return nil, ErrRecordingJobStorage
	}
	child, err := root.OpenRoot(id)
	if err != nil {
		return nil, ErrRecordingJobStorage
	}
	file, err := child.Open(".")
	if err != nil {
		child.Close()
		return nil, ErrRecordingJobStorage
	}
	after, err := file.Stat()
	file.Close()
	if err != nil || !recordingJobDirectory(after) || !os.SameFile(before, after) {
		child.Close()
		return nil, ErrRecordingJobStorage
	}
	return child, nil
}

func recordingJobSync(root *os.Root) error {
	file, err := root.Open(".")
	if err != nil {
		return ErrRecordingJobStorage
	}
	defer file.Close()
	return file.Sync()
}

func recordingJobSpace(root *os.Root, required uint64) error {
	file, err := root.Open(".")
	if err != nil {
		return ErrRecordingJobStorage
	}
	defer file.Close()
	var stat syscall.Statfs_t
	if syscall.Fstatfs(int(file.Fd()), &stat) != nil || stat.Bsize <= 0 || stat.Bavail > ^uint64(0)/uint64(stat.Bsize) {
		return ErrRecordingJobStorage
	}
	if stat.Bavail*uint64(stat.Bsize) < required {
		return ErrRecordingJobLimit
	}
	return nil
}

func (s *RecordingJobs) admit(extra uint64) error {
	if extra > ^uint64(0)-s.reserve {
		return ErrRecordingJobLimit
	}
	return s.free(s.root, s.reserve+extra)
}

func recordingJobWrite(root *os.Root, name string, data []byte) error {
	file, err := recordingJobFile(root, name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, int64(len(data)))
	if err != nil {
		return err
	}
	count, writeErr := file.Write(data)
	if count != len(data) && writeErr == nil {
		writeErr = io.ErrShortWrite
	}
	if writeErr == nil && !recordingJobBound(root, name, file, int64(len(data))) {
		writeErr = ErrRecordingJobStorage
	}
	err = errors.Join(writeErr, file.Sync(), file.Close(), recordingJobSync(root))
	return err // a failed/partial original remains, never removed or repaired
}

func recordingJobRead(root *os.Root, name string, max int64) ([]byte, error) {
	file, err := recordingJobFile(root, name, os.O_RDONLY, max)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, max+1))
	if err != nil || int64(len(data)) > max || !recordingJobBound(root, name, file, int64(len(data))) {
		return nil, ErrRecordingJobStorage
	}
	return data, nil
}

func recordingJobSHA(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// Create fsyncs the job and lock plus both directories before acknowledging it.
// Cross-process admission is serialized; a quota check cannot race another
// manager into exceeding maxJobs. Existing/incomplete directories refuse.
func (s *RecordingJobs) Create(job ManagedRecordingJob) (string, error) {
	if !recordingJobRootValid(s.root) {
		return "", ErrRecordingJobStorage
	}
	data, err := job.Bytes()
	if err != nil {
		return "", err
	}
	if err = s.admit(uint64(len(data) + 16384)); err != nil {
		return "", err
	}
	lock, err := recordingJobFile(s.root, "admission.lock", os.O_RDWR|os.O_CREATE, 0)
	if err != nil {
		return "", ErrRecordingJobStorage
	}
	defer lock.Close()
	if syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) != nil {
		return "", ErrRecordingJobActive
	}
	if !recordingJobBound(s.root, "admission.lock", lock, 0) {
		return "", ErrRecordingJobStorage
	}
	directory, err := s.root.Open(".")
	if err != nil {
		return "", ErrRecordingJobStorage
	}
	entries, err := directory.ReadDir(s.maxJobs + 2)
	directory.Close()
	if err != nil && err != io.EOF {
		return "", ErrRecordingJobStorage
	}
	count := 0
	for _, entry := range entries {
		if entry.Name() == "admission.lock" {
			continue
		}
		if !recordingJobHex(entry.Name(), 32) || !entry.IsDir() {
			return "", ErrRecordingJobStorage
		}
		count++
	}
	if count >= s.maxJobs {
		return "", ErrRecordingJobLimit
	}
	if err = lock.Sync(); err != nil {
		return "", err
	}
	if err = s.root.Mkdir(job.ID, 0700); err != nil {
		return "", ErrRecordingJobStorage
	}
	child, err := recordingJobChild(s.root, job.ID)
	if err != nil {
		return "", err
	}
	defer child.Close()
	if err = recordingJobWrite(child, "job.json", data); err != nil {
		return "", err
	}
	if err = recordingJobWrite(child, "worker.lock", nil); err != nil {
		return "", err
	}
	if err = errors.Join(recordingJobSync(child), recordingJobSync(s.root)); err != nil {
		return "", err
	}
	if !recordingJobChildBound(s.root, job.ID, child) {
		return "", ErrRecordingJobStorage
	}
	return recordingJobSHA(data), nil
}

func (s *RecordingJobs) read(root *os.Root, id, digest string) (ManagedRecordingJob, error) {
	if !recordingJobRootValid(s.root) || !recordingJobChildBound(s.root, id, root) {
		return ManagedRecordingJob{}, ErrRecordingJobStorage
	}
	if !recordingJobHex(digest, 64) {
		return ManagedRecordingJob{}, ErrRecordingJob
	}
	raw, err := recordingJobRead(root, "job.json", MaxRecordingJobBytes)
	if err != nil || recordingJobSHA(raw) != digest {
		return ManagedRecordingJob{}, ErrRecordingJobStorage
	}
	job, err := ReadManagedRecordingJob(raw)
	if err != nil || job.ID != id {
		return ManagedRecordingJob{}, ErrRecordingJob
	}
	return job, nil
}

// Read is lookup only. It contacts no engine or service and changes no file.
func (s *RecordingJobs) Read(id, digest string) (ManagedRecordingJob, error) {
	root, err := recordingJobChild(s.root, id)
	if err != nil {
		return ManagedRecordingJob{}, err
	}
	defer root.Close()
	return s.read(root, id, digest)
}

type RecordingWorkerIdentity struct {
	ExecutableSHA256 string                   `json:"executable_sha256"`
	PID              uint32                   `json:"pid"`
	HostBootID       string                   `json:"host_boot_id"`
	Namespaces       recording.HostNamespaces `json:"namespaces"`
}

func (i RecordingWorkerIdentity) valid() bool {
	b := i.HostBootID
	return recordingJobHex(i.ExecutableSHA256, 64) && i.PID > 1 && len(b) == 36 && b[8] == '-' && b[13] == '-' && b[18] == '-' && b[23] == '-' && recordingJobHex(stringsWithoutBootDashes(b), 32) && i.Namespaces.User != 0 && i.Namespaces.Mount != 0 && i.Namespaces.PID != 0 && i.Namespaces.Network != 0
}

func stringsWithoutBootDashes(s string) string {
	var out []byte
	for _, b := range []byte(s) {
		if b != '-' {
			out = append(out, b)
		}
	}
	return string(out)
}

type recordingJobClaim struct {
	Version   int                     `json:"version"`
	JobID     string                  `json:"job_id"`
	JobSHA256 string                  `json:"job_sha256"`
	Worker    RecordingWorkerIdentity `json:"worker"`
}

type RecordingJobReady struct {
	Version        int       `json:"version"`
	JobID          string    `json:"job_id"`
	JobSHA256      string    `json:"job_sha256"`
	SessionID      string    `json:"session_id"`
	MetadataSHA256 string    `json:"metadata_sha256"`
	AttachedAt     time.Time `json:"attached_at"`
}

type RecordingJobLease struct {
	mu               sync.Mutex
	store            *RecordingJobs
	root             *os.Root
	lock             *os.File
	job              ManagedRecordingJob
	digest, claimSHA string
	worker           RecordingWorkerIdentity
	closed           bool
}

// Claim requires actual worker observations, never caller-supplied RPC facts.
// Verify the executable before any claim is written. Fsync the immutable claim
// before native admission/attachment. A closed/crashed claim is never replayed.
func (s *RecordingJobs) Claim(id, digest string, worker RecordingWorkerIdentity) (_ *RecordingJobLease, err error) {
	if !worker.valid() {
		return nil, ErrRecordingJob
	}
	root, err := recordingJobChild(s.root, id)
	if err != nil {
		return nil, err
	}
	lease := &RecordingJobLease{root: root, store: s, worker: worker, digest: digest}
	defer func() {
		if err != nil {
			if lease.lock != nil {
				lease.lock.Close()
			}
			root.Close()
		}
	}()
	lease.job, err = s.read(root, id, digest)
	if err != nil {
		return nil, err
	}
	if worker.ExecutableSHA256 != lease.job.ExecutableSHA256 {
		return nil, ErrRecordingJob
	}
	lease.lock, err = recordingJobFile(root, "worker.lock", os.O_RDWR, 0)
	if err != nil {
		return nil, ErrRecordingJobStorage
	}
	if syscall.Flock(int(lease.lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) != nil {
		return nil, ErrRecordingJobActive
	}
	if !recordingJobBound(root, "worker.lock", lease.lock, 0) {
		return nil, ErrRecordingJobStorage
	}
	if _, e := root.Lstat("claim.json"); !errors.Is(e, os.ErrNotExist) {
		return nil, ErrRecordingJobClaimed
	}
	if err = s.admit(8192); err != nil {
		return nil, err
	}
	data, _ := json.Marshal(recordingJobClaim{Version: 1, JobID: id, JobSHA256: digest, Worker: worker})
	if err = recordingJobWrite(root, "claim.json", data); err != nil {
		return nil, err
	}
	lease.claimSHA = recordingJobSHA(data)
	return lease, nil
}

func (l *RecordingJobLease) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return nil
	}
	l.closed = true
	return errors.Join(l.lock.Close(), l.root.Close())
}

// Ready is an immutable durable attachment acknowledgement, not Start/exit or
// full-coverage evidence. Only the independently admitted worker may call it.
func (l *RecordingJobLease) Ready(m recording.Metadata) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed || !recordingJobBound(l.root, "worker.lock", l.lock, 0) || !l.job.MatchesAdmission(m) || m.Identity.HostBootID != l.worker.HostBootID || m.Identity.RecorderNamespaces != l.worker.Namespaces {
		return ErrRecordingJobStorage
	}
	if _, err := l.store.read(l.root, l.job.ID, l.digest); err != nil {
		return err
	}
	claim, err := recordingJobRead(l.root, "claim.json", 4096)
	if err != nil || recordingJobSHA(claim) != l.claimSHA {
		return ErrRecordingJobStorage
	}
	if err = l.store.admit(8192); err != nil {
		return err
	}
	digest, _ := m.Digest()
	data, _ := json.Marshal(RecordingJobReady{Version: 1, JobID: l.job.ID, JobSHA256: l.digest, SessionID: m.SessionID, MetadataSHA256: hex.EncodeToString(digest[:]), AttachedAt: m.AttachedAt.UTC()})
	return recordingJobWrite(l.root, "ready.json", data)
}

// ReadReady returns saved durable evidence only. It establishes neither current
// worker/process health nor permission to Start; the manager must separately
// verify its lifecycle fence, current native admission and exact live worker.
func (s *RecordingJobs) ReadReady(id, digest string) (RecordingJobReady, error) {
	root, err := recordingJobChild(s.root, id)
	if err != nil {
		return RecordingJobReady{}, err
	}
	defer root.Close()
	job, err := s.read(root, id, digest)
	if err != nil {
		return RecordingJobReady{}, err
	}
	data, err := recordingJobRead(root, "ready.json", 4096)
	if errors.Is(err, os.ErrNotExist) {
		return RecordingJobReady{}, ErrRecordingJobNotReady
	}
	if err != nil {
		return RecordingJobReady{}, err
	}
	var ready RecordingJobReady
	if json.Unmarshal(data, &ready) != nil || ready.Version != 1 || ready.JobID != id || ready.JobSHA256 != digest || !recordingJobHex(ready.SessionID, 32) || !recordingJobHex(ready.MetadataSHA256, 64) || ready.AttachedAt.Before(time.Unix(0, 1)) || ready.AttachedAt.After(time.Unix(0, math.MaxInt64)) {
		return RecordingJobReady{}, ErrRecordingJobStorage
	}
	canonical, _ := json.Marshal(ready)
	if !bytes.Equal(canonical, data) {
		return RecordingJobReady{}, ErrRecordingJobStorage
	}
	claimData, err := recordingJobRead(root, "claim.json", 4096)
	var claim recordingJobClaim
	if err != nil || json.Unmarshal(claimData, &claim) != nil || claim.Version != 1 || claim.JobID != id || claim.JobSHA256 != digest || !claim.Worker.valid() || claim.Worker.ExecutableSHA256 != job.ExecutableSHA256 {
		return RecordingJobReady{}, ErrRecordingJobStorage
	}
	canonical, _ = json.Marshal(claim)
	if !bytes.Equal(canonical, claimData) {
		return RecordingJobReady{}, ErrRecordingJobStorage
	}
	return ready, nil
}
