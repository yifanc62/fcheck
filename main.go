package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	defaultFileListName = "f.json"
	defaultDiffListName = "diff.json"
	defaultPackDirName  = "diff-package"
)

var (
	WorkDirectory  = flag.String("d", "", "Directory Path")
	OutputFilePath = flag.String("o", "", "Output file list json path")
	InputFilePath  = flag.String("i", "", "Input file list json path")
	Overwrite      = flag.Bool("y", false, "Confirm overwritten")
	Generate       = flag.Bool("g", false, "Generate file list json")
	Pack           = flag.Bool("p", false, "Pack file from file list json")
	ShowVersion    = flag.Bool("v", false, "Show version")
	ShowUsage      = flag.Bool("h", false, "Show usage")

	Tag        = "undefined"
	BuildTime  = "undefined"
	CommitHash = "undefined"
)

func main() {
	flag.Usage = showUsage
	flag.Parse()
	if *ShowUsage {
		flag.Usage()
		return
	}
	if *ShowVersion {
		showVersion()
		return
	}
	if *Generate && *Pack {
		fmt.Printf("Argument -g and -p is conflict.\n\n")
		flag.Usage()
		return
	}
	if len(*WorkDirectory) == 0 {
		dir, err := os.Getwd()
		if err != nil {
			panic("get work directory failed: " + err.Error())
		}
		*WorkDirectory = dir
	}

	pathPtr := InputFilePath
	if *Generate {
		pathPtr = OutputFilePath
	}

	filePath := defaultFileListName
	if *Pack {
		filePath = defaultDiffListName
	}
	if len(*pathPtr) == 0 {
		if len(flag.Args()) > 1 {
			fmt.Printf("Invalid arguments.\n\n")
			flag.Usage()
			return
		}
		if len(flag.Args()) > 0 {
			filePath = flag.Arg(0)
		}
	} else {
		filePath = *pathPtr
	}

	diffFilePath := defaultDiffListName
	if !*Generate && !*Pack {
		if len(*OutputFilePath) > 0 {
			diffFilePath = *OutputFilePath
		}
	}

	packDirPath := defaultPackDirName
	if *Pack {
		if len(*OutputFilePath) > 0 {
			packDirPath = *OutputFilePath
		}
	}

	if !DirectoryExist(*WorkDirectory) {
		panic("source directory path '" + filePath + "' not exists")
	}

	if *Generate {
		if notExist, isDir := PathNotExist(filePath); !notExist {
			if isDir {
				panic("output file path '" + filePath + "' is a directory")
			}
			if !*Overwrite {
				panic("output file path '" + filePath + "' exists, use -y if you want to overwrite it")
			}
			fmt.Printf("Warning: Output file path '%s' exists, will be overwritten.\n", filePath)
			err := os.Remove(filePath)
			if err != nil {
				panic("delete file '" + filePath + "' failed: " + err.Error())
			}
		}
		generate(*WorkDirectory, filePath)
		return
	}

	if !FileExist(filePath) {
		panic("input file '" + filePath + "' not exist")
	}

	if *Pack {
		if notExist, isDir := PathNotExist(packDirPath); !notExist {
			if !isDir {
				panic("pack path '" + packDirPath + "' is not a directory")
			}
			if !*Overwrite {
				panic("pack path '" + packDirPath + "' exists, use -y if you want to delete it and repack")
			}
			fmt.Printf("Warning: Pack path '%s' exists, will be deleted and repack.\n", packDirPath)
			err := os.RemoveAll(packDirPath)
			if err != nil {
				panic("delete directory '" + packDirPath + "' failed: " + err.Error())
			}
		}
		pack(*WorkDirectory, filePath, packDirPath)
		return
	}

	if notExist, isDir := PathNotExist(diffFilePath); !notExist {
		if isDir {
			panic("output file path '" + diffFilePath + "' is a directory")
		}
		if !*Overwrite {
			panic("output file path '" + diffFilePath + "' exists, use -y if you want to overwrite it")
		}
		fmt.Printf("Warning: Output file path '%s' exists, will be overwritten.\n", diffFilePath)
	}
	check(*WorkDirectory, filePath, diffFilePath)
}

func generate(dirPath, outputFilePath string) {
	result := &FCheckList{Time: time.Now().Unix()}
	err := filepath.Walk(dirPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		result.Files = append(result.Files, &FCheckFile{
			Path: GetSerializablePath(dirPath, filePath),
			SHA1: HashFileSHA1(filePath),
			Size: info.Size(),
		})
		return nil
	})
	if err != nil {
		panic("walk through path '" + dirPath + "' failed: " + err.Error())
	}

	var outBytes []byte
	outBytes, err = json.MarshalIndent(result, "", "\t")
	if err != nil {
		panic("inner json marshal error: " + err.Error())
	}
	err = os.WriteFile(outputFilePath, outBytes, os.ModePerm)
	if err != nil {
		panic("save result file '" + outputFilePath + "' failed: " + err.Error())
	}

	fmt.Printf("Generate success, save result to '%s'.", outputFilePath)
}

