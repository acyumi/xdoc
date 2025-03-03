package feishu

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkdrive "github.com/larksuite/oapi-sdk-go/v3/service/drive/v1"
	"github.com/samber/lo"
	"github.com/samber/oops"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/acyumi/doc-exporter/component/argument"
	"github.com/acyumi/doc-exporter/component/constant"
	"github.com/acyumi/doc-exporter/component/progress"
)

func TestTaskImplSuite(t *testing.T) {
	suite.Run(t, new(TaskImplTestSuite))
}

type TaskImplTestSuite struct {
	suite.Suite
	*mockRunArgs
}

func (s *TaskImplTestSuite) SetupSuite() {
	cleanSleep()
}

func (s *TaskImplTestSuite) SetupTest() {
	s.mockRunArgs = getMockRunArgs(s.T())
}

func (s *TaskImplTestSuite) TearDownTest() {
	if s.task.completed == nil || s.task.completed.Load() || s.task.queue == nil || s.task.wait == nil {
		return
	}
	s.task.Close()
}

// 定义测试用例。
func (s *TaskImplTestSuite) TestTaskImpl_Validate() {
	tests := []struct {
		name     string
		task     TaskImpl
		expected string
	}{
		{
			name:     "整体校验不通过",
			task:     TaskImpl{},
			expected: `Client: cannot be blank; Docs: cannot be blank; ProgramConstructor: cannot be blank.`,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := tt.task.Validate()
			if tt.expected == "" {
				s.Require().NoError(err, tt.name)
			} else {
				s.Require().Error(err, tt.name)
				s.IsType(oops.OopsError{}, err, tt.name)
				var actualError oops.OopsError
				yes := errors.As(err, &actualError)
				s.Require().True(yes, tt.name)
				s.Equal("InvalidArgument", actualError.Code(), tt.name)
				s.Equal(tt.expected, actualError.Error(), tt.name)
			}
		})
	}
}

type mockRunArgs struct {
	task         TaskImpl
	mockClient   *MockClient
	mockProgram  *MockProgram
	mockExporter *MockExporter
}

func getMockRunArgs(t *testing.T) *mockRunArgs {
	var args mockRunArgs
	args.mockClient = NewMockClient(t)
	args.mockProgram = NewMockProgram(t)
	args.mockExporter = NewMockExporter(t)
	args.task = TaskImpl{
		Client:  args.mockClient, // 模拟Client
		program: args.mockProgram,
		ProgramConstructor: func(stats progress.Stats) progress.IProgram {
			return args.mockProgram
		},
		completed: &atomic.Bool{},
		exporter:  args.mockExporter,
	}
	return &args
}

