package main

import (
	"embed"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

//go:embed swagger_templates/*
var templatesFS embed.FS

func writeTemplates(destinationRoot string) error {
	err := os.RemoveAll(destinationRoot)
	if err != nil && !os.IsNotExist(err) {
		return errors.Wrapf(err, "cannot remove directory %s", destinationRoot)
	}

	walkDirFn := func(path string, d os.DirEntry, err error) (e error) {
		if path == "." {
			return nil
		}

		if d.IsDir() {
			err := os.MkdirAll(filepath.Join(destinationRoot, path), 0777)
			if err != nil && !os.IsExist(err) {
				return err
			}
		} else {
			src, err := templatesFS.Open(path)
			if err != nil {
				return errors.Wrapf(err, "Cannot open embeded file %s for reading", path)
			}
			dstFileName := filepath.Join(destinationRoot, path)
			dst, err := os.Create(dstFileName)
			if err != nil {
				return errors.Wrapf(err, "Cannot open %s for writing contents of %s",
					dstFileName, path)
			}
			_, err = io.Copy(dst, src)
			if err != nil {
				return errors.Wrapf(err, "Cannot copy %s contents to %s",
					path, dstFileName)
			}

			if err := dst.Close(); err != nil {
				return errors.Wrapf(err, "error closing file %s", dstFileName)
			}
			_ = src.Close()
		}
		return nil
	}
	if err := fs.WalkDir(templatesFS, ".", walkDirFn); err != nil {
		return err
	}

	return nil
}
