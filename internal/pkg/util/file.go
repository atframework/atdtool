package util

import (
	"bufio"
	"os"
	"path/filepath"
)

// PathExist check input path is exist
func PathExist(file string) bool {
	if file == "" {
		return false
	}
	info, err := os.Stat(file)
	return err == nil && info.IsDir()
}

// FileExist check input file is exist
func FileExist(file string) bool {
	info, err := os.Stat(file)
	return err == nil && !info.IsDir()
}

// GetLines get lines from file
func GetLines(file string) ([]string, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	var lines []string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

// WriteFile writes data to file
func WriteFile(b []byte, name string) (err error) {
	path := filepath.Dir(name)
	if !PathExist(path) {
		err = os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return
		}
	}
	return os.WriteFile(name, b, os.ModePerm)
}
