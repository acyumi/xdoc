package feishu

import (
	"fmt"
	"io"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	teaProgress "github.com/charmbracelet/bubbles/progress"
	validation "github.com/go-ozzo/ozzo-validation"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/samber/lo"
	"github.com/samber/oops"

	"github.com/acyumi/xdoc/component/app"
	"github.com/acyumi/xdoc/component/progress"
)

type TaskImpl struct {
	Client             Client                                 //
	Docs               []*DocumentNode                        //
	ProgramConstructor func(progress.Stats) progress.IProgram //

	canDownloadList []*DocumentInfo    //
	program         progress.IProgram  //
	countDown       *atomic.Int32      //
	completed       *atomic.Bool       // 任务整体是否完成（导出+下载）
	queue           chan *exportResult //
	wait            chan struct{}      //
	exporter        IExporter          //
}

func (t TaskImpl) Validate() (err error) {
	return oops.Code("InvalidArgument").Wrap(
		validation.ValidateStruct(&t,
			validation.Field(&t.Docs, validation.Required),
			validation.Field(&t.Client, validation.Required),
			validation.Field(&t.ProgramConstructor, validation.Required),
		))
}

func (t *TaskImpl) Run() (err error) {
	startTime := time.Now()
	fmt.Println("阶段2: 下载飞书云文档")
	fmt.Println("--------------------------")
	defer func() {
		fmt.Println("--------------------------")
		fmt.Printf("阶段2, 耗时: %s\n", time.Since(startTime).String())
	}()

	args := t.Client.GetArgs()
	// 将树结构转为平铺的列表
	infoList := documentNodesToInfoList(t.Docs, args.SaveDir)

	// 初始化必要参数备用
	t.canDownloadList = lo.Filter(infoList, func(di *DocumentInfo, _ int) bool { return di.CanDownload })
	canDownloadCount := len(t.canDownloadList)
	t.program = t.ProgramConstructor(calculateOverallProgress(canDownloadCount))
	t.countDown = &atomic.Int32{}
	t.countDown.Store(int32(canDownloadCount))
	t.completed = &atomic.Bool{}
	t.queue = make(chan *exportResult, 20)
	t.wait = make(chan struct{})
	t.exporter = &exporter{client: t.Client, program: t.program, completed: t.completed}

	// 开启下载UI程序
	go func() {
		// 下载UI程序退出就退出主程序
		defer func() {
			t.Complete()
		}()
		// 启动 BubbleTea
		if _, err = t.program.Run(); err != nil {
			fmt.Println("下载UI程序运行出错:", err)
			return
		}
		fmt.Println("退出下载UI程序")
	}()

	// 开启5个协程同时创建导出任务
	_ = t.exportDocuments()

	// 开启3个协诚同时下载文件
	_ = t.downloadDocuments()

	// 等待中断触发或批量下载完成
	<-t.wait

	return err
}

func (t *TaskImpl) Close() {
	if t.queue != nil {
		close(t.queue)
	}
	if t.wait != nil {
		close(t.wait)
	}
}

func (t *TaskImpl) Interrupt() {
	// 先关掉下载UI程序，再让它所在的goroutine退出时触发调用t.Complete()
	t.program.Quit()
}

func (t *TaskImpl) Complete() {
	t.completed.Store(true)
	t.wait <- struct{}{}
}

