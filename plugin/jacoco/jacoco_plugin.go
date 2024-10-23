package jacoco

import (
	"fmt"
	pd "github.com/harness-community/drone-coverage-report/plugin/plugin_defs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type JacocoPlugin struct {
	pd.CoveragePluginArgs
	InputArgs *pd.Args
	JacocoPluginStateStore
}

type JacocoPluginStateStore struct {
	BuildRootPath string

	ExecFilePathsWithPrefixList []pd.PathWithPrefix

	ClassesInfoStoreList []pd.FilesInfoStore
	FinalizedClassesList []pd.IncludeExcludesMerged

	SourcesInfoStoreList []pd.FilesInfoStore
	FinalizedSourcesList []pd.IncludeExcludesMerged

	JacocoWorkSpaceDir         string
	ExecFilesFinalCompletePath []string
	JacocoJarPath              string
	CoverageThresholds         JacocoCoverageThresholdsValues
}

type JacocoCoverageThresholds struct {
	InstructionCoverageThreshold string
	BranchCoverageThreshold      string
	LineCoverageThreshold        string
	ComplexityCoverageThreshold  int
	MethodCoverageThreshold      string
	ClassCoverageThreshold       string
}

type JacocoCoverageThresholdsValues struct {
	InstructionCoverageThreshold float64
	BranchCoverageThreshold      float64
	LineCoverageThreshold        float64
	ComplexityCoverageThreshold  int
	MethodCoverageThreshold      float64
	ClassCoverageThreshold       float64
}

func (p *JacocoPlugin) Init(args *pd.Args) error {

	pd.LogPrintln(p, "JacocoPlugin Init")

	p.InputArgs = args

	err := p.SetBuildRoot("")
	if err != nil {
		pd.LogPrintln(p, "JacocoPlugin Error in Init: "+err.Error())
		return err
	}

	err = p.CreateNewWorkspace()
	if err != nil {
		pd.LogPrintln(p, "JacocoPlugin Error in Init: "+err.Error())
		return err
	}

	err = p.SetJarPath()
	if err != nil {
		pd.LogPrintln(p, "JacocoPlugin Error in Init: "+err.Error())
		return err
	}

	return nil
}

func (p *JacocoPlugin) SetJarPath() error {
	p.JacocoJarPath = os.Getenv("JACOCO_JAR_PATH")
	if p.JacocoJarPath == "" {
		p.JacocoJarPath = DefaultJacocoJarPath
	}

	_, err := os.Stat(p.JacocoJarPath)
	if err != nil {
		cwd, err := os.Getwd()
		if err != nil {
			pd.LogPrintln(p, "JacocoPlugin Error in SetJarPath: "+err.Error())
			return pd.GetNewError("Error in SetJarPath: " + err.Error())
		}

		p.JacocoJarPath = filepath.Join(cwd, TestJacocoJarPath)
		_, err = os.Stat(p.JacocoJarPath)
		if err != nil {
			pd.LogPrintln(p, "JacocoPlugin Error in SetJarPath: "+err.Error())
			return pd.GetNewError("Error in SetJarPath: " + err.Error())
		}
	}

	return nil
}

func (p *JacocoPlugin) CreateNewWorkspace() error {

	jacocoWorkSpaceDir, err := p.GetBuildRootPath()
	if err != nil {
		pd.LogPrintln(p, "JacocoPlugin Error in Creat`eNewWorkspace: "+err.Error())
		return pd.GetNewError("Error in CreateNewWorkspace: " + err.Error())
	}

	p.JacocoWorkSpaceDir = jacocoWorkSpaceDir

	err = pd.CreateDir(p.JacocoWorkSpaceDir)
	if err != nil {
		pd.LogPrintln(p, "JacocoPlugin CreateNewWorkspace Error: "+err.Error())
		return err
	}

	err = pd.CreateDir(p.GetOutputReportsWorkSpaceDir())
	if err != nil {
		pd.LogPrintln(p, "JacocoPlugin CreateNewWorkspace Error: "+err.Error())
		return err
	}

	return nil
}

