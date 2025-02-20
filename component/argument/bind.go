package argument

import (
	"time"

	"github.com/samber/oops"
)

var (
	StartTime time.Time

	ConfigFile        string            // 用于存储 --config 参数的值
	AppID             string            // 飞书应用ID
	AppSecret         string            // 飞书应用密钥
	DocURL            string            // 文档地址
	SaveDir           string            // 文档存放目录(本地)
	FileExtensions    map[string]string // 文档扩展名映射, 用于指定文档下载后的文件类型
	ListOnly          bool              // 是否只列出云文档信息不进行导出下载
	QuitAutomatically bool              // 是否在下载完成后自动退出程序
)

func Validate() error {
	if AppID == "" {
		return oops.New("app-id是必需参数")
	}
	if AppSecret == "" {
		return oops.New("app-secret是必需参数")
	}
	if DocURL == "" {
		return oops.New("url是必需参数")
	}
	if SaveDir == "" {
		return oops.New("dir是必需参数")
	}
	return nil
}
