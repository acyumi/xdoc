package argument

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/samber/oops"

	"github.com/acyumi/doc-exporter/component/constant"
)

type Args struct {
	StartTime time.Time // 程序开始时间

	ConfigFile        string                                // 用于存储 --config 参数的值
	Verbose           bool                                  // 是否显示详细日志
	AppID             string                                // 应用ID
	AppSecret         string                                // 应用密钥
	DocURL            string                                // 文档地址
	SaveDir           string                                // 文档存放目录(本地)
	FileExtensions    map[constant.DocType]constant.FileExt // 文档扩展名映射, 用于指定文档下载后的文件类型
	ListOnly          bool                                  // 是否只列出云文档信息不进行导出下载
	QuitAutomatically bool                                  // 是否在下载完成后自动退出程序
}

func (a Args) Validate() error {
	return oops.Code("InvalidArgument").Wrap(
		validation.ValidateStruct(&a,
			validation.Field(&a.AppID, validation.Required.Error("app-id是必需参数")),
			validation.Field(&a.AppSecret, validation.Required.Error("app-secret是必需参数")),
			validation.Field(&a.DocURL, validation.Required.Error("url是必需参数")),
			validation.Field(&a.SaveDir, validation.Required.Error("dir是必需参数")),
		))
}

func (a *Args) SetFileExtensions(fes map[string]string) {
	if a.FileExtensions == nil {
		a.FileExtensions = map[constant.DocType]constant.FileExt{}
	}
	for k, v := range fes {
		a.FileExtensions[constant.DocType(k)] = constant.FileExt(v)
	}
}