func (s *TaskImplTestSuite) TestTaskImpl_Run() {
	tests := []struct {
		name          string
		setupMock     func(args *mockRunArgs)
		expectedError error
	}{
		{
			name: "只列出文件树",
			setupMock: func(args *mockRunArgs) {
				args.task.Docs = &DocumentNode{
					DocumentInfo: DocumentInfo{
						Name:             "folder1",
						Token:            "folder1_token",
						Type:             constant.DocTypeFolder,
						DownloadDirectly: false,
						FileExtension:    "folder",
						CanDownload:      false,
						FilePath:         "folder1_path",
					},
					Children: []*DocumentNode{
						{
							DocumentInfo: DocumentInfo{
								Name:             "doc1",
								Token:            "doc1_token",
								Type:             constant.DocTypeDoc,
								DownloadDirectly: false,
								FileExtension:    constant.FileExtDocx,
								CanDownload:      true,
								FilePath:         "doc1_path.docx",
							},
						},
						{
							DocumentInfo: DocumentInfo{
								Name:             "doc2",
								Token:            "doc2_token",
								Type:             constant.DocTypeDocx,
								DownloadDirectly: true,
								FileExtension:    constant.FileExtPDF,
								CanDownload:      true,
								FilePath:         "doc2_path.pdf",
							},
						},
					},
				}
				args.mockClient.EXPECT().GetArgs().Return(&argument.Args{
					SaveDir:  "/tmp",
					ListOnly: true,
				}).Once()
			},
			expectedError: nil,
		},
		{
			name: "正常执行",
			setupMock: func(args *mockRunArgs) {
				args.task.Docs = &DocumentNode{
					DocumentInfo: DocumentInfo{
						Name:             "folder1",
						Token:            "folder1_token",
						Type:             constant.DocTypeFolder,
						DownloadDirectly: false,
						FileExtension:    "folder",
						CanDownload:      false,
						FilePath:         "folder1_path",
					},
					Children: []*DocumentNode{
						{
							DocumentInfo: DocumentInfo{
								Name:             "doc1",
								Token:            "doc1_token",
								Type:             constant.DocTypeDoc,
								DownloadDirectly: false,
								FileExtension:    constant.FileExtDocx,
								CanDownload:      false, // 设置成不能下载，不跑 exportDocuments 和 downloadDocuments
								FilePath:         "doc1_path.docx",
							},
						},
						{
							DocumentInfo: DocumentInfo{
								Name:             "doc2",
								Token:            "doc2_token",
								Type:             constant.DocTypeDocx,
								DownloadDirectly: false,
								FileExtension:    constant.FileExtPDF,
								CanDownload:      false, // 设置成不能下载，不跑 exportDocuments 和 downloadDocuments
								FilePath:         "doc2_path.pdf",
							},
						},
					},
				}
				args.mockClient.EXPECT().GetArgs().Return(&argument.Args{
					SaveDir:           "/tmp",
					ListOnly:          false,
					QuitAutomatically: true,
				}).Maybe()
				args.mockProgram.EXPECT().Run().Return(nil, nil).Once()
				args.mockProgram.EXPECT().Quit().Maybe()
			},
			expectedError: nil,
		},
		{
			name: "下载UI程序报错",
			setupMock: func(args *mockRunArgs) {
				args.task.Docs = &DocumentNode{
					DocumentInfo: DocumentInfo{
						Name:             "folder1",
						Token:            "folder1_token",
						Type:             constant.DocTypeFolder,
						DownloadDirectly: false,
						FileExtension:    "folder",
						CanDownload:      false,
						FilePath:         "folder1_path",
					},
					Children: []*DocumentNode{
						{
							DocumentInfo: DocumentInfo{
								Name:             "doc1",
								Token:            "doc1_token",
								Type:             constant.DocTypeDoc,
								DownloadDirectly: false,
								FileExtension:    constant.FileExtDocx,
								CanDownload:      false, // 设置成不能下载，不跑 exportDocuments 和 downloadDocuments
								FilePath:         "doc1_path.docx",
							},
						},
						{
							DocumentInfo: DocumentInfo{
								Name:             "doc2",
								Token:            "doc2_token",
								Type:             constant.DocTypeDocx,
								DownloadDirectly: false,
								FileExtension:    constant.FileExtPDF,
								CanDownload:      false, // 设置成不能下载，不跑 exportDocuments 和 downloadDocuments
								FilePath:         "doc2_path.pdf",
							},
						},
					},
				}
				args.mockClient.EXPECT().GetArgs().Return(&argument.Args{
					SaveDir:           "/tmp",
					ListOnly:          false,
					QuitAutomatically: true,
				}).Maybe() // 可能结束太快没调用到，所以用maybe
				args.mockProgram.EXPECT().Run().Return(nil, oops.New("下载UI程序报错")).Once()
				args.mockProgram.EXPECT().Quit().Maybe()
			},
			expectedError: oops.New("下载UI程序报错"),
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			// 不共用一个args，避免测试间相互影响 DATA RACE
			args := getMockRunArgs(s.T())
			// 设置mock
			tt.setupMock(args)
			// 执行测试
			err := args.task.Run()
			defer os.Remove(filepath.Join("/tmp", "document-tree.json"))
			// 验证结果
			if tt.expectedError != nil {
				s.Require().Error(err, tt.name)
				s.IsType(oops.OopsError{}, err, tt.name)
				var actualError oops.OopsError
				yes := errors.As(err, &actualError)
				s.Require().True(yes, tt.name)
				s.Equal("", actualError.Code(), tt.name)
				s.Equal(tt.expectedError.Error(), actualError.Error(), tt.name)
			} else {
				s.Require().NoError(err, tt.name)
			}
		})
	}
}

func (s *TaskImplTestSuite) TestTaskImpl_Close() {
	s.task.queue = make(chan *exportResult, 1)
	s.task.wait = make(chan struct{})
	s.task.Close()
	s.task.completed.Store(true)
}

func (s *TaskImplTestSuite) TestTaskImpl_Interrupt() {
	s.mockProgram.EXPECT().Quit().Once()
	s.task.Interrupt()
}

