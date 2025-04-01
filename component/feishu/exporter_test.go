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
	"bytes"
	"errors"
	"io"
	"net/http"
	"sync/atomic"
	"testing"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkdrive "github.com/larksuite/oapi-sdk-go/v3/service/drive/v1"
	"github.com/samber/oops"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/acyumi/xdoc/component/constant"
	"github.com/acyumi/xdoc/component/progress"
)

func Test_exporterSuite(t *testing.T) {
	suite.Run(t, new(exporterTestSuite))
}

type exporterTestSuite struct {
	suite.Suite
	task        exporter
	mockClient  *MockClient
	mockProgram *MockProgram
}

func (s *exporterTestSuite) SetupTest() {
	s.mockClient = &MockClient{}
	s.mockProgram = &MockProgram{}
	s.task = exporter{
		client:    s.mockClient, // 模拟Client
		program:   s.mockProgram,
		completed: &atomic.Bool{},
	}
}

func (s *exporterTestSuite) TearDownTest() {
}

// TestDoExport 测试创建导出任务。
func (s *exporterTestSuite) Test_exporter_doExport() {
	tests := []struct {
		name           string
		di             *DocumentInfo
		setupMock      func(di *DocumentInfo)
		expectedTicket string
		expectedError  error
	}{
		{
			name: "成功创建导出任务",
			di: &DocumentInfo{
				Token: "file_token",
				Type:  "doc",
			},
			setupMock: func(di *DocumentInfo) {
				s.mockProgram.EXPECT().Update(mock.Anything, 0.0, progress.StatusExporting, "请求%d次", mock.Anything).Return().Once()
				s.mockClient.EXPECT().ExportCreate(
					mock.Anything,
					mock.MatchedBy(func(req *larkdrive.CreateExportTaskReq) bool {
						return larkcore.StringValue(req.ExportTask.Token) == di.Token &&
							larkcore.StringValue(req.ExportTask.Type) == string(di.Type) &&
							larkcore.StringValue(req.ExportTask.FileExtension) == string(di.FileExtension)
					}),
				).Return(&larkdrive.CreateExportTaskResp{
					Data: &larkdrive.CreateExportTaskRespData{
						Ticket: larkcore.StringPtr("export_ticket_123"),
					},
				}, nil).Once()
			},
			expectedTicket: "export_ticket_123",
			expectedError:  nil,
		},
		{
			name: "客户端返回错误",
			di: &DocumentInfo{
				Token: "invalid_token",
				Type:  "doc",
			},
			setupMock: func(di *DocumentInfo) {
				s.mockProgram.EXPECT().Update(mock.Anything, 0.0, progress.StatusExporting, "请求%d次", mock.Anything).Return().Once()
				s.mockClient.EXPECT().ExportCreate(
					mock.Anything,
					mock.MatchedBy(func(req *larkdrive.CreateExportTaskReq) bool {
						return larkcore.StringValue(req.ExportTask.Token) == di.Token &&
							larkcore.StringValue(req.ExportTask.Type) == string(di.Type) &&
							larkcore.StringValue(req.ExportTask.FileExtension) == string(di.FileExtension)
					}),
				).Return(nil, errors.New("API调用失败")).Once()
			},
			expectedError: errors.New("API调用失败"),
		},
		{
			name: "无效文档类型",
			di: &DocumentInfo{
				Token: "file_token",
				Type:  "unknown_type",
			},
			setupMock: func(di *DocumentInfo) {
				s.mockProgram.EXPECT().Update(mock.Anything, 0.0, progress.StatusExporting, "请求%d次", mock.Anything).Return().Once()
				resp := &larkdrive.CreateExportTaskResp{
					ApiResp: &larkcore.ApiResp{
						Header: http.Header{
							larkcore.HttpHeaderKeyLogId: []string{"1111111111"},
						},
					},
					// 模拟API返回错误响应
					CodeError: larkcore.CodeError{Code: 400, Msg: "无效类型"},
				}
				s.mockClient.EXPECT().ExportCreate(
					mock.Anything,
					mock.MatchedBy(func(req *larkdrive.CreateExportTaskReq) bool {
						return larkcore.StringValue(req.ExportTask.Token) == di.Token &&
							larkcore.StringValue(req.ExportTask.Type) == string(di.Type) &&
							larkcore.StringValue(req.ExportTask.FileExtension) == string(di.FileExtension)
					}),
				).Return(checkResp(resp, nil)).Once()
			},
			expectedError: oops.New("logId: \x1b]8;;https://open.feishu.cn/search?q=1111111111\x1b\\, 操作: 创建导出任务, 响应错误: msg:无效类型,code:400"),
		},
		{
			name: "空Token参数",
			di: &DocumentInfo{
				Token: "",
				Type:  "doc",
			},
			setupMock: func(di *DocumentInfo) {
				s.mockProgram.EXPECT().Update(mock.Anything, 0.0, progress.StatusExporting, "请求%d次", mock.Anything).Return().Once()
				s.mockClient.EXPECT().ExportCreate(
					mock.Anything,
					mock.MatchedBy(func(req *larkdrive.CreateExportTaskReq) bool {
						return larkcore.StringValue(req.ExportTask.Token) == di.Token &&
							larkcore.StringValue(req.ExportTask.Type) == string(di.Type) &&
							larkcore.StringValue(req.ExportTask.FileExtension) == string(di.FileExtension)
					}),
				).Return(nil, oops.New("Token不能为空")).Once()
			},
			expectedError: oops.New("Token不能为空"),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// 设置mock
			tt.setupMock(tt.di)

			// 执行测试
			ticket, err := s.task.doExport(tt.di)

			// 验证结果
			if tt.expectedError != nil {
				s.Require().Error(err, tt.name)
				s.IsType(oops.OopsError{}, err, tt.name)
				var actualError oops.OopsError
				yes := errors.As(err, &actualError)
				s.Require().True(yes, tt.name)
				s.Equal("", actualError.Code(), tt.name)
				s.Equal(tt.expectedError.Error(), actualError.Error(), tt.name)
				s.Empty(ticket, tt.name)
			} else {
				s.Require().NoError(err, tt.name)
				s.Equal(tt.expectedTicket, ticket, tt.name)
			}
		})
	}
}

