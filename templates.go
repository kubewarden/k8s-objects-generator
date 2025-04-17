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

	walkDirFn := func(path string, d os.DirEntry, walkErr error) error {
		if path == "." {
			return nil
		}

		if d.IsDir() {
			err2 := os.MkdirAll(filepath.Join(destinationRoot, path), 0o777)
			if err2 != nil && !os.IsExist(err2) {
				return err2
			}
			return nil
		}
		src, err := templatesFS.Open(path)
		if err != nil {
			return errors.Wrapf(err, "Cannot open embedded file %s for reading", path)
		}
		defer src.Close()

		dstFileName := filepath.Join(destinationRoot, path)
		dst, err := os.Create(dstFileName)
		if err != nil {
			return errors.Wrapf(err, "Cannot open %s for writing contents of %s",
				dstFileName, path)
		}
		defer dst.Close()

		_, err = io.Copy(dst, src)
		if err != nil {
			return errors.Wrapf(err, "Cannot copy %s contents to %s",
				path, dstFileName)
		}

		return nil
	}
	if err := fs.WalkDir(templatesFS, ".", walkDirFn); err != nil {
		return err
	}

	return nil
}