func (s *TaskImplTestSuite) TestTaskImpl_Complete() {
	s.task.queue = make(chan *exportResult, 1)
	s.task.wait = make(chan struct{})
	go func() {
		<-s.task.wait
	}()
	s.task.Complete()
}

func (s *TaskImplTestSuite) TestTaskImpl_exportDocuments() {
	// 注意涉及到协程的测试，要让单测保持串行，所以需要让每个用例等待处理完再往后执行
	tests := []struct {
		name      string
		setupMock func(name string) (args []any)
		want      func(name string, completed *atomic.Bool, args []any)
	}{
		{
			name: "正常执行[直接完成]",
			setupMock: func(name string) (args []any) {
				s.task.completed.Store(true)
				return nil
			},
			want: func(name string, completed *atomic.Bool, args []any) {
				s.waitToContinue(completed)
				t := s.T()
				s.mockProgram.AssertNotCalled(t, "Add")
				s.mockProgram.AssertNotCalled(t, "Update")
				s.mockExporter.AssertNotCalled(t, "doExport")
				s.mockExporter.AssertNotCalled(t, "checkExport")
			},
		},
		{
			name: "正常执行[无数据]",
			setupMock: func(name string) (args []any) {
				return nil
			},
			want: func(name string, completed *atomic.Bool, args []any) {
				s.waitToContinue(completed)
				t := s.T()
				s.mockProgram.AssertNotCalled(t, "Add")
				s.mockProgram.AssertNotCalled(t, "Update")
				s.mockExporter.AssertNotCalled(t, "doExport")
				s.mockExporter.AssertNotCalled(t, "checkExport")
			},
		},
		{
			name: "正常执行[直接下载一条数据]",
			setupMock: func(name string) (args []any) {
				s.task.Docs = &DocumentNode{
					DocumentInfo: DocumentInfo{
						Name:             "doc1",
						Token:            "doc1_token",
						Type:             constant.DocTypeDoc,
						DownloadDirectly: true,
						FileExtension:    constant.FileExtDocx,
						CanDownload:      true,
						FilePath:         "doc1_path.docx",
					},
					Children: []*DocumentNode(nil),
				}
				infoList := documentTreeToInfoList(s.task.Docs, "/tmp")
				// 初始化必要参数备用
				s.task.canDownloadList = lo.Filter(infoList, func(di *DocumentInfo, _ int) bool { return di.CanDownload })
				s.task.countDown = &atomic.Int32{}
				s.task.countDown.Store(int32(len(s.task.canDownloadList)))
				s.task.completed.Store(false)
				s.task.queue = make(chan *exportResult, 1)
				s.mockProgram.EXPECT().Add(s.task.Docs.FilePath, "doc1.docx").Once()
				return []any{&exportResult{DocumentInfo: infoList[0], result: nil}}
			},
			want: func(name string, completed *atomic.Bool, args []any) {
				got := s.receiveFromQueue(1)
				s.waitToContinue(completed)
				t := s.T()
				s.mockProgram.AssertNotCalled(t, "Update")
				s.mockExporter.AssertNotCalled(t, "doExport")
				s.mockExporter.AssertNotCalled(t, "checkExport")
				s.Subset([]*exportResult{args[0].(*exportResult)}, got, name)
			},
		},
		{
			name: "正常执行[有数据]",
			setupMock: func(name string) (args []any) {
				di1 := &DocumentNode{
					DocumentInfo: DocumentInfo{
						Name:             "doc1",
						Token:            "doc1_token",
						Type:             constant.DocTypeDoc,
						DownloadDirectly: false,
						FileExtension:    constant.FileExtDocx,
						CanDownload:      true,
						FilePath:         "doc1_path.docx",
					},
				}
				di2 := &DocumentNode{
					DocumentInfo: DocumentInfo{
						Name:             "doc2",
						Token:            "doc2_token",
						Type:             constant.DocTypeDocx,
						DownloadDirectly: false,
						FileExtension:    constant.FileExtPDF,
						CanDownload:      true,
						FilePath:         "doc2_path.pdf",
					},
				}
				s.task.Docs = &DocumentNode{
					DocumentInfo: DocumentInfo{
						Name:             "folder1",
						Token:            "folder1_token",
						Type:             constant.DocTypeFolder,
						DownloadDirectly: false,
						FileExtension:    "folder",
						CanDownload:      false,
						FilePath:         "folder1_path",
					},
					Children: []*DocumentNode{di1, di2},
				}
				infoList := documentTreeToInfoList(s.task.Docs, "/tmp")
				exportResults := []*exportResult{
					{
						DocumentInfo: infoList[1],
						result: &larkdrive.ExportTask{
							Token:         larkcore.StringPtr(di1.Token),
							FileExtension: larkcore.StringPtr(string(di1.FileExtension)),
						},
					},
					{
						DocumentInfo: infoList[2],
						result: &larkdrive.ExportTask{
							FileToken:     larkcore.StringPtr(di2.Token),
							FileExtension: larkcore.StringPtr(string(di2.FileExtension)),
						},
					},
				}
				// 初始化必要参数备用
				s.task.canDownloadList = lo.Filter(infoList, func(di *DocumentInfo, _ int) bool { return di.CanDownload })
				s.task.countDown = &atomic.Int32{}
				s.task.countDown.Store(int32(len(s.task.canDownloadList)))
				s.task.completed.Store(false)
				s.task.queue = make(chan *exportResult, 2)

				s.mockProgram.EXPECT().Add(di1.FilePath, "doc1.docx").Once()
				s.mockExporter.EXPECT().doExport(&di1.DocumentInfo).Return("ticket1", nil).Once()
				s.mockProgram.EXPECT().Update(di1.FilePath, 0.05, progress.StatusExporting).Once()
				s.mockExporter.EXPECT().checkExport(&di1.DocumentInfo, "ticket1").Return(exportResults[0], progress.StatusExported, nil).Once()
				s.mockProgram.EXPECT().Update(di1.FilePath, 0.15, progress.StatusExported).Once()
				s.mockProgram.EXPECT().Update(di1.FilePath, 0.15, progress.StatusWaiting).Once()

				s.mockProgram.EXPECT().Add(di2.FilePath, "doc2.pdf").Once()
				s.mockExporter.EXPECT().doExport(&di2.DocumentInfo).Return("ticket2", nil).Once()
				s.mockProgram.EXPECT().Update(di2.FilePath, 0.05, progress.StatusExporting).Once()
				s.mockExporter.EXPECT().checkExport(&di2.DocumentInfo, "ticket2").Return(exportResults[1], progress.StatusExported, nil).Once()
				s.mockProgram.EXPECT().Update(di2.FilePath, 0.15, progress.StatusExported).Once()
				s.mockProgram.EXPECT().Update(di2.FilePath, 0.15, progress.StatusWaiting).Once()
				return []any{exportResults[0], exportResults[1]}
			},
			want: func(name string, completed *atomic.Bool, args []any) {
				got := s.receiveFromQueue(2)
				s.waitToContinue(completed)
				s.Subset([]*exportResult{args[0].(*exportResult), args[1].(*exportResult)}, got, name)
			},
		},
		{
			name: "创建导出任务失败",
			setupMock: func(name string) (args []any) {
				di1 := &DocumentNode{
					DocumentInfo: DocumentInfo{
						Name:             "doc1",
						Token:            "doc1_token",
						Type:             constant.DocTypeDoc,
						DownloadDirectly: false,
						FileExtension:    constant.FileExtDocx,
						CanDownload:      true,
						FilePath:         "doc1_path.docx",
					},
				}
				s.task.Docs = &DocumentNode{
					DocumentInfo: DocumentInfo{
						Name:             "folder1",
						Token:            "folder1_token",
						Type:             constant.DocTypeFolder,
						DownloadDirectly: false,
						FileExtension:    "folder",
						CanDownload:      false,
						FilePath:         "folder1_path",
					},
					Children: []*DocumentNode{di1},
				}
				infoList := documentTreeToInfoList(s.task.Docs, "/tmp")
				exportResults := []*exportResult{
					{
						DocumentInfo: infoList[1],
						result: &larkdrive.ExportTask{
							Token:         larkcore.StringPtr(di1.Token),
							FileExtension: larkcore.StringPtr(string(di1.FileExtension)),
						},
					},
				}
				// 初始化必要参数备用
				s.task.canDownloadList = lo.Filter(infoList, func(di *DocumentInfo, _ int) bool { return di.CanDownload })
				s.task.countDown = &atomic.Int32{}
				s.task.countDown.Store(int32(len(s.task.canDownloadList)))
				s.task.completed.Store(false)
				s.task.queue = make(chan *exportResult, 2)

				s.mockProgram.EXPECT().Add(di1.FilePath, "doc1.docx").Once()
				s.mockExporter.EXPECT().doExport(&di1.DocumentInfo).Return("", oops.New("创建导出任务失败")).Once()
				s.mockProgram.EXPECT().Update(di1.FilePath, 0.05, progress.StatusFailed, "创建导出任务失败").Once()
				return []any{exportResults[0]}
			},
			want: func(name string, completed *atomic.Bool, args []any) {
				s.waitToContinue(completed)
				t := s.T()
				s.mockExporter.AssertNotCalled(t, "checkExport")
			},
		},
		{
			name: "查询导出任务结果失败",
			setupMock: func(name string) (args []any) {
				di1 := &DocumentNode{
					DocumentInfo: DocumentInfo{
						Name:             "doc1",
						Token:            "doc1_token",
						Type:             constant.DocTypeDoc,
						DownloadDirectly: false,
						FileExtension:    constant.FileExtDocx,
						CanDownload:      true,
						FilePath:         "doc1_path.docx",
					},
				}
				s.task.Docs = &DocumentNode{
					DocumentInfo: DocumentInfo{
						Name:             "folder1",
						Token:            "folder1_token",
						Type:             constant.DocTypeFolder,
						DownloadDirectly: false,
						FileExtension:    "folder",
						CanDownload:      false,
						FilePath:         "folder1_path",
					},
					Children: []*DocumentNode{di1},
				}
				infoList := documentTreeToInfoList(s.task.Docs, "/tmp")
				exportResults := []*exportResult{
					{
						DocumentInfo: infoList[1],
						result: &larkdrive.ExportTask{
							Token:         larkcore.StringPtr(di1.Token),
							FileExtension: larkcore.StringPtr(string(di1.FileExtension)),
						},
					},
				}
				// 初始化必要参数备用
				s.task.canDownloadList = lo.Filter(infoList, func(di *DocumentInfo, _ int) bool { return di.CanDownload })
				s.task.countDown = &atomic.Int32{}
				s.task.countDown.Store(int32(len(s.task.canDownloadList)))
				s.task.completed.Store(false)
				s.task.queue = make(chan *exportResult, 2)

				s.mockProgram.EXPECT().Add(di1.FilePath, "doc1.docx").Once()
				s.mockExporter.EXPECT().doExport(&di1.DocumentInfo).Return("ticket1", nil).Once()
				s.mockProgram.EXPECT().Update(di1.FilePath, 0.05, progress.StatusExporting).Once()
				s.mockExporter.EXPECT().checkExport(&di1.DocumentInfo, "ticket1").Return(nil, progress.StatusFailed, oops.New("查询导出任务结果失败")).Once()
				s.mockProgram.EXPECT().Update(di1.FilePath, 0.10, progress.StatusFailed, "查询导出任务结果失败").Once()
				return []any{exportResults[0]}
			},
			want: func(name string, completed *atomic.Bool, args []any) {
				s.waitToContinue(completed)
			},
		},
		{
			name: "查询导出任务结果中断",
			setupMock: func(name string) (args []any) {
				di1 := &DocumentNode{
					DocumentInfo: DocumentInfo{
						Name:             "doc1",
						Token:            "doc1_token",
						Type:             constant.DocTypeDoc,
						DownloadDirectly: false,
						FileExtension:    constant.FileExtDocx,
						CanDownload:      true,
						FilePath:         "doc1_path.docx",
					},
				}
				s.task.Docs = &DocumentNode{
					DocumentInfo: DocumentInfo{
						Name:             "folder1",
						Token:            "folder1_token",
						Type:             constant.DocTypeFolder,
						DownloadDirectly: false,
						FileExtension:    "folder",
						CanDownload:      false,
						FilePath:         "folder1_path",
					},
					Children: []*DocumentNode{di1},
				}
				infoList := documentTreeToInfoList(s.task.Docs, "/tmp")
				exportResults := []*exportResult{
					{
						DocumentInfo: infoList[1],
						result: &larkdrive.ExportTask{
							Token:         larkcore.StringPtr(di1.Token),
							FileExtension: larkcore.StringPtr(string(di1.FileExtension)),
						},
					},
				}
				// 初始化必要参数备用
				s.task.canDownloadList = lo.Filter(infoList, func(di *DocumentInfo, _ int) bool { return di.CanDownload })
				s.task.countDown = &atomic.Int32{}
				s.task.countDown.Store(int32(len(s.task.canDownloadList)))
				s.task.completed.Store(false)
				s.task.queue = make(chan *exportResult, 2)

				s.mockProgram.EXPECT().Add(di1.FilePath, "doc1.docx").Once()
				s.mockExporter.EXPECT().doExport(&di1.DocumentInfo).Return("ticket1", nil).Once()
				s.mockProgram.EXPECT().Update(di1.FilePath, 0.05, progress.StatusExporting).Once()
				s.mockExporter.EXPECT().checkExport(&di1.DocumentInfo, "ticket1").Return(nil, progress.StatusInterrupted, nil).Once()
				return []any{exportResults[0]}
			},
			want: func(name string, completed *atomic.Bool, args []any) {
				s.waitToContinue(completed)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			// 设置mock
			args := tt.setupMock(tt.name)
			// 执行测试
			completed := s.task.exportDocuments()
			// 验证结果
			tt.want(tt.name, completed, args)
		})
	}
}

