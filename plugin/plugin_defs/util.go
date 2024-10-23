// Copyright 2020 the Drone Authors. All rights reserved.
// Use of this source code is governed by the Blue Oak Model License
// that can be found in the LICENSE file.

package plugin_defs

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

func writeCard(path, schema string, card interface{}) {
	data, _ := json.Marshal(map[string]interface{}{
		"schema": schema,
		"data":   card,
	})
	switch {
	case path == "/dev/stdout":
		writeCardTo(os.Stdout, data)
	case path == "/dev/stderr":
		writeCardTo(os.Stderr, data)
	case path != "":
		ioutil.WriteFile(path, data, 0644)
	}
}

func writeCardTo(out io.Writer, data []byte) {
	encoded := base64.StdEncoding.EncodeToString(data)
	io.WriteString(out, "\u001B]1338;")
	io.WriteString(out, encoded)
	io.WriteString(out, "\u001B]0m")
	io.WriteString(out, "\n")
}

func GetNewError(s string) error {
	return errors.New(s)
}

func LogPrintln(p Plugin, args ...interface{}) {
	//
	//if !IsDevTestingMode() {
	//	return
	//}
	//
	//if p != nil {
	//	if p.IsQuiet() {
	//		return
	//	}
	//}

	logrus.Println(append([]interface{}{"Plugin Info:"}, args...)...)
}

func LogPrintf(p Plugin, format string, v ...interface{}) {

	//if !IsDevTestingMode() {
	//	return
	//}
	//
	//if p != nil {
	//	if p.IsQuiet() {
	//		return
	//	}
	//}
	logrus.Printf(format, v...)
}

func IsDirExists(dir string) (bool, error) {
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return false, err
	}
	return info.IsDir(), nil
}

func GetAllJacocoExecFilesFromGlobPattern(rootDir, globPatterns string) ([]PathWithPrefix, error) {

	var execFilesPathWithPrefixList []PathWithPrefix

	patterns := strings.Split(globPatterns, ",")

	fmt.Println("KKKKKKKKKKKKKKKKKKKKKKKKKKKKKKKKKKKK")
	fmt.Println(rootDir)
	fmt.Println(patterns)
	fmt.Println(globPatterns)

	for _, pattern := range patterns {
		rootSearchDirFS := os.DirFS(rootDir)

		relPattern := strings.TrimPrefix(pattern, rootDir+"/")

		fmt.Println("VVVVVVVVVVVVVVVV")
		fmt.Println("pattern: ", pattern)
		fmt.Println("relPattern: ", relPattern)
		fmt.Println("rootSearchDirFS: ", rootSearchDirFS)

		matchedDirs, err := doublestar.Glob(rootSearchDirFS, relPattern)
		if err != nil {
			fmt.Println("GetAllJacocoExecFilesFromGlobPattern Error in doublestar.Glob: ", err.Error())
			return execFilesPathWithPrefixList, err
		}

		for _, match := range matchedDirs {
			execFilesPathWithPrefix := PathWithPrefix{
				CompletePathPrefix: rootDir,
				RelativePath:       match,
			}
			execFilesPathWithPrefixList = append(execFilesPathWithPrefixList, execFilesPathWithPrefix)
		}
	}

	fmt.Println(execFilesPathWithPrefixList)

	return execFilesPathWithPrefixList, nil
}

func FilterFileOrDirUsingGlobPatterns(rootSearchDir string, dirsGlobList []string,
	includeGlobPatternCsvStr, excludeGlobPatternCsvStr string, autoFillIncludePattern string) ([]FilesInfoStore, error) {

	fmt.Println("TTTTTTTTTTTTTTTTTTTTTTTTTTTT")
	fmt.Println(rootSearchDir)
	fmt.Println(dirsGlobList)
	fmt.Println(includeGlobPatternCsvStr)
	fmt.Println(excludeGlobPatternCsvStr)
	fmt.Println(autoFillIncludePattern)

	var filesStoreList []FilesInfoStore

	if len(includeGlobPatternCsvStr) == 0 {
		if len(autoFillIncludePattern) > 0 {
			includeGlobPatternCsvStr = autoFillIncludePattern
		}
	}

	includeGlobPatternStrList := ToStringArrayFromCsvString(includeGlobPatternCsvStr)
	excludeGlobPatternStrList := ToStringArrayFromCsvString(excludeGlobPatternCsvStr)

	for _, dirPattern := range dirsGlobList {

		rootSearchDirFS := os.DirFS(rootSearchDir)

		relPattern := strings.TrimPrefix(dirPattern, rootSearchDir+"/")

		matchedDirs, err := doublestar.Glob(rootSearchDirFS, relPattern)

		fmt.Println("CCCCCCCCCCCCCCCCCCCC")
		fmt.Println("rootSearchDirFS: ", rootSearchDirFS)
		fmt.Println("dirPattern: ", dirPattern)
		fmt.Println("relPattern: ", relPattern)
		fmt.Println("matchedDirs: ", matchedDirs)

		if err != nil {
			return filesStoreList, err
		}
		for _, relativePath := range matchedDirs {
			completePath := filepath.Join(rootSearchDir, relativePath)
			filesInfoStore, err := WalkDir2(completePath, relativePath, rootSearchDir+"/",
				includeGlobPatternStrList, excludeGlobPatternStrList)
			if err != nil {
				LogPrintln(nil, "Error in WalkDir: ", err.Error())
				return filesStoreList, err
			}
			filesStoreList = append(filesStoreList, filesInfoStore)
		}

	}

	return filesStoreList, nil
}

