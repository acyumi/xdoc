// Copyright 2025 acyumi <417064257@qq.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build !test

package main

import (
	"fmt"
	"os"

	"github.com/acyumi/xdoc/cmd"
)

// SDK 使用文档：https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/server-side-sdk/golang-sdk-guide/preparations
func main() {
	// TODO 支持-g --generate-config生成默认配置文件到当前目录，当打印使用命令辅助使用
	// TODO 导出和下载的协程数量支持配置
	// TODO 执行日志输出到文件
	// TODO docx 和 pdf 下载后自动去除水印
	// TODO 下载UI程序支持快速滚动到顶部和底部、按ctrl+↑向上滚动10%、按ctrl+↓向下滚动10%
	// TODO 补充更多使用说明，如主程序参数、下载UI状态下的快捷键说明
	// TODO 支持跳过已下载文件（将下载进度保存到缓存文件中，每次执行都做一下检查）
	// 执行命令
	err := cmd.Execute()
	if err != nil {
		fmt.Println("----------------------------------------------")
		if cmd.GetArgs().Verbose {
			_, _ = fmt.Fprintf(os.Stderr, "%+v\n", err)
		} else {
			_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		}
		os.Exit(1)
	}
}
