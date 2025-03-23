package feishu

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	larkdrive "github.com/larksuite/oapi-sdk-go/v3/service/drive/v1"
	"github.com/samber/oops"
	"github.com/xlab/treeprint"

	"github.com/acyumi/xdoc/component/argument"
	"github.com/acyumi/xdoc/component/cloud"
	"github.com/acyumi/xdoc/component/constant"
)

type DocumentInfo struct {
	Name          string           `json:"name"          yaml:"name" bson:"name" gorm:"dddd"` // 文档名
	Type          constant.DocType `json:"type"          yaml:"type" bson:"name" gorm:"dddd"` // 文档类型
	Token         string           `json:"token"`                                             // 文档token
	FileExtension constant.FileExt `json:"fileExtension"`                                     // 文件扩展名，如果是目录type=folder，则为空
	CanDownload   bool             `json:"canDownload"`                                       // 是否可下载

	DownloadDirectly bool   `json:"downloadDirectly"` // 是否使用【下载文件】API直接下载
	URL              string `json:"url"`              // 在浏览器中查看的链接

	NodeToken string `json:"nodeToken"` // 知识节点ID
	SpaceID   string `json:"spaceId"`   // 知识空间ID

	FilePath string // 文件保存路径
}

type DocumentNode struct {
	DocumentInfo `json:",inline"`
	Children     []*DocumentNode `json:"children"`
}

func (di *DocumentInfo) GetFileName() string {
	return fmt.Sprintf("%s.%s", di.Name, di.FileExtension)
}

type exportResult struct {
	*DocumentInfo
	result *larkdrive.ExportTask // 如果 DocumentInfo.DownloadDirectly=true，则 result 为空
}

