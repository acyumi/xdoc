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

	"github.com/acyumi/xdoc/component/app"
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

func setFileExtension(dn *DocumentNode, args *argument.Args) {
	// 如果是file，则需要通过文件名获取文件类型，再继续下成的switch处理
	if dn.Type == constant.DocTypeFile {
		dn.CanDownload = true
		dn.DownloadDirectly = true
		ext := filepath.Ext(dn.Name)
		if len(ext) > 0 {
			e := constant.DocType(ext[1:])
			switch e {
			case constant.DocTypeXlsx, constant.DocTypeXls:
				dn.Type = constant.DocTypeSheet
			default:
				dn.Type = e
			}
			dn.Name = dn.Name[:len(dn.Name)-len(ext)]
		}
	}
	var setOrDefault = func(def constant.FileExt) {
		dn.FileExtension = args.FileExtensions[dn.Type]
		if dn.FileExtension == "" {
			dn.FileExtension = def
		}
	}
	switch dn.Type {
	// 具体查看支持的类型: https://open.feishu.cn/document/server-docs/docs/drive-v1/export_task/export-user-guide
	// 目前为四种:
	// docx：新版飞书文档。支持导出扩展名为 docx 和 pdf 格式的文件。
	// doc：旧版飞书文档。支持导出扩展名为 docx 和 pdf 的文件。已不推荐使用。
	// sheet：飞书电子表格。支持导出扩展名为 xlsx 和 csv 的文件。
	// bitable：飞书多维表格。支持导出扩展名为 xlsx 和 csv 格式的文件。
	case constant.DocTypeDocx, constant.DocTypeDoc:
		dn.CanDownload = true
		setOrDefault(constant.FileExtDocx)
	case constant.DocTypeBitable, constant.DocTypeSheet:
		dn.CanDownload = true
		setOrDefault(constant.FileExtXlsx)
	default:
		setOrDefault(constant.FileExt(dn.Type))
	}
}

func deduplication(dns []*DocumentNode) (fdns []*DocumentNode) {
	if len(dns) < 2 {
		return dns
	}
	rejectedIndices := map[int]bool{}
	for i, dni := range dns {
		if rejectedIndices[i] {
			continue
		}
		li := documentNodeToInfoList(dni)
		for j, dnj := range dns {
			if rejectedIndices[i] {
				break
			}
			if j == i || rejectedIndices[j] {
				continue
			}
			for _, ii := range li {
				// 如果ii与dnj相同，那就是dni树包含了dnj树，则丢弃dnj
				if ii.Type == dnj.Type && ii.Token == dnj.Token {
					rejectedIndices[j] = true
					break
				}
			}
			if rejectedIndices[j] {
				continue
			}
			lj := documentNodeToInfoList(dnj)
			for _, ij := range lj {
				// 如果ij与dni相同，那就是dnj树包含了dni树，则丢弃dni
				if ij.Type == dni.Type && ij.Token == dni.Token {
					rejectedIndices[i] = true
					break
				}
			}
		}
	}
	for i, dn := range dns {
		if rejectedIndices[i] {
			continue
		}
		fdns = append(fdns, dn)
	}
	return fdns
}

// documentNodeToInfoList 一棵未完整的文档树转为一维列表。
func documentNodeToInfoList(dn *DocumentNode) []*DocumentInfo {
	var infoList []*DocumentInfo
	infoList = append(infoList, &dn.DocumentInfo)
	for _, child := range dn.Children {
		infoList = append(infoList, documentNodeToInfoList(child)...)
	}
	return infoList
}

// documentNodesToInfoList 多棵完整的文档树转为一维列表。
func documentNodesToInfoList(dns []*DocumentNode, saveDir string) []*DocumentInfo {
	var infoList []*DocumentInfo
	for _, dn := range dns {
		if dn.Type == constant.DocTypeFolder {
			dn.FilePath = filepath.Join(saveDir, dn.Name)
		} else {
			dn.FilePath = filepath.Join(saveDir, dn.Name+"."+string(dn.FileExtension))
		}
		infoList = append(infoList, &dn.DocumentInfo)
		infoList = append(infoList, documentNodesToInfoList(dn.Children, filepath.Join(saveDir, dn.Name))...)
	}
	return infoList
}

// 递归打印目录结构及文件名，打印过程中会调整空文件名为"未命名xxxn.xxx"格式
// tree：需要在调用前构造好传进来，以后也不要想着改造成传nil再在第一次处理时从函数内部构造
// 返回值：tc: totalCount, cdc: canDownloadCount。
func printTree(logWriter io.Writer, tree treeprint.Tree, dns []*DocumentNode, totalCount, canDownloadCount int) (tc, cdc int) {
	if totalCount == 0 {
		root := tree
		defer func() {
			_, _ = fmt.Fprint(logWriter, root.String())
		}()
		_, _ = fmt.Fprint(logWriter, "\n")
	}
	temp := map[string]int{}
	for _, child := range dns {
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
			totalCount, canDownloadCount = printTree(logWriter, branch, child.Children, totalCount, canDownloadCount)
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
	defer app.Sleep(time.Second * 2)
	defer task.Close()

	err = task.Run()
	return oops.Wrap(err)
}
