package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/gofunky/semver"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "请使用子命令 init、reload 或 build")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		if err := initProject(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

	case "build":
		if err := buildProject(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

	case "reload":
		if err := reloadProject(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

	default:
		fmt.Fprintln(os.Stderr, "请使用子命令 init、reload 或 build")
		os.Exit(1)
	}
}

func reloadProject(args []string) error {
	var dirs = args
	if len(args) == 0 {
		dirs = findBuildDirs()
	} else {
		var newDirs []string
		for _, dir := range dirs {
			if findBuildDir(dir) {
				newDirs = append(newDirs, dir)
			}
		}
		dirs = newDirs
	}
	if len(dirs) == 0 {
		return errNoBuildDirs
	} else {
		fmt.Println("找到目标目录：", dirs)
	}

	cfg := &buildConfig{}
	for _, dir := range dirs {
		cfgFilePath := filepath.Join(dir, defaultBuildConfig)
		if err := cfg.ReadFrom(cfgFilePath); err != nil {
			return err
		}

		if err := initFromConfig(cfg, dir, false); err != nil {
			return err
		}
	}

	return nil
}

func buildProject(args []string) error {
	flagSet := flag.NewFlagSet("ndk-cmake build", flag.ExitOnError)
	var target string
	flagSet.StringVar(&target, "target", "", "makefile target name")
	flagSet.Parse(args)

	var dirs = flagSet.Args()
	if len(dirs) == 0 {
		dirs = findBuildDirs()
	} else {
		var newDirs []string
		for _, dir := range dirs {
			if findBuildDir(dir) {
				newDirs = append(newDirs, dir)
			}
		}
		dirs = newDirs
	}
	if len(dirs) == 0 {
		return errNoBuildDirs
	} else {
		fmt.Println("找到目标目录：", dirs)
	}

	cfg := &buildConfig{}
	for _, dir := range dirs {
		cfgFilePath := filepath.Join(dir, defaultBuildConfig)
		if err := cfg.ReadFrom(cfgFilePath); err != nil {
			return err
		}
		cmakeArgs := []string{"--build", dir}
		if len(target) > 0 {
			cmakeArgs = append(cmakeArgs, "--target", target)
		}
		cmakeArgs = append(cmakeArgs, "--", "-j", fmt.Sprintf("%d", runtime.NumCPU()))
		cmd := exec.Command(cfg.CMake, cmakeArgs...)
		fmt.Println(cmd)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}

func findBuildDirs() []string {
	d, _ := filepath.Abs(".")
	if b := findBuildDir(d); b {
		return []string{d}
	}

	infos, err := ioutil.ReadDir(d)
	if err != nil {
		return nil
	}
	var ret []string
	for _, info := range infos {
		fullPath := filepath.Join(d, info.Name())
		if info.IsDir() && findBuildDir(fullPath) {
			ret = append(ret, fullPath)
		}
	}
	return ret
}

func findBuildDir(dir string) bool {
	infos, err := ioutil.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, info := range infos {
		if !info.IsDir() && info.Name() == defaultBuildConfig {
			return true
		}
	}
	return false
}

func initProject() error {
	sdk := ""
	cmake := ""
	ndk := ""
	project := ""
	outputDir := ""
	abi := 0
	armMode := 0
	neon := 0
	ld := 0
	platform := 0
	stl := 0
	buildMode := 0
	var err error

	type step int
	const (
		stepStart step = iota
		stepCMake
		stepNDK
		stepProject
		stepABI
		stepArmMode
		stepNeon
		stepLd
		stepPlatform
		stepSTL
		stepBuildMode
		stepOutput
		stepInvalid step = -1
	)
	state := stepStart

	for state >= stepStart {
		switch state {

		case stepStart: // 设置 Android SDK 的路径
			sdk, err = findAndroidSDK()
			if err != nil {
				sdk, err = readString("请输入 Android SDK 路径：")
			}
			if err == nil {
				fmt.Println("找到 Android SDK：", sdk)
				state = stepCMake
			}

		case stepCMake: // 查找 CMake 路径
			cmake, err = findCMake(sdk)
			if err != nil {
				cmake, err = readString("请输入 CMake 路径：")
			}
			if err == nil {
				fmt.Println("找到 CMake：", cmake)
				state = stepNDK
			}

		case stepNDK: // 查找 NDK 路径
			ndk, err = findNDK(sdk)
			if err != nil {
				ndk, err = readString("请输入 NDK 路径：")
			}
			if err == nil {
				fmt.Println("找到 NDK：", ndk)
				state = stepProject
			}

		case stepProject: // 查找工程路径
			cmakeFile, _ := filepath.Abs("CMakeLists.txt")
			if _, err = os.Stat(cmakeFile); err != nil {
				abs, _ := filepath.Abs(".")
				project, err = readString(fmt.Sprintf("请输入工程路径，默认为 %s：", abs))
				if len(project) == 0 {
					project = abs
				}
				cmakeFile, _ = filepath.Abs(filepath.Join(project, "CMakeLists.txt"))
				if _, err = os.Stat(cmakeFile); err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
			}
			project = filepath.Dir(cmakeFile)
			fmt.Println("找到工程目录：", project)
			state = stepABI

		case stepOutput: // 选择输出目录名
			buffers := new(strings.Builder)
			fmt.Fprintf(buffers, "cmake")
			if abi != 1 {
				fmt.Fprintf(buffers, "-%s", androidABIs[abi])
			} else {
				fmt.Fprintf(buffers, "-armeabi-v7a-with-NEON")
			}
			if abi < 2 {
				fmt.Fprintf(buffers, "-%s", androidArmMode[armMode])
			}
			fmt.Fprintf(buffers, "-%s", androidStl[stl])
			fmt.Fprintf(buffers, "-android%d", platform)
			fmt.Fprintf(buffers, "-%s", cmakeBuildModes[buildMode])
			defaultBuildDirName := buffers.String()
			outputDir, _ = readString(fmt.Sprintf("请输入 CMake 生成文件目录，默认为 %s：", defaultBuildDirName))
			if len(outputDir) == 0 {
				outputDir = defaultBuildDirName
			}
			state = stepInvalid

		case stepABI: // 设置 ANDROID_ABI
			abi, _ = readInt("请输入 ANDROID_ABI，默认为 armeabi-v7a：\n\t1: armeabi-v7a with NEON\n\t2: arm64-v8a\n\t3: x86\n\t4: x86_64")
			if abi == 0 || abi == 1 {
				state = stepArmMode
			} else {
				state = stepLd
			}

		case stepArmMode: // 设置 ANDROID_ARM_MODE
			armMode, _ = readInt("请输入 ANDROID_ARM_MODE，默认为 thumb：\n\t1: arm")
			state = stepNeon

		case stepNeon: // 设置 ANDROID_ARM_NEON
			if abi == 0 {
				neon, _ = readInt("请输入 ANDROID_ARM_NEON，默认为不选择：\n\t1: TRUE\n\t2: FALSE")
			}
			state = stepLd

		case stepLd: // 设置 ANDROID_LD
			ld, _ = readInt("请输入 ANDROID_LD，默认为不选择：\n\t1: lld\n\t2: default")
			state = stepPlatform

		case stepPlatform: // 设置 ANDROID_PLATFORM
			versions, _ := findPlatformVersions(ndk, abi)
			defaultVersion := 21
			desc := "请输入 ANDROID_PLATFORM，默认为 %d："
			if len(versions) > 0 {
				builders := new(strings.Builder)
				fmt.Fprintf(builders, "请输入 ANDROID_PLATFORM，\n")
				for _, version := range versions {
					fmt.Fprintf(builders, "\t%d: %s\n", version, androidVersions[version])
				}
				fmt.Fprint(builders, "默认为 %d:")
				desc = builders.String()
			}
			platform, _ = readInt(fmt.Sprintf(desc, defaultVersion))
			found := false
			for i := range versions {
				if platform == versions[i] {
					found = true
					break
				}
			}
			if !found {
				platform = defaultVersion
			}
			state = stepSTL

		case stepSTL: // 设置 ANDROID_STL
			stl, _ = readInt("请输入 ANDROID_STL，默认为 c++_static：\n\t1: c++_shared\n\t2: none\n\t3: system")
			state = stepBuildMode

		case stepBuildMode: // 设置 build mode
			buildMode, _ = readInt("请输入构建模式，默认为 Debug：\n\t1: Release\n\t2: RelWithDebInfo\n\t3: MinSizeRel")
			state = stepOutput
		}
	}

	cfg := &buildConfig{
		Project:    project,
		AndroidSDK: sdk,
		CMake:      cmake,
		AndroidNDK: ndk,
		ABI:        androidABIs[abi],
		ArmMode:    androidArmMode[armMode],
		Neon:       androidArmNeon[neon],
		Ld:         androidLd[ld],
		Platform:   platform,
		Stl:        androidStl[stl],
		BuildMode:  cmakeBuildModes[buildMode],
	}

	fmt.Println("--------------------------------")
	fmt.Println(cfg)
	fmt.Println("--------------------------------")

	buildTargetDir := filepath.Join(project, outputDir)

	os.RemoveAll(buildTargetDir)
	os.MkdirAll(buildTargetDir, 0775)

	if err = initFromConfig(cfg, buildTargetDir, false); err != nil {
		return err
	}

	return cfg.WriteTo(filepath.Join(buildTargetDir, defaultBuildConfig))
}

func initFromConfig(cfg *buildConfig, buildDir string, ninja bool) error {
	arguments := []string{
		fmt.Sprintf("-DCMAKE_BUILD_TYPE=%s", cfg.BuildMode),
		"-DCMAKE_VERBOSE_MAKEFILE=ON",
		fmt.Sprintf("-DCMAKE_TOOLCHAIN_FILE=%s", filepath.Join(cfg.AndroidNDK, "build/cmake/android.toolchain.cmake")),
		fmt.Sprintf("-DANDROID_ABI=%s", cfg.ABI),
		fmt.Sprintf("-DANDROID_ARM_MODE=%s", cfg.ArmMode),
		fmt.Sprintf("-DANDROID_PLATFORM=%d", cfg.Platform),
		fmt.Sprintf("-DANDROID_STL=%s", cfg.Stl),
	}

	if len(cfg.Neon) > 0 {
		arguments = append(arguments, fmt.Sprintf("-DANDROID_ARM_NEON=%s", cfg.Neon))
	}

	if len(cfg.Ld) > 0 {
		arguments = append(arguments, fmt.Sprintf("-DANDROID_LD=%s", cfg.Ld))
	}

	if ninja {
		arguments = append(arguments,
			"-DANDROID_TOOLCHAIN=clang",
			fmt.Sprintf("-DCMAKE_MAKE_PROGRAM=%s", filepath.Join(filepath.Dir(cfg.CMake), "ninja")),
			"-G", "Android Gradle - Ninja",
		)
	} else {
		arguments = append(arguments,
			"-G", "CodeBlocks - Unix Makefiles",
		)
	}

	arguments = append(arguments, cfg.Project)

	cmd := exec.Command(cfg.CMake, arguments...)
	fmt.Println(cmd)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Dir = buildDir

	return cmd.Run()
}

type buildConfig struct {
	Project    string `json:"project"`
	AndroidSDK string `json:"sdk"`
	CMake      string `json:"cmake"`
	AndroidNDK string `json:"ndk"`
	ABI        string `json:"abi"`
	ArmMode    string `json:"arm_mode"`
	Neon       string `json:"neon"`
	Ld         string `json:"ld"`
	Platform   int    `json:"platform"`
	Stl        string `json:"stl"`
	BuildMode  string `json:"build_mode"`
}

//func (b *buildConfig) String() string {
//	var builder strings.Builder
//	fmt.Fprintf(&builder, "Project: %q\n", b.Project)
//	fmt.Fprintf(&builder, "Android SDK: %q\n", b.AndroidSDK)
//	fmt.Fprintf(&builder, "CMake: %q\n", b.CMake)
//	fmt.Fprintf(&builder, "NDK: %q\n", b.AndroidNDK)
//	fmt.Fprintf(&builder, "ANDROID_ABI: %q\n", b.ABI)
//	fmt.Fprintf(&builder, "ANDROID_ARM_MODE: %q\n", b.ArmMode)
//	fmt.Fprintf(&builder, "ANDROID_ARM_NEON: %q\n", b.Neon)
//	fmt.Fprintf(&builder, "ANDROID_LD: %q\n", b.Ld)
//	fmt.Fprintf(&builder, "ANDROID_PLATFORM: %d\n", b.Platform)
//	fmt.Fprintf(&builder, "ANDROID_STL: %q\n", b.Stl)
//	fmt.Fprintf(&builder, "构建模式: %q", b.BuildMode)
//	return builder.String()
//}

func (b *buildConfig) String() string {
	marshalIndent, _ := json.MarshalIndent(b, "", "  ")
	return string(marshalIndent)
}

func (b *buildConfig) WriteTo(s string) error {
	content, err := json.Marshal(b)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(s, content, 0664)
}

func (b *buildConfig) ReadFrom(s string) error {
	content, err := ioutil.ReadFile(s)
	if err != nil {
		return err
	}
	return json.Unmarshal(content, b)
}

var (
	errAndroidSDKNotFound = errors.New("no Android SDK found")
	errNoVersionDirFound  = errors.New("no version dir found")
	errNoNDKFound         = errors.New("no Android NDK found")
	errNoBuildDirs        = errors.New("no build dirs found")
	errNoPlatforms        = errors.New("no platforms found")
)

var buf = bufio.NewReader(os.Stdin)

//const defaultBuildDir = "cmake-android-build"

const defaultBuildConfig = ".cmake-android-build.cfg"

func findAndroidSDK() (string, error) {
	if env, b := os.LookupEnv("ANDROID_HOME"); b {
		return env, nil
	}
	return "", errAndroidSDKNotFound
}

func findCMake(sdk string) (string, error) {
	if len(sdk) > 0 {
		cmakeDir, err := findMaxVersionDir(filepath.Join(sdk, "cmake"))
		if err == nil {
			return filepath.Join(cmakeDir, "bin", "cmake"), nil
		}
	}
	return exec.LookPath("cmake")
}

func findNDK(sdk string) (string, error) {
	if len(sdk) > 0 {
		ndkDir, err := findMaxVersionDir(filepath.Join(sdk, "ndk"))
		if err == nil {
			return ndkDir, nil
		}
	}
	ndkBuildPath, err := exec.LookPath("ndk-build")
	if err != nil {
		return "", errNoNDKFound
	}
	return filepath.Dir(ndkBuildPath), nil
}

func findPlatformVersions(ndk string, abi int) ([]int, error) {
	infos, err := ioutil.ReadDir(filepath.Join(ndk, "platforms"))
	if err != nil {
		return findPlatformVersionsOverNDKr22(ndk, abi)
	}
	ret := []int(nil)
	for _, info := range infos {
		dirName := info.Name()
		if strings.HasPrefix(dirName, "android-") {
			if code, err := strconv.Atoi(dirName[len("android-"):]); err == nil {
				ret = append(ret, code)
			}
		}
	}
	if len(ret) <= 0 {
		return nil, errNoPlatforms
	} else {
		sort.Sort(sort.IntSlice(ret))
	}
	return ret, nil
}

func findPlatformVersionsOverNDKr22(ndk string, _ int) ([]int, error) {
	jsonContent, err := os.ReadFile(filepath.Join(ndk, "meta", "platforms.json"))
	if err != nil {
		return nil, err
	}
	obj := make(map[string]interface{})
	if err = json.Unmarshal(jsonContent, &obj); err != nil {
		return nil, err
	}

	min, max, err := func(m map[string]interface{}) (min int, max int, err error) {
		if minVal, ok := m["min"]; ok {
			min = int(minVal.(float64))
		} else {
			return 0, 0, fmt.Errorf("min not found: %s", m)
		}

		if maxVal, ok := m["max"]; ok {
			max = int(maxVal.(float64))
		} else {
			return 0, 0, fmt.Errorf("max not found: %s", m)
		}

		return
	}(obj)
	if err != nil {
		return nil, err
	}
	return func(min, max int) []int {
		var ret []int
		for i := min; i <= max; i++ {
			ret = append(ret, i)
		}
		return ret
	}(min, max), nil
}

func findMaxVersionDir(d string) (string, error) {
	file, err := os.Open(d)
	if err != nil {
		return "", err
	}
	defer file.Close()
	children, err := file.Readdir(-1)
	if err != nil {
		return "", err
	}
	var max = struct {
		ver  *semver.Version
		path string
	}{
		ver: nil, path: "",
	}
	for _, info := range children {
		if info.IsDir() {
			newVer := semver.MustParse(info.Name())
			if max.ver == nil || max.ver.LT(newVer) {
				max.ver = &newVer
				max.path = filepath.Join(d, info.Name())
			}
		}
	}
	if len(max.path) == 0 {
		return "", errNoVersionDirFound
	}
	return max.path, nil
}

//func compareVersion(s1, s2 string) int {
//	if s1 == s2 {
//		return 0
//	}
//
//	var pre1, pre2 string
//	var post1, post2 string
//	if index1 := strings.Index(s1, "."); index1 == -1 {
//		pre1 = s1
//	} else {
//		pre1 = s1[:index1]
//		post1 = s1[index1+1:]
//	}
//	if index2 := strings.Index(s2, "."); index2 == -1 {
//		pre2 = s2
//	} else {
//		pre2 = s2[:index2]
//		post2 = s2[index2+1:]
//	}
//	var i1, i2 int
//	i1, _ = strconv.Atoi(pre1)
//	i2, _ = strconv.Atoi(pre2)
//	if i1 == i2 {
//		return compareVersion(post1, post2)
//	} else if i1 > i2 {
//		return 1
//	} else {
//		return -1
//	}
//}

func readInt(txt string) (int, error) {
	if len(txt) > 0 {
		fmt.Println(txt)
	}
	s, err := readString("")
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(s)
}

func readString(txt string) (string, error) {
	if len(txt) > 0 {
		fmt.Println(txt)
	}
	s, err := buf.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(s), nil
}

var androidABIs = []string{
	"armeabi-v7a",
	"armeabi-v7a with NEON",
	"arm64-v8a",
	"x86",
	"x86_64",
}

var androidArmMode = []string{
	"thumb", "arm",
}

var androidArmNeon = []string{
	"", "TRUE", "FALSE",
}

var androidLd = []string{
	"", "lld", "default",
}

var androidStl = []string{
	"c++_static", "c++_shared", "none", "system",
}

var cmakeBuildModes = []string{
	"Debug", "Release", "RelWithDebInfo", "MinSizeRel",
}

var androidVersions = map[int]string{
	1:  "Android 1.0",
	2:  "Android 1.1",
	3:  "Android 1.5",
	4:  "Android 1.6",
	5:  "Android 2.0",
	6:  "Android 2.0.1",
	7:  "Android 2.1",
	8:  "Android 2.2",
	9:  "Android 2.3",
	10: "Android 2.3.3",
	11: "Android 3.0",
	12: "Android 3.1",
	13: "Android 3.2",
	14: "Android 4.0",
	15: "Android 4.0.3",
	16: "Android 4.1",
	17: "Android 4.2",
	18: "Android 4.3",
	19: "Android 4.4",
	20: "Android 4.4W",
	21: "Android 5.0 Lollipop",
	22: "Android 5.1 Lollipop",
	23: "Android 6.0 Marshmallow",
	24: "Android 7.0 Nougat",
	25: "Android 7.1 Nougat",
	26: "Android 8.0 Oreo",
	27: "Android 8.1 Oreo",
	28: "Android 9.0 Pie",
	29: "Android 10.0 Q",
	30: "Android 11.0 R",
	31: "Android 12.0",
}