func (s *TaskImplTestSuite) waitToContinue(completed *atomic.Bool) {
	timeout := time.After(time.Second * 2)
	for {
		select {
		case <-timeout:
			return
		default:
			if completed.Load() {
				return
			}
			time.Sleep(time.Millisecond * 100)
		}
	}
}

func (s *TaskImplTestSuite) receiveFromQueue(length int) (got []*exportResult) {
	// 新开一个协程，从 s.task.queue 中读取数据，需要读取到2个 或者 等到超时才结束
	var wg sync.WaitGroup
	wg.Add(1)
	var lock sync.Mutex
	go func() {
		defer wg.Done()
		timeout := time.After(time.Second * 2)
		for {
			if len(got) == length {
				return
			}
			select {
			case er := <-s.task.queue:
				lock.Lock()
				got = append(got, er)
				lock.Unlock()
			case <-timeout:
				return
			default:
			}
		}
	}()
	wg.Wait()
	return got
}

func (s *TaskImplTestSuite) TestTaskImpl_downloadDocuments() {
	tests := []struct {
		name      string
		setupMock func(name string) (args []any)
		want      func(name string, completed *atomic.Bool, args []any)
	}{
		{
			name: "直接完成",
			setupMock: func(name string) (args []any) {
				s.task.completed.Store(true)
				return nil
			},
			want: func(name string, completed *atomic.Bool, args []any) {
				s.waitToContinue(completed)
				t := s.T()
				s.mockExporter.AssertNotCalled(t, "doDownloadDirectly")
				s.mockExporter.AssertNotCalled(t, "doDownloadExported")
				s.mockProgram.AssertNotCalled(t, "Update")
				s.mockProgram.AssertNotCalled(t, "Quit")
				s.mockClient.AssertNotCalled(t, "GetArgs")
			},
		},
		{
			name: "队列关闭了",
			setupMock: func(name string) (args []any) {
				s.task.countDown = &atomic.Int32{}
				s.task.completed.Store(false)
				s.task.queue = make(chan *exportResult, 2)
				s.mockClient.EXPECT().GetArgs().Return(&argument.Args{QuitAutomatically: true}).Maybe()
				s.mockProgram.EXPECT().Quit().Maybe()
				return nil
			},
			want: func(name string, completed *atomic.Bool, args []any) {
				close(s.task.queue)
				s.waitToContinue(completed)
				t := s.T()
				s.mockExporter.AssertNotCalled(t, "doDownloadDirectly")
				s.mockExporter.AssertNotCalled(t, "doDownloadExported")
				s.mockProgram.AssertNotCalled(t, "Update")
			},
		},
		{
			name: "正常下载[直接下载文件和下载导出的文档]",
			setupMock: func(name string) (args []any) {
				di1 := &DocumentNode{
					DocumentInfo: DocumentInfo{
						Name:             "doc1",
						Token:            "doc1_token",
						Type:             constant.DocTypeDoc,
						DownloadDirectly: true,
						FileExtension:    constant.FileExtDocx,
						CanDownload:      true,
						FilePath:         "doc1_path.docx",
					},
				}
				di2 := &DocumentNode{
					DocumentInfo: DocumentInfo{
						Name:             "doc2",
						Token:            "doc2_token",
						Type:             constant.DocTypeDocx,
						DownloadDirectly: false,
						FileExtension:    constant.FileExtPDF,
						CanDownload:      true,
						FilePath:         "doc2_path.pdf",
					},
				}
				s.task.Docs = &DocumentNode{
					DocumentInfo: DocumentInfo{
						Name:             "folder1",
						Token:            "folder1_token",
						Type:             constant.DocTypeFolder,
						DownloadDirectly: false,
						FileExtension:    "folder",
						CanDownload:      false,
						FilePath:         "folder1_path",
					},
					Children: []*DocumentNode{di1, di2},
				}
				infoList := documentTreeToInfoList(s.task.Docs, "/tmp")
				exportedContent := "mock导出的文件内容"
				exportResults := []*exportResult{
					{
						DocumentInfo: infoList[1],
						result:       nil,
					},
					{
						DocumentInfo: infoList[2],
						result: &larkdrive.ExportTask{
							FileToken:     larkcore.StringPtr(di2.Token),
							FileExtension: larkcore.StringPtr(string(di2.FileExtension)),
							FileSize:      larkcore.IntPtr(len(exportedContent)),
						},
					},
				}
				// 初始化必要参数备用
				s.task.canDownloadList = lo.Filter(infoList, func(di *DocumentInfo, _ int) bool { return di.CanDownload })
				s.task.countDown = &atomic.Int32{}
				s.task.countDown.Store(int32(len(s.task.canDownloadList)))
				s.task.completed.Store(false)
				s.task.queue = make(chan *exportResult, 2)
				directlyContent := "mock直接下载的文件内容"
				directlyFileSize := int64(len(directlyContent))
				s.mockExporter.EXPECT().doDownloadDirectly(di1.FilePath, di1.Token).
					Return(strings.NewReader(directlyContent), directlyFileSize, nil).Once()
				s.mockExporter.EXPECT().doDownloadExported(
					di2.FilePath, larkcore.StringValue(exportResults[1].result.FileToken),
				).Return(strings.NewReader(exportedContent), nil).Once()
				s.mockProgram.EXPECT().Update(di1.FilePath, 0.20, progress.StatusDownloading).Once()
				s.mockProgram.EXPECT().Update(di2.FilePath, 0.20, progress.StatusDownloading).Once()
				fn := func(pg float64) bool {
					return pg >= 0.20 && pg <= 1.00
				}
				s.mockProgram.EXPECT().Update(di1.FilePath, mock.MatchedBy(fn), progress.StatusDownloading,
					"total: %d, wrote: %d", directlyFileSize, mock.Anything).Maybe()
				s.mockProgram.EXPECT().Update(di2.FilePath, mock.MatchedBy(fn), progress.StatusDownloading,
					"total: %d, wrote: %d", int64(len(exportedContent)), mock.Anything).Maybe()
				s.mockProgram.EXPECT().Update(di1.FilePath, 1.00, progress.StatusCompleted).Once()
				s.mockProgram.EXPECT().Update(di2.FilePath, 1.00, progress.StatusCompleted).Once()
				s.mockClient.EXPECT().GetArgs().Return(&argument.Args{QuitAutomatically: true}).Maybe()
				s.mockProgram.EXPECT().Quit().Maybe()
				return []any{exportResults}
			},
			want: func(name string, completed *atomic.Bool, args []any) {
				exportResults := args[0].([]*exportResult)
				for _, er := range exportResults {
					s.task.queue <- er
				}
				s.waitToContinue(completed)
				os.Remove(filepath.Join("/tmp", exportResults[0].FilePath))
				os.Remove(filepath.Join("/tmp", exportResults[1].FilePath))
			},
		},
		{
			name: "下载失败",
			setupMock: func(name string) (args []any) {
				di1 := &DocumentNode{
					DocumentInfo: DocumentInfo{
						Name:             "doc1",
						Token:            "doc1_token",
						Type:             constant.DocTypeDoc,
						DownloadDirectly: false,
						FileExtension:    constant.FileExtDocx,
						CanDownload:      true,
						FilePath:         "doc1_path.docx",
					},
				}
				exportedContent := "mock导出的文件内容"
				exportResults := []*exportResult{
					{
						DocumentInfo: &di1.DocumentInfo,
						result: &larkdrive.ExportTask{
							FileToken:     larkcore.StringPtr(di1.Token),
							FileExtension: larkcore.StringPtr(string(di1.FileExtension)),
							FileSize:      larkcore.IntPtr(len(exportedContent)),
						},
					},
				}
				s.task.canDownloadList = []*DocumentInfo{&di1.DocumentInfo}
				s.task.countDown = &atomic.Int32{}
				s.task.countDown.Store(int32(len(s.task.canDownloadList)))
				s.task.completed.Store(false)
				s.task.queue = make(chan *exportResult, 2)
				s.mockExporter.EXPECT().doDownloadExported(di1.FilePath, di1.Token).Return(nil, oops.New("下载失败")).Once()
				s.mockProgram.EXPECT().Update(di1.FilePath, 0.18, progress.StatusFailed, "下载失败").Once()
				s.mockClient.EXPECT().GetArgs().Return(&argument.Args{QuitAutomatically: true}).Maybe()
				s.mockProgram.EXPECT().Quit().Maybe()
				return []any{exportResults}
			},
			want: func(name string, completed *atomic.Bool, args []any) {
				exportResults := args[0].([]*exportResult)
				for _, er := range exportResults {
					s.task.queue <- er
				}
				s.waitToContinue(completed)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			// 设置mock
			args := tt.setupMock(tt.name)
			// 执行测试
			completed := s.task.downloadDocuments()
			// 验证结果
			tt.want(tt.name, completed, args)
		})
	}
}

