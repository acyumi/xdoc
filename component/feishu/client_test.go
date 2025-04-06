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
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/h2non/gock"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkdrive "github.com/larksuite/oapi-sdk-go/v3/service/drive/v1"
	larkwiki "github.com/larksuite/oapi-sdk-go/v3/service/wiki/v2"
	"github.com/samber/oops"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/suite"

	"github.com/acyumi/xdoc/component/app"
	"github.com/acyumi/xdoc/component/argument"
	"github.com/acyumi/xdoc/component/cloud"
	"github.com/acyumi/xdoc/component/progress"
)

func TestClientImplSuite(t *testing.T) {
	suite.Run(t, new(ClientImplTestSuite))
}

type ClientImplTestSuite struct {
	suite.Suite
	client              *ClientImpl
	args                *Args
	originMarshalIndent func(v any, prefix, indent string) ([]byte, error)
	memFs               *afero.Afero
	mockTask            *MockTask
}

func (s *ClientImplTestSuite) SetupSuite() {
	s.originMarshalIndent = app.MarshalIndent
	useMemMapFs()
	s.memFs = app.Fs
}

func (s *ClientImplTestSuite) SetupTest() {
	s.client = NewClient(&Args{
		AppID:     "cli_xxx",
		AppSecret: "xxx",
		Args: &argument.Args{
			StartTime: time.Now(),
		},
	}).(*ClientImpl)
	s.args = s.client.GetArgs()
	s.mockTask = NewMockTask(s.T())
	s.client.TaskCreator = func(args *Args, docs []*DocumentNode) cloud.Task {
		return s.mockTask
	}
}

func (s *ClientImplTestSuite) TearDownTest() {
	// TODO 开发自动在跑单测过程中生成gock的mock代码的功能，这样后续mock代码不对时只需要跑一下单测就能打印出来直接替换
	defer gock.Off()
}

func (s *ClientImplTestSuite) TearDownSuite() {
}

func (s *ClientImplTestSuite) TestClientImpl_Validate() {
	tests := []struct {
		name      string
		client    *ClientImpl
		wantError string
	}{
		{
			name: "正常",
			client: func() *ClientImpl {
				var c ClientImpl
				c.SetArgs(&Args{
					AppID:     "cli_xxx",
					AppSecret: "xxx",
					DocURLs:   []string{"x"},
					SaveDir:   "/tmp",
				})
				return &c
			}(),
			wantError: "",
		},
		{
			name:      "Client和Args为空",
			client:    &ClientImpl{},
			wantError: "Args: cannot be blank; Client: cannot be blank.",
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := tt.client.Validate()
			if err != nil || tt.wantError != "" {
				s.Require().EqualError(err, tt.wantError, tt.name)
			}
		})
	}
}

type mockFs struct {
	afero.Fs
	openFileHasErr bool
}

func (f *mockFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	if f.openFileHasErr {
		return nil, oops.New("创建文件失败")
	}
	return f.Fs.OpenFile(name, flag, perm)
}

