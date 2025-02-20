package feishu

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	teaProgress "github.com/charmbracelet/bubbles/progress"
	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkdrive "github.com/larksuite/oapi-sdk-go/v3/service/drive/v1"
	"github.com/samber/lo"
	"github.com/samber/oops"

	"acyumi.com/feishu-doc-exporter/component/argument"
	"acyumi.com/feishu-doc-exporter/component/progress"
)

type DocumentInfo struct {
	Name          string `json:"name"`          // 文档名
	Type          string `json:"type"`          // 文档类型
	Token         string `json:"token"`         // 文档token
	FileExtension string `json:"fileExtension"` // 文件扩展名，如果是目录type=folder，则为空
	CanDownload   bool   `json:"canDownload"`   // 是否可下载

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

func setFileExtension(di *DocumentNode) {
	// 如果是file，则需要通过文件名获取文件类型，再继续下成的switch处理
	if di.Type == "file" {
		di.DownloadDirectly = true
		ext := filepath.Ext(di.Name)
		if len(ext) > 0 {
			e := ext[1:]
			switch e {
			case "xlsx", "xls":
				di.Type = "sheet"
			default:
				di.Type = e
			}
			di.Name = di.Name[:len(di.Name)-len(ext)]
		}
	}
	var setOrDefault = func(d string) {
		di.FileExtension = argument.FileExtensions[di.Type]
		if di.FileExtension == "" {
			di.FileExtension = d
		}
	}
	switch di.Type {
	// 具体查看支持的类型: https://open.feishu.cn/document/server-docs/docs/drive-v1/export_task/export-user-guide
	// 目前为四种:
	// docx：新版飞书文档。支持导出扩展名为 docx 和 pdf 格式的文件。
	// doc：旧版飞书文档。支持导出扩展名为 docx 和 pdf 的文件。已不推荐使用。
	// sheet：飞书电子表格。支持导出扩展名为 xlsx 和 csv 的文件。
	// bitable：飞书多维表格。支持导出扩展名为 xlsx 和 csv 格式的文件。
	case "docx", "doc":
		di.CanDownload = true
		setOrDefault("docx")
	case "bitable", "sheet":
		di.CanDownload = true
		setOrDefault("xlsx")
	default:
		di.CanDownload = false
		setOrDefault(di.Type)
	}
}

func documentTreeToInfoList(di *DocumentNode, saveDir string) []*DocumentInfo {
	if di.Type == "folder" {
		di.FilePath = filepath.Join(saveDir, di.Name)
	} else {
		di.FilePath = filepath.Join(saveDir, di.Name+"."+di.FileExtension)
	}
	var infoList []*DocumentInfo
	infoList = append(infoList, &di.DocumentInfo)
	for _, child := range di.Children {
		infoList = append(infoList, documentTreeToInfoList(child, filepath.Join(saveDir, di.Name))...)
	}
	return infoList
}

// 递归打印目录结构及文件名，打印过程中会调整空文件名为"未命名xxxn.xxx"格式
// 返回值：tc: totalCount, cdc: canDownloadCount
func printTree(di *DocumentNode, prefix string, totalCount, canDownloadCount int) (tc, cdc int) {
	if prefix == "" {
		totalCount++
		name := getName(di.Name, di.Type, map[string]int{})
		suffix := di.FileExtension
		if di.CanDownload {
			canDownloadCount++
		} else {
			suffix += "（不可下载）"
		}
		if di.Type != "folder" {
			fmt.Println(prefix + "├─ " + name + "." + suffix) // 文件
		}
		if len(di.Children) > 0 {
			fmt.Println(prefix + "├─ " + name) // 目录
		}
		prefix = prefix + "│  "
	}
	temp := map[string]int{}
	for i, child := range di.Children {
		totalCount++
		suffix := child.FileExtension
		if child.CanDownload {
			canDownloadCount++
		} else {
			suffix += "（不可下载）"
		}
		child.Name = getName(child.Name, child.Type, temp)
		if len(child.Children) > 0 {
			if child.Type != "folder" {
				fmt.Println(prefix + "├─ " + child.Name + "." + suffix) // 文件
			}
			fmt.Println(prefix + "├─ " + child.Name) // 目录
			totalCount, canDownloadCount = printTree(child, prefix+"│  ", totalCount, canDownloadCount)
			continue
		}
		if i == len(di.Children)-1 {
			fmt.Println(prefix + "└─ " + child.Name + "." + suffix)
		} else {
			fmt.Println(prefix + "├─ " + child.Name + "." + suffix)
		}
	}
	return totalCount, canDownloadCount
}

func getName(name, typ string, duplicateNameIndexMap map[string]int) string {
	if name != "" {
		index, ok := duplicateNameIndexMap[name]
		if !ok {
			duplicateNameIndexMap[name] = 0
			return name
		}
		duplicateNameIndexMap[name] = index + 1
		return fmt.Sprintf("%s%d", name, index+1)
	}
	unnamedIndex := duplicateNameIndexMap["[Unnamed]"+typ] + 1
	duplicateNameIndexMap["[Unnamed]"+typ] = unnamedIndex
	switch typ {
	case "docx":
		return fmt.Sprintf("未命名新版文档%d", unnamedIndex)
	case "doc":
		return fmt.Sprintf("未命名旧版文档%d", unnamedIndex)
	case "sheet":
		return fmt.Sprintf("未命名电子表格%d", unnamedIndex)
	case "bitable":
		return fmt.Sprintf("未命名多维表格%d", unnamedIndex)
	case "mindnote":
		return fmt.Sprintf("未命名思维笔记%d", unnamedIndex)
	case "slides":
		return fmt.Sprintf("未命名幻灯片%d", unnamedIndex)
	default:
		return fmt.Sprintf("未命名飞书文档%d", unnamedIndex)
	}
}

func doExportAndDownload(client *lark.Client, di *DocumentNode) error {
	// 将查询到的文档树信息保存到文件中
	// 创建目录
	err := os.MkdirAll(argument.SaveDir, 0o755)
	filePath := argument.SaveDir + "\\document-tree.json"
	diBytes, err := json.MarshalIndent(di, "", "  ")
	if err != nil {
		return oops.Wrap(err)
	}
	err = os.WriteFile(filePath, diBytes, 0o644)
	if err != nil {
		return oops.Errorf("写入文件失败: %+v", err)
	}

	fmt.Println("下列目录或文件将保存到:", argument.SaveDir)
	totalCount, canDownloadCount := printTree(di, "", 0, 0)
	fmt.Printf("\n查询总数量: %d, 可下载文档数量: %d\n", totalCount, canDownloadCount)
	fmt.Println("----------------------------------------------")
	if argument.ListOnly {
		return nil
	}

	fmt.Println("阶段2: 下载飞书知识库云文档")
	fmt.Println("--------------------------")
	// 将树结构转为平铺的列表（复制为两个列表，一个供导出任务使用，一个供下载文件使用）
	infoList := documentTreeToInfoList(di, argument.SaveDir)
	canDownloadList := lo.Filter(infoList, func(di *DocumentInfo, _ int) bool { return di.CanDownload })
	canDownloadCount = len(canDownloadList)

	// 创建下载UI程序备用
	totalProgress := teaProgress.New(
		teaProgress.WithDefaultGradient(), // 使用默认渐变颜色
		teaProgress.WithWidth(60),         // 设置进度条宽度
	) // 整体进度
	program := progress.NewProgram(nil, func(total, downloaded, failed int) string {
		remaining := total - downloaded - failed
		statsInfo := fmt.Sprintf("可下载: %d, 已提交: %d, 已下载: %d, 未下载: %d, 已失败: %d", canDownloadCount, total, downloaded, remaining, failed)
		tp := totalProgress.ViewAs(float64(downloaded+failed) / float64(canDownloadCount))
		return progress.TipsStyle.Render(statsInfo) + "\n" + tp
	})
	task := &Task{
		canDownloadList: canDownloadList,
		client:          client,
		program:         program,
		completed:       &atomic.Bool{},
		queue:           make(chan *exportResult, 20),
	}
	defer close(task.queue)

	wait := make(chan struct{})
	go func() {
		// 下载UI程序退出就退出主程序
		defer func() {
			task.completed.Store(true)
			wait <- struct{}{}
		}()
		// 启动 BubbleTea
		if _, err = program.Run(); err != nil {
			fmt.Println("下载UI程序运行出错:", err)
			return
		}
		fmt.Println("退出下载UI程序")
	}()

	task.Run()

	<-wait

	time.Sleep(time.Second * 2)
	return nil
}