// exportDocuments 批量创建和检查导出任务。
func (t *TaskImpl) exportDocuments() (completed *atomic.Bool) {
	completed = &atomic.Bool{}
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
	total := 5
	doneCount := &atomic.Int32{}
	for i := 0; i < total; i++ {
		go func() {
			defer func() {
				// 如果所有协程都执行完了，则发送完成信号
				if doneCount.Add(1) == int32(total) {
					completed.Store(true)
				}
			}()
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
				ticket, err := t.exporter.doExport(di)
				if err != nil {
					t.program.Update(di.FilePath, 0.05, progress.StatusFailed, cleanEnter(err))
					t.countDown.Add(-1)
					continue // 注意这里是continue而不是return
				}
				t.program.Update(di.FilePath, 0.05, progress.StatusExporting)

				// 查询导出任务结果
				exportResult, status, err := t.exporter.checkExport(di, ticket)
				if err != nil {
					t.program.Update(di.FilePath, 0.10, progress.StatusFailed, cleanEnter(err))
					t.countDown.Add(-1)
					continue // 注意这里是continue而不是return
				}
				if status == progress.StatusInterrupted {
					return
				}
				t.program.Update(di.FilePath, 0.15, status)

				// 随机睡眠1到3秒
				app.Sleep(time.Second * time.Duration(rand.Intn(2)+1))

				t.program.Update(di.FilePath, 0.15, progress.StatusWaiting)
				t.queue <- exportResult
			}
		}()
	}
	return completed
}

// downloadDocuments 批量下载已导出的文件并显示下载进度。
func (t *TaskImpl) downloadDocuments() (completed *atomic.Bool) {
	completed = &atomic.Bool{}
	total := 3
	doneCount := atomic.Int32{}
	for range total {
		go func() {
			defer func() {
				// 如果所有协程都执行完了，则发送完成信号
				if doneCount.Add(1) == int32(total) {
					completed.Store(true)
				}
			}()
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
						file, fileSize, err = t.exporter.doDownloadDirectly(value.FilePath, value.Token)
					} else {
						fileSize = int64(larkcore.IntValue(value.result.FileSize))
						fileToken := larkcore.StringValue(value.result.FileToken)
						file, err = t.exporter.doDownloadExported(value.FilePath, fileToken)
					}
					if err != nil {
						t.program.Update(value.FilePath, 0.18, progress.StatusFailed, cleanEnter(err))
						t.countDown.Add(-1)
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
						t.countDown.Add(-1)
						continue // 注意这里是continue而不是return
					}
					t.program.Update(value.FilePath, pw.Progress(), progress.StatusCompleted)

					// 随机睡眠1到3秒
					app.Sleep(time.Second * time.Duration(rand.Intn(2)+1))
					t.countDown.Add(-1)
				default:
					if t.countDown.Load() <= 0 {
						if t.Client.GetArgs().QuitAutomatically {
							// 随机睡眠1到3秒
							app.Sleep(time.Second * time.Duration(rand.Intn(2)+1))
							t.Interrupt()
						}
						return
					}
				}
			}
		}()
	}
	return completed
}

// calculateOverallProgress 计算整体进度。
func calculateOverallProgress(canDownloadCount int) progress.Stats {
	// 创建下载UI程序备用
	totalProgress := teaProgress.New(
		teaProgress.WithDefaultGradient(), // 使用默认渐变颜色
		teaProgress.WithWidth(60),         // 设置进度条宽度
	) // 整体进度
	return func(total, downloaded, failed int) string {
		remaining := total - downloaded - failed
		statsInfo := fmt.Sprintf("可下载: %d, 已提交: %d, 已下载: %d, 未下载: %d, 已失败: %d", canDownloadCount, total, downloaded, remaining, failed)
		tp := totalProgress.ViewAs(float64(downloaded+failed) / float64(canDownloadCount))
		return progress.TipsStyle.Render(statsInfo) + "\n" + tp
	}
}

// toErrMsg 将错误转换为错误信息。
func toErrMsg(err error, operation string) string {
	logID := getLogID(err)
	errMsg := cleanEnter(err)
	if logID == "" {
		return fmt.Sprintf("响应错误: %s", errMsg)
	}
	return fmt.Sprintf("logId: %s, 操作: %s, 响应错误: %s", logID, operation, errMsg)
}

// cleanEnter 清除换行符。
func cleanEnter(err error) string {
	return strings.ReplaceAll(err.Error(), "\n", " ")
}
