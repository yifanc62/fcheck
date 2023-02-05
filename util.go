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

func PathNotExist(path string) (notExist bool, isDir bool) {
	info, err := os.Stat(path)
	if err != nil {
		notExist := os.IsNotExist(err)
		if notExist {
			return true, true
		}
		panic("check path '" + path + "' failed: " + err.Error())
	}
	return false, info.IsDir()
}

func DirectoryExist(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		notExist := os.IsNotExist(err)
		if notExist {
			return false
		}
		panic("check path '" + path + "' failed: " + err.Error())
	}
	return info.IsDir()
}

func FileExist(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		notExist := os.IsNotExist(err)
		if notExist {
			return false
		}
		panic("check path '" + path + "' failed: " + err.Error())
	}
	return !info.IsDir()
}

func HashFileSHA1(path string) string {
	h := sha1.New()
	f, err := os.Open(path)
	if err != nil {
		panic("open path '" + path + "' failed: " + err.Error())
	}
	defer func() {
		_ = f.Close()
	}()
	_, err = io.Copy(h, f)
	if err != nil {
		panic("read path '" + path + "' failed: " + err.Error())
	}
	return hex.EncodeToString(h.Sum(nil))
}

func CompareFileSHA1(path, sha1Str string) (exist bool, match bool) {
	if _, err := os.Stat(path); err != nil {
		notExist := os.IsNotExist(err)
		if notExist {
			return false, false
		}
		panic("check path '" + path + "' failed: " + err.Error())
	}
	sha1Byte, err := hex.DecodeString(sha1Str)
	if err != nil {
		panic("path '" + path + "' SHA1 value '" + sha1Str + "' is invalid")
	}
	f, err := os.Open(path)
	if err != nil {
		panic("open path '" + path + "' failed: " + err.Error())
	}
	defer func() {
		_ = f.Close()
	}()
	h := sha1.New()
	_, err = io.Copy(h, f)
	if err != nil {
		panic("read path '" + path + "' failed: " + err.Error())
	}
	if bytes.Compare(h.Sum(nil), sha1Byte) == 0 {
		return true, true
	}
	return true, false
}

func CopyFile(dst, src string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file '%s' not exists", src)
		}
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("file '%s' is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		_ = source.Close()
	}()

	err = CreateDir(filepath.Dir(dst))
	if err != nil {
		return err
	}
	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		_ = destination.Close()
	}()

	_, err = io.Copy(destination, source)
	return err
}

func CreateDir(path string) error {
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(path, os.ModePerm)
		}
		return err
	}
	if stat.IsDir() {
		return nil
	}
	return fmt.Errorf("path '%s' is not directory", path)
}
