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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	validation "github.com/go-ozzo/ozzo-validation"
	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkdrive "github.com/larksuite/oapi-sdk-go/v3/service/drive/v1"
	larkwiki "github.com/larksuite/oapi-sdk-go/v3/service/wiki/v2"
	"github.com/samber/oops"
	"github.com/xlab/treeprint"

	"github.com/acyumi/xdoc/component/app"
	"github.com/acyumi/xdoc/component/cloud"
	"github.com/acyumi/xdoc/component/constant"
	"github.com/acyumi/xdoc/component/progress"
)

type ClientImpl struct {
	*lark.Client
	Args        *Args
	TaskCreator func(args *Args, docs []*DocumentNode) cloud.Task
}

func NewClient(args *Args) cloud.Client[*Args] {
	var c ClientImpl
	c.SetArgs(args)
	return &c
}

func (c *ClientImpl) SetArgs(args *Args) {
	c.Client = lark.NewClient(args.AppID, args.AppSecret)
	c.Args = args
}

func (c *ClientImpl) GetArgs() *Args {
	return c.Args
}

func (c ClientImpl) Validate() error {
	return oops.Code("InvalidArgument").Wrap(
		validation.ValidateStruct(&c,
			validation.Field(&c.Client, validation.Required),
			validation.Field(&c.Args, validation.Required),
		))
}

func (c *ClientImpl) DownloadDocuments(docSources []*cloud.DocumentSource) error {
	fmt.Println("阶段1: 读取飞书云文档信息")
	fmt.Println("--------------------------")
	var dns []*DocumentNode
	for _, ds := range docSources {
		dn, err := c.QueryDocuments(ds.Type, ds.Token)
		if err != nil {
			return oops.Wrap(err)
		}
		dns = append(dns, dn)
	}
	// 去重，可能dns中的树是互相包含的关系
	dns = deduplication(dns)

	// 将查询到的文档树信息保存到文件中
	// 创建目录
	err := app.Fs.MkdirAll(c.Args.SaveDir, 0o755)
	if err != nil {
		return oops.Wrap(err)
	}
	// 将文档树信息保存到document-tree.json文件中
	filePath := filepath.Join(c.Args.SaveDir, "document-tree.json")
	diBytes, err := app.MarshalIndent(dns, "", "  ")
	if err != nil {
		return oops.Wrap(err)
	}
	err = app.Fs.WriteFile(filePath, diBytes, 0o644)
	if err != nil {
		return oops.Wrapf(err, "写入文件失败")
	}

	fmt.Println("预计将目录或文件保存如下:")
	tree := treeprint.NewWithRoot(c.Args.SaveDir)
	totalCount, canDownloadCount := printTree(os.Stdout, tree, dns, 0, 0)
	fmt.Printf("\n查询总数量: %d, 可下载文档数量: %d\n", totalCount, canDownloadCount)
	fmt.Println("--------------------------")
	fmt.Printf("阶段1, 耗时: %s\n", time.Since(c.Args.StartTime).String())
	fmt.Println("----------------------------------------------")
	if c.Args.ListOnly {
		return nil
	}

	task := c.CreateTask(dns, progress.NewProgram)
	return doExportAndDownload(task)
}

func (c *ClientImpl) QueryDocuments(typ, token string) (dn *DocumentNode, err error) {
	switch typ {
	case "/wiki":
		fmt.Printf("飞书云文档源: 知识库, 类型: %s, token: %s\n", typ, token)
		dn, err = c.QueryWikiDocuments(token)
	case "/wiki/settings":
		fmt.Printf("飞书云文档源: 知识库, 类型: %s, token: %s\n", typ, token)
		dn, err = c.QueryWikiSpaceDocuments(token)
	case "/drive/folder", "/docs", "/docx", "/sheets", "/file":
		fmt.Printf("飞书云文档源: 云空间, 类型: %s, token: %s\n", typ, token)
		var docType constant.DocType
		switch typ {
		case "/docs":
			docType = constant.DocTypeDoc
		case "/docx":
			docType = constant.DocTypeDocx
		case "/sheets":
			docType = constant.DocTypeSheet
		case "/file":
			docType = constant.DocTypeFile
		default:
			// "/drive/folder"
			docType = constant.DocTypeFolder
		}
		dn, err = c.QueryDriveDocuments(docType, token)
	default:
		err = oops.Code("InvalidArgument").Errorf("不支持的飞书云文档类型: %s\n", typ)
	}
	return dn, err
}