func (p *JacocoPlugin) GetWorkspaceDir() string {

	p.JacocoWorkSpaceDir = os.Getenv(pd.DefaultWorkSpaceDirEnvVarKey)
	if p.JacocoWorkSpaceDir == "" {
		p.JacocoWorkSpaceDir = pd.GetTestWorkSpaceDir()
	}

	return p.JacocoWorkSpaceDir
}

func (p *JacocoPlugin) InspectProcessArgs(argNamesList []string) (map[string]interface{}, error) {

	m := map[string]interface{}{}
	for _, argName := range argNamesList {
		switch argName {
		case ClassesInfoStoreListParamKey:
			m[argName] = p.ClassesInfoStoreList
		case FinalizedSourcesListParamKey:
			m[argName] = p.SourcesInfoStoreList
		case WorkSpaceCompletePathKeyStr:
			nm := map[string]string{}
			nm["classes"] = p.GetClassesWorkSpaceDir()
			nm["sources"] = p.GetSourcesWorkSpaceDir()
			nm["execFiles"] = p.GetExecFilesWorkSpaceDir()
			nm["workspace"] = p.GetWorkspaceDir()
			m[argName] = nm
		}

	}
	return m, nil
}

func (p *JacocoPlugin) GetBuildRootPath() (string, error) {
	buildRootPath := os.Getenv(pd.DefaultWorkSpaceDirEnvVarKey)

	if buildRootPath == "" {
		return pd.GetTestBuildRootDir(), nil
	}

	return buildRootPath, nil
}

func (p *JacocoPlugin) SetBuildRoot(buildRootPath string) error {

	var err error

	if buildRootPath == "" {
		buildRootPath, err = p.GetBuildRootPath()
		if err != nil {
			pd.LogPrintln(p, "JacocoPlugin Error in SetBuildRoot: "+err.Error())
			return err
		}
	}

	ok, err := pd.IsDirExists(buildRootPath)

	if err != nil {
		pd.LogPrintln(p, "JacocoPlugin Error in SetBuildRoot: "+err.Error())
		return err
	}

	if !ok {
		pd.LogPrintln(p, "JacocoPlugin Error in SetBuildRoot: Build root path does not exist")
		return pd.GetNewError("Error in SetBuildRoot: Build root path does not exist")
	}

	p.BuildRootPath = buildRootPath
	return nil
}

func (p *JacocoPlugin) DeInit() error {
	pd.LogPrintln(p, "JacocoPlugin DeInit")
	return nil
}

func (p *JacocoPlugin) ValidateAndProcessArgs(args pd.Args) error {
	pd.LogPrintln(p, "JacocoPlugin BuildAndValidateArgs")

	err := p.IsExecFileArgOk(args)
	if err != nil {
		pd.LogPrintln(p, "JacocoPlugin Error in ValidateAndProcessArgs: "+err.Error())
		return err
	}

	err = p.IsClassArgOk(args)
	if err != nil {
		pd.LogPrintln(p, "JacocoPlugin Error in ValidateAndProcessArgs: "+err.Error())
		return err
	}

	err = p.IsSourceArgOk(args)
	if err != nil {
		pd.LogPrintln(p, "JacocoPlugin Error in ValidateAndProcessArgs: "+err.Error())
		return err
	}

	return nil
}

func (p *JacocoPlugin) GetClassesList() []pd.IncludeExcludesMerged {
	return p.FinalizedClassesList
}

func (p *JacocoPlugin) GetSourcesList() []pd.IncludeExcludesMerged {
	return p.FinalizedSourcesList
}

func (p *JacocoPlugin) DoPostArgsValidationSetup(args pd.Args) error {
	pd.LogPrintln(p, "JacocoPlugin DoPostArgsValidationSetup")

	err := p.CopyClassesToWorkspace()
	if err != nil {
		pd.LogPrintln(p, "JacocoPlugin Error in DoPostArgsValidationSetup: "+err.Error())
		return err
	}

	err = p.CopySourcesToWorkspace()
	if err != nil {
		pd.LogPrintln(p, "JacocoPlugin Error in DoPostArgsValidationSetup: "+err.Error())
		return err
	}

	err = p.CopyJacocoExecFilesToWorkspace()
	if err != nil {
		pd.LogPrintln(p, "JacocoPlugin Error in DoPostArgsValidationSetup: "+err.Error())
		return err
	}

	return nil
}

