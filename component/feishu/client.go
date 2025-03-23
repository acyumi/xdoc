package feishu

import (
	"context"
	"fmt"

	validation "github.com/go-ozzo/ozzo-validation"
	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkdrive "github.com/larksuite/oapi-sdk-go/v3/service/drive/v1"
	larkwiki "github.com/larksuite/oapi-sdk-go/v3/service/wiki/v2"
	"github.com/samber/oops"

	"github.com/acyumi/xdoc/component/argument"
	"github.com/acyumi/xdoc/component/cloud"
	"github.com/acyumi/xdoc/component/constant"
	"github.com/acyumi/xdoc/component/progress"
)

type ClientImpl struct {
	*lark.Client
	Args        *argument.Args
	TaskCreator func(args *argument.Args, docs *DocumentNode) cloud.Task
}

func NewClient(args *argument.Args) cloud.Client {
	var c ClientImpl
	c.SetArgs(args)
	return &c
}

func (c *ClientImpl) SetArgs(args *argument.Args) {
	c.Client = lark.NewClient(args.AppID, args.AppSecret)
	c.Args = args
}

func (c *ClientImpl) GetArgs() *argument.Args {
	return c.Args
}

func (c ClientImpl) Validate() error {
	return oops.Code("InvalidArgument").Wrap(
		validation.ValidateStruct(&c,
			validation.Field(&c.Client, validation.Required),
			validation.Field(&c.Args, validation.Required),
		))
}

func (c *ClientImpl) DownloadDocuments(typ, token string) (err error) {
	switch typ {
	case "/wiki":
		fmt.Printf("飞书云文档源: 知识库, 类型: %s, token: %s\n", typ, token)
		err = c.DownloadWikiDocuments(token)
	case "/wiki/settings":
		fmt.Printf("飞书云文档源: 知识库, 类型: %s, token: %s\n", typ, token)
		err = c.DownloadWikiSpaceDocuments(token)
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
		case "/drive/folder":
			docType = constant.DocTypeFolder
		default:
			return oops.Code("InvalidArgument").Errorf("不支持的飞书云文档类型: %s\n", typ)
		}
		err = c.DownloadDriveDocuments(docType, token)
	default:
		err = oops.Code("InvalidArgument").Errorf("不支持的飞书云文档类型: %s\n", typ)
	}
	return err
}

func (c *ClientImpl) CreateTask(docs *DocumentNode, programConstructor func(progress.Stats) progress.IProgram) cloud.Task {
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
