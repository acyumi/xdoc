package cloud

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation"

	"github.com/acyumi/doc-exporter/component/argument"
)

var Sleep = func(duration time.Duration) { time.Sleep(duration) } // 睡眠等待函数

// Client 云客户端。
type Client interface {
	validation.Validatable
	// SetArgs 设置参数，可以在此方法中初始化客户端
	SetArgs(args *argument.Args)
	// GetArgs 获取参数
	GetArgs() *argument.Args
	// 	DownloadDocuments 下载文档，下载过程中可通过实现和创建 Task 来执行批量下载和获取下载进度
	DownloadDocuments(typ, token string) error
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