// TestCheckExport 测试导出任务状态检查。
func (s *exporterTestSuite) Test_exporter_checkExport() {
	s.task.completed = &atomic.Bool{}
	// s.task.queue = make(chan *exportResult, 20)
	// s.task.wait = make(chan struct{})
	// 测试用例
	di := &DocumentInfo{
		Name:             "doc1",
		Token:            "doc1_token",
		Type:             constant.DocTypeDoc,
		DownloadDirectly: false,
		FileExtension:    constant.FileExtDocx,
		CanDownload:      true,
		FilePath:         "doc1_path",
	}
	tests := []struct {
		name           string
		di             *DocumentInfo
		ticket         string
		setupMock      func(di *DocumentInfo, ticket string)
		expectedResult *exportResult
		expectedStat   progress.Status
		expectedError  error
	}{
		{
			name:   "成功",
			di:     di,
			ticket: "doc1_ticket",
			setupMock: func(di *DocumentInfo, ticket string) {
				s.mockProgram.EXPECT().Update(di.FilePath, 0.10, progress.StatusExporting, "查询%d次", mock.Anything).Once()
				req := larkdrive.NewGetExportTaskReqBuilder().Ticket(ticket).Token(di.Token).Build()
				s.mockClient.EXPECT().ExportGet(mock.Anything, req).Return(
					&larkdrive.GetExportTaskResp{
						Data: &larkdrive.GetExportTaskRespData{
							Result: &larkdrive.ExportTask{
								JobStatus:   larkcore.IntPtr(0),
								JobErrorMsg: larkcore.StringPtr(""),
							},
						},
					},
					nil,
				).Once()
			},
			expectedResult: &exportResult{
				DocumentInfo: di,
				result: &larkdrive.ExportTask{
					JobStatus:   larkcore.IntPtr(0),
					JobErrorMsg: larkcore.StringPtr(""),
				},
			},
			expectedStat:  progress.StatusExported,
			expectedError: nil,
		},
		{
			name:   "直接中断",
			di:     di,
			ticket: "doc1_ticket",
			setupMock: func(di *DocumentInfo, ticket string) {
				s.task.completed.Store(true)
			},
			expectedResult: nil,
			expectedStat:   progress.StatusInterrupted,
			expectedError:  nil,
		},
		{
			name:   "初始化, 尝试5次后超时失败",
			di:     di,
			ticket: "doc1_ticket",
			setupMock: func(di *DocumentInfo, ticket string) {
				s.mockProgram.EXPECT().Update(di.FilePath, 0.10, progress.StatusExporting, "查询%d次", mock.Anything).Times(5)
				req := larkdrive.NewGetExportTaskReqBuilder().Ticket(ticket).Token(di.Token).Build()
				s.mockClient.EXPECT().ExportGet(mock.Anything, req).Return(
					&larkdrive.GetExportTaskResp{
						Data: &larkdrive.GetExportTaskRespData{
							Result: &larkdrive.ExportTask{
								JobStatus:   larkcore.IntPtr(1),
								JobErrorMsg: larkcore.StringPtr(""),
							},
						},
					},
					nil,
				).Times(5)
				s.mockProgram.EXPECT().Update(di.FilePath, 0.10, progress.StatusExporting, "等待完成导出任务").Times(5)
			},
			expectedResult: &exportResult{
				DocumentInfo: di,
				result: &larkdrive.ExportTask{
					JobStatus:   larkcore.IntPtr(1),
					JobErrorMsg: larkcore.StringPtr(""),
				},
			},
			expectedStat:  progress.StatusFailed,
			expectedError: oops.New("经过多次尝试取不到导出任务结果"),
		},
		{
			name:   "请求报错",
			di:     di,
			ticket: "doc1_ticket",
			setupMock: func(di *DocumentInfo, ticket string) {
				s.mockProgram.EXPECT().Update(di.FilePath, 0.10, progress.StatusExporting, "查询%d次", mock.Anything).Once()
				req := larkdrive.NewGetExportTaskReqBuilder().Ticket(ticket).Token(di.Token).Build()
				s.mockClient.EXPECT().ExportGet(mock.Anything, req).Return(nil, errors.New("请求报错")).Once()
			},
			expectedResult: nil,
			expectedStat:   progress.StatusFailed,
			expectedError:  oops.New("请求报错"),
		},
		{
			name:   "请求失败",
			di:     di,
			ticket: "doc1_ticket",
			setupMock: func(di *DocumentInfo, ticket string) {
				s.mockProgram.EXPECT().Update(di.FilePath, 0.10, progress.StatusExporting, "查询%d次", mock.Anything).Once()
				req := larkdrive.NewGetExportTaskReqBuilder().Ticket(ticket).Token(di.Token).Build()
				resp := &larkdrive.GetExportTaskResp{
					ApiResp: &larkcore.ApiResp{
						Header: http.Header{
							larkcore.HttpHeaderKeyLogId: []string{"1111111111"},
						},
					},
					CodeError: larkcore.CodeError{Code: 111},
					Data: &larkdrive.GetExportTaskRespData{
						Result: &larkdrive.ExportTask{
							JobStatus:   larkcore.IntPtr(3),
							JobErrorMsg: larkcore.StringPtr("请求失败"),
						},
					},
				}
				s.mockClient.EXPECT().ExportGet(mock.Anything, req).Return(checkResp(resp, nil)).Once()
			},
			expectedResult: nil,
			expectedStat:   progress.StatusFailed,
			expectedError:  oops.New("logId: \x1b]8;;https://open.feishu.cn/search?q=1111111111\x1b\\, 操作: 查询导出任务结果, 响应错误: msg:,code:111"),
		},
		{
			name:   "请求成功，但服务端返回失败",
			di:     di,
			ticket: "doc1_ticket",
			setupMock: func(di *DocumentInfo, ticket string) {
				s.mockProgram.EXPECT().Update(di.FilePath, 0.10, progress.StatusExporting, "查询%d次", mock.Anything).Once()
				req := larkdrive.NewGetExportTaskReqBuilder().Ticket(ticket).Token(di.Token).Build()
				s.mockClient.EXPECT().ExportGet(mock.Anything, req).Return(
					&larkdrive.GetExportTaskResp{
						Data: &larkdrive.GetExportTaskRespData{
							Result: &larkdrive.ExportTask{
								JobStatus:   larkcore.IntPtr(3),
								JobErrorMsg: larkcore.StringPtr("错误信息"),
							},
						},
					},
					nil,
				).Once()
			},
			expectedResult: nil,
			expectedStat:   progress.StatusFailed,
			expectedError:  oops.New("错误信息"),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// 设置mock
			tt.setupMock(tt.di, tt.ticket)

			// 执行测试
			result, status, err := s.task.checkExport(tt.di, tt.ticket)
			defer s.task.completed.Store(false)

			// 验证结果
			s.Equal(tt.expectedStat, status)
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
			if result != nil {
				s.Equal(tt.expectedResult, result, tt.name)
			}
		})
	}
}