func (p *JacocoPlugin) GetExecFilesWorkSpaceDir() string {
	return filepath.Join(p.GetWorkspaceDir(), "execFiles")
}

func (p *JacocoPlugin) GetClassesWorkSpaceDir() string {
	return filepath.Join(p.GetWorkspaceDir(), "classes")
}

func (p *JacocoPlugin) GetOutputReportsWorkSpaceDir() string {
	return filepath.Join(p.GetWorkspaceDir(), JacocoReportsDirName)
}

func (p *JacocoPlugin) GetSourcesWorkSpaceDir() string {
	return filepath.Join(p.GetWorkspaceDir(), "sources")
}

func (p *JacocoPlugin) CopyJacocoExecFilesToWorkspace() error {
	uniqueDirs, err := p.GetJacocoExecFilesUniqueDirs()
	if err != nil {
		pd.LogPrintln(p, "JacocoPlugin Error in CopyJacocoExecFilesToWorkspace: "+err.Error())
		return err
	}

	execFilesDir := p.GetExecFilesWorkSpaceDir()
	pd.LogPrintln(p, "JacocoPlugin Copying Exec files to workspace: "+execFilesDir)
	err = pd.CreateDir(execFilesDir)
	if err != nil {
		pd.LogPrintln(p, "JacocoPlugin Error in CopyJacocoExecFilesToWorkspace: "+err.Error())
		return pd.GetNewError("Error in CopyJacocoExecFilesToWorkspace: " + err.Error())
	}

	for _, dir := range uniqueDirs {
		newDir := filepath.Join(execFilesDir, dir)
		err = pd.CreateDir(newDir)
		if err != nil {
			pd.LogPrintln(p, "JacocoPlugin Error in CopyJacocoExecFilesToWorkspace: "+err.Error())
			return pd.GetNewError("Error in CopyJacocoExecFilesToWorkspace: " + err.Error())
		}
	}

	for _, execFilePathsWithPrefix := range p.ExecFilePathsWithPrefixList {
		relPath := execFilePathsWithPrefix.RelativePath
		srcFilePath := filepath.Join(execFilePathsWithPrefix.CompletePathPrefix, execFilePathsWithPrefix.RelativePath)
		dstFilePath := filepath.Join(execFilesDir, relPath)
		err = pd.CopyFile(srcFilePath, dstFilePath)
		if err != nil {
			pd.LogPrintln(p, "JacocoPlugin Error in CopyJacocoExecFilesToWorkspace: "+err.Error())
			return pd.GetNewError("Error in CopyJacocoExecFilesToWorkspace: " + err.Error())
		}

		p.ExecFilesFinalCompletePath = append(p.ExecFilesFinalCompletePath, dstFilePath)
	}

	return nil
}

func (p *JacocoPlugin) GetJacocoExecFilesUniqueDirs() ([]string, error) {

	uniqueDirMap := map[string]bool{}

	for _, execFilePathsWithPrefix := range p.ExecFilePathsWithPrefixList {
		dir := filepath.Dir(execFilePathsWithPrefix.RelativePath)
		uniqueDirMap[dir] = true
	}

	execFilesDirList := []string{}

	for dir, _ := range uniqueDirMap {
		execFilesDirList = append(execFilesDirList, dir)
	}

	return execFilesDirList, nil
}

