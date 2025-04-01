#!/usr/bin/env bash
# Copyright 2025 acyumi <417064257@qq.com>
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

###
# 为什么不用Makefile，因为windows系统下默认不支持make，需要自行安装，还挺麻烦
# 而大家基本上都会安装git，有gitbash就可以用bash执行shell脚本了
# 或者后续考虑研究使用 https://github.com/go-task/task
###

# 参数功能定义
# 0、init: 安装构建依赖工具，如github.com/incu6us/goimports-reviser/v3@latest
# 1、build: 构建go程序，指向当前shell脚本所在目录下的output目录，构建多种系统的二进制文件
# 1.1、windows: xdoc.exe
# 1.2、linux: xdoc
# 1.3、macos: xdoc.darwin
# 2、clean: 清理output目录
# 3、run: 使用 go run 运行 main.go，运行时指定配置文件，默认为当前shell脚本所在目录下的config.yaml
# 4、test: 运行单元测试
# 5、fmt: 格式化代码
# 5.1、执行: gofmt -w -r 'interface{} -> any' . 2>&1
# 5.2、执行: goimports-reviser -project-name "go.mod中的module值" ./... 2>&1
# 5.2、执行: go mod tidy
# 6、lint: 运行golangci-lint
# 6.1、执行: golangci-lint run ./...
# 7、mock: 生成接口的mock实现代码
# 8、help: 基于以上的功能，提供帮助文档
# 使用示例：
# ./builder.sh build
# ./builder.sh fmt

# 任何命令失败（返回非0）立即终止脚本（-e）
# 管道命令中任意环节失败都会导致整个管道失败（pipefail）
set -eo pipefail

# 脚本所在目录
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" &>/dev/null && pwd -P)
# 构建输出目录
BUILD_DIR="${SCRIPT_DIR}/output"
# 依赖列表，格式 "工具名:库@版本"
DEPENDENCIES=(
    # 代码格式化工具
    "goimports-reviser:github.com/incu6us/goimports-reviser/v3@v3.8.2"
    # 标签对齐工具
    "tagalign:github.com/4meepo/tagalign/cmd/tagalign@v1.4.2"
    # 代码检测工具
    "golangci-lint:github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.6"
    # Mock生成工具
    "mockery:github.com/vektra/mockery/v2@v2.53.2"
    # License添加工具
    "addlicense:github.com/google/addlicense@latest"
)

# 获取工具库版本
function get_tool_repo_version() {
    local tool_name="$1"
    local repo_version=""

    # 遍历所有依赖项查找对应仓库地址
    for entry in "${DEPENDENCIES[@]}"; do
        IFS=':' read -r tool repo <<< "$entry"
        if [[ "$tool" == "$tool_name" ]]; then
            repo_version="$repo"
            break
        fi
    done

    # 返回"库@版本"
    echo "$repo_version"
}

# 安装工具依赖
function install_tool() {
    local tool_name="$1"
    local repo_version=`get_tool_repo_version "${tool_name}"`

    if [[ -z "$repo_version" ]]; then
        echo "错误：未找到工具 $tool_name 的依赖信息"
        exit 1
    fi

    # 提取版本号显示
    local version="${repo_version#*@}"
    echo "正在安装 ${tool_name}@${version}"
    # echo go install "$repo_version"
    go install "$repo_version"
}

# 检查工具是否存在并安装指定版本
function check_and_install_tool() {
    local tool_name="$1"
    local args="$2"
    local match_str="$3"
    local cmd=`basename "$tool_name"`
    local repo_version=`get_tool_repo_version "$tool_name"`
    local expected_version="${repo_version#*@}"
    # 普通工具通过--version检查版本
    if [[ -z "$args" ]]; then
        args="--version"
    fi
    if [[ -z "$match_str" ]]; then
        match_str="$expected_version"
    fi

    # 检查命令是否存在
    if ! command -v "$cmd" &>/dev/null; then
        echo "警告: 找不到 $tool_name 命令"
        install_tool "$tool_name"
        return
    fi

    local current_version=`$cmd "$args" 2>&1`
    # echo "当前版本: $current_version"
    if [[ "$current_version" != *"$match_str"* ]]; then
        echo "警告: $tool_name 当前版本不匹配，需要 $expected_version"
        install_tool "$tool_name"
    fi
}