// TestDoDownloadExported 测试导出文件下载。
func (s *exporterTestSuite) Test_exporter_doDownloadExported() {
	tests := []struct {
		name          string
		filePath      string
		fileToken     string
		setupMock     func(filePath, fileToken string)
		expectedFile  io.Reader
		expectedError error
	}{
		{
			name:      "成功下载",
			filePath:  "file_path",
			fileToken: "file_token",
			setupMock: func(filePath, fileToken string) {
				s.mockProgram.EXPECT().Update(filePath, 0.18, progress.StatusDownloading, "请求%d次", mock.Anything).Once()
				req := larkdrive.NewDownloadExportTaskReqBuilder().FileToken(fileToken).Build()
				s.mockClient.EXPECT().ExportDownload(mock.Anything, req).Return(
					&larkdrive.DownloadExportTaskResp{
						File: io.NopCloser(bytes.NewBufferString("exported data")),
					},
					nil,
				).Once()
			},
			expectedFile: bytes.NewBufferString("exported data"),
		},
		{
			name:      "请求报错，下载失败",
			filePath:  "file_path",
			fileToken: "file_token",
			setupMock: func(filePath, fileToken string) {
				s.mockProgram.EXPECT().Update(filePath, 0.18, progress.StatusDownloading, "请求%d次", mock.Anything).Once()
				req := larkdrive.NewDownloadExportTaskReqBuilder().FileToken(fileToken).Build()
				s.mockClient.EXPECT().ExportDownload(mock.Anything, req).Return(
					nil,
					errors.New("下载失败"),
				).Once()
			},
			expectedError: errors.New("下载失败"),
		},
		{
			name:      "请求成功，服务端返回失败",
			filePath:  "file_path",
			fileToken: "file_token",
			setupMock: func(filePath, fileToken string) {
				s.mockProgram.EXPECT().Update(filePath, 0.18, progress.StatusDownloading, "请求%d次", mock.Anything).Once()
				req := larkdrive.NewDownloadExportTaskReqBuilder().FileToken(fileToken).Build()
				resp := &larkdrive.DownloadExportTaskResp{
					ApiResp: &larkcore.ApiResp{
						Header: http.Header{
							larkcore.HttpHeaderKeyLogId: []string{"1111111111"},
						},
					},
					CodeError: larkcore.CodeError{Code: 111},
					File:      nil,
					FileName:  "",
				}
				s.mockClient.EXPECT().ExportDownload(mock.Anything, req).Return(checkResp(resp, nil)).Once()
			},
			expectedError: errors.New("logId: \x1b]8;;https://open.feishu.cn/search?q=1111111111\x1b\\, 操作: 下载导出文件, 响应错误: msg:,code:111"),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock(tt.filePath, tt.fileToken)

			file, err := s.task.doDownloadExported("file_path", "file_token")

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
				expectedBytes, _ := io.ReadAll(tt.expectedFile)
				actualBytes, _ := io.ReadAll(file)
				s.Equal(expectedBytes, actualBytes, tt.name)
			}
		})
	}
}