func (c *ClientImpl) CreateTask(docs []*DocumentNode, programConstructor func(progress.Stats) progress.IProgram) cloud.Task {
	if c.TaskCreator != nil {
		return c.TaskCreator(c.Args, docs)
	}
	return &TaskImpl{Client: c, Docs: docs, ProgramConstructor: programConstructor}
}

func checkResp[R error](resp R, err error) (R, error) {
	if err != nil {
		return resp, err
	}
	r, ok := any(resp).(interface{ Success() bool })
	if !ok {
		return resp, nil
	}
	if !r.Success() {
		codeError := getCodeError(resp)
		if codeError == nil {
			return resp, nil
		}
		return resp, oops.Errorf("logId: %s, error response: \n%s", getLogID(resp), larkcore.Prettify(codeError))
	}
	return resp, nil
}

// getLogID 获取错误日志ID。
func getLogID(err error) string {
	r, ok := err.(interface{ RequestId() string })
	if !ok {
		return ""
	}
	logID := r.RequestId()
	// 渲染为链接，方便查看分析报错原因
	logID = progress.URLStyleRender(fmt.Sprintf("https://open.feishu.cn/search?q=%s", logID))
	return logID
}

// DriveBatchQuery 批量查询文件元信息。
func (c *ClientImpl) DriveBatchQuery(ctx context.Context, req *larkdrive.BatchQueryMetaReq, options ...larkcore.RequestOptionFunc) (*larkdrive.BatchQueryMetaResp, error) {
	resp, err := c.Drive.V1.Meta.BatchQuery(ctx, req, options...)
	return checkResp(resp, err)
}

// DriveList 获取文件夹中的文件清单。
func (c *ClientImpl) DriveList(ctx context.Context, req *larkdrive.ListFileReq, options ...larkcore.RequestOptionFunc) (*larkdrive.ListFileResp, error) {
	resp, err := c.Drive.V1.File.List(ctx, req, options...)
	return checkResp(resp, err)
}

func (c *ClientImpl) DriveDownload(ctx context.Context, req *larkdrive.DownloadFileReq, options ...larkcore.RequestOptionFunc) (*larkdrive.DownloadFileResp, error) {
	resp, err := c.Drive.V1.File.Download(ctx, req, options...)
	return checkResp(resp, err)
}

func (c *ClientImpl) WikiGetNode(ctx context.Context, req *larkwiki.GetNodeSpaceReq, options ...larkcore.RequestOptionFunc) (*larkwiki.GetNodeSpaceResp, error) {
	resp, err := c.Wiki.V2.Space.GetNode(ctx, req, options...)
	return checkResp(resp, err)
}

func (c *ClientImpl) WikiGetSpace(ctx context.Context, req *larkwiki.GetSpaceReq, options ...larkcore.RequestOptionFunc) (*larkwiki.GetSpaceResp, error) {
	resp, err := c.Wiki.V2.Space.Get(ctx, req, options...)
	return checkResp(resp, err)
}

func (c *ClientImpl) WikiNodeList(ctx context.Context, req *larkwiki.ListSpaceNodeReq, options ...larkcore.RequestOptionFunc) (*larkwiki.ListSpaceNodeResp, error) {
	resp, err := c.Wiki.V2.SpaceNode.List(ctx, req, options...)
	return checkResp(resp, err)
}

func (c *ClientImpl) ExportCreate(ctx context.Context, req *larkdrive.CreateExportTaskReq, options ...larkcore.RequestOptionFunc) (*larkdrive.CreateExportTaskResp, error) {
	resp, err := c.Drive.V1.ExportTask.Create(ctx, req, options...)
	return checkResp(resp, err)
}

func (c *ClientImpl) ExportGet(ctx context.Context, req *larkdrive.GetExportTaskReq, options ...larkcore.RequestOptionFunc) (*larkdrive.GetExportTaskResp, error) {
	resp, err := c.Drive.V1.ExportTask.Get(ctx, req, options...)
	return checkResp(resp, err)
}

func (c *ClientImpl) ExportDownload(ctx context.Context, req *larkdrive.DownloadExportTaskReq, options ...larkcore.RequestOptionFunc) (*larkdrive.DownloadExportTaskResp, error) {
	resp, err := c.Drive.V1.ExportTask.Download(ctx, req, options...)
	return checkResp(resp, err)
}