# 安装依赖
function init() {
    echo "开始安装构建依赖工具..."
    for entry in "${DEPENDENCIES[@]}"; do
        IFS=':' read -r tool repo <<< "$entry"
        install_tool "$tool"
    done
    echo "所有依赖安装完成"
}

# 构建多平台二进制文件
function build() {
    mkdir -p "${BUILD_DIR}"

    platforms=(
        "windows/amd64"
        "windows/arm64"
        "linux/amd64"
        "linux/arm64"
        "darwin/amd64"
        "darwin/arm64"
    )

    for platform in "${platforms[@]}"; do
        GOOS=${platform%/*}
        GOARCH=${platform#*/}
        output_name="xdoc"

        if [ "${GOARCH}" = "arm64" ]; then
            output_name+=".arm64"
        fi
        if [ "${GOOS}" = "windows" ]; then
            output_name+=".exe"
        elif [ "${GOOS}" = "darwin" ]; then
            output_name+=".darwin"
        fi

        echo "构建 ${GOOS}/${GOARCH} -> ${output_name}"
        env GOOS="${GOOS}" GOARCH="${GOARCH}" go build -o "${BUILD_DIR}/${output_name}" "${SCRIPT_DIR}/main.go"
    done

    echo "构建完成，输出目录: ${BUILD_DIR}"
}

# 运行测试
function test() {
    echo "运行单元测试统计覆盖率--------------------------------"
    # 跑单测并统计覆盖率到html
    # main.go添加了!test构建标签，这里用于将main.go排除到单测覆盖之外
    go test -tags=test -race -coverprofile=coverage.out -covermode=atomic "${SCRIPT_DIR}/..."
    go tool cover -html=coverage.out -o coverage.html
    echo "覆盖率存放在coverage.html中---------------------------"
}

# 运行程序
function run() {
    local config_file="${SCRIPT_DIR}/config.yaml"
    if [ ! -f "${config_file}" ]; then
        echo "警告: 配置文件 ${config_file} 不存在"
    fi
    go run "${SCRIPT_DIR}/main.go" --config "${config_file}"
}