func (p *JacocoPlugin) CopyClassesToWorkspace() error {
	fmt.Println("GGGGGGGGGGGGGGGGGGGGG")
	classesList := p.GetClassesList()
	if len(classesList) < 1 {
		pd.LogPrintln(p, "JacocoPlugin Error in CopyClassesToWorkspace: No class files to copy")
		return pd.GetNewError("Error in CopyClassesToWorkspace: No class files to copy")
	}

	d, err1 := os.Getwd()
	if err1 != nil {
		pd.LogPrintln(p, "JacocoPlugin Error in CopyClassesToWorkspace: "+err1.Error())
	}

	fmt.Println("HHHHHHHHHHHHHHHHHHHHH ")

	fmt.Println(d)
	fmt.Println(classesList)

	dstClassesDir := p.GetClassesWorkSpaceDir()
	err := pd.CreateDir(dstClassesDir)
	if err != nil {
		pd.LogPrintln(p, "JacocoPlugin Error in CopyClassesToWorkspace: "+err.Error())
		return pd.GetNewError("Error in CopyClassesToWorkspace: " + err.Error())
	}

	for _, classInfo := range classesList {

		err := classInfo.CopyTo(dstClassesDir, p.BuildRootPath)
		if err != nil {
			continue
		}
	}

	return nil
}

func (p *JacocoPlugin) CopySourcesToWorkspace() error {

	if p.InputArgs.SkipCopyOfSrcFiles {
		pd.LogPrintln(p, "JacocoPlugin Skipping copying of source files")
		return nil
	}

	dstSourcesDir := p.GetSourcesWorkSpaceDir()
	pd.LogPrintln(p, "JacocoPlugin Copying sources to workspace: "+dstSourcesDir)
	err := pd.CreateDir(dstSourcesDir)
	if err != nil {
		pd.LogPrintln(p, "JacocoPlugin Error in CopySourcesToWorkspace: "+err.Error())
		return pd.GetNewError("Error in CopySourcesToWorkspace: " + err.Error())
	}

	sourcesList := p.GetSourcesList()
	for _, sourceInfo := range sourcesList {
		err := sourceInfo.CopySourceTo(dstSourcesDir, p.BuildRootPath)
		if err != nil {
			continue
		}
	}

	return nil
}

func (p *JacocoPlugin) GetClassPatternsStrArray() []string {
	return pd.ToStringArrayFromCsvString(p.InputArgs.ClassPatterns)
}

func (p *JacocoPlugin) GetSourcePatternsStrArray() []string {
	return pd.ToStringArrayFromCsvString(p.InputArgs.SourcePattern)
}

func (p *JacocoPlugin) IsSourceArgOk(args pd.Args) error {
	pd.LogPrintln(p, "JacocoPlugin BuildAndValidateArgs")

	if p.InputArgs.SkipCopyOfSrcFiles {
		pd.LogPrintln(p, "JacocoPlugin Skipping copying of source files")
		return nil
	}

	if args.SourcePattern == "" {
		return pd.GetNewError("Error in IsSourceArgOk: SourcePattern is empty")
	}
	p.InputArgs.SourcePattern = args.SourcePattern
	p.InputArgs.SourceInclusionPattern = args.SourceInclusionPattern
	p.InputArgs.SourceExclusionPattern = args.SourceExclusionPattern

	sourcesInfoStoreList, err :=
		pd.FilterFileOrDirUsingGlobPatterns(p.BuildRootPath, p.GetSourcePatternsStrArray(),
			p.InputArgs.SourceInclusionPattern, p.InputArgs.SourceExclusionPattern, AllSourcesAutoFillGlob)

	if err != nil {
		pd.LogPrintln(p, "JacocoPlugin Error in IsSourceArgOk: "+err.Error())
		return pd.GetNewError("Error in IsSourceArgOk: " + err.Error())
	}

	p.SourcesInfoStoreList = sourcesInfoStoreList
	p.FinalizedSourcesList = pd.MergeIncludeExcludeFileCompletePaths(p.SourcesInfoStoreList)

	return nil

}

