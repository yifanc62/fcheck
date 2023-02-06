package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func GetSerializablePath(dirPath, filePath string) string {
	return filepath.ToSlash(strings.TrimPrefix(strings.TrimPrefix(filePath, dirPath), string(filepath.Separator)))
}

func GetFromSerializablePath(path string) string {
	return filepath.FromSlash(path)
}

func CheckPathNotExists(path string) (notExist bool, isDir bool, err error) {
	info, err := os.Stat(path)
	if err != nil {
		notExist = os.IsNotExist(err)
		if notExist {
			return true, true, nil
		}
		err = fmt.Errorf("failed to get info of path '%s': %w", path, err)
		return
	}
	return false, info.IsDir(), nil
}

func CheckDirectoryExists(path string) (exist bool, err error) {
	info, err := os.Stat(path)
	if err != nil {
		notExist := os.IsNotExist(err)
		if notExist {
			return false, nil
		}
		err = fmt.Errorf("failed to get info of path '%s': %w", path, err)
		return
	}
	return info.IsDir(), nil
}

func CheckFileExists(path string) (exist bool, err error) {
	info, err := os.Stat(path)
	if err != nil {
		notExist := os.IsNotExist(err)
		if notExist {
			return false, nil
		}
		err = fmt.Errorf("failed to get info of path '%s': %w", path, err)
		return
	}
	return !info.IsDir(), nil
}

func HashFileSHA1(path string) (result string, err error) {
	h := sha1.New()
	f, err := os.Open(path)
	if err != nil {
		err = fmt.Errorf("failed to open '%s': %w", path, err)
		return
	}
	defer func() {
		_ = f.Close()
	}()
	_, err = io.Copy(h, f)
	if err != nil {
		err = fmt.Errorf("failed to read '%s': %w", path, err)
		return
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func CompareFileSHA1(path, sha1Str string) (exist bool, match bool, err error) {
	if _, err = os.Stat(path); err != nil {
		notExist := os.IsNotExist(err)
		if notExist {
			return false, false, nil
		}
		err = fmt.Errorf("failed to get info of path '%s': %w", path, err)
		return
	}
	sha1Byte, err := hex.DecodeString(sha1Str)
	if err != nil {
		err = fmt.Errorf("SHA1 value '%s' of path '%s' is invalid", sha1Str, path)
		return
	}
	f, err := os.Open(path)
	if err != nil {
		err = fmt.Errorf("failed to open '%s': %w", path, err)
		return
	}
	defer omitError(f.Close)
	h := sha1.New()
	_, err = io.Copy(h, f)
	if err != nil {
		err = fmt.Errorf("failed to read '%s': %w", path, err)
		return
	}
	return true, bytes.Compare(h.Sum(nil), sha1Byte) == 0, nil
}

func CopyFileWithPath(dst, src string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file '%s' not exists", src)
		}
		return fmt.Errorf("failed to get info of path '%s': %w", src, err)
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("path '%s' is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open '%s': %w", src, err)
	}
	defer omitError(source.Close)

	dir := filepath.Dir(dst)
	err = CreateDir(dir)
	if err != nil {
		return fmt.Errorf("failed to create directory '%s': %w", dir, err)
	}
	destination, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create file '%s': %w", dst, err)
	}
	defer omitError(destination.Close)

	_, err = io.Copy(destination, source)
	if err != nil {
		return fmt.Errorf("failed to copy file '%s' to '%s': %w", src, dst, err)
	}
	return nil
}

func CreateDir(path string) error {
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(path, os.ModePerm)
			if err != nil {
				return fmt.Errorf("failed to mkdir '%s': %w", path, err)
			}
			return nil
		}
		return fmt.Errorf("failed to get info of path '%s': %w", path, err)
	}
	if stat.IsDir() {
		return nil
	}
	return fmt.Errorf("path '%s' is not a directory", path)
}

func omitError(f func() error) {
	_ = f()
}