func (s *TaskImplTestSuite) Test_calculateOverallProgress() {
	tests := []struct {
		name                      string
		canDownloadCount          int
		total, downloaded, failed int
		expected                  string
	}{
		{
			name:             "部分完成有失败1",
			canDownloadCount: 10,
			total:            8,
			downloaded:       6,
			failed:           2,
			expected:         "可下载: 10, 已提交: 8, 已下载: 6, 未下载: 0, 已失败: 2\n████████████████████████████████████████████░░░░░░░░░░░  80%",
		},
		{
			name:             "全部完成有失败2",
			canDownloadCount: 10,
			total:            10,
			downloaded:       8,
			failed:           2,
			expected:         "可下载: 10, 已提交: 10, 已下载: 8, 未下载: 0, 已失败: 2\n███████████████████████████████████████████████████████ 100%",
		},
		{
			name:             "全部完成",
			canDownloadCount: 10,
			total:            10,
			downloaded:       10,
			failed:           0,
			expected:         "可下载: 10, 已提交: 10, 已下载: 10, 未下载: 0, 已失败: 0\n███████████████████████████████████████████████████████ 100%",
		},
		{
			name:             "部分完成",
			canDownloadCount: 10,
			total:            7,
			downloaded:       3,
			failed:           0,
			expected:         "可下载: 10, 已提交: 7, 已下载: 3, 未下载: 4, 已失败: 0\n█████████████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  30%",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			stats := calculateOverallProgress(tt.canDownloadCount)
			actual := stats(tt.total, tt.downloaded, tt.failed)
			s.Require().Equal(tt.expected, actual, tt.name)
		})
	}
}