func pack(dirPath, filePath, outputDirPath string) {
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		panic("open file '" + filePath + "' failed: " + err.Error())
	}

	var diffList FCheckDiffList
	err = json.Unmarshal(fileData, &diffList)
	if err != nil {
		panic("file '" + filePath + "' is invalid: " + err.Error())
	}

	fmt.Printf("File diff list generate at: %s.\n", time.Unix(diffList.Time, 0).Format("2006-01-02 15:04:05"))

	// packDir can not be the parent of sourceDir, or the sourceDir could be deleted
	srcAbsPath, err := filepath.Abs(dirPath)
	if err != nil {
		panic("deal with path '" + dirPath + "' failed: " + err.Error())
	}
	dstAbsPath, err := filepath.Abs(outputDirPath)
	if err != nil {
		panic("deal with path '" + outputDirPath + "' failed: " + err.Error())
	}
	if strings.HasPrefix(srcAbsPath, dstAbsPath) {
		panic("output path '" + outputDirPath + "' is parent of dir path '" + dirPath + "'")
	}

	for _, path := range diffList.Mismatching {
		err = CopyFile(filepath.Join(outputDirPath, path), filepath.Join(dirPath, path))
		if err != nil {
			panic("file '" + path + "' copy failed: " + err.Error())
		}
	}
	for _, path := range diffList.Missing {
		err = CopyFile(filepath.Join(outputDirPath, path), filepath.Join(dirPath, path))
		if err != nil {
			panic("file '" + path + "' copy failed: " + err.Error())
		}
	}
	if len(diffList.Redundant) > 0 {
		fileName := "remove_" + strconv.FormatInt(diffList.Time, 10) + ".bat"
		var sb strings.Builder
		sb.WriteString("@echo off\r\n")
		for _, path := range diffList.Redundant {
			sb.WriteString(fmt.Sprintf("del /f \"%s\"\r\n", filepath.FromSlash(path)))
		}
		writeFilePath := filepath.Join(outputDirPath, fileName)
		if notExist, _ := PathNotExist(writeFilePath); !notExist {
			panic("file '" + fileName + "' exists")
		}
		err = os.WriteFile(writeFilePath, []byte(sb.String()), os.ModePerm)
		if err != nil {
			panic("file '" + fileName + "' create failed: " + err.Error())
		}
	}

	fmt.Printf("Pack success, save package to '%s'.", outputDirPath)
}

func check(dirPath, filePath, diffFilePath string) {
	filePathInfo, err := os.Stat(filePath)
	if err != nil {
		panic("open file '" + filePath + "' failed: " + err.Error())
	}
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		panic("open file '" + filePath + "' failed: " + err.Error())
	}

	var checkList FCheckList
	err = json.Unmarshal(fileData, &checkList)
	if err != nil {
		panic("file '" + filePath + "' is invalid: " + err.Error())
	}

	fmt.Printf("File list generate at: %s.\n", time.Unix(checkList.Time, 0).Format("2006-01-02 15:04:05"))

	fileMap := make(map[string]*FCheckFile)
	notCheckedFileMap := make(map[string]*FCheckFile)
	for _, f := range checkList.Files {
		fileMap[f.Path] = f
		notCheckedFileMap[f.Path] = f
	}

	mismatchingFiles, missingFiles, redundantFiles := make(map[string]bool), make(map[string]bool), make(map[string]bool)
	var passCount int
	err = filepath.Walk(dirPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if os.SameFile(filePathInfo, info) {
			return nil
		}
		path := GetSerializablePath(dirPath, filePath)
		file := fileMap[path]
		if file == nil {
			redundantFiles[path] = true
			PrintStatus(false, "REDUNDANT", path)
			return nil
		}
		delete(notCheckedFileMap, path)
		exist, match := CompareFileSHA1(filePath, file.SHA1)
		if !exist {
			missingFiles[path] = true
			PrintStatus(false, "NOT FOUND", path)
			return nil
		}
		if !match || file.Size != info.Size() {
			mismatchingFiles[path] = true
			PrintStatus(false, "MISMATCH", path)
			return nil
		}
		passCount++
		PrintStatus(true, "PASS", path)
		return nil
	})
	for path := range notCheckedFileMap {
		missingFiles[path] = true
		PrintStatus(false, "NOT FOUND", path)
	}
	if err != nil {
		panic("walk through path '" + dirPath + "' failed: " + err.Error())
	}
	if len(missingFiles)+len(mismatchingFiles)+len(redundantFiles) == 0 {
		fmt.Printf("All %d files matched.\n", passCount)
		return
	}
	fmt.Printf("Passed: %d\nMismatched: %d\nNot found: %d\nRedundant: %d\n", passCount, len(mismatchingFiles), len(missingFiles), len(redundantFiles))

	result := &FCheckDiffList{Time: time.Now().Unix()}
	for mismatchingFile := range mismatchingFiles {
		result.Mismatching = append(result.Mismatching, mismatchingFile)
	}
	for missingFile := range missingFiles {
		result.Missing = append(result.Missing, missingFile)
	}
	for redundantFile := range redundantFiles {
		result.Redundant = append(result.Redundant, redundantFile)
	}

	var outBytes []byte
	outBytes, err = json.MarshalIndent(result, "", "\t")
	if err != nil {
		panic("inner json marshal error: " + err.Error())
	}
	err = os.WriteFile(diffFilePath, outBytes, os.ModePerm)
	if err != nil {
		panic("save result file '" + diffFilePath + "' failed: " + err.Error())
	}
}

func showVersion() {
	fmt.Printf("fcheck %s(%s)\n", Tag, CommitHash)
	fmt.Printf("build: %s\n", BuildTime)
}

func showUsage() {
	showVersion()
	fmt.Println()
	fmt.Printf("Usage(check): fcheck [-d dirPath] [-o output.json] [-i] input.json\n")
	fmt.Printf("Usage(generate): fcheck -g [-d dirPath] [-o] output.json\n")
	fmt.Printf("Usage(pack): fcheck -p [-d dirPath] [-o outputDir] [-i] diff.json\n")
}
