package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

type entry struct {
	Title    string `json:"title,omitempty"`
	TypeTag  string `json:"type_tag,omitempty"`
	PageData struct {
		Part string `json:"part,omitempty"`
	} `json:"page_data,omitempty"`
}

var (
	input  string
	output string
	ffmpeg string

	once sync.Once
	wg   sync.WaitGroup
)

func init() {
	flag.StringVar(&input, "input", "", "input dir")
	flag.StringVar(&output, "output", "output", "output dir")
	flag.StringVar(&ffmpeg, "ffmpeg", "ffmpeg", "ffmpeg bin file")
}

func main() {
	flag.Parse()
	if len(input) == 0 {
		fmt.Fprintln(os.Stderr, "input must not be empty")
		return
	}
	paths, err := filepath.Glob(input + "/*")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	wg.Add(len(paths))
	for _, path := range paths {
		go func(path string) {
			defer wg.Done()
			entry, err := getEntry(path)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}
			if err := execFFmpeg(path, entry); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}
		}(path)
	}
	wg.Wait()
	fmt.Fprintln(os.Stdout, "all success")
}

func getEntry(path string) (*entry, error) {
	var entry *entry
	f, err := os.Open(filepath.Join(path, "entry.json"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	bs, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(bs, &entry); err != nil {
		return nil, err
	}
	return entry, nil
}

func isExist(name string) bool {
	_, err := os.Stat(name)
	if err == nil {
		return true
	}
	return os.IsExist(err)
}

func execFFmpeg(path string, entry *entry) error {
	audio := filepath.Join(path, entry.TypeTag, "audio.m4s")
	video := filepath.Join(path, entry.TypeTag, "video.m4s")
	outputDir := filepath.Join(output, entry.Title)
	once.Do(func() {
		if !isExist(outputDir) {
			_ = os.Mkdir(outputDir, os.ModeDir)
		}
	})
	cmd := exec.Command(ffmpeg,
		"-y", // overwrite yes
		"-i", audio,
		"-i", video,
		"-codec", "copy",
		filepath.Join(outputDir, entry.PageData.Part+".mp4"),
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if _, err := cmd.Output(); err != nil {
		return fmt.Errorf("stderr: %s, err: %w", stderr.String(), err)
	}
	return nil
}
