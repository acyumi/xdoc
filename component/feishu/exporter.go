package feishu

import (
	"context"
	"io"
	"math/rand"
	"strings"
	"sync/atomic"
	"time"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkdrive "github.com/larksuite/oapi-sdk-go/v3/service/drive/v1"
	"github.com/samber/oops"
	"github.com/spf13/cast"

	"github.com/acyumi/doc-exporter/component/cloud"
	"github.com/acyumi/doc-exporter/component/progress"
)

type exporter struct {
	client    Client            //
	program   progress.IProgram //
	completed *atomic.Bool      //
}

// doExport 创建导出任务。
func (e *exporter) doExport(di *DocumentInfo) (string, error) {
	// 发送请求创建导出任务
	exportTask := larkdrive.NewExportTaskBuilder().
		FileExtension(string(di.FileExtension)).
		Token(di.Token).
		Type(string(di.Type)).
		Build()
	req := larkdrive.NewCreateExportTaskReqBuilder().
		ExportTask(exportTask).
		Build()
	req.ExportTask = exportTask
	resp, err := SendWithRetry(func(count int) (*larkdrive.CreateExportTaskResp, error) {
		e.program.Update(di.FilePath, 0, progress.StatusExporting, "请求%d次", count)
		return e.client.ExportCreate(context.Background(), req)
	})
	if err != nil {
		if resp != nil && !resp.Success() {
			return "", oops.New(toErrMsg(resp, "创建导出任务"))
		}
		return "", oops.Wrap(err)
	}
	return larkcore.StringValue(resp.Data.Ticket), nil
}

// checkExport 查询导出任务结果。
func (e *exporter) checkExport(di *DocumentInfo, ticket string) (*exportResult, progress.Status, error) {
	for i := 0; i < 5; i++ {
		if e.completed.Load() {
			return nil, progress.StatusInterrupted, nil
		}
		// 发送请求查询导出任务结果
		req := larkdrive.NewGetExportTaskReqBuilder().Ticket(ticket).Token(di.Token).Build()
		resp, err := SendWithRetry(func(count int) (*larkdrive.GetExportTaskResp, error) {
			e.program.Update(di.FilePath, 0.10, progress.StatusExporting, "查询%d次", count)
			return e.client.ExportGet(context.Background(), req)
		})
		if err != nil {
			if resp != nil && !resp.Success() {
				return nil, progress.StatusFailed, oops.New(toErrMsg(resp, "查询导出任务结果"))
			}
			return nil, progress.StatusFailed, oops.Wrap(err)
		}

		// https://open.feishu.cn/document/server-docs/docs/drive-v1/export_task/get
		// 0：成功，1：初始化，2：处理中，大于2的其他状态为异常状态
		result := resp.Data.Result
		jobStatus := larkcore.IntValue(result.JobStatus)
		if jobStatus < 1 {
			return &exportResult{DocumentInfo: di, result: result}, progress.StatusExported, nil
		}
		if jobStatus > 2 {
			jobErrorMsg := larkcore.StringValue(result.JobErrorMsg)
			return nil, progress.StatusFailed, oops.New(strings.ReplaceAll(jobErrorMsg, "\n", " "))
		}
		e.program.Update(di.FilePath, 0.10, progress.StatusExporting, "等待完成导出任务")
		// 随机睡眠1到5秒
		cloud.Sleep(time.Second * time.Duration(rand.Intn(4)+1))
	}
	return nil, progress.StatusFailed, oops.New("经过多次尝试取不到导出任务结果")
}

// doDownloadExported 下载导出的文件。
func (e *exporter) doDownloadExported(filePath, fileToken string) (io.Reader, error) {
	req := larkdrive.NewDownloadExportTaskReqBuilder().FileToken(fileToken).Build()
	resp, err := SendWithRetry(func(count int) (*larkdrive.DownloadExportTaskResp, error) {
		e.program.Update(filePath, 0.18, progress.StatusDownloading, "请求%d次", count)
		return e.client.ExportDownload(context.Background(), req)
	})
	if err != nil {
		if resp != nil && !resp.Success() {
			return nil, oops.New(toErrMsg(resp, "下载导出文件"))
		}
		return nil, oops.Wrap(err)
	}
	return resp.File, nil
}

// doDownloadDirectly 不需要经过导出操作，直接下载文件。
func (e *exporter) doDownloadDirectly(filePath, fileToken string) (io.Reader, int64, error) {
	req := larkdrive.NewDownloadFileReqBuilder().FileToken(fileToken).Build()
	resp, err := SendWithRetry(func(count int) (*larkdrive.DownloadFileResp, error) {
		e.program.Update(filePath, 0.18, progress.StatusDownloading, "请求%d次", count)
		return e.client.DriveDownload(context.Background(), req)
	})
	if err != nil {
		if resp != nil && !resp.Success() {
			return nil, 0, oops.New(toErrMsg(resp, "下载导出文件"))
		}
		return nil, 0, oops.Wrap(err)
	}
	contentLength := resp.Header.Get("Content-Length")
	return resp.File, cast.ToInt64(contentLength), nil
}
