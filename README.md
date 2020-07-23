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
找到 CMake： /Users/xxx/Library/Android/sdk/cmake/3.10.2.4988404/bin/cmake
找到 NDK： /Users/xxx/Library/Android/sdk/ndk/21.3.6528147
找到工程目录： /Users/xxx/dev/source/cprojects/home/hello_world
请输入 CMake 生成文件目录，默认为 cmake-android-build：

请输入 ANDROID_ABI，默认为 armeabi-v7a：
        1: armeabi-v7a with NEON
        2: arm64-v8a
        3: x86
        4: x86_64
2
请输入 ANDROID_LD，默认为不选择：
        1: lld
        2: default

请输入 ANDROID_PLATFORM，默认为 16：

请输入 ANDROID_STL，默认为 c++_static：
        0: c++_static
        1: c++_shared
        2: none
        3: system

请输入构建模式，默认为 Debug：
        1: Release
        2: RelWithDebInfo
        3: MinSizeRel
1
--------------------------------
Project: "/Users/xxx/dev/source/cprojects/home/hello_world"
Android SDK: "/Users/xxx/Library/Android/sdk"
CMake: "/Users/xxx/Library/Android/sdk/cmake/3.10.2.4988404/bin/cmake"
NDK: "/Users/xxx/Library/Android/sdk/ndk/21.3.6528147"
ANDROID_ABI: "arm64-v8a"
ANDROID_ARM_MODE: "thumb"
ANDROID_ARM_NEON: ""
ANDROID_LD: ""
ANDROID_PLATFORM: 16
ANDROID_STL: "c++_static"
构建模式: "Release"
--------------------------------
/Users/xxx/Library/Android/sdk/cmake/3.10.2.4988404/bin/cmake -DCMAKE_BUILD_TYPE=Release -DCMAKE_VERBOSE_MAKEFILE=ON -DCMAKE_TOOLCHAIN_FILE=/Users/xxx/Library/Android/sdk/ndk/21.3.6528147/build/cmake/android.toolchain.cmake -DANDROID_ABI=arm64-v8a -DANDROID_ARM_MODE=thumb -DANDROID_PLATFORM=16 -DANDROID_STL=c++_static -G CodeBlocks - Unix Makefiles /Users/xxx/dev/source/cprojects/home/hello_world
-- Check for working C compiler: /Users/xxx/Library/Android/sdk/ndk/21.3.6528147/toolchains/llvm/prebuilt/darwin-x86_64/bin/clang
-- Check for working C compiler: /Users/xxx/Library/Android/sdk/ndk/21.3.6528147/toolchains/llvm/prebuilt/darwin-x86_64/bin/clang -- works
-- Detecting C compiler ABI info
-- Detecting C compiler ABI info - done
-- Detecting C compile features
-- Detecting C compile features - done
-- Check for working CXX compiler: /Users/xxx/Library/Android/sdk/ndk/21.3.6528147/toolchains/llvm/prebuilt/darwin-x86_64/bin/clang++
-- Check for working CXX compiler: /Users/xxx/Library/Android/sdk/ndk/21.3.6528147/toolchains/llvm/prebuilt/darwin-x86_64/bin/clang++ -- works
-- Detecting CXX compiler ABI info
-- Detecting CXX compiler ABI info - done
-- Detecting CXX compile features
-- Detecting CXX compile features - done
-- Configuring done
-- Generating done
-- Build files have been written to: /Users/xxx/dev/source/cprojects/home/hello_world/cmake-android-build
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
找到目标目录： [/Users/xxx/dev/source/cprojects/home/hello_world/cmake-android-build]
...
```

最后在 `cmake-android-build` 目录中生成目标可执行文件。

### 运行

使用我的 [arun](https://github.com/ClarkGuan/arun) 工具即可：

```bash
$ arun cmake-android-build/hello_world
prepare to push /Users/xxx/dev/source/cprojects/home/hello_world/cmake-android-build/hello_world to device
/Users/xxx/dev/source/cprojects/home/hello_world/cmake-android-build/hello_world: 1 file pushed, 0 skipped. 46.5 MB/s (4713888 bytes in 0.097s)
[程序输出如下]
Hello world!!!
[程序执行完毕]
    0m00.03s real     0m00.02s user     0m00.00s system
```
