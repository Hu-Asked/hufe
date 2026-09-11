package fileops

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

type Phase uint8

const (
	PhasePreparing Phase = iota
	PhaseCopying
)

type Progress struct {
	Phase          Phase
	CompletedBytes int64
	TotalBytes     int64
	CompletedItems int
	TotalItems     int
	CurrentPath    string
}

type Result struct {
	Target string
}

type Reporter func(Progress)

type manifestEntry struct {
	source     string
	relative   string
	mode       fs.FileMode
	modifiedAt time.Time
	linkTarget string
}

func Copy(ctx context.Context, source, destinationDirectory string, report Reporter) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	source, err := filepath.Abs(source)
	if err != nil {
		return Result{}, fmt.Errorf("resolve source: %w", err)
	}
	destinationDirectory, err = filepath.Abs(destinationDirectory)
	if err != nil {
		return Result{}, fmt.Errorf("resolve destination: %w", err)
	}
	if report != nil {
		report(Progress{Phase: PhasePreparing, CurrentPath: filepath.Base(source)})
	}

	destinationInfo, err := os.Stat(destinationDirectory)
	if err != nil {
		return Result{}, fmt.Errorf("read destination: %w", err)
	}
	if !destinationInfo.IsDir() {
		return Result{}, fmt.Errorf("destination is not a directory: %s", destinationDirectory)
	}

	target := filepath.Join(destinationDirectory, filepath.Base(source))
	if _, err := os.Lstat(target); err == nil {
		return Result{}, fmt.Errorf("destination already exists: %w", fs.ErrExist)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return Result{}, fmt.Errorf("check destination: %w", err)
	}

	entries, totalBytes, err := buildManifest(ctx, source)
	if err != nil {
		return Result{}, err
	}
	if len(entries) == 0 {
		return Result{}, fmt.Errorf("source does not exist: %s", source)
	}

	stagingDirectory, err := os.MkdirTemp(destinationDirectory, ".hufe-paste-")
	if err != nil {
		return Result{}, fmt.Errorf("create staging directory: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = os.RemoveAll(stagingDirectory)
		}
	}()

	tracker := newProgressTracker(totalBytes, len(entries), report)
	tracker.emit(PhaseCopying, "")
	stagingRoot := filepath.Join(stagingDirectory, filepath.Base(source))

	var directories []manifestEntry
	var regularFiles []manifestEntry
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		targetPath := stagingPath(stagingRoot, entry.relative)
		switch {
		case entry.mode.IsDir():
			if err := os.Mkdir(targetPath, 0o700); err != nil {
				return Result{}, fmt.Errorf("create directory %s: %w", entry.relative, err)
			}
			directories = append(directories, entry)
			tracker.itemDone(entry.relative)
		case entry.mode.IsRegular():
			regularFiles = append(regularFiles, entry)
		case entry.mode&fs.ModeSymlink != 0:
			if err := os.Symlink(entry.linkTarget, targetPath); err != nil {
				return Result{}, fmt.Errorf("copy symbolic link %s: %w", entry.relative, err)
			}
			tracker.itemDone(entry.relative)
		}
	}

	if err := copyRegularFiles(ctx, stagingRoot, regularFiles, tracker); err != nil {
		return Result{}, err
	}

	for index := len(directories) - 1; index >= 0; index-- {
		entry := directories[index]
		targetPath := stagingPath(stagingRoot, entry.relative)
		if err := os.Chmod(targetPath, entry.mode.Perm()); err != nil {
			return Result{}, fmt.Errorf("set directory permissions %s: %w", entry.relative, err)
		}
		if err := os.Chtimes(targetPath, entry.modifiedAt, entry.modifiedAt); err != nil {
			return Result{}, fmt.Errorf("set directory timestamp %s: %w", entry.relative, err)
		}
	}

	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if _, err := os.Lstat(target); err == nil {
		return Result{}, fmt.Errorf("destination appeared while copying: %w", fs.ErrExist)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return Result{}, fmt.Errorf("recheck destination: %w", err)
	}
	if err := os.Rename(stagingRoot, target); err != nil {
		return Result{}, fmt.Errorf("finish paste: %w", err)
	}
	committed = true
	_ = os.Remove(stagingDirectory)
	tracker.finish()

	return Result{Target: target}, nil
}

func buildManifest(ctx context.Context, source string) ([]manifestEntry, int64, error) {
	var entries []manifestEntry
	var totalBytes int64
	err := filepath.WalkDir(source, func(path string, directoryEntry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}

		info, err := directoryEntry.Info()
		if err != nil {
			return err
		}
		mode := info.Mode()
		if !mode.IsDir() && !mode.IsRegular() && mode&fs.ModeSymlink == 0 {
			return fmt.Errorf("unsupported file type: %s", path)
		}

		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		entry := manifestEntry{
			source:     path,
			relative:   relative,
			mode:       mode,
			modifiedAt: info.ModTime(),
		}
		if mode.IsRegular() {
			totalBytes += info.Size()
		}
		if mode&fs.ModeSymlink != 0 {
			entry.linkTarget, err = os.Readlink(path)
			if err != nil {
				return err
			}
		}
		entries = append(entries, entry)
		return nil
	})
	if err != nil {
		return nil, 0, fmt.Errorf("scan source: %w", err)
	}
	return entries, totalBytes, nil
}

func stagingPath(root, relative string) string {
	if relative == "." {
		return root
	}
	return filepath.Join(root, relative)
}

