package progress

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/samber/oops"
	"github.com/spf13/cast"

	"github.com/acyumi/xdoc/component/argument"
	"github.com/acyumi/xdoc/component/cloud"
)

type TestClient struct {
	p IProgram
}

func NewTestClient(args *argument.Args) cloud.Client {
	// 创建模型
	testClient := &TestClient{p: NewProgram(nil)}
	testClient.SetArgs(args)
	return testClient
}

func (c *TestClient) SetArgs(_ *argument.Args) {}

func (c *TestClient) GetArgs() *argument.Args {
	return nil
}

func (c *TestClient) Validate() error {
	return nil
}

func (c *TestClient) DownloadDocuments(_, _ string) error {
	return testProgramAddUpdate(c.p)
}

// 模拟下载任务。
func downloadTask(index int, p IProgram) {
	for i := 1; i <= 100; i++ {
		var style = lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color(cast.ToString(i)))
		cloud.Sleep(time.Duration(rand.Intn(3000)) * time.Millisecond) // 模拟下载延迟
		link := style.Render(fmt.Sprintf("\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\", "https://baidu.com", "ctrl+单击查看问题"))
		p.Update(fmt.Sprintf("f%d", index), float64(i)/100.0, StatusDownloading, link)
	}
}

// 模拟动态增加文件。
func addFileTask(p IProgram) {
	for i := 0; i < 100; i++ {
		cloud.Sleep(1 * time.Second) // 模拟动态增加文件的延迟
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
