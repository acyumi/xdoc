package feishu

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkdrive "github.com/larksuite/oapi-sdk-go/v3/service/drive/v1"
	"github.com/samber/oops"
	"github.com/spf13/cast"

	"acyumi.com/feishu-doc-exporter/component/argument"
	"acyumi.com/feishu-doc-exporter/component/progress"
)

type Task struct {
	canDownloadList []*DocumentInfo    //
	client          *lark.Client       //
	program         *progress.Program  //
	completed       *atomic.Bool       //
	queue           chan *exportResult //
}

func (t *Task) Run() {
	// 开启5个协程同时创建导出任务
	t.exportDocuments()
	// 开启3个协诚同时下载文件
	t.downloadDocuments()
}

func (t *Task) exportDocuments() {
	lock := &sync.Mutex{}
	docIdx := -1
	canDownloadCount := len(t.canDownloadList)
	getNextDoc := func() *DocumentInfo {
		lock.Lock()
		defer lock.Unlock()
		docIdx++
		if docIdx >= canDownloadCount {
			return nil
		}
		return t.canDownloadList[docIdx]
	}
	for i := 0; i < 5; i++ {
		go func() {
			for {
				if t.completed.Load() {
					return
				}
				di := getNextDoc()
				if di == nil {
					return
				}
				// 发送新文件到program
				fileName := di.GetFileName()
				t.program.Add(di.FilePath, fileName)

				if di.DownloadDirectly {
					t.queue <- &exportResult{DocumentInfo: di, result: nil}
					continue // 注意这里是continue而不是return
				}

				// 创建导出任务
				ticket, err := t.doExport(di)
				if err != nil {
					t.program.Update(di.FilePath, 0.05, progress.StatusFailed, cleanEnter(err))
					continue // 注意这里是continue而不是return
				}
				t.program.Update(di.FilePath, 0.05, progress.StatusExporting)

				// 查询导出任务结果
				exportResult, status, err := t.checkExport(di, ticket)
				if err != nil {
					t.program.Update(di.FilePath, 0.10, progress.StatusFailed, cleanEnter(err))
					continue // 注意这里是continue而不是return
				}
				if status == progress.StatusInterrupted {
					return
				}
				t.program.Update(di.FilePath, 0.15, status)
				// 随机睡眠1到3秒
				time.Sleep(time.Second * time.Duration(rand.Intn(2)+1))
				t.program.Update(di.FilePath, 0.15, progress.StatusWaiting)
				t.queue <- exportResult
			}
		}()
	}
}

func (t *Task) downloadDocuments() {
	var canDownloadCount = len(t.canDownloadList)
	var counter atomic.Int32
	for i := 0; i < 3; i++ {
		go func(id int) {
			for {
				if t.completed.Load() {
					return
				}
				select {
				case value, ok := <-t.queue:
					if !ok {
						return
					}

					// 开始下载文件，写入到saveDir目录中
					var fileSize int64
					var file io.Reader
					var err error
					if value.DownloadDirectly {
						file, fileSize, err = t.doDownloadDirectly(value.FilePath, value.Token)
					} else {
						fileSize = int64(larkcore.IntValue(value.result.FileSize))
						fileToken := larkcore.StringValue(value.result.FileToken)
						file, err = t.doDownloadExported(value.FilePath, fileToken)
					}
					if err != nil {
						t.program.Update(value.FilePath, 0.18, progress.StatusFailed, cleanEnter(err))
						counter.Add(1)
						continue // 注意这里是continue而不是return
					}
					t.program.Update(value.FilePath, 0.20, progress.StatusDownloading)

					pw := &progress.Writer{
						FileKey:  value.Token,
						FilePath: value.FilePath,
						Program:  t.program,
						Total:    fileSize,
						Walked:   0.2,
					}
					if err = pw.WriteFile(file); err != nil {
						t.program.Update(value.FilePath, pw.Progress(), progress.StatusFailed, cleanEnter(err))
						counter.Add(1)
						continue // 注意这里是continue而不是return
					}
					t.program.Update(value.FilePath, pw.Progress(), progress.StatusCompleted)

					// 随机睡眠1到3秒
					time.Sleep(time.Second * time.Duration(rand.Intn(2)+1))
					counter.Add(1)
				default:
					if counter.Load() == int32(canDownloadCount) {
						if argument.QuitAutomatically {
							// 随机睡眠1到3秒
							time.Sleep(time.Second * time.Duration(rand.Intn(2)+1))
							t.program.Quit()
						}
						return
					}
				}
			}
		}(i)
	}
}