func (p *JacocoPlugin) IsClassArgOk(args pd.Args) error {

	pd.LogPrintln(p, "JacocoPlugin IsClassArgOk")

	if args.ClassPatterns == "" {
		return pd.GetNewError("Error in IsClassArgOk: ClassPatterns is empty")
	}
	p.InputArgs.ClassPatterns = args.ClassPatterns
	p.InputArgs.ClassInclusionPatterns = args.ClassInclusionPatterns
	p.InputArgs.ClassExclusionPatterns = args.ClassExclusionPatterns

	classesInfoStoreList, err :=
		pd.FilterFileOrDirUsingGlobPatterns(p.BuildRootPath, p.GetClassPatternsStrArray(),
			p.InputArgs.ClassInclusionPatterns, p.InputArgs.ClassExclusionPatterns, AllClassesAutoFillGlob)

	if err != nil {
		pd.LogPrintln(p, "JacocoPlugin Error in IsClassArgOk: "+err.Error())
		return pd.GetNewError("Error in IsClassArgOk: " + err.Error())
	}

	p.ClassesInfoStoreList = classesInfoStoreList
	p.FinalizedClassesList = pd.MergeIncludeExcludeFileCompletePaths(p.ClassesInfoStoreList)

	if len(p.FinalizedClassesList) < 1 {
		pd.LogPrintln(p, "Error in IsClassArgOk: No class inferred from class patterns")
		return pd.GetNewError("Error in IsClassArgOk: No class inferred from class patterns")
	}
	return nil
}

func (p *JacocoPlugin) IsExecFileArgOk(args pd.Args) error {

	pd.LogPrintln(p, "JacocoPlugin BuildAndValidateArgs")

	if args.ExecFilesPathPattern == "" {
		return pd.GetNewError("Error in IsExecFileArgOk: ExecFilesPathPattern is empty")
	}

	execFilesPathList, err := pd.GetAllJacocoExecFilesFromGlobPattern(p.BuildRootPath, args.ExecFilesPathPattern)
	if err != nil {
		pd.LogPrintln(p, "JacocoPlugin Error in IsExecFileArgOk: "+err.Error())
		return pd.GetNewError("Error in IsExecFileArgOk: " + err.Error())
	}

	p.ExecFilePathsWithPrefixList = execFilesPathList

	if len(p.ExecFilePathsWithPrefixList) < 1 {
		pd.LogPrintln(p, "JacocoPlugin Error in IsExecFileArgOk: No jacoco exec files found")
		return pd.GetNewError("Error in IsExecFileArgOk: No jacoco exec files found")
	}

	return nil
}

func (p *JacocoPlugin) GetExecFilesList() []pd.PathWithPrefix {
	return p.ExecFilePathsWithPrefixList
}

/*
Usage example:
java -jar jacoco.jar \
    report   ./gameoflife-core/target/jacoco.exec   ./gameoflife-web/target/jacoco.exec   \
    --classfiles ./gameoflife-core/target/classes   \
    --sourcefiles ./gameoflife-core/src/main/java   \
    --html ./gameoflife-core/target/site/jacoco_html   \
    --xml ./gameoflife-core/target/site/jacoco.xml
*/

func (p *JacocoPlugin) Run() error {
	pd.LogPrintln(p, "JacocoPlugin Run")

	err := p.GenerateJacocoReports()
	if err != nil {
		pd.LogPrintln(p, "JacocoPlugin Error in Run: "+err.Error())
		return err
	}

	err = p.AnalyzeJacocoCoverageThresholds()
	if err != nil {
		pd.LogPrintln(p, "JacocoPlugin Error in Run: "+err.Error())
		return err
	}

	return nil
}

func (p *JacocoPlugin) AnalyzeJacocoCoverageThresholds() error {

	if p.PluginFailIfNoReports {
		_, err := os.Stat(p.GetJacocoXmlReportFilePath())
		if err != nil {
			pd.LogPrintln(p, "JacocoPlugin Error in AnalyzeJacocoCoverageThresholds: "+err.Error())
			return pd.GetNewError("Error in AnalyzeJacocoCoverageThresholds: " + err.Error())
		}
		_, err = os.Stat(p.GetJacocoHtmlReportFilePath())
		if err != nil {
			pd.LogPrintln(p, "JacocoPlugin Error in AnalyzeJacocoCoverageThresholds: "+err.Error())
			return pd.GetNewError("Error in AnalyzeJacocoCoverageThresholds: " + err.Error())
		}
	}

	p.CoverageThresholds = GetJacocoCoverageThresholds(p.GetJacocoXmlReportFilePath())

	if p.PluginFailOnThreshold == false {
		pd.LogPrintln(p, "JacocoPlugin PluginFailOnThreshold is false, so skipping threshold check")
		return nil
	}

	if p.IsThresholdValuesGood() == false {
		pd.LogPrintln(p, "JacocoPlugin Error in AnalyzeJacocoCoverageThresholds: Threshold values not good")
		return pd.GetNewError("Error in AnalyzeJacocoCoverageThresholds: Threshold values not good")
	}

	return nil
}