func copyRegularFiles(ctx context.Context, stagingRoot string, entries []manifestEntry, tracker *progressTracker) error {
	if len(entries) == 0 {
		return nil
	}

	workerContext, cancel := context.WithCancel(ctx)
	defer cancel()
	workerCount := min(len(entries), min(8, max(2, runtime.GOMAXPROCS(0))))
	jobs := make(chan manifestEntry)
	var workers sync.WaitGroup
	var firstError error
	var errorOnce sync.Once

	for range workerCount {
		workers.Add(1)
		go func() {
			defer workers.Done()
			buffer := make([]byte, 256*1024)
			for entry := range jobs {
				if err := copyRegularFile(workerContext, stagingRoot, entry, tracker, buffer); err != nil {
					errorOnce.Do(func() {
						firstError = err
						cancel()
					})
					return
				}
			}
		}()
	}

sendJobs:
	for _, entry := range entries {
		select {
		case jobs <- entry:
		case <-workerContext.Done():
			break sendJobs
		}
	}
	close(jobs)
	workers.Wait()

	if firstError != nil {
		return firstError
	}
	return ctx.Err()
}

func copyRegularFile(ctx context.Context, stagingRoot string, entry manifestEntry, tracker *progressTracker, buffer []byte) error {
	sourceFile, err := os.Open(entry.source)
	if err != nil {
		return fmt.Errorf("open %s: %w", entry.relative, err)
	}
	defer sourceFile.Close()

	targetPath := stagingPath(stagingRoot, entry.relative)
	targetFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("create %s: %w", entry.relative, err)
	}

	_, copyErr := io.CopyBuffer(&trackingWriter{
		ctx:     ctx,
		writer:  targetFile,
		tracker: tracker,
		current: entry.relative,
	}, &contextReader{ctx: ctx, reader: sourceFile}, buffer)
	closeErr := targetFile.Close()
	if copyErr != nil {
		return fmt.Errorf("copy %s: %w", entry.relative, copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close %s: %w", entry.relative, closeErr)
	}
	if err := os.Chmod(targetPath, entry.mode.Perm()); err != nil {
		return fmt.Errorf("set file permissions %s: %w", entry.relative, err)
	}
	if err := os.Chtimes(targetPath, entry.modifiedAt, entry.modifiedAt); err != nil {
		return fmt.Errorf("set file timestamp %s: %w", entry.relative, err)
	}
	tracker.itemDone(entry.relative)
	return nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (reader *contextReader) Read(buffer []byte) (int, error) {
	if err := reader.ctx.Err(); err != nil {
		return 0, err
	}
	return reader.reader.Read(buffer)
}

type trackingWriter struct {
	ctx     context.Context
	writer  io.Writer
	tracker *progressTracker
	current string
}

func (writer *trackingWriter) Write(buffer []byte) (int, error) {
	if err := writer.ctx.Err(); err != nil {
		return 0, err
	}
	written, err := writer.writer.Write(buffer)
	if written > 0 {
		writer.tracker.bytesDone(int64(written), writer.current)
	}
	return written, err
}

type progressTracker struct {
	mutex          sync.Mutex
	report         Reporter
	completedBytes int64
	totalBytes     int64
	completedItems int
	totalItems     int
	currentPath    string
	lastReport     time.Time
}

func newProgressTracker(totalBytes int64, totalItems int, report Reporter) *progressTracker {
	return &progressTracker{
		report:     report,
		totalBytes: totalBytes,
		totalItems: totalItems,
	}
}

func (tracker *progressTracker) bytesDone(count int64, current string) {
	tracker.mutex.Lock()
	defer tracker.mutex.Unlock()
	tracker.completedBytes += count
	tracker.currentPath = current
	shouldReport := time.Since(tracker.lastReport) >= 50*time.Millisecond
	if shouldReport {
		tracker.lastReport = time.Now()
	}
	if shouldReport && tracker.report != nil {
		tracker.report(tracker.snapshotLocked(PhaseCopying))
	}
}

func (tracker *progressTracker) itemDone(current string) {
	tracker.mutex.Lock()
	defer tracker.mutex.Unlock()
	tracker.completedItems++
	tracker.currentPath = current
	tracker.lastReport = time.Now()
	if tracker.report != nil {
		tracker.report(tracker.snapshotLocked(PhaseCopying))
	}
}

func (tracker *progressTracker) emit(phase Phase, current string) {
	tracker.mutex.Lock()
	defer tracker.mutex.Unlock()
	tracker.currentPath = current
	tracker.lastReport = time.Now()
	if tracker.report != nil {
		tracker.report(tracker.snapshotLocked(phase))
	}
}

func (tracker *progressTracker) finish() {
	tracker.mutex.Lock()
	defer tracker.mutex.Unlock()
	tracker.completedBytes = tracker.totalBytes
	tracker.completedItems = tracker.totalItems
	if tracker.report != nil {
		tracker.report(tracker.snapshotLocked(PhaseCopying))
	}
}

func (tracker *progressTracker) snapshotLocked(phase Phase) Progress {
	return Progress{
		Phase:          phase,
		CompletedBytes: tracker.completedBytes,
		TotalBytes:     tracker.totalBytes,
		CompletedItems: tracker.completedItems,
		TotalItems:     tracker.totalItems,
		CurrentPath:    tracker.currentPath,
	}
}