func (s *ClientImplTestSuite) TestClientImpl_DownloadDocuments() {
	checkAuthenticated()
	tests := []struct {
		name         string
		typ          string
		token        string
		setupMock    func(mt *MockTask, name string)
		teardownMock func(mt *MockTask, name string)
		wantCode     string
		wantError    string
	}{
		{
			name:  "只列出文件树",
			typ:   "/wiki/settings",
			token: "6946843325487912366",
			setupMock: func(mt *MockTask, name string) {
				s.mockWikiSettingsServer("6946843325487912366")
				s.args.ListOnly = true
			},
			teardownMock: func(mt *MockTask, name string) {
				s.args.SaveDir = ""
				s.args.ListOnly = false
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
			wantError: "",
		},
		{
			name:  "列出文件树后创建任务执行",
			typ:   "/wiki/settings",
			token: "6946843325487912366",
			setupMock: func(mt *MockTask, name string) {
				s.mockWikiSettingsServer("6946843325487912366")
				s.args.ListOnly = false
				s.mockTask.EXPECT().Validate().Return(nil).Once()
				s.mockTask.EXPECT().Run().Return(nil).Once()
				s.mockTask.EXPECT().Close().Return().Once()
			},
			teardownMock: func(mt *MockTask, name string) {
				s.args.SaveDir = ""
				s.args.ListOnly = false
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
			wantError: "",
		},
		{
			name:  "创建目录失败",
			typ:   "/wiki/settings",
			token: "6946843325487912366",
			setupMock: func(mt *MockTask, name string) {
				s.mockWikiSettingsServer("6946843325487912366")
				s.args.ListOnly = true
				useFs(&afero.Afero{Fs: afero.NewReadOnlyFs(app.Fs)})
			},
			teardownMock: func(mt *MockTask, name string) {
				s.args.SaveDir = ""
				s.args.ListOnly = false
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
			wantError: "operation not permitted",
		},
		{
			name:  "MarshalIndent失败",
			typ:   "/wiki/settings",
			token: "6946843325487912366",
			setupMock: func(mt *MockTask, name string) {
				s.mockWikiSettingsServer("6946843325487912366")
				s.args.ListOnly = true
				app.MarshalIndent = func(v any, prefix, indent string) ([]byte, error) {
					return nil, errors.New("MarshalIndent失败")
				}
			},
			teardownMock: func(mt *MockTask, name string) {
				s.args.SaveDir = ""
				s.args.ListOnly = false
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
			wantError: "MarshalIndent失败",
		},
		{
			name:  "写入文件失败",
			typ:   "/wiki/settings",
			token: "6946843325487912345",
			setupMock: func(mt *MockTask, name string) {
				s.mockWikiSettingsServer("6946843325487912345")
				s.args.ListOnly = true
				useFs(&afero.Afero{Fs: &mockFs{Fs: afero.NewMemMapFs(), openFileHasErr: true}})
			},
			teardownMock: func(mt *MockTask, name string) {
				s.args.SaveDir = ""
				s.args.ListOnly = false
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
			wantError: "写入文件失败: 创建文件失败",
		},
		{
			name:  "/wiki",
			typ:   "/wiki",
			token: "token233",
			setupMock: func(mt *MockTask, name string) {
				gock.New("https://open.feishu.cn").
					Get("/open-apis/wiki/v2/spaces/get_node").
					MatchParam("obj_type", "wiki").
					MatchParam("token", "token233").
					Reply(500).
					AddHeader(larkcore.HttpHeaderKeyLogId, "xyz").
					JSON(`{"code": 500,"msg": "something wrong"}`)
			},
			teardownMock: func(mt *MockTask, name string) {
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
			wantError: "logId: \x1b]8;;https://open.feishu.cn/search?q=xyz\x1b\\, error response: \n{\n  Code: 500,\n  Msg: \"something wrong\"\n}",
		},
		{
			name:  "/wiki/settings",
			typ:   "/wiki/settings",
			token: "token233",
			setupMock: func(mt *MockTask, name string) {
				gock.New("https://open.feishu.cn").
					Get("/open-apis/wiki/v2/spaces/token233").
					PathParam("spaces", "token233").
					Reply(500).
					AddHeader(larkcore.HttpHeaderKeyLogId, "xyz").
					JSON(`{"code": 500,"msg": "something wrong"}`)
			},
			teardownMock: func(mt *MockTask, name string) {
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
			wantError: "logId: \x1b]8;;https://open.feishu.cn/search?q=xyz\x1b\\, error response: \n{\n  Code: 500,\n  Msg: \"something wrong\"\n}",
		},
		{
			name:  "/drive/folder",
			typ:   "/drive/folder",
			token: "token",
			setupMock: func(mt *MockTask, name string) {
				gock.New("https://open.feishu.cn").
					Post("/open-apis/drive/v1/metas/batch_query").
					// 指定请求的Content-Type(主要是多了utf-8部分)，否则会报错 gock.MockMatcher 匹配不上
					MatchType("application/json; charset=utf-8").
					JSON(`{
  "request_docs" : [ {
    "doc_token" : "token",
    "doc_type" : "folder"
  } ],
  "with_url" : true
}`).
					Reply(500).
					AddHeader(larkcore.HttpHeaderKeyLogId, "xyz").
					JSON(`{"code": 500,"msg": "something wrong"}`)
			},
			teardownMock: func(mt *MockTask, name string) {
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
			wantError: "logId: \x1b]8;;https://open.feishu.cn/search?q=xyz\x1b\\, error response: \n{\n  Code: 500,\n  Msg: \"something wrong\"\n}",
		},
		{
			name:  "/docs",
			typ:   "/docs",
			token: "token",
			setupMock: func(mt *MockTask, name string) {
				gock.New("https://open.feishu.cn").
					Post("/open-apis/drive/v1/metas/batch_query").
					// 指定请求的Content-Type(主要是多了utf-8部分)，否则会报错 gock.MockMatcher 匹配不上
					MatchType("application/json; charset=utf-8").
					JSON(`{
  "request_docs" : [ {
    "doc_token" : "token",
    "doc_type" : "doc"
  } ],
  "with_url" : true
}`).
					Reply(500).
					AddHeader(larkcore.HttpHeaderKeyLogId, "xyz").
					JSON(`{"code": 500,"msg": "something wrong"}`)
			},
			teardownMock: func(mt *MockTask, name string) {
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
			wantError: "logId: \x1b]8;;https://open.feishu.cn/search?q=xyz\x1b\\, error response: \n{\n  Code: 500,\n  Msg: \"something wrong\"\n}",
		},
		{
			name:  "/docx",
			typ:   "/docx",
			token: "token",
			setupMock: func(mt *MockTask, name string) {
				gock.New("https://open.feishu.cn").
					Post("/open-apis/drive/v1/metas/batch_query").
					// 指定请求的Content-Type(主要是多了utf-8部分)，否则会报错 gock.MockMatcher 匹配不上
					MatchType("application/json; charset=utf-8").
					JSON(`{
  "request_docs" : [ {
    "doc_token" : "token",
    "doc_type" : "docx"
  } ],
  "with_url" : true
}`).
					Reply(500).
					AddHeader(larkcore.HttpHeaderKeyLogId, "xyz").
					JSON(`{"code": 500,"msg": "something wrong"}`)
			},
			teardownMock: func(mt *MockTask, name string) {
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
			wantError: "logId: \x1b]8;;https://open.feishu.cn/search?q=xyz\x1b\\, error response: \n{\n  Code: 500,\n  Msg: \"something wrong\"\n}",
		},
		{
			name:  "/sheets",
			typ:   "/sheets",
			token: "token",
			setupMock: func(mt *MockTask, name string) {
				gock.New("https://open.feishu.cn").
					Post("/open-apis/drive/v1/metas/batch_query").
					// 指定请求的Content-Type(主要是多了utf-8部分)，否则会报错 gock.MockMatcher 匹配不上
					MatchType("application/json; charset=utf-8").
					JSON(`{
  "request_docs" : [ {
    "doc_token" : "token",
    "doc_type" : "sheet"
  } ],
  "with_url" : true
}`).
					Reply(500).
					AddHeader(larkcore.HttpHeaderKeyLogId, "xyz").
					JSON(`{"code": 500,"msg": "something wrong"}`)
			},
			teardownMock: func(mt *MockTask, name string) {
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
			wantError: "logId: \x1b]8;;https://open.feishu.cn/search?q=xyz\x1b\\, error response: \n{\n  Code: 500,\n  Msg: \"something wrong\"\n}",
		},
		{
			name:  "/file",
			typ:   "/file",
			token: "token",
			setupMock: func(mt *MockTask, name string) {
				gock.New("https://open.feishu.cn").
					Post("/open-apis/drive/v1/metas/batch_query").
					// 指定请求的Content-Type(主要是多了utf-8部分)，否则会报错 gock.MockMatcher 匹配不上
					MatchType("application/json; charset=utf-8").
					JSON(`{
  "request_docs" : [ {
    "doc_token" : "token",
    "doc_type" : "file"
  } ],
  "with_url" : true
}`).
					Reply(500).
					AddHeader(larkcore.HttpHeaderKeyLogId, "xyz").
					JSON(`{"code": 500,"msg": "something wrong"}`)
			},
			teardownMock: func(mt *MockTask, name string) {
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
			wantError: "logId: \x1b]8;;https://open.feishu.cn/search?q=xyz\x1b\\, error response: \n{\n  Code: 500,\n  Msg: \"something wrong\"\n}",
		},
		{
			name:  "不支持的类型",
			typ:   "/xxx",
			token: "token",
			setupMock: func(mt *MockTask, name string) {
			},
			teardownMock: func(mt *MockTask, name string) {
			},
			wantCode:  "InvalidArgument",
			wantError: "不支持的飞书云文档类型: /xxx\n",
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer func() {
				app.MarshalIndent = s.originMarshalIndent
				useFs(s.memFs)
			}()
			tt.setupMock(s.mockTask, tt.name)
			err := s.client.DownloadDocuments([]*cloud.DocumentSource{{Type: tt.typ, Token: tt.token}})
			if err != nil || tt.wantError != "" {
				s.Require().Error(err, tt.name)
				s.IsType(oops.OopsError{}, err, tt.name)
				var actualError oops.OopsError
				yes := errors.As(err, &actualError)
				s.Require().True(yes, tt.name)
				s.Equal(tt.wantCode, actualError.Code(), tt.name)
				s.Equal(tt.wantError, actualError.Error(), tt.name)
			}
		})
	}
}

func (s *ClientImplTestSuite) mockWikiSettingsServer(spaceID string) {
	s.args.SaveDir = "/tmp"
	gock.New("https://open.feishu.cn").
		Get("/open-apis/wiki/v2/spaces/"+spaceID).
		PathParam("spaces", spaceID).
		Reply(200).
		AddHeader(larkcore.HttpHeaderKeyLogId, "xyz").
		JSON(`{
    "code": 0,
    "msg": "success",
    "data": {
        "space": {
            "name": "知识空间",
            "description": "知识空间描述",
            "space_id": "6946843325487912366"
        }
    }
}`)
	// 模拟 WikiNodeList 的 API 响应
	gock.New("https://open.feishu.cn").
		Get(fmt.Sprintf("/open-apis/wiki/v2/spaces/%s/nodes", spaceID)).
		PathParam("spaces", spaceID).
		MatchParams(map[string]string{
			"parent_node_token": "",
			"page_token":        "",
			"page_size":         "50",
		}).
		Reply(200).
		JSON(fmt.Sprintf(`{
    "code": 0,
    "msg": "success",
    "data": {
        "items": [
            {
                "space_id": "%s",
                "node_token": "wikcnKQ1k3pxxxxxx8Vabceg",
                "obj_token": "doccnzAaODxxxxxxWabcdeg",
                "obj_type": "doc",
                "parent_node_token": "wikcnKQ1k3pxxxxxx8Vabceg",
                "node_type": "origin",
                "origin_node_token": "wikcnKQ1k3pxxxxxx8Vabceg",
                "origin_space_id": "6946843325487912356",
                "has_child": false,
                "title": "标题",
                "obj_create_time": "1642402428",
                "obj_edit_time": "1642402428",
                "node_create_time": "1642402428",
                "creator": "ou_xxxxx",
                "owner": "ou_xxxxx",
                "node_creator": "ou_xxxxx"
            }
        ],
        "page_token": "",
        "has_more": false
    }
}`, spaceID))
}

func (s *ClientImplTestSuite) TestClientImpl_CreateTask() {
	origin := s.client.TaskCreator
	defer func() {
		s.client.TaskCreator = origin
	}()

	s.mockTask.On("Validate").Return(errors.New("test error"))
	task := s.client.CreateTask(nil, nil)
	err := task.Validate()
	s.Require().Error(err)
	s.Require().EqualError(err, "test error")
	s.mockTask.AssertCalled(s.T(), "Validate")

	s.client.TaskCreator = nil
	task = s.client.CreateTask(nil, nil)
	err = task.Validate()
	s.Require().Error(err)
	s.Require().EqualError(err, "Client: Args: DocURLs: urls是必需参数; SaveDir: dir是必需参数..; Docs: cannot be blank; ProgramConstructor: cannot be blank.")
}

type MockSuccess struct {
	success bool
	error   string
}

func (s *MockSuccess) Error() string {
	return s.error
}

func (s *MockSuccess) Success() bool {
	return s.success
}

// TestGetLogID 测试日志ID提取。
func (s *ClientImplTestSuite) Test_checkResp() {
	tests := []struct {
		name      string
		resp      error // 这个是飞书客户端的Response，也是error的实现
		err       error
		wantResp  error
		wantError string
	}{
		{
			name:      "err不为空",
			resp:      nil,
			err:       errors.New("发生错误"),
			wantResp:  nil,
			wantError: "发生错误",
		},
		{
			name: "resp服务错误",
			resp: &larkdrive.BatchQueryMetaResp{
				ApiResp: &larkcore.ApiResp{
					StatusCode: 500,
					Header: http.Header{
						larkcore.HttpHeaderKeyLogId: []string{"xyz"},
					},
					RawBody: []byte(`{code:1061002,msg:"发生错误"}`),
				},
				CodeError: larkcore.CodeError{
					Code: 1061002,
					Msg:  "发生错误",
				},
			},
			err: nil,
			wantResp: &larkdrive.BatchQueryMetaResp{
				ApiResp: &larkcore.ApiResp{
					StatusCode: 500,
					Header: http.Header{
						larkcore.HttpHeaderKeyLogId: []string{"xyz"},
					},
					RawBody: []byte(`{code:1061002,msg:"发生错误"}`),
				},
				CodeError: larkcore.CodeError{
					Code: 1061002,
					Msg:  "发生错误",
				},
			},
			wantError: "logId: \x1b]8;;https://open.feishu.cn/search?q=xyz\x1b\\, error response: \n{\n  Code: 1061002,\n  Msg: \"发生错误\"\n}",
		},
		{
			name:      "非飞书的响应",
			resp:      viper.ConfigFileNotFoundError{},
			err:       nil,
			wantResp:  viper.ConfigFileNotFoundError{},
			wantError: "",
		},
		{
			name:      "MockSuccess",
			resp:      &MockSuccess{success: false},
			err:       nil,
			wantResp:  &MockSuccess{success: false},
			wantError: "",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			resp, err := checkResp(tt.resp, tt.err)
			s.Equal(tt.wantResp, resp, tt.name)
			if err == nil && tt.wantError == "" {
				return
			}
			s.Require().EqualError(err, tt.wantError, tt.name)
		})
	}
}

// TestGetLogID 测试日志ID提取。
func (s *ClientImplTestSuite) Test_getLogID() {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "支持RequestId接口",
			err:      &mockError{logID: "456"},
			expected: progress.URLStyleRender("https://open.feishu.cn/search?q=456"),
		},
		{
			name:     "不支持RequestId接口",
			err:      errors.New("no log"),
			expected: "",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			logID := getLogID(tt.err)
			s.Equal(tt.expected, logID, tt.name)
		})
	}
}

func (s *ClientImplTestSuite) TestClientImpl_DriveBatchQuery() {
	checkAuthenticated()
	gock.New("https://open.feishu.cn").
		Post("/open-apis/drive/v1/metas/batch_query").
		// 指定请求的Content-Type(主要是多了utf-8部分)，否则会报错 gock.MockMatcher 匹配不上
		MatchType("application/json; charset=utf-8").
		JSON(`{
  "request_docs" : [ {
    "doc_token" : "token",
    "doc_type" : "type"
  } ],
  "with_url" : true
}`).
		Reply(200).
		JSON(`{
  "code": 0,
  "data": {
    "metas": [
      {
        "create_time": "1740147877",
        "doc_token": "zzz",
        "doc_type": "folder",
        "latest_modify_time": "1740301528",
        "latest_modify_user": "ou_xxx",
        "owner_id": "ou_xxx",
        "title": "xxx",
        "url": "https://sample.feishu.cn/drive/folder/xxxxx"
      }
    ]
  },
  "msg": "Success"
}`)
	req := larkdrive.NewBatchQueryMetaReqBuilder().
		MetaRequest(larkdrive.NewMetaRequestBuilder().
			RequestDocs([]*larkdrive.RequestDoc{
				larkdrive.NewRequestDocBuilder().
					DocToken("token").
					DocType("type").
					Build(),
			}).
			WithUrl(true).
			Build()).
		Build()
	resp, err := s.client.DriveBatchQuery(context.Background(), req)
	s.Require().NoError(err)
	s.NotNil(resp)
	s.Equal("zzz", larkcore.StringValue(resp.Data.Metas[0].DocToken))
	s.True(gock.IsDone())
}

func (s *ClientImplTestSuite) TestClientImpl_DriveList() {
	checkAuthenticated()
	// 模拟 DriveList 的 API 响应
	gock.New("https://open.feishu.cn").
		Get("/open-apis/drive/v1/files").
		MatchParams(map[string]string{
			"folder_token": "folderToken",
			"page_token":   "pageToken",
			"order_by":     "EditedTime",
			"direction":    "DESC",
		}).
		Reply(200).
		JSON(`{
    "code":0,
    "data":{
        "files":[
            {
                "name":"test pdf.pdf",
                "parent_token":"fldbcO1UuPz8VwnpPx5a9abcef",
                "token":"boxbc0dGSMu23m7QkC1bvabcef",
                "type":"file",
                "created_time":"1679277808",
                "modified_time":"1679277808",
                "owner_id":"ou_20b31734443364ec8a1df89fdf325b44",
                "url":"https://feishu.cn/file/boxbc0dGSMu23m7QkC1bvabcef"
            }
        ],
        "has_more":false
    },
    "msg":"success"
}`)
	// 构造请求参数
	req := larkdrive.NewListFileReqBuilder().
		FolderToken("folderToken").
		PageToken("pageToken").
		OrderBy(`EditedTime`).
		Direction(`DESC`).
		PageSize(200).
		Build()
	// 执行 DriveList 方法
	resp, err := s.client.DriveList(context.Background(), req)

	// 断言验证
	s.Require().NoError(err)
	s.NotNil(resp)
	s.Len(resp.Data.Files, 1)
	s.Equal("boxbc0dGSMu23m7QkC1bvabcef", larkcore.StringValue(resp.Data.Files[0].Token))

	// 测试错误响应
	gock.New("https://open.feishu.cn").
		Get("/open-apis/drive/v1/files").
		MatchParams(map[string]string{
			"folder_token": "233333333",
			"page_token":   "",
			"order_by":     "EditedTime",
			"direction":    "DESC",
		}).
		Reply(400).
		AddHeader(larkcore.HttpHeaderKeyLogId, "202503132148180A516BE43C33D1078308").
		JSON(`{
  "code": 1061002,
  "msg": "params error.",
  "error": {
    "log_id": "202503132148180A516BE43C33D1078308",
    "troubleshooter": "排查建议查看(Troubleshooting suggestions): https://open.feishu.cn/search?log_id=202503132148180A516BE43C33D1078308"
  }
}`)
	// 构造请求参数
	req = larkdrive.NewListFileReqBuilder().
		FolderToken("233333333").
		PageToken("").
		OrderBy(`EditedTime`).
		Direction(`DESC`).
		PageSize(200).
		Build()
	// 执行 DriveList 方法
	resp, err = s.client.DriveList(context.Background(), req)

	// 断言验证
	s.Require().Error(err)
	s.IsType(oops.OopsError{}, err)
	var actualError oops.OopsError
	yes := errors.As(err, &actualError)
	s.Require().True(yes)
	s.Equal("", actualError.Code())
	s.Equal("logId: \x1b]8;;https://open.feishu.cn/search?q=202503132148180A516BE43C33D1078308\x1b\\, "+
		"error response: \n{\n  Code: 1061002,\n  Msg: \"params error.\",\n  "+
		"Err: {\n    LogID: \"202503132148180A516BE43C33D1078308\",\n    "+
		"Troubleshooter: \"排查建议查看(Troubleshooting suggestions): "+
		"https://open.feishu.cn/search?log_id=202503132148180A516BE43C33D1078308\"\n  }\n}", actualError.Error())
	s.NotNil(resp)
	s.Equal(1061002, resp.Code)
	s.Equal("params error.", resp.Msg)
	s.Nil(resp.Data)
	s.True(gock.IsDone())
}

// TestClientImpl_DriveDownload 测试文件下载功能。
func (s *ClientImplTestSuite) TestClientImpl_DriveDownload() {
	checkAuthenticated()
	gock.New("https://open.feishu.cn").
		Get("/open-apis/drive/v1/files/fileToken123/download").
		PathParam("files", "fileToken123").
		Reply(200).
		AddHeader("Content-Disposition", `attachment; filename="xxxxxx.docx"`).
		BodyString(`FileContentXxx`)

	req := larkdrive.NewDownloadFileReqBuilder().FileToken("fileToken123").Build()
	resp, err := s.client.DriveDownload(context.Background(), req)
	s.Require().NoError(err)
	s.Equal("xxxxxx.docx", resp.FileName)
	all, err := io.ReadAll(resp.File)
	s.Require().NoError(err)
	s.Equal(`FileContentXxx`, string(all))
	s.True(gock.IsDone())
}

// TestClientImpl_WikiGetNode 测试获取知识库节点。
func (s *ClientImplTestSuite) TestClientImpl_WikiGetNode() {
	checkAuthenticated()
	gock.New("https://open.feishu.cn").
		Get("/open-apis/wiki/v2/spaces/get_node").
		MatchParam("obj_type", "wiki").
		MatchParam("token", "token1").
		Reply(200).
		JSON(`{
    "code": 0,
    "msg": "success",
    "data": {
        "node": {
            "space_id": "6946843325487912356",
            "node_token": "wikcnKQ1k3p******8Vabcef",
            "obj_token": "doccnzAaOD******Wabcdef",
            "obj_type": "doc",
            "parent_node_token": "wikcnKQ1k3p******8Vabcef",
            "node_type": "origin",
            "origin_node_token": "wikcnKQ1k3p******8Vabcef",
            "origin_space_id": "6946843325487912356",
            "has_child": false,
            "title": "标题",
            "obj_create_time": "1642402428",
            "obj_edit_time": "1642402428",
            "node_create_time": "1642402428",
            "creator": "ou_xxxxx",
            "owner": "ou_xxxxx",
            "node_creator": "ou_xxxxx"
        }
    }
}`)
	req := larkwiki.NewGetNodeSpaceReqBuilder().Token("token1").ObjType(`wiki`).Build()
	resp, err := s.client.WikiGetNode(context.Background(), req)
	s.Require().NoError(err)
	s.Equal("wikcnKQ1k3p******8Vabcef", larkcore.StringValue(resp.Data.Node.NodeToken))
	s.Equal("doccnzAaOD******Wabcdef", larkcore.StringValue(resp.Data.Node.ObjToken))
	s.True(gock.IsDone())
}

// TestClientImpl_WikiGetSpace 测试获取知识库信息。
func (s *ClientImplTestSuite) TestClientImpl_WikiGetSpace() {
	checkAuthenticated()
	gock.New("https://open.feishu.cn").
		Get("/open-apis/wiki/v2/spaces/space_123").
		PathParam("spaces", "space_123").
		Reply(200).
		JSON(`{
    "code": 0,
    "msg": "success",
    "data": {
        "space": {
            "name": "知识空间",
            "description": "知识空间描述",
            "space_id": "1565676577122621"
        }
    }
}`)
	req := larkwiki.NewGetSpaceReqBuilder().
		SpaceId("space_123").
		Build()
	resp, err := s.client.WikiGetSpace(context.Background(), req)
	s.Require().NoError(err)
	s.Equal("1565676577122621", larkcore.StringValue(resp.Data.Space.SpaceId))
	s.Equal("知识空间", larkcore.StringValue(resp.Data.Space.Name))
	s.True(gock.IsDone())
}

// TestClientImpl_WikiNodeList 测试获取知识库节点列表。
func (s *ClientImplTestSuite) TestClientImpl_WikiNodeList() {
	checkAuthenticated()
	gock.New("https://open.feishu.cn").
		Get("/open-apis/wiki/v2/spaces/space_123/nodes").
		PathParam("spaces", "space_123").
		MatchParam("page_size", "10").
		Reply(200).
		JSON(`{
    "code": 0,
    "msg": "success",
    "data": {
        "items": [
            {
                "space_id": "6946843325487912356",
                "node_token": "wikcnKQ1k3p******8Vabcef",
                "obj_token": "doccnzAaOD******Wabcdef",
                "obj_type": "doc",
                "parent_node_token": "wikcnKQ1k3p******8Vabcef",
                "node_type": "origin",
                "origin_node_token": "wikcnKQ1k3p******8Vabcef",
                "origin_space_id": "6946843325487912356",
                "has_child": false,
                "title": "标题",
                "obj_create_time": "1642402428",
                "obj_edit_time": "1642402428",
                "node_create_time": "1642402428",
                "creator": "ou_xxxxx",
                "owner": "ou_xxxxx",
                "node_creator": "ou_xxxxx"
            }
        ],
        "page_token": "6946843325487906839",
        "has_more": true
    }
}`)
	req := larkwiki.NewListSpaceNodeReqBuilder().
		SpaceId("space_123").
		PageSize(10).
		Build()
	resp, err := s.client.WikiNodeList(context.Background(), req)
	s.Require().NoError(err)
	s.Len(resp.Data.Items, 1)
	s.Equal("doccnzAaOD******Wabcdef", larkcore.StringValue(resp.Data.Items[0].ObjToken))
	s.True(larkcore.BoolValue(resp.Data.HasMore))
	s.True(gock.IsDone())
}

// TestClientImpl_ExportCreate 测试导出任务创建。
func (s *ClientImplTestSuite) TestClientImpl_ExportCreate() {
	checkAuthenticated()
	gock.New("https://open.feishu.cn").
		Post("/open-apis/drive/v1/export_tasks").
		Reply(200).
		JSON(`{
    "code": 0,
    "msg": "success",
    "data": {
        "ticket": "6933093124755423251"
    }
}`)
	exportTask := larkdrive.NewExportTaskBuilder().
		FileExtension("docx").
		Token("token").
		Type("type").
		Build()
	req := larkdrive.NewCreateExportTaskReqBuilder().
		ExportTask(exportTask).
		Build()
	req.ExportTask = exportTask
	resp, err := s.client.ExportCreate(context.Background(), req)
	s.Require().NoError(err)
	s.Equal("6933093124755423251", larkcore.StringValue(resp.Data.Ticket))
	s.True(gock.IsDone())
}

// TestClientImpl_ExportGet 测试获取导出任务状态。
func (s *ClientImplTestSuite) TestClientImpl_ExportGet() {
	checkAuthenticated()
	gock.New("https://open.feishu.cn").
		Get("/open-apis/drive/v1/export_tasks/ticket").
		PathParam("export_tasks", "ticket").
		MatchParam("token", "token").
		Reply(200).
		JSON(`{
    "code": 0,
    "msg": "success",
    "data": {
        "result": {
            "file_extension": "pdf",
            "type": "doc",
            "file_name": "task_456",
            "file_token": "boxcnxe5OdjlAkNgSNdsJvabcef",
            "file_size": 34356,
            "job_error_msg": "success",
            "job_status": 0
        }
    }
}`)
	req := larkdrive.NewGetExportTaskReqBuilder().Ticket("ticket").Token("token").Build()
	resp, err := s.client.ExportGet(context.Background(), req)
	s.Require().NoError(err)
	s.Equal("task_456", larkcore.StringValue(resp.Data.Result.FileName))
	s.Equal(0, larkcore.IntValue(resp.Data.Result.JobStatus))
	s.Equal(34356, larkcore.IntValue(resp.Data.Result.FileSize))
	s.True(gock.IsDone())
}

// TestClientImpl_ExportDownload 测试导出文件下载。
func (s *ClientImplTestSuite) TestClientImpl_ExportDownload() {
	checkAuthenticated()
	gock.New("https://open.feishu.cn").
		Get("/open-apis/drive/v1/export_tasks/file/fileToken/download").
		PathParam("file", "fileToken").
		Reply(200).
		AddHeader("Content-Disposition", `attachment; filename="FileNameXxx.pdf"`).
		BodyString(`FileContentXyz`)

	req := larkdrive.NewDownloadExportTaskReqBuilder().FileToken("fileToken").Build()
	resp, err := s.client.ExportDownload(context.Background(), req)
	s.Require().NoError(err)
	s.Equal(`FileNameXxx.pdf`, resp.FileName)
	all, err := io.ReadAll(resp.File)
	s.Require().NoError(err)
	s.Equal(`FileContentXyz`, string(all))
	s.True(gock.IsDone())
}