func (p *JacocoPlugin) IsThresholdValuesGood() bool {

	type ThresholdsCompare struct {
		ObservedValue float64
		ExpectedValue float64
		ThresholdType string
	}

	thresholdsCompareList := []ThresholdsCompare{
		{ObservedValue: p.CoverageThresholds.InstructionCoverageThreshold,
			ExpectedValue: p.InputArgs.MinimumInstructionCoverage, ThresholdType: "InstructionCoverage"},
		{ObservedValue: p.CoverageThresholds.BranchCoverageThreshold,
			ExpectedValue: p.InputArgs.MinimumBranchCoverage, ThresholdType: "BranchCoverage"},
		{ObservedValue: p.CoverageThresholds.LineCoverageThreshold,
			ExpectedValue: p.InputArgs.MinimumLineCoverage, ThresholdType: "LineCoverage"},
		{ObservedValue: p.CoverageThresholds.MethodCoverageThreshold,
			ExpectedValue: p.InputArgs.MinimumMethodCoverage, ThresholdType: "MethodCoverage"},
		{ObservedValue: p.CoverageThresholds.ClassCoverageThreshold,
			ExpectedValue: p.InputArgs.MinimumClassCoverage, ThresholdType: "ClassCoverage"},
	}

	for _, thresholdCompare := range thresholdsCompareList {
		if thresholdCompare.ObservedValue <= thresholdCompare.ExpectedValue {
			pd.LogPrintln(p, "JacocoPlugin "+thresholdCompare.ThresholdType+" threshold not met",
				" expected = ", thresholdCompare.ExpectedValue, " observed = ", thresholdCompare.ObservedValue)

			return false
		}
	}

	if p.CoverageThresholds.ComplexityCoverageThreshold > p.InputArgs.MinimumComplexityCoverage {
		pd.LogPrintln(p, "JacocoPlugin ComplexityCoverage threshold not met",
			" expected = ", p.InputArgs.MinimumComplexityCoverage,
			" observed = ", p.CoverageThresholds.ComplexityCoverageThreshold)
		return false
	}

	return true
}

func (p *JacocoPlugin) GenerateJacocoReports() error {

	args := []string{}

	args = append(args, "java"+" ")
	args = append(args, "-jar"+" "+p.JacocoJarPath+" ")
	args = append(args, p.GetReportArgs()+" ")
	args = append(args, p.GetClassFilesPathArgs()+" ")

	if p.InputArgs.SkipCopyOfSrcFiles == false {
		args = append(args, p.GetSourceFilesPathArgs()+" ")
	}

	args = append(args, p.GetHtmlReportArgs()+" ")
	args = append(args, p.GetXmlReportArgs()+" ")

	cmdStr := strings.Join(args, " ")
	pd.LogPrintln(p, "JacocoPlugin Running command: ")
	pd.LogPrintln(p, cmdStr)

	parts := strings.Fields(cmdStr)

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		pd.LogPrintln(p, "JacocoPlugin Error in Run: "+err.Error())
		return pd.GetNewError("Error in Run: " + err.Error())
	} else {
		pd.LogPrintln(p, "Command executed successfully.")
	}

	return nil
}

func (p *JacocoPlugin) GetReportArgs() string {
	reportArg := "report"
	for _, execFilePath := range p.ExecFilesFinalCompletePath {
		reportArg = reportArg + " " + execFilePath
	}
	return reportArg
}

func (p *JacocoPlugin) GetClassFilesPathArgs() string {
	classFilePathArg := "--classfiles"
	classFilePathArg = classFilePathArg + " " + p.GetClassesWorkSpaceDir()
	return classFilePathArg
}

