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

package cloud

import (
	validation "github.com/go-ozzo/ozzo-validation"

	"github.com/acyumi/xdoc/component/argument"
)

// DocumentSource 文档源信息。
type DocumentSource struct {
	Type  string
	Token string
}

// Client 云客户端。
type Client interface {
	validation.Validatable
	// SetArgs 设置参数，可以在此方法中初始化客户端
	SetArgs(args *argument.Args)
	// GetArgs 获取参数
	GetArgs() *argument.Args
	// 	DownloadDocuments 下载文档，下载过程中可通过实现和创建 Task 来执行批量下载和获取下载进度
	DownloadDocuments([]*DocumentSource) error
}

// Task 云任务。
type Task interface {
	validation.Validatable
	// Run 运行任务
	Run() error
	// Close 关闭任务资源，一般配合defer使用
	Close()
	// Interrupt 中断任务
	Interrupt()
	// Complete 完成任务
	Complete()
}
