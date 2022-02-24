# ndk-cmake

该工具主要是为了简化 [CMake 编译的参数传递](https://developer.android.com/ndk/guides/cmake)而诞生的。

主要分 init 和 build 两大子命令：

1. init 子命令用于创建 CMake-generated project binary tree
2. build 子命令用于构建不同 CMake-generated project binary tree 目录的内容

## 安装

```bash
$ git clone https://github.com/ClarkGuan/ndk-cmake
$ cd ndk-cmake
$ go install
```

需要依赖：

1. Golang 环境
2. 将 GOBIN 或 GOPATH/bin 添加到 PATH 环境变量中

## 使用

### 创建 CMake-generated project binary tree

创建一个 cpp 的 hello world 工程：

main.cpp 内容如下：

```cpp
#include <iostream>
#include <android/log.h>

using namespace std;

#define LOGI(TAG, ...) ((void)__android_log_print(ANDROID_LOG_INFO, TAG, __VA_ARGS__))
#define TAG "hello-world"

int main(int argc, char **argv, char **env) {
    cout << "Hello world!!!" << endl;
    LOGI("hello-world", "Hello world!!!");
    return 0;
}
```

CMakeLists.txt 内容如下：

```
cmake_minimum_required(VERSION 3.6)
project(hello_world)
set(CMAKE_CXX_STANDARD 14)
add_executable(hello_world main.cpp)
target_link_libraries(hello_world log)
```

在该目录下运行命令

```bash
$ ndk-cmake init
找到 Android SDK： /Users/xxx/Library/Android/sdk
找到 CMake： /Users/xxx/Library/Android/sdk/cmake/3.18.1/bin/cmake
找到 NDK： /Users/xxx/Library/Android/sdk/ndk/21.4.7075529
找到工程目录： /Users/xxx/source/cpro/home/hello
请输入 ANDROID_ABI，默认为 armeabi-v7a：
        1: armeabi-v7a with NEON
        2: arm64-v8a
        3: x86
        4: x86_64
2
请输入 ANDROID_LD，默认为不选择：
        1: lld
        2: default

请输入 ANDROID_PLATFORM，
        16: Android 4.1
        17: Android 4.2
        18: Android 4.3
        19: Android 4.4
        21: Android 5.0 Lollipop
        22: Android 5.1 Lollipop
        23: Android 6.0 Marshmallow
        24: Android 7.0 Nougat
        26: Android 8.0 Oreo
        27: Android 8.1 Oreo
        28: Android 9.0 Pie
        29: Android 10.0 Q
        30: Android 11.0 R
默认为 21:

请输入 ANDROID_STL，默认为 c++_static：
        1: c++_shared
        2: none
        3: system

请输入构建模式，默认为 Debug：
        1: Release
        2: RelWithDebInfo
        3: MinSizeRel
1
请输入 CMake 生成文件目录，默认为 cmake-arm64-v8a-c++_static-android21-Release：

--------------------------------
{
  "project": "/Users/xxx/source/cpro/home/hello",
  "sdk": "/Users/xxx/Library/Android/sdk",
  "cmake": "/Users/xxx/Library/Android/sdk/cmake/3.18.1/bin/cmake",
  "ndk": "/Users/xxx/Library/Android/sdk/ndk/21.4.7075529",
  "abi": "arm64-v8a",
  "arm_mode": "thumb",
  "neon": "",
  "ld": "",
  "platform": 21,
  "stl": "c++_static",
  "build_mode": "Release"
}
--------------------------------
/Users/xxx/Library/Android/sdk/cmake/3.18.1/bin/cmake -DCMAKE_BUILD_TYPE=Release -DCMAKE_VERBOSE_MAKEFILE=ON -DCMAKE_TOOLCHAIN_FILE=/Users/xxx/Library/Android/sdk/ndk/21.4.7075529/build/cmake/android.toolchain.cmake -DANDROID_ABI=arm64-v8a -DANDROID_ARM_MODE=thumb -DANDROID_PLATFORM=21 -DANDROID_STL=c++_static -G CodeBlocks - Unix Makefiles /Users/xxx/source/cpro/home/hello
-- Detecting C compiler ABI info
-- Detecting C compiler ABI info - done
-- Check for working C compiler: /Users/xxx/Library/Android/sdk/ndk/21.4.7075529/toolchains/llvm/prebuilt/darwin-x86_64/bin/clang - skipped
-- Detecting C compile features
-- Detecting C compile features - done
-- Detecting CXX compiler ABI info
-- Detecting CXX compiler ABI info - done
-- Check for working CXX compiler: /Users/xxx/Library/Android/sdk/ndk/21.4.7075529/toolchains/llvm/prebuilt/darwin-x86_64/bin/clang++ - skipped
-- Detecting CXX compile features
-- Detecting CXX compile features - done
-- Configuring done
-- Generating done
-- Build files have been written to: /Users/xxx/source/cpro/home/hello/cmake-arm64-v8a-c++_static-android21-Release
```

主要做几件事：

1. 确定 Android SDK 的位置
2. 确定 Android NDK 的位置
3. 确定使用的 cmake 命令的位置
4. 确定 cmake 命令的输入参数

整个过程采用交互的方式，你也不用记乱七八糟的名字了。

### 交叉编译

编译很简单，只需要运行：

```bash
$ ndk-cmake build
找到目标目录： [/Users/xxx/source/cpro/home/hello/cmake-arm64-v8a-c++_static-android21-Release]
...
```

最后在 `cmake-arm64-v8a-c++_static-android21-Release` 目录中生成目标可执行文件。

### 运行

使用我的 [arun](https://github.com/ClarkGuan/arun) 工具即可：

```bash
$ arun cmake-arm64-v8a-c++_static-android21-Release/hello
Hello world!!!
============================
[exit status:(0)]
    0m00.02s real     0m00.01s user     0m00.02s system
```

## 其他

1. 寻找 Android SDK 路径时使用 `$ANDROID_HOME` 环境变量，如果没有，会让用户手动输入
2. 优先使用 Android SDK 中的 NDK 和 CMake 工具；如果没有下载，需要提前下载好
3. 由于 NDKr22 之后对于官方 CMake 的支持加强了，因此本工具目前仅支持最大的 NDK 版本为 r21
