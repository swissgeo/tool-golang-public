package completioncache

import (
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const defaultLocation = "/.cache/"
const defaultFileName = "completions"

// CompletionCache reads and writes command completions using a file in "~/.cache/<command-name>/completions".
type CompletionCache interface {
	Write(key string, completions []cobra.Completion, compDir cobra.ShellCompDirective) error
	Read(key string) ([]cobra.Completion, cobra.ShellCompDirective, error)
	Delete() error
}

// NewCache creates a new cacher.
func NewCache(cmdName string, validDuration time.Duration) (CompletionCache, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	filePath, err := filepath.Abs(home + defaultLocation + cmdName)
	if err != nil {
		return nil, err
	}
	err = os.MkdirAll(filePath, os.ModePerm)
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filePath+"/"+defaultFileName, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dec := gob.NewDecoder(f)
	c := make(map[string]content)
	err = dec.Decode(&c)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	return &fileCache{
		filePath:      filePath + "/" + defaultFileName,
		validDuration: validDuration,
		cache:         c,
	}, nil
}

type fileCache struct {
	filePath      string
	validDuration time.Duration
	cache         map[string]content

	deleted bool
}

var (
	ErrNotFound     = errors.New("no cache entry found")
	ErrCacheDeleted = errors.New("cache has been deleted")
)

// Read completion from cache. If content is not found or expired, ErrNotFound is returned.
func (c *fileCache) Read(key string) ([]cobra.Completion, cobra.ShellCompDirective, error) {
	if c.deleted {
		return nil, cobra.ShellCompDirectiveError, ErrCacheDeleted
	}
	content, ok := c.cache[key]
	if !ok || c.contentExpired(content) {
		return nil, cobra.ShellCompDirectiveError, ErrNotFound
	}
	return content.Completions, content.CompDir, nil
}

// Write completion completion to cache. key should be unique for command.
func (c *fileCache) Write(key string, completions []cobra.Completion, compDir cobra.ShellCompDirective) error {
	if c.deleted {
		return ErrCacheDeleted
	}
	f, err := os.OpenFile(c.filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer f.Close()

	c.cache[key] = content{
		CreatedAt:   time.Now(),
		Completions: completions,
		CompDir:     compDir,
	}

	enc := gob.NewEncoder(f)
	err = enc.Encode(c.cache)
	if err != nil {
		return err
	}
	return nil
}

type content struct {
	CreatedAt   time.Time
	Completions []cobra.Completion
	CompDir     cobra.ShellCompDirective
}

func (c fileCache) contentExpired(v content) bool {
	return v.CreatedAt.Add(c.validDuration).Before(time.Now())
}

// Delete removes the cache file
func (c *fileCache) Delete() error {
	cacheFolder, _ := strings.CutSuffix(c.filePath, "/"+defaultFileName)
	fmt.Println(cacheFolder)
	err := os.RemoveAll(cacheFolder)
	if err != nil {
		return err
	}
	c.deleted = true
	return nil
}