func (s *exporterTestSuite) Test_exporter_doDownloadDirectly() {
	tests := []struct {
		name           string
		filePath       string
		fileToken      string
		setupMock      func(filePath, fileToken string)
		expectedFile   io.Reader
		expectedLength int64
		expectedError  error
	}{
		{
			name:      "成功下载",
			filePath:  "file_path",
			fileToken: "file_token",
			setupMock: func(filePath, fileToken string) {
				s.mockProgram.EXPECT().Update(filePath, 0.18, progress.StatusDownloading, "请求%d次", mock.Anything).Return().Once()
				req := larkdrive.NewDownloadFileReqBuilder().FileToken(fileToken).Build()
				resp := &larkdrive.DownloadFileResp{
					ApiResp: &larkcore.ApiResp{
						Header: http.Header{"Content-Length": []string{"123"}},
					},
					File: io.NopCloser(bytes.NewBufferString("test content")),
				}
				s.mockClient.EXPECT().DriveDownload(mock.Anything, req).Return(resp, nil).Once()
			},
			expectedFile:   bytes.NewBufferString("test content"),
			expectedLength: 123,
			expectedError:  nil,
		},
		{
			name:      "客户端调用失败",
			filePath:  "file_path",
			fileToken: "file_token",
			setupMock: func(filePath, fileToken string) {
				s.mockProgram.EXPECT().Update(filePath, 0.18, progress.StatusDownloading, "请求%d次", mock.Anything).Return().Once()
				req := larkdrive.NewDownloadFileReqBuilder().FileToken(fileToken).Build()
				s.mockClient.EXPECT().DriveDownload(mock.Anything, req).Return(nil, errors.New("API调用失败")).Once()
			},
			expectedError: errors.New("API调用失败"),
		},
		{
			name:      "API响应不成功",
			filePath:  "file_path",
			fileToken: "file_token",
			setupMock: func(filePath, fileToken string) {
				s.mockProgram.EXPECT().Update(filePath, 0.18, progress.StatusDownloading, "请求%d次", mock.Anything).Return().Once()
				req := larkdrive.NewDownloadFileReqBuilder().FileToken(fileToken).Build()
				resp := &larkdrive.DownloadFileResp{
					ApiResp: &larkcore.ApiResp{
						Header: http.Header{
							"Content-Length":            []string{"456"},
							larkcore.HttpHeaderKeyLogId: []string{"1111111111"},
						},
						StatusCode: 404,
					},
					CodeError: larkcore.CodeError{Code: 112233},
				}
				s.mockClient.EXPECT().DriveDownload(mock.Anything, req).Return(checkResp(resp, nil)).Once()
			},
			expectedError: errors.New("logId: \x1b]8;;https://open.feishu.cn/search?q=1111111111\x1b\\, 操作: 下载导出文件, 响应错误: msg:,code:112233"),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// 设置模拟行为
			tt.setupMock(tt.filePath, tt.fileToken)

			// 执行测试
			file, length, err := s.task.doDownloadDirectly(tt.filePath, tt.fileToken)

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

			// 比较文件内容
			if tt.expectedFile != nil {
				expectedBytes, _ := io.ReadAll(tt.expectedFile)
				actualBytes, _ := io.ReadAll(file)
				s.Equal(expectedBytes, actualBytes, tt.name)
			}

			s.Equal(tt.expectedLength, length, tt.name)
		})
	}
}
