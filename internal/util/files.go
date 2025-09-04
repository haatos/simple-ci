package util

import (
	"archive/zip"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
)

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	return false, err
}

func ArchiveDirectory(dirPath string) (string, error) {
	_, folderName := path.Split(dirPath)
	archive, err := os.Create(path.Join("artifacts", folderName+".zip"))
	if err != nil {
		return "", err
	}
	defer archive.Close()

	paths := make([]string, 0)
	filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			paths = append(paths, path)
		}
		return nil
	})

	zw := zip.NewWriter(archive)
	for _, p := range paths {
		if err := copyToArchive(zw, p); err != nil {
			return "", nil
		}
	}

	if err := zw.Close(); err != nil {
		return "", err
	}

	return archive.Name(), nil
}

func copyToArchive(zw *zip.Writer, p string) error {
	// open file to archive
	f, err := os.Open(p)
	if err != nil {
		return err
	}
	defer f.Close()

	// open file in archive
	zf, err := zw.Create(p)
	if err != nil {
		return err
	}

	// copy file to archive
	if _, err := io.Copy(zf, f); err != nil {
		return err
	}
	return nil
}