// mockError Mock types for testing。
type mockError struct {
	logID string
	msg   string
}

func (m *mockError) Error() string {
	return m.msg
}

// 因为飞书的SDK中有命名为RequestId的函数，这里需要mock，所以针对性禁用var-naming规则
// revive:disable:var-naming
func (m *mockError) RequestId() string {
	// revive:enable:var-naming
	return m.logID
}

// TestToErrMsg 测试错误信息格式化。
func (s *TaskImplTestSuite) Test_toErrMsg() {
	tests := []struct {
		name      string
		err       error
		operation string
		expected  string
	}{
		{
			name: "带有日志ID",
			err: &mockError{
				logID: "123",
				msg:   "error\nwith\nlines",
			},
			operation: "upload",
			expected: fmt.Sprintf("logId: %s, 操作: upload, 响应错误: error with lines",
				progress.URLStyleRender("https://open.feishu.cn/search?q=123")),
		},
		{
			name:      "无日志ID",
			err:       errors.New("some error"),
			operation: "download",
			expected:  "响应错误: some error",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := toErrMsg(tt.err, tt.operation)
			s.Equal(tt.expected, result, tt.name)
		})
	}
}

// TestCleanEnter 测试清除换行符。
func (s *TaskImplTestSuite) Test_cleanEnter() {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "包含换行符",
			err:      errors.New("error\nwith\nlines"),
			expected: "error with lines",
		},
		{
			name:     "无换行符",
			err:      errors.New("error message"),
			expected: "error message",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			cleaned := cleanEnter(tt.err)
			s.Equal(tt.expected, cleaned, tt.name)
		})
	}
}
