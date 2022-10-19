package controllers

import (
	"os"
	"path/filepath"
)

func FilePathFromHome(path string) string {
	return os.Getenv("HOME") + "/" + path
}
func CreateFileFromBytes(path string, buffer []byte) error {
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, 0666)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	f.Chmod(0600)
	_, err = f.Write(buffer)
	if err != nil {
		return err
	}
	return nil
}

func DeleteFile(path string) error {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return err
	}
	if fileInfo.IsDir() {
		return os.RemoveAll(path)
	}
	return os.Remove(path)
}
