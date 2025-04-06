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

package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/acyumi/xdoc/component/argument"
)

// command 命令接口。
type command interface {
	// init 初始化命令。
	init(vip *viper.Viper, args *argument.Args)
	// bind 绑定命令参数。
	// 注意：不要在此文件中读取和使用配置，应该仅做参数绑定。
	//       参数的读取一般放在运行阶段。
	bind() error
	// get 获取命令。
	get() *cobra.Command
	// children 获取子命令集。
	// 初始化子命令时会也调用此方法（注意保持多次调用的一致性）。
	children() []command
	// exec 执行命令。
	// 如果需要提供给父命令执行，可以将命令执行逻辑写在此方法中（注意保持逻辑纯净）。
	exec() error
}