type FilesInfoStore struct {
	IncludedPathsListWithPrefix []PathWithPrefix
	ExcludedPathsListWithPrefix []PathWithPrefix
}

type IncludeExcludesMerged struct {
	CompletePathsWithPrefixList []PathWithPrefix
}

func (i *IncludeExcludesMerged) CopyTo(toDstPathPrefix, buildRootPath string) error {

	err := i.CreateUniqueDirs(toDstPathPrefix)
	if err != nil {
		LogPrintln(nil, "Error in CreateUniqueDirs: ", err.Error())
		return err
	}

	for _, pathWithPrefix := range i.CompletePathsWithPrefixList {
		prefix := pathWithPrefix.CompletePathPrefix
		relPath := pathWithPrefix.RelativePath

		srcPath := filepath.Join(prefix, relPath)
		dstPath := filepath.Join(toDstPathPrefix, relPath)

		err := CopyFile(srcPath, dstPath)
		if err != nil {
			LogPrintln(nil, "** Error **:  in copying file: ", err.Error())
		}
	}

	return nil
}

func (i *IncludeExcludesMerged) CopySourceTo(toDstPathPrefix, buildRootPath string) error {

	uniqueDirs := i.GetAllUniqueDirsForSource(toDstPathPrefix, buildRootPath)
	for _, dir := range uniqueDirs {
		err := CreateDir(dir)
		if err != nil {
			return err
		}
	}

	for _, prefixWithPath := range i.CompletePathsWithPrefixList {

		prefix := prefixWithPath.CompletePathPrefix
		relPath := prefixWithPath.RelativePath
		srcPath := filepath.Join(prefix, relPath)

		pos := strings.Index(srcPath, buildRootPath)
		if pos == -1 {
			LogPrintln(nil, "Target not found in the path")
			continue
		}

		if pos == 0 {
			skipLen := len(buildRootPath)
			tmpFilePath := srcPath[skipLen:]
			dstFile := filepath.Join(toDstPathPrefix, tmpFilePath)

			err := CopyFile(srcPath, dstFile)
			if err != nil {
				LogPrintln(nil, "Error in copying file: ", err.Error())
				continue
			}
		}
	}

	return nil
}

func (i *IncludeExcludesMerged) GetAllUniqueDirsForSource(toDstPathPrefix, buildRootPath string) []string {

	sourceFilesDirMap := map[string]bool{}

	for _, prefixWithPath := range i.CompletePathsWithPrefixList {
		prefix := prefixWithPath.CompletePathPrefix
		relPath := prefixWithPath.RelativePath

		srcPath := filepath.Join(prefix, relPath)
		pos := strings.Index(srcPath, buildRootPath)
		if pos == -1 {
			LogPrintln(nil, "Target not found in the path")
			continue
		}

		if pos == 0 {
			skipLen := len(buildRootPath)
			tmpFilePath := srcPath[skipLen:]
			dstFile := filepath.Join(toDstPathPrefix, tmpFilePath)

			dirPath := filepath.Dir(dstFile)
			sourceFilesDirMap[dirPath] = true
		}

	}

	var uniqueDirs []string
	for dir := range sourceFilesDirMap {
		uniqueDirs = append(uniqueDirs, dir)
	}

	return uniqueDirs
}

func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy content: %w", err)
	}

	err = destFile.Sync()
	if err != nil {
		return fmt.Errorf("failed to sync destination file: %w", err)
	}

	return nil
}

