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
	defaultFileListJsonName = "f.json"
	defaultFileDiffJsonName = "diff.json"
	defaultPackDirName      = "diff-package"

	maxLabelLength = len(redundantLabel)
	redundantLabel = "REDUNDANT"
	notFoundLabel  = "NOT FOUND"
	mismatchLabel  = "MISMATCH"
	passLabel      = "PASS"
)

var (
	WorkDirectory = flag.String("d", "", "Directory Path")
	OutputPath    = flag.String("o", "", "Output path")
	InputPath     = flag.String("i", "", "Input json file path")
	Overwrite     = flag.Bool("y", false, "Confirm overwriting")
	Generate      = flag.Bool("g", false, "Generate a file list json")
	Pack          = flag.Bool("p", false, "Pack files from a file diff json")
	ShowVersion   = flag.Bool("v", false, "Show version")
	ShowUsage     = flag.Bool("h", false, "Show usage")

	Tag        = "undefined"
	BuildTime  = "undefined"
	CommitHash = "undefined"
)

func main() {
	defer func() {
		err := recover()
		if err != nil {
			fmt.Printf("panic: %v\n", err)
		}
	}()
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
		fmt.Printf("Arguments -g and -p are conflict.\n\n")
		flag.Usage()
		return
	}
	if len(*WorkDirectory) == 0 {
		dir, err := os.Getwd()
		if err != nil {
			panic("failed to get working directory: " + err.Error())
		}
		*WorkDirectory = dir
	}

	pathPtr := InputPath
	if *Generate {
		pathPtr = OutputPath
	}

	filePath := defaultFileListJsonName
	if *Pack {
		filePath = defaultFileDiffJsonName
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

	diffFilePath := defaultFileDiffJsonName
	if !*Generate && !*Pack {
		if len(*OutputPath) > 0 {
			diffFilePath = *OutputPath
		}
	}

	packDirPath := defaultPackDirName
	if *Pack {
		if len(*OutputPath) > 0 {
			packDirPath = *OutputPath
		}
	}

	if exist, err := CheckDirectoryExists(*WorkDirectory); err != nil {
		panic("failed to check directory '" + *WorkDirectory + "' existence: " + err.Error())
	} else if !exist {
		panic("source directory path '" + filePath + "' not exists")
	}

	if *Generate {
		if notExist, isDir, err := CheckPathNotExists(filePath); err != nil {
			panic("failed to check path '" + filePath + "' existence: " + err.Error())
		} else if !notExist {
			if isDir {
				panic("output file path '" + filePath + "' is a directory")
			}
			if !*Overwrite {
				panic("output file path '" + filePath + "' exists, specify -y to overwrite")
			}
			fmt.Printf("Warning: Output file path '%s' exists, will be overwritten.\n", filePath)
			err := os.Remove(filePath)
			if err != nil {
				panic("failed to delete file '" + filePath + "': " + err.Error())
			}
		}
		generate(*WorkDirectory, filePath)
		return
	}

	if exist, err := CheckFileExists(filePath); err != nil {
		panic("failed to check file '" + filePath + "' existence: " + err.Error())
	} else if !exist {
		panic("input file '" + filePath + "' not exists")
	}

	if *Pack {
		if notExist, isDir, err := CheckPathNotExists(packDirPath); err != nil {
			panic("failed to check path '" + packDirPath + "' existence: " + err.Error())
		} else if !notExist {
			if !isDir {
				panic("output path '" + packDirPath + "' is not a directory")
			}
			if !*Overwrite {
				panic("output directory path '" + packDirPath + "' exists, specify -y to delete and repack")
			}
			fmt.Printf("Warning: Output directory path '%s' exists, will be deleted and repack.\n", packDirPath)
			err := os.RemoveAll(packDirPath)
			if err != nil {
				panic("failed to delete directory '" + packDirPath + "': " + err.Error())
			}
		}
		pack(*WorkDirectory, filePath, packDirPath)
		return
	}

	if notExist, isDir, err := CheckPathNotExists(diffFilePath); err != nil {
		panic("failed to check path '" + diffFilePath + "' existence: " + err.Error())
	} else if !notExist {
		if isDir {
			panic("output file path '" + diffFilePath + "' is a directory")
		}
		if !*Overwrite {
			panic("output file path '" + diffFilePath + "' exists, specify -y to overwrite")
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
		sha1Str, err := HashFileSHA1(filePath)
		if err != nil {
			return err
		}
		result.Files = append(result.Files, &FCheckFile{
			Path: GetSerializablePath(dirPath, filePath),
			SHA1: sha1Str,
			Size: info.Size(),
		})
		return nil
	})
	if err != nil {
		panic("failed to walk through path '" + dirPath + "': " + err.Error())
	}

	var outBytes []byte
	outBytes, err = json.MarshalIndent(result, "", "\t")
	if err != nil {
		panic("json marshal error: " + err.Error())
	}
	err = os.WriteFile(outputFilePath, outBytes, os.ModePerm)
	if err != nil {
		panic("failed to save result to '" + outputFilePath + "': " + err.Error())
	}

	fmt.Printf("Generate successfully, result has been saved to '%s'.\n", outputFilePath)
}