func (t *Task) doExport(di *DocumentInfo) (string, error) {
	// 发送请求创建导出任务
	req := larkdrive.NewCreateExportTaskReqBuilder().
		ExportTask(larkdrive.NewExportTaskBuilder().
			FileExtension(di.FileExtension).
			Token(di.Token).
			Type(di.Type).
			Build()).
		Build()
	resp, err := SendWithRetry(func(count int) (*larkdrive.CreateExportTaskResp, error) {
		t.program.Update(di.FilePath, 0, progress.StatusExporting, "请求%d次", count)
		return t.client.Drive.V1.ExportTask.Create(context.Background(), req)
	})
	if err != nil {
		return "", oops.Wrap(err)
	}
	if !resp.Success() {
		return "", oops.New(toErrMsg(resp, "创建导出任务"))
	}
	return larkcore.StringValue(resp.Data.Ticket), nil
}

func (t *Task) checkExport(di *DocumentInfo, ticket string) (*exportResult, progress.Status, error) {
	for i := 0; i < 5; i++ {
		if t.completed.Load() {
			return nil, progress.StatusInterrupted, nil
		}
		// 发送请求查询导出任务结果
		etReq := larkdrive.NewGetExportTaskReqBuilder().Ticket(ticket).Token(di.Token).Build()
		etResp, err := SendWithRetry(func(count int) (*larkdrive.GetExportTaskResp, error) {
			t.program.Update(di.FilePath, 0.10, progress.StatusExporting, "查询%d次", count)
			return t.client.Drive.V1.ExportTask.Get(context.Background(), etReq)
		})
		if err != nil {
			return nil, progress.StatusFailed, oops.Wrap(err)
		}
		if !etResp.Success() {
			return nil, progress.StatusFailed, oops.New(toErrMsg(etResp, "查询导出任务结果"))
		}

		// https://open.feishu.cn/document/server-docs/docs/drive-v1/export_task/get
		// 0：成功，1：初始化，2：处理中，大于2的其他状态为异常状态
		result := etResp.Data.Result
		jobStatus := larkcore.IntValue(result.JobStatus)
		if jobStatus < 1 {
			return &exportResult{DocumentInfo: di, result: result}, progress.StatusExported, nil
		}
		if jobStatus > 2 {
			jobErrorMsg := larkcore.StringValue(result.JobErrorMsg)
			return nil, progress.StatusFailed, oops.New(strings.ReplaceAll(jobErrorMsg, "\n", " "))
		}
		t.program.Update(di.FilePath, 0.10, progress.StatusExporting, "等待完成导出任务")
		// 随机睡眠1到5秒
		time.Sleep(time.Second * time.Duration(rand.Intn(4)+1))
	}
	return nil, progress.StatusFailed, oops.New("经过多次尝试取不到导出任务结果")
}

func (t *Task) doDownloadExported(filePath, fileToken string) (io.Reader, error) {
	req := larkdrive.NewDownloadExportTaskReqBuilder().FileToken(fileToken).Build()
	resp, err := SendWithRetry(func(count int) (*larkdrive.DownloadExportTaskResp, error) {
		t.program.Update(filePath, 0.18, progress.StatusDownloading, "请求%d次", count)
		return t.client.Drive.V1.ExportTask.Download(context.Background(), req)
	})
	if err != nil {
		return nil, oops.Wrap(err)
	}
	if !resp.Success() {
		return nil, oops.New(toErrMsg(resp, "下载导出文件"))
	}
	return resp.File, nil
}

func (t *Task) doDownloadDirectly(filePath, token string) (io.Reader, int64, error) {
	req := larkdrive.NewDownloadFileReqBuilder().FileToken(token).Build()
	resp, err := SendWithRetry(func(count int) (*larkdrive.DownloadFileResp, error) {
		t.program.Update(filePath, 0.18, progress.StatusDownloading, "请求%d次", count)
		return t.client.Drive.V1.File.Download(context.Background(), req)
	})
	if err != nil {
		return nil, 0, oops.Wrap(err)
	}
	if !resp.Success() {
		return nil, 0, oops.New(toErrMsg(resp, "下载导出文件"))
	}
	contentLength := resp.Header.Get("Content-Length")
	return resp.File, cast.ToInt64(contentLength), nil
}

func toErrMsg(err error, operation string) string {
	r, ok := err.(interface{ LogId() string })
	errMsg := cleanEnter(err)
	if !ok {
		return fmt.Sprintf("响应错误: %s", errMsg)
	}
	// TODO 可以加个颜色
	return fmt.Sprintf("logId: %s, 操作: %s, 响应错误: %s", r.LogId(), operation, errMsg)
}

func cleanEnter(err error) string {
	return strings.ReplaceAll(err.Error(), "\n", " ")
}