func (i *IncludeExcludesMerged) CreateUniqueDirs(toDstPathPrefix string) error {
	uniqueDirs := i.GetAllUniqueDirs(toDstPathPrefix)
	for _, dir := range uniqueDirs {
		newDir := filepath.Join(toDstPathPrefix, dir)
		err := CreateDir(newDir)
		if err != nil {
			return err
		}
	}
	return nil
}

func (i *IncludeExcludesMerged) GetAllUniqueDirs(toDstPathPrefix string) []string {

	uniqueDirsMap := map[string]bool{}
	_ = uniqueDirsMap

	for _, pathWithPrefix := range i.CompletePathsWithPrefixList {
		curDir := filepath.Dir(pathWithPrefix.RelativePath)
		uniqueDirsMap[curDir] = true
	}

	uniqDirsList := []string{}
	for dir := range uniqueDirsMap {
		uniqDirsList = append(uniqDirsList, dir)
	}

	return uniqDirsList
}

func MergeIncludeExcludeFileCompletePaths(filesInfoStore []FilesInfoStore) []IncludeExcludesMerged {

	validFileList := []string{}
	excludeMap := make(map[string]bool)
	var validFilesListWithPrefix []PathWithPrefix

	for _, fileInfo := range filesInfoStore {
		for _, excludedPathWithPrefix := range fileInfo.ExcludedPathsListWithPrefix {
			excludeFileCompletePath := filepath.Join(excludedPathWithPrefix.CompletePathPrefix,
				excludedPathWithPrefix.RelativePath)
			excludeMap[excludeFileCompletePath] = true
		}
	}

	for _, fileInfo := range filesInfoStore {
		for _, includedPathWithPrefix := range fileInfo.IncludedPathsListWithPrefix {
			includedFileCompletePath := filepath.Join(includedPathWithPrefix.CompletePathPrefix,
				includedPathWithPrefix.RelativePath)
			if _, excluded := excludeMap[includedFileCompletePath]; !excluded {
				validFileList = append(validFileList, includedFileCompletePath)
				validFilesListWithPrefix = append(validFilesListWithPrefix, includedPathWithPrefix)
			}
		}
	}

	var result []IncludeExcludesMerged
	includeExcludesMerged := IncludeExcludesMerged{
		CompletePathsWithPrefixList: validFilesListWithPrefix,
	}
	result = append(result, includeExcludesMerged)
	return result
}

type PathWithPrefix struct {
	CompletePathPrefix string
	RelativePath       string
}

func WalkDir2(completePath, relativePath, completePathPrefix string,
	includeGlobPatternStrList, excludeGlobPatternStrList []string) (FilesInfoStore, error) {

	relativeIncludePathsList := []string{}
	relativeExcludePathsList := []string{}

	includedCompletePathsList := []string{}
	excludedCompletePathsList := []string{}

	var includedPathsListWithPrefix []PathWithPrefix
	var excludedPathsListWithPrefix []PathWithPrefix

	for _, includeGlobPatternStr := range includeGlobPatternStrList {

		rootSearchDirFS := os.DirFS(completePath)
		relPattern := strings.TrimPrefix(includeGlobPatternStr, completePath+"/")
		matchedFiles, err := doublestar.Glob(rootSearchDirFS, relPattern)

		if err != nil {
			LogPrintln(nil, "Error in doublestar.Glob: ", err.Error())
			return FilesInfoStore{}, err
		}

		for _, matchedFile := range matchedFiles {
			matchedFileCompletePath := filepath.Join(completePath, matchedFile)
			includedCompletePathsList = append(includedCompletePathsList, matchedFileCompletePath)
			pathWithPrefix := PathWithPrefix{
				CompletePathPrefix: completePath,
				RelativePath:       matchedFile,
			}
			includedPathsListWithPrefix = append(includedPathsListWithPrefix, pathWithPrefix)
		}

		relativeIncludePathsList = append(relativeIncludePathsList, matchedFiles...)
	}

	for _, excludeGlobPatternStr := range excludeGlobPatternStrList {
		rootSearchDirFS := os.DirFS(completePath)
		relPattern := strings.TrimPrefix(excludeGlobPatternStr, completePath+"/")
		matchedFiles, err := doublestar.Glob(rootSearchDirFS, relPattern)

		if err != nil {
			LogPrintln(nil, "Error in doublestar.Glob: ", err.Error())
		}

		for _, matchedFile := range matchedFiles {
			matchedFileCompletePath := filepath.Join(completePath, matchedFile)
			excludedCompletePathsList = append(excludedCompletePathsList, matchedFileCompletePath)
			pathWithPrefix := PathWithPrefix{
				CompletePathPrefix: completePath,
				RelativePath:       matchedFile,
			}
			excludedPathsListWithPrefix = append(excludedPathsListWithPrefix, pathWithPrefix)
		}

		relativeExcludePathsList = append(relativeExcludePathsList, matchedFiles...)
	}

	classInfoStore := FilesInfoStore{
		IncludedPathsListWithPrefix: includedPathsListWithPrefix,
		ExcludedPathsListWithPrefix: excludedPathsListWithPrefix,
	}

	return classInfoStore, nil
}

