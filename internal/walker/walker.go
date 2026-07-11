package walker

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
)

// Entry is one filesystem entry discovered by the walker.
type Entry struct {
	RelPath string // relative to root, forward-slash separated
	IsDir   bool
	Size    int64
	Depth   int
}

// Options controls walk behavior.
type Options struct {
	MaxDepth         int // 0 = unlimited
	RespectGitignore bool
	ShowHidden       bool
	DirsOnly         bool
	FilesOnly        bool
}

// job is a directory queued for a worker to process.
type job struct {
	absPath string
	relPath string
	depth   int
	ignores *IgnoreSet
}

// Walk performs a parallel recursive walk of root and returns all discovered entries.
// Directories matched by ignore rules are pruned before being opened (never read),
// which is the main speed lever versus a naive recursive ls.
func Walk(root string, opt Options) ([]Entry, error) {
	root = filepath.Clean(root)

	baseIgnores := NewIgnoreSet()
	if opt.RespectGitignore {
		_ = baseIgnores.LoadGitignore(filepath.Join(root, ".gitignore"))
	}

	numWorkers := max(runtime.NumCPU(), 1)

	jobs := make(chan job, 1024)

	var wg sync.WaitGroup // tracks in-flight jobs (not workers)
	var workerWG sync.WaitGroup

	var mu sync.Mutex
	var results []Entry
	var walkErr error

	enqueue := func(j job) {
		wg.Add(1)
		jobs <- j
	}

	process := func(j job) {
		defer wg.Done()

		dirEntries, err := os.ReadDir(j.absPath)
		if err != nil {
			mu.Lock()
			if walkErr == nil {
				walkErr = err
			}
			mu.Unlock()
			return
		}

		// Local ignore set, extended with this directory's own .gitignore if present.
		localIgnores := j.ignores
		if opt.RespectGitignore {
			gi := filepath.Join(j.absPath, ".gitignore")
			if _, statErr := os.Stat(gi); statErr == nil {
				localIgnores = j.ignores.Clone()
				_ = localIgnores.LoadGitignore(gi)
			}
		}

		sort.Slice(dirEntries, func(a, b int) bool {
			return dirEntries[a].Name() < dirEntries[b].Name()
		})

		var localResults []Entry

		for _, de := range dirEntries {
			name := de.Name()
			if !opt.ShowHidden && strings.HasPrefix(name, ".") {
				continue
			}

			relPath := name
			if j.relPath != "" {
				relPath = j.relPath + "/" + name
			}

			isDir := de.IsDir()

			if opt.RespectGitignore && localIgnores.Match(relPath, isDir) {
				continue
			}

			var size int64
			if !isDir {
				if info, err := de.Info(); err == nil {
					size = info.Size()
				}
			}

			include := (isDir && !opt.FilesOnly) || (!isDir && !opt.DirsOnly)
			if include {
				localResults = append(localResults, Entry{
					RelPath: relPath,
					IsDir:   isDir,
					Size:    size,
					Depth:   j.depth + 1,
				})
			}

			if isDir {
				withinDepth := opt.MaxDepth == 0 || j.depth+1 < opt.MaxDepth
				if withinDepth {
					enqueue(job{
						absPath: filepath.Join(j.absPath, name),
						relPath: relPath,
						depth:   j.depth + 1,
						ignores: localIgnores,
					})
				}
			}
		}

		if len(localResults) > 0 {
			mu.Lock()
			results = append(results, localResults...)
			mu.Unlock()
		}
	}

	for i := 0; i < numWorkers; i++ {
		workerWG.Add(1)
		go func() {
			defer workerWG.Done()
			for j := range jobs {
				process(j)
			}
		}()
	}

	enqueue(job{absPath: root, relPath: "", depth: 0, ignores: baseIgnores})

	// Closer goroutine: once all in-flight jobs finish, close the channel so workers exit.
	go func() {
		wg.Wait()
		close(jobs)
	}()

	workerWG.Wait()

	if walkErr != nil {
		return nil, walkErr
	}
	return results, nil
}
