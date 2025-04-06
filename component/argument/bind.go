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

package argument

import (
	"time"
)

// Args 程序参数，优先级：命令行参数 > 环境变量 > 配置文件 > 默认值。
type Args struct {
	StartTime time.Time // 程序开始时间

	ConfigFile        string // 用于存储 --config 参数的值
	Verbose           bool   // 是否显示详细日志
	GenerateConfig    bool   // 是否在程序目录生成config.yaml
	QuitAutomatically bool   // 是否在程序跑完后自动退出
}

func (a Args) Validate() error {
	// TODO
	return nil
}
