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
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "请使用子命令 init 或 build")
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

	default:
		fmt.Fprintln(os.Stderr, "请使用子命令 init 或 build")
		os.Exit(1)
	}
}

func buildProject(args []string) error {
	var dir string

	buildFlags := flag.NewFlagSet("ndk-cmake build", flag.ExitOnError)
	buildFlags.StringVar(&dir, "p", ".", "cmake 工程路径，默认为 $PWD")
	buildFlags.Parse(args)

	cfg := &buildConfig{}
	buildTargetDir := filepath.Join(dir, defaultBuildDir)
	cfgFilePath := filepath.Join(buildTargetDir, defaultBuildConfig)
	if err := cfg.ReadFrom(cfgFilePath); err != nil {
		return err
	}

	cmd := exec.Command(cfg.CMake, "--build", buildTargetDir, "--", "-j", fmt.Sprintf("%d", runtime.NumCPU()))
	fmt.Println(cmd)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func initProject() error {
	state := 0
	sdk := ""
	cmake := ""
	ndk := ""
	project := ""
	abi := 0
	armMode := 0
	neon := 0
	ld := 0
	platform := 0
	stl := 0
	buildMode := 0
	ninja := false
	var err error

	for state != -1 {
		switch state {

		case 0: // 设置 Android SDK 的路径
			sdk, err = findAndroidSDK()
			if err != nil {
				sdk, err = readString("请输入 Android SDK 路径：")
			}
			if err == nil {
				state = 1
			}

		case 1: // 查找 CMake 路径
			cmake, err = findCMake(sdk)
			if err != nil {
				cmake, err = readString("请输入 CMake 路径：")
			}
			if err == nil {
				state = 2
			}

		case 2: // 查找 NDK 路径
			ndk, err = findNDK(sdk)
			if err != nil {
				ndk, err = readString("请输入 NDK 路径：")
			}
			if err == nil {
				state = 3
			}

		case 3: // 查找工程路径
			abs, _ := filepath.Abs(".")
			project, err = readString(fmt.Sprintf("请输入工程路径，默认为 %s：", abs))
			if len(project) == 0 {
				project = abs
			}
			cmakeFile := filepath.Join(project, "CMakeLists.txt")
			if _, err = os.Stat(cmakeFile); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			state = 4

		case 4: // 设置 ANDROID_ABI
			abi, _ = readInt("请输入 ANDROID_ABI，默认为 armeabi-v7a：\n\t0: armeabi-v7a\n\t1: armeabi-v7a with NEON\n\t2: arm64-v8a\n\t3: x86\n\t4: x86_64")
			state = 5

		case 5: // 设置 ANDROID_ARM_MODE
			armMode, _ = readInt("请输入 ANDROID_ARM_MODE，默认为 thumb：\n\t0: thumb\n\t1: arm")
			state = 6

		case 6: // 设置 ANDROID_ARM_NEON
			neon, _ = readInt("请输入 ANDROID_ARM_NEON，默认为不选择：\n\t1: TRUE\n\t2: FALSE")
			state = 7

		case 7: // 设置 ANDROID_LD
			ld, _ = readInt("请输入 ANDROID_LD，默认为不选择：\n\t1: lld\n\t2: default")
			state = 8

		case 8: // 设置 ANDROID_PLATFORM
			platform, _ = readInt("请输入 ANDROID_PLATFORM，默认为 16：")
			if platform < 16 {
				platform = 16
			}
			state = 9

		case 9: // 设置 ANDROID_STL
			stl, _ = readInt("请输入 ANDROID_STL，默认为 c++_static：\n\t0: c++_static\n\t1: c++_shared\n\t2: none\n\t3: system")
			state = 10

		case 10: // 设置 build mode
			buildMode, _ = readInt("请输入构建模式，默认为 Debug：\n\t0: Debug\n\t1: Release\n\t2: RelWithDebInfo\n\t3: MinSizeRel")
			state = -1
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

	buildTargetDir := filepath.Join(project, defaultBuildDir)

	os.RemoveAll(buildTargetDir)
	os.MkdirAll(buildTargetDir, 0775)

	arguments := []string{
		fmt.Sprintf("-DCMAKE_BUILD_TYPE=%s", cfg.BuildMode),
		fmt.Sprintf("-DCMAKE_TOOLCHAIN_FILE=%s", filepath.Join(ndk, "build/cmake/android.toolchain.cmake")),
		fmt.Sprintf("-DANDROID_ABI=%s", cfg.ABI),
		fmt.Sprintf("-DANDROID_ARM_MODE=%s", cfg.ArmMode),
		fmt.Sprintf("-DANDROID_PLATFORM=%d", cfg.Platform),
		fmt.Sprintf("-DANDROID_STL=%s", cfg.Stl),
	}

	if neon != 0 {
		arguments = append(arguments, fmt.Sprintf("-DANDROID_ARM_NEON=%s", cfg.Neon))
	}

	if ld != 0 {
		arguments = append(arguments, fmt.Sprintf("-DANDROID_LD=%s", cfg.Ld))
	}

	if ninja {
		arguments = append(arguments,
			"-DANDROID_TOOLCHAIN=clang",
			fmt.Sprintf("-DCMAKE_MAKE_PROGRAM=%s", filepath.Join(filepath.Dir(cmake), "ninja")),
			"-G", "Android Gradle - Ninja",
		)
	} else {
		arguments = append(arguments,
			"-G", "CodeBlocks - Unix Makefiles",
		)
	}

	arguments = append(arguments, project)

	cmd := exec.Command(cmake, arguments...)
	fmt.Println(cmd)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Dir = buildTargetDir

	if err = cmd.Run(); err != nil {
		return err
	}

	return cfg.WriteTo(filepath.Join(buildTargetDir, defaultBuildConfig))
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

func (b *buildConfig) String() string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "Project: %q\n", b.Project)
	fmt.Fprintf(&builder, "Android SDK: %q\n", b.AndroidSDK)
	fmt.Fprintf(&builder, "CMake: %q\n", b.CMake)
	fmt.Fprintf(&builder, "NDK: %q\n", b.AndroidNDK)
	fmt.Fprintf(&builder, "ANDROID_ABI: %q\n", b.ABI)
	fmt.Fprintf(&builder, "ANDROID_ARM_MODE: %q\n", b.ArmMode)
	fmt.Fprintf(&builder, "ANDROID_ARM_NEON: %q\n", b.Neon)
	fmt.Fprintf(&builder, "ANDROID_LD: %q\n", b.Ld)
	fmt.Fprintf(&builder, "ANDROID_PLATFORM: %d\n", b.Platform)
	fmt.Fprintf(&builder, "ANDROID_STL: %q\n", b.Stl)
	fmt.Fprintf(&builder, "构建模式: %q", b.BuildMode)
	return builder.String()
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
)

var buf = bufio.NewReader(os.Stdin)

const defaultBuildDir = "cmake-android-build"

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
	max := ""
	for _, info := range children {
		if info.IsDir() {
			newPath := filepath.Join(d, info.Name())
			if compareVersion(max, newPath) < 0 {
				max = newPath
			}
		}
	}
	if len(max) == 0 {
		return "", errNoVersionDirFound
	}
	return max, nil
}

func compareVersion(s1, s2 string) int {
	if s1 == s2 {
		return 0
	}

	var pre1, pre2 string
	var post1, post2 string
	if index1 := strings.Index(s1, "."); index1 == -1 {
		pre1 = s1
	} else {
		pre1 = s1[:index1]
		post1 = s1[index1+1:]
	}
	if index2 := strings.Index(s2, "."); index2 == -1 {
		pre2 = s2
	} else {
		pre2 = s2[:index2]
		post2 = s2[index2+1:]
	}
	var i1, i2 int
	i1, _ = strconv.Atoi(pre1)
	i2, _ = strconv.Atoi(pre2)
	if i1 == i2 {
		return compareVersion(post1, post2)
	} else if i1 > i2 {
		return 1
	} else {
		return -1
	}
}

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