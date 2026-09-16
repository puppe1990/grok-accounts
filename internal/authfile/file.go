package authfile

import (
	"os"
	"path/filepath"
)

func Read(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func WriteAtomic(path string, raw []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".auth.json.tmp-*")
	if err != nil {
		return err
	}
	name := tmp.Name()

	err = writeContents(tmp, raw)
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(name, path)
	}
	if err != nil {
		_ = os.Remove(name)
	}
	return err
}

func writeContents(f *os.File, raw []byte) error {
	if err := f.Chmod(0o600); err != nil {
		return err
	}
	if _, err := f.Write(raw); err != nil {
		return err
	}
	return f.Sync()
}