func (p *JacocoPlugin) GetSourceFilesPathArgs() string {
	sourceFilePathArg := "--sourcefiles"
	sourceFilePathArg = sourceFilePathArg + " " + p.GetSourcesWorkSpaceDir()
	return sourceFilePathArg
}

func (p *JacocoPlugin) GetHtmlReportArgs() string {
	htmlReportArg := "--html"
	htmlReportArg = htmlReportArg + " " + p.GetOutputReportsWorkSpaceDir() + "/" + "jacoco_html" + " "
	return htmlReportArg
}

func (p *JacocoPlugin) GetXmlReportArgs() string {
	xmlReportArg := "--xml"
	xmlReportArg = xmlReportArg + " " + p.GetJacocoXmlReportFilePath() + " "
	return xmlReportArg
}

func (p *JacocoPlugin) GetJacocoXmlReportFilePath() string {
	return filepath.Join(p.GetOutputReportsWorkSpaceDir(), "jacoco.xml")
}

func (p *JacocoPlugin) GetJacocoHtmlReportFilePath() string {
	return filepath.Join(p.GetOutputReportsWorkSpaceDir(), "jacoco.html")
}

func (p *JacocoPlugin) PersistResults() error {
	pd.LogPrintln(p, "JacocoPlugin StoreResults")
	return nil
}

func (p *JacocoPlugin) WriteOutputVariables() error {
	pd.LogPrintln(p, "JacocoPlugin WriteOutputVariables to ", pd.GetOutputVariablesStorageFilePath())

	type EnvKvPair struct {
		Key   string
		Value interface{}
	}

	var kvPairs = []EnvKvPair{
		{Key: "INSTRUCTION_COVERAGE", Value: p.CoverageThresholds.InstructionCoverageThreshold},
		{Key: "BRANCH_COVERAGE", Value: p.CoverageThresholds.BranchCoverageThreshold},
		{Key: "LINE_COVERAGE", Value: p.CoverageThresholds.LineCoverageThreshold},
		{Key: "COMPLEXITY_COVERAGE", Value: p.CoverageThresholds.ComplexityCoverageThreshold},
		{Key: "METHOD_COVERAGE", Value: p.CoverageThresholds.MethodCoverageThreshold},
		{Key: "CLASS_COVERAGE", Value: p.CoverageThresholds.ClassCoverageThreshold},
	}

	var retErr error = nil

	for _, kvPair := range kvPairs {
		err := pd.WriteEnvVariableAsString(kvPair.Key, kvPair.Value)
		if err != nil {
			retErr = err
		}
	}

	s, err := pd.ReadFileAsString(pd.GetOutputVariablesStorageFilePath())
	if err != nil {
		pd.LogPrintln(p, "JacocoPlugin Error in WriteOutputVariables: "+err.Error())
		return pd.GetNewError("Error in WriteOutputVariables: " + err.Error())
	}

	fmt.Println("\n\nReading JacocoPlugin Output Variables file ", pd.GetOutputVariablesStorageFilePath())
	fmt.Println(s)
	fmt.Println("Reading Complete")

	return retErr
}

func (p *JacocoPlugin) IsQuiet() bool {
	return false
}

func (p *JacocoPlugin) GetPluginType() string {
	return pd.JacocoPluginType
}

func GetNewJacocoPlugin() JacocoPlugin {
	return JacocoPlugin{}
}

const (
	JacocoReportsDirName         = "jacoco_reports_dir"
	ClassFilesListParamKey       = "ClassFilesList"
	ClassesInfoStoreListParamKey = "ClassesInfoStoreList"
	FinalizedSourcesListParamKey = "FinalizedSourcesList"
	WorkSpaceCompletePathKeyStr  = "WorkSpaceCompletePathKeyStr"
	AllClassesAutoFillGlob       = "**/*.class"
	AllSourcesAutoFillGlob       = "**/*.java"
	DefaultJacocoJarPath         = "/opt/harness/plugins-deps/jacoco/0.8.12/jacoco.jar"
	TestJacocoJarPath            = "../test/tmp_workspace/jacoco.jar"
)

//
//