func pack(dirPath, filePath, outputDirPath string) {
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		panic("failed to read file '" + filePath + "': " + err.Error())
	}

	var diffList FCheckDiffList
	err = json.Unmarshal(fileData, &diffList)
	if err != nil {
		panic("file '" + filePath + "' content is invalid: " + err.Error())
	}

	fmt.Printf("File diff json was generated at: %s.\n", time.Unix(diffList.Time, 0).Format("2006-01-02 15:04:05"))

	// packDir can not be the parent of sourceDir, or the sourceDir would be deleted
	srcAbsPath, err := filepath.Abs(dirPath)
	if err != nil {
		panic("failed to get absolute path of '" + dirPath + "': " + err.Error())
	}
	dstAbsPath, err := filepath.Abs(outputDirPath)
	if err != nil {
		panic("failed to get absolute path of '" + outputDirPath + "': " + err.Error())
	}
	if strings.HasPrefix(srcAbsPath, dstAbsPath) {
		panic("output path '" + outputDirPath + "' is the parent of source directory path '" + dirPath + "'")
	}

	for _, path := range diffList.Mismatching {
		err = CopyFileWithPath(filepath.Join(outputDirPath, path), filepath.Join(dirPath, path))
		if err != nil {
			panic("failed to copy mismatching file '" + path + "': " + err.Error())
		}
	}
	for _, path := range diffList.Missing {
		err = CopyFileWithPath(filepath.Join(outputDirPath, path), filepath.Join(dirPath, path))
		if err != nil {
			panic("failed to copy missing file '" + path + "': " + err.Error())
		}
	}
	if len(diffList.Redundant) > 0 {
		fileName := "remove_" + strconv.FormatInt(diffList.Time, 10) + ".bat"
		var sb strings.Builder
		sb.WriteString("@echo off\r\n")
		for _, path := range diffList.Redundant {
			sb.WriteString(fmt.Sprintf("del /f \"%s\"\r\n", GetFromSerializablePath(path)))
		}
		writeFilePath := filepath.Join(outputDirPath, fileName)
		if notExist, _, err := CheckPathNotExists(writeFilePath); err != nil {
			panic("failed to check path '" + writeFilePath + "' existence: " + err.Error())
		} else if !notExist {
			panic("batch file '" + fileName + "' to be generated exists")
		}
		err = os.WriteFile(writeFilePath, []byte(sb.String()), os.ModePerm)
		if err != nil {
			panic("failed to write batch file '" + fileName + "': " + err.Error())
		}
	}

	fmt.Printf("Pack successfully, package has been saved to '%s'.\n", outputDirPath)
}

func check(dirPath, filePath, diffFilePath string) {
	filePathInfo, err := os.Stat(filePath)
	if err != nil {
		panic("failed to get info of file '" + filePath + "': " + err.Error())
	}
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		panic("failed to read file '" + filePath + "': " + err.Error())
	}

	var checkList FCheckList
	err = json.Unmarshal(fileData, &checkList)
	if err != nil {
		panic("file '" + filePath + "' content is invalid: " + err.Error())
	}

	fmt.Printf("File list json was generated at: %s.\n", time.Unix(checkList.Time, 0).Format("2006-01-02 15:04:05"))

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
		// skip for the source json file
		if os.SameFile(filePathInfo, info) {
			return nil
		}
		path := GetSerializablePath(dirPath, filePath)
		file := fileMap[path]
		if file == nil {
			redundantFiles[path] = true
			PrintStatus(false, redundantLabel, path)
			return nil
		}
		delete(notCheckedFileMap, path)
		exist, match, err := CompareFileSHA1(filePath, file.SHA1)
		if err != nil {
			return err
		}
		if !exist {
			missingFiles[path] = true
			PrintStatus(false, notFoundLabel, path)
			return nil
		}
		if !match || file.Size != info.Size() {
			mismatchingFiles[path] = true
			PrintStatus(false, mismatchLabel, path)
			return nil
		}
		passCount++
		PrintStatus(true, passLabel, path)
		return nil
	})
	for path := range notCheckedFileMap {
		missingFiles[path] = true
		PrintStatus(false, notFoundLabel, path)
	}
	if err != nil {
		panic("failed to walk through path '" + dirPath + "': " + err.Error())
	}
	if len(missingFiles)+len(mismatchingFiles)+len(redundantFiles) == 0 {
		if passCount > 1 {
			fmt.Printf("All %d files match.\n", passCount)
		} else {
			fmt.Printf("All %d file matches.\n", passCount)
		}
		return
	}
	fmt.Printf("\nPassed: %d\nMismatched: %d\nNot found: %d\nRedundant: %d\n", passCount, len(mismatchingFiles), len(missingFiles), len(redundantFiles))

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
		panic("json marshal error: " + err.Error())
	}
	err = os.WriteFile(diffFilePath, outBytes, os.ModePerm)
	if err != nil {
		panic("failed to save result to '" + diffFilePath + "': " + err.Error())
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