func cleanName(name string) string {
	// windows目录或文件名不允许使用的字符比其他系统更多
	// windows目录或文件名不允许使用这些字符: \/:*?"<>|
	name = strings.ReplaceAll(name, `\`, "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, ":", "_")
	name = strings.ReplaceAll(name, "*", "_")
	name = strings.ReplaceAll(name, "?", "_")
	name = strings.ReplaceAll(name, `"`, "_")
	name = strings.ReplaceAll(name, "<", "_")
	name = strings.ReplaceAll(name, ">", "_")
	name = strings.ReplaceAll(name, "|", "_")
	return name
}

func setFileExtension(di *DocumentNode, args *argument.Args) {
	// 如果是file，则需要通过文件名获取文件类型，再继续下成的switch处理
	if di.Type == constant.DocTypeFile {
		di.DownloadDirectly = true
		ext := filepath.Ext(di.Name)
		if len(ext) > 0 {
			e := constant.DocType(ext[1:])
			switch e {
			case constant.DocTypeXlsx, constant.DocTypeXls:
				di.Type = constant.DocTypeSheet
			default:
				di.Type = e
			}
			di.Name = di.Name[:len(di.Name)-len(ext)]
		}
	}
	var setOrDefault = func(def constant.FileExt) {
		di.FileExtension = args.FileExtensions[di.Type]
		if di.FileExtension == "" {
			di.FileExtension = def
		}
	}
	switch di.Type {
	// 具体查看支持的类型: https://open.feishu.cn/document/server-docs/docs/drive-v1/export_task/export-user-guide
	// 目前为四种:
	// docx：新版飞书文档。支持导出扩展名为 docx 和 pdf 格式的文件。
	// doc：旧版飞书文档。支持导出扩展名为 docx 和 pdf 的文件。已不推荐使用。
	// sheet：飞书电子表格。支持导出扩展名为 xlsx 和 csv 的文件。
	// bitable：飞书多维表格。支持导出扩展名为 xlsx 和 csv 格式的文件。
	case constant.DocTypeDocx, constant.DocTypeDoc:
		di.CanDownload = true
		setOrDefault(constant.FileExtDocx)
	case constant.DocTypeBitable, constant.DocTypeSheet:
		di.CanDownload = true
		setOrDefault(constant.FileExtXlsx)
	default:
		di.CanDownload = false
		setOrDefault(constant.FileExt(di.Type))
	}
}

func documentTreeToInfoList(di *DocumentNode, saveDir string) []*DocumentInfo {
	if di.Type == constant.DocTypeFolder {
		di.FilePath = filepath.Join(saveDir, di.Name)
	} else {
		di.FilePath = filepath.Join(saveDir, di.Name+"."+string(di.FileExtension))
	}
	var infoList []*DocumentInfo
	infoList = append(infoList, &di.DocumentInfo)
	for _, child := range di.Children {
		infoList = append(infoList, documentTreeToInfoList(child, filepath.Join(saveDir, di.Name))...)
	}
	return infoList
}

// 递归打印目录结构及文件名，打印过程中会调整空文件名为"未命名xxxn.xxx"格式
// 返回值：tc: totalCount, cdc: canDownloadCount。
func printTree(logWriter io.Writer, tree treeprint.Tree, di *DocumentNode, totalCount, canDownloadCount int) (tc, cdc int) {
	if totalCount == 0 {
		root := tree
		defer func() {
			_, _ = fmt.Fprint(logWriter, root.String())
		}()
		totalCount++
		name := getName(di.Name, di.Type, map[string]int{})
		suffix := string(di.FileExtension)
		if di.CanDownload {
			canDownloadCount++
		} else {
			suffix += "（不可下载）"
		}
		_, _ = fmt.Fprint(logWriter, "\n")
		if di.Type != constant.DocTypeFolder {
			tree.AddNode(name + "." + suffix) // 文件
		}
		if len(di.Children) > 0 {
			tree = tree.AddBranch(name) // 目录
		}
	}
	temp := map[string]int{}
	for _, child := range di.Children {
		totalCount++
		suffix := string(child.FileExtension)
		if child.CanDownload {
			canDownloadCount++
		} else {
			suffix += "（不可下载）"
		}
		child.Name = getName(child.Name, child.Type, temp)
		if len(child.Children) > 0 {
			if child.Type != constant.DocTypeFolder {
				tree.AddNode(child.Name + "." + suffix) // 文件
			}
			branch := tree.AddBranch(child.Name) // 目录
			totalCount, canDownloadCount = printTree(logWriter, branch, child, totalCount, canDownloadCount)
			continue
		}
		if child.Type == constant.DocTypeFolder {
			tree.AddNode(child.Name) // 目录
			continue
		}
		tree.AddNode(child.Name + "." + suffix) // 文件
	}
	return totalCount, canDownloadCount
}

func getName(name string, typ constant.DocType, duplicateNameIndexMap map[string]int) string {
	if name != "" {
		index, ok := duplicateNameIndexMap[name]
		if !ok {
			duplicateNameIndexMap[name] = 0
			return name
		}
		duplicateNameIndexMap[name] = index + 1
		return fmt.Sprintf("%s%d", name, index+1)
	}
	unnamedType := string("[Unnamed]" + typ)
	unnamedIndex := duplicateNameIndexMap[unnamedType] + 1
	duplicateNameIndexMap[unnamedType] = unnamedIndex
	switch typ {
	case constant.DocTypeDocx:
		return fmt.Sprintf("未命名新版文档%d", unnamedIndex)
	case constant.DocTypeDoc:
		return fmt.Sprintf("未命名旧版文档%d", unnamedIndex)
	case constant.DocTypeSheet:
		return fmt.Sprintf("未命名电子表格%d", unnamedIndex)
	case constant.DocTypeBitable:
		return fmt.Sprintf("未命名多维表格%d", unnamedIndex)
	case constant.DocTypeMindNote:
		return fmt.Sprintf("未命名思维笔记%d", unnamedIndex)
	case constant.DocTypeSlides:
		return fmt.Sprintf("未命名幻灯片%d", unnamedIndex)
	default:
		return fmt.Sprintf("未命名飞书文档%d", unnamedIndex)
	}
}

func doExportAndDownload(task cloud.Task) error {
	err := task.Validate()
	if err != nil {
		return oops.Wrap(err)
	}
	defer cloud.Sleep(time.Second * 2)
	defer task.Close()

	err = task.Run()
	return oops.Wrap(err)
}
