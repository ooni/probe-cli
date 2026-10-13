package main

//
// unpack-maven: Maven's distribution zip (already checked against Signal-Server's pinned SHA-256),
// unpacked so that its single top directory becomes DIR, with bin/* executable.
//

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func unpackMaven(zipPath, dir string) error {
	z, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer z.Close()
	if len(z.File) == 0 {
		return fmt.Errorf("%s is empty", zipPath)
	}
	top, _, _ := strings.Cut(z.File[0].Name, "/")
	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("%s exists already", dir)
	}
	tmp := dir + ".tmp"
	if err := os.RemoveAll(tmp); err != nil {
		return err
	}
	defer os.RemoveAll(tmp) // left only by a failure
	for _, f := range z.File {
		if !filepath.IsLocal(f.Name) || !strings.HasPrefix(f.Name, top+"/") {
			return fmt.Errorf("%s: %q is not under %s/", zipPath, f.Name, top)
		}
		if err := unpackFile(f, filepath.Join(tmp, f.Name)); err != nil {
			return err
		}
	}
	if err := os.Rename(filepath.Join(tmp, top), dir); err != nil {
		return err
	}
	bin, err := os.ReadDir(filepath.Join(dir, "bin"))
	if err != nil {
		return err
	}
	for _, e := range bin {
		if err := os.Chmod(filepath.Join(dir, "bin", e.Name()), 0o755); err != nil { // #nosec G302 -- Maven's scripts
			return err
		}
	}
	return nil
}

func unpackFile(f *zip.File, path string) error {
	if f.FileInfo().IsDir() {
		return os.MkdirAll(path, 0o750)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	r, err := f.Open()
	if err != nil {
		return err
	}
	defer r.Close()
	w, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644) // #nosec G304 G302 -- checked local above
	if err != nil {
		return err
	}
	if _, err := io.Copy(w, r); err != nil { // #nosec G110 -- a distribution checked against its pinned SHA-256
		w.Close()
		return err
	}
	return w.Close()
}