func TrimStrings(input []string) []string {
	var trimmed []string
	for _, str := range input {
		trimmed = append(trimmed, strings.TrimSpace(str))
	}
	return trimmed
}

func ToStringArrayFromCsvString(input string) []string {
	sl := strings.Split(input, ",")

	return TrimStrings(sl)
}

func IsMapHasAllStrings(m map[string]interface{}, strList []string) bool {
	for _, str := range strList {
		found := false
		for _, val := range m {
			if valStr, ok := val.(string); ok && valStr == str {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func ToJsonStringFromStruct[T any](v T) (string, error) {
	jsonBytes, err := json.Marshal(v)

	if err == nil {
		return string(jsonBytes), nil
	}

	return "", err
}

func ToJsonStringFromMap[T any](m T) (string, error) {
	outBytes, err := json.Marshal(m)
	if err == nil {
		return string(outBytes), nil
	}
	return "", err
}

func ToStructFromJsonString[T any](jsonStr string) (T, error) {
	var v T
	err := json.Unmarshal([]byte(jsonStr), &v)
	return v, err
}

func GetRandomDir(absolutePrefixPath, workspacePrefixStr string) (string, error) {
	randomStr, err := generateRandomString(8)
	if err != nil {
		return "", err
	}

	dirName := fmt.Sprintf("%s-%s", workspacePrefixStr, randomStr)
	completePath := filepath.Join(absolutePrefixPath, dirName)

	if _, err := os.Stat(completePath); os.IsNotExist(err) {
		if err := os.MkdirAll(completePath, os.ModePerm); err != nil {
			return "", fmt.Errorf("failed to create directory: %w", err)
		}
		return completePath, nil
	} else if err != nil {
		return "", fmt.Errorf("failed to check directory: %w", err)
	}

	return GetRandomDir(absolutePrefixPath, workspacePrefixStr)
}

func GetRandomTmpFileName(absolutePrefixPath, workspacePrefixStr string) (string, error) {
	randomStr, err := generateRandomString(8)
	if err != nil {
		return "", err
	}

	fileName := fmt.Sprintf("%s-%s", workspacePrefixStr, randomStr)
	completePath := filepath.Join(absolutePrefixPath, fileName)

	return completePath, nil
}

func GetRandomJacocoWorkspaceDir(absolutePrefixPath string) (string, error) {
	return GetRandomDir(absolutePrefixPath, "jacoco-workspace-")
}

func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

func CreateDir(absolutePath string) error {
	if absolutePath == "" || absolutePath == "." || absolutePath == ".." {
		return nil
	}

	err := os.MkdirAll(absolutePath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", absolutePath, err)
	}
	return nil
}

func GetOutputVariablesStorageFilePath() string {
	if IsDevTestingMode() {
		return filepath.Join("/tmp", "drone-output")
	}
	return os.Getenv("DRONE_OUTPUT")
}

func ReadFileAsString(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func WriteEnvVariableAsString(key string, value interface{}) error {

	outputFile, err := os.OpenFile(GetOutputVariablesStorageFilePath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open output file: %w", err)
	}
	defer outputFile.Close()

	valueStr := fmt.Sprintf("%v", value)

	_, err = fmt.Fprintf(outputFile, "%s=%s\n", key, valueStr)
	if err != nil {
		return fmt.Errorf("failed to write to env: %w", err)
	}

	return nil
}

func IsDevTestingMode() bool {
	return os.Getenv("DEV_TEST_d6c9b463090c") == "true"
}

func StructToJSONWithEnvKeys(v interface{}) (string, error) {
	val := reflect.ValueOf(v)
	typ := reflect.TypeOf(v)

	data := make(map[string]interface{})

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		key := field.Tag.Get("envconfig")
		if key != "" {
			data[key] = val.Field(i).Interface()
		}
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}

func GetTestWorkSpaceDir() string {
	return TestWorkSpaceDir
}

func GetTestBuildRootDir() string {
	return GetTestWorkSpaceDir()
}

const (
	DefaultWorkSpaceDirEnvVarKey = "DRONE_WORKSPACE"
	TestWorkSpaceDir             = "../test/tmp_workspace"
)

//
//