# 格式化代码
function fmt() {
    # 定位go.mod文件
    local go_mod_file="${SCRIPT_DIR}/go.mod"

    # 检查文件是否存在
    if [ ! -f "${go_mod_file}" ]; then
        echo "错误: go.mod 文件不存在于 ${SCRIPT_DIR}"
        exit 1
    fi

    # 提取模块名称
    local module_name=$(awk '/^module / {print $2; exit}' "${go_mod_file}")  # 精确匹配module行

    # 验证模块名有效性
    if [[ -z "${module_name}" ]]; then
        echo "错误: 无法从 go.mod 中解析模块名"
        exit 1
    fi

    echo "检测到项目模块名: ${module_name}"
    echo "执行 gofmt..."
    gofmt -w -r 'interface{} -> any' "${SCRIPT_DIR}" 2>&1

    echo "整理 imports..."
    echo "执行 goimports-reviser -project-name "${module_name}" ./... 2>&1"
    check_and_install_tool "goimports-reviser"
    goimports-reviser -project-name "${module_name}" ./... 2>&1

    echo "整理 tag (对齐、排序)，注意排序应与 .golangci.yaml 中配置的 tagalign 的配置一致..."
    check_and_install_tool "tagalign" "-V=full" "devel"
    # 如果有代码修改，tagalign 会返回非0结果让脚本直接结束，所以加上 || true
    result=`tagalign -fix -sort -order "json,yaml,yml,toml,xml,mapstructure,binding,validate" -strict ./... 2>&1 || true`
    if [[ -n "${result}" ]]; then
        # 将 result 中的 tag is not aligned, should be 改为 tag is not aligned, modify to
        result=${result//tag is not aligned, should be/tag未对齐, 已修改为}
        echo "tagalign 修改如下:"
        echo "${result}"
    fi

    echo "检查添加 license..."
    addlicense -c "acyumi <417064257@qq.com>" .

    echo "整理 go.mod..."
    go mod tidy
}

# 静态代码分析
function lint() {
    # 检查 golangci-lint 的版本，如果不是想要的版本（$LINT_VERSION），则退出
    # golangci-lint --version 的结果参考如下：
    # golangci-lint has version v1.56.2 built with go1.22.0 from (unknown, mod sum: "h1:dgQzlWHgNbCqJjuxRJhFEnHDVrrjuTGQHJ3RIZMpp/o=") on (unknown)
    check_and_install_tool "golangci-lint"
    # 使用 sed 的目的是 将路径中的反斜杠替换为正斜杠，否则在 Windows 下不能点击直达代码片段
    if [[ "$1" == "--verbose" ]]; then
        # \是转义符号，用\\\\才能打印出两个\
        echo "执行 golangci-lint run ./... $1 | sed -E '/^[^:]+\.go:[0-9]+(:[0-9]+)?/ s#\\\\#/#g'"
        golangci-lint run ./... $1 | sed -E '/^[^:]+\.go:[0-9]+(:[0-9]+)?/ s#\\#/#g'
    else
        echo "执行 golangci-lint run ./... | sed -E '/^[^:]+\.go:[0-9]+(:[0-9]+)?/ s#\\\\#/#g'"
        golangci-lint run ./... | sed -E '/^[^:]+\.go:[0-9]+(:[0-9]+)?/ s#\\#/#g'
    fi
}

# 清理构建目录
# 函数名不用 mockery 否则会死循环
function do_mockery() {
    echo "执行 mockery"
    # v3版本使用: check_and_install_tool "mockery" "version"
    check_and_install_tool "mockery" "--version"
    mockery
}

# 清理构建目录
function clean() {
    echo "清理构建目录: ${BUILD_DIR}"
    rm -rf "${BUILD_DIR}"
}

# 清理构建目录
function progress() {
    echo "测试进度条程序--------------------------------"
    go run "${SCRIPT_DIR}/main.go" export --url https://progress.test/xx/yy --app-id "1" --app-secret "2"
}

# 帮助文档
function usage() {
    echo "Usage: $0 <command>"
    echo ""
    echo "Available commands:"
    echo "  init    安装构建依赖工具 (如goimports-reviser)"
    echo "  build   构建多平台二进制文件到 build 目录"
    echo "  test    运行单元测试"
    echo "  run     使用默认配置运行程序"
    echo "  fmt     格式化代码并整理依赖"
    echo "  lint    运行静态代码分析"
    echo "  mockery 生成接口mock实现代码"
    echo "  clean   清理构建目录"
    echo "  help    显示帮助信息"
    echo ""
    exit 1
}

# 参数解析
case "$1" in
    init)
        init
        ;;
    build)
        build
        ;;
    clean)
        clean
        ;;
    run)
        run
        ;;
    test)
        if [[ "$2" != "-t" ]]; then
            fmt
            lint $2
        fi
        test
        ;;
    fmt)
        fmt
        ;;
    lint)
        fmt
        lint $2
        ;;
    mockery)
        do_mockery
        fmt
        ;;
    progress)
        progress
        ;;
    help|--help|-h)
        usage
        ;;
    *)
        if [ -z "$1" ]; then
            usage
        else
            echo "错误: 未知命令 '$1'"
            echo ""
            usage
        fi
        ;;
esac