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

package progress

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/samber/oops"
	"github.com/spf13/cast"

	"github.com/acyumi/xdoc/component/app"
	"github.com/acyumi/xdoc/component/cloud"
)

type TestClient struct {
	p IProgram
}

func NewTestClient() cloud.Client[any] {
	// 创建模型
	testClient := &TestClient{p: NewProgram(nil)}
	testClient.SetArgs(nil)
	return testClient
}

func (c *TestClient) SetArgs(_ any) {}

func (c *TestClient) GetArgs() any {
	return nil
}

func (c *TestClient) Validate() error {
	return nil
}

func (c *TestClient) DownloadDocuments(dss []*cloud.DocumentSource) error {
	if len(dss) == 0 {
		return oops.New("文档源为空")
	}
	return testProgramAddUpdate(c.p)
}

// 模拟下载任务。
func downloadTask(index int, p IProgram) {
	for i := 1; i <= 100; i++ {
		var style = lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color(cast.ToString(i)))
		app.Sleep(time.Duration(rand.Intn(3000)) * time.Millisecond) // 模拟下载延迟
		link := style.Render(fmt.Sprintf("\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\", "https://baidu.com", "ctrl+单击查看问题"))
		p.Update(fmt.Sprintf("f%d", index), float64(i)/100.0, StatusDownloading, link)
	}
}

// 模拟动态增加文件。
func addFileTask(p IProgram) {
	for i := 0; i < 100; i++ {
		app.Sleep(1 * time.Second) // 模拟动态增加文件的延迟
		p.Add(fmt.Sprintf("add%d", i), fmt.Sprintf("新增文件%d.docx", i))
	}
}

// testProgramAddUpdate 测试框架看不到控制台的效果，所以通过main函数运行。
func testProgramAddUpdate(p IProgram) error {
	fileNames := []string{
		"file1.txt",
		"这是一个非常长的中文文件名超过20个字符.txt",
		"中等长度的中文文件名.txt",
		"短.txt",
		"另一个超长的中文文件名示例.doc",
		"测试文件.zip",
		"图片.png",
		"文档.pdf",
		"数据.csv",
		"压缩包1.tar.gz",
		"压缩包2.tar.gz",
		"压缩包3.tar.gz",
	}

	// 启动多个下载任务
	for i := 0; i < len(fileNames); i++ {
		p.Add(fmt.Sprintf("f%d", i), fileNames[i])
	}
	for i := 0; i < len(fileNames); i++ {
		go downloadTask(i, p)
	}

	// 启动动态增加文件任务
	go addFileTask(p)

	// 启动 BubbleTea
	_, err := p.Run()
	if err != nil {
		fmt.Println("Error running program:", err)
	}
	return oops.Wrap(err)
}
