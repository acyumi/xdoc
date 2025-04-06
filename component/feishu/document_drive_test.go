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
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/h2non/gock"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkdrive "github.com/larksuite/oapi-sdk-go/v3/service/drive/v1"
	"github.com/samber/oops"
	"github.com/stretchr/testify/suite"

	"github.com/acyumi/xdoc/component/argument"
	"github.com/acyumi/xdoc/component/constant"
)

func TestDocumentDriveSuite(t *testing.T) {
	suite.Run(t, new(DocumentDriveTestSuite))
}

type DocumentDriveTestSuite struct {
	suite.Suite
	client *ClientImpl
}

func (s *DocumentDriveTestSuite) SetupSuite() {
	initBackOff = testInitBackOff
	cleanSleep()
}

func (s *DocumentDriveTestSuite) SetupTest() {
	s.client = NewClient(&Args{
		AppID:     "cli_xxx",
		AppSecret: "xxx",
		Args: &argument.Args{
			StartTime: time.Now(),
		},
	}).(*ClientImpl)
}

func (s *DocumentDriveTestSuite) TearDownTest() {
}

func (s *DocumentDriveTestSuite) TearDownSuite() {
	initBackOff = initExponentialBackOff
}

func (s *DocumentDriveTestSuite) TestQueryDriveDocuments() {
	tests := []struct {
		name         string
		typ          constant.DocType
		token        string
		setupMock    func(name string)
		teardownMock func(name string)
		wantError    string
		want         *DocumentNode
	}{
		{
			name:  "请求报错",
			typ:   constant.DocTypeFolder,
			token: "token",
			setupMock: func(name string) {
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
			teardownMock: func(name string) {
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
			wantError: "logId: \x1b]8;;https://open.feishu.cn/search?q=xyz\x1b\\, error response: \n{\n  Code: 500,\n  Msg: \"something wrong\"\n}",
		},
		{
			name:  "服务端响应有failed_list",
			typ:   constant.DocTypeFile,
			token: "fileToken",
			setupMock: func(name string) {
				gock.New("https://open.feishu.cn").
					Post("/open-apis/drive/v1/metas/batch_query").
					// 指定请求的Content-Type(主要是多了utf-8部分)，否则会报错 gock.MockMatcher 匹配不上
					MatchType("application/json; charset=utf-8").
					JSON(`{
  "request_docs" : [ {
    "doc_token" : "fileToken",
    "doc_type" : "file"
  } ],
  "with_url" : true
}`).
					Reply(200).
					AddHeader(larkcore.HttpHeaderKeyLogId, "xyz").
					JSON(`{
  "code": 0,
  "msg": "success",
  "data": {
    "metas": [],
    "failed_list": [
      {
        "token": "boxcnrHpsg1QDqXAAAyachabcef",
        "code": 970005
      }
    ]
  }
}`)
			},
			teardownMock: func(name string) {
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
			wantError: "获取文件夹元数据失败: code: 970005, token: boxcnrHpsg1QDqXAAAyachabcef",
		},
		{
			name:  "请求成功，读到文件列表",
			typ:   constant.DocTypeFolder,
			token: "folderToken",
			setupMock: func(name string) {
				gock.New("https://open.feishu.cn").
					Post("/open-apis/drive/v1/metas/batch_query").
					// 指定请求的Content-Type(主要是多了utf-8部分)，否则会报错 gock.MockMatcher 匹配不上
					MatchType("application/json; charset=utf-8").
					JSON(`{
  "request_docs" : [ {
    "doc_token" : "folderToken",
    "doc_type" : "folder"
  } ],
  "with_url" : true
}`).
					Reply(200).
					AddHeader(larkcore.HttpHeaderKeyLogId, "xyz").
					JSON(`{
  "code": 0,
  "msg": "success",
  "data": {
    "metas": [
      {
        "doc_token": "folderToken",
        "doc_type": "folder",
        "title": "sampletitle",
        "owner_id": "ou_b13d41c02edc52ce66aaae67bf1abcef",
        "create_time": "1652066345",
        "latest_modify_user": "ou_b13d41c02edc52ce66aaae67bf1abcef",
        "latest_modify_time": "1652066345",
        "url": "https://sample.feishu.cn/docs/folderToken",
        "sec_label_name": "L2-内部"
      }
    ],
    "failed_list": []
  }
}`)
				// 模拟 DriveList 的 API 响应
				gock.New("https://open.feishu.cn").
					Get("/open-apis/drive/v1/files").
					MatchParams(map[string]string{
						"folder_token": "folderToken",
						"page_token":   "",
						"order_by":     "EditedTime",
						"direction":    "DESC",
						"page_size":    "200",
					}).
					Reply(200).
					JSON(`{
    "code":0,
    "data":{
        "files":[
            {
                "name":"test docx",
                "parent_token":"folderToken",
                "token":"boxbc0dGSMu23m7QkC1bvabcef",
                "type":"docx",
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
			},
			teardownMock: func(name string) {
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
			wantError: "",
			want: &DocumentNode{
				DocumentInfo: DocumentInfo{
					Name:             "sampletitle",
					Type:             "folder",
					Token:            "folderToken",
					FileExtension:    "",
					CanDownload:      false,
					DownloadDirectly: false,
					URL:              "",
					NodeToken:        "",
					SpaceID:          "",
					FilePath:         "",
				},
				Children: []*DocumentNode{
					{
						DocumentInfo: DocumentInfo{
							Name:             "test docx",
							Type:             "docx",
							Token:            "boxbc0dGSMu23m7QkC1bvabcef",
							FileExtension:    "docx",
							CanDownload:      true,
							DownloadDirectly: false,
							URL:              "https://feishu.cn/file/boxbc0dGSMu23m7QkC1bvabcef",
							NodeToken:        "",
							SpaceID:          "",
							FilePath:         "",
						},
					},
				},
			},
		},
		{
			name:  "读到文件列表失败",
			typ:   constant.DocTypeFolder,
			token: "folderToken",
			setupMock: func(name string) {
				gock.New("https://open.feishu.cn").
					Post("/open-apis/drive/v1/metas/batch_query").
					// 指定请求的Content-Type(主要是多了utf-8部分)，否则会报错 gock.MockMatcher 匹配不上
					MatchType("application/json; charset=utf-8").
					JSON(`{
  "request_docs" : [ {
    "doc_token" : "folderToken",
    "doc_type" : "folder"
  } ],
  "with_url" : true
}`).
					Reply(200).
					AddHeader(larkcore.HttpHeaderKeyLogId, "xyz").
					JSON(`{
  "code": 0,
  "msg": "success",
  "data": {
    "metas": [
      {
        "doc_token": "folderToken",
        "doc_type": "folder",
        "title": "sampletitle",
        "owner_id": "ou_b13d41c02edc52ce66aaae67bf1abcef",
        "create_time": "1652066345",
        "latest_modify_user": "ou_b13d41c02edc52ce66aaae67bf1abcef",
        "latest_modify_time": "1652066345",
        "url": "https://sample.feishu.cn/docs/folderToken",
        "sec_label_name": "L2-内部"
      }
    ],
    "failed_list": []
  }
}`)
				// 模拟 DriveList 的 API 响应
				gock.New("https://open.feishu.cn").
					Get("/open-apis/drive/v1/files").
					MatchParams(map[string]string{
						"folder_token": "folderToken",
						"page_token":   "",
						"order_by":     "EditedTime",
						"direction":    "DESC",
						"page_size":    "200",
					}).
					Reply(400).
					AddHeader(larkcore.HttpHeaderKeyLogId, "xyz").
					JSON(`{"code":400,"msg":"Failed"}`)
			},
			teardownMock: func(name string) {
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
			wantError: "logId: \x1b]8;;https://open.feishu.cn/search?q=xyz\x1b\\, error response: \n{\n  Code: 400,\n  Msg: \"Failed\"\n}",
		},
		{
			name:  "非目录，请求成功",
			typ:   constant.DocTypeFile,
			token: "fileToken",
			setupMock: func(name string) {
				gock.New("https://open.feishu.cn").
					Post("/open-apis/drive/v1/metas/batch_query").
					// 指定请求的Content-Type(主要是多了utf-8部分)，否则会报错 gock.MockMatcher 匹配不上
					MatchType("application/json; charset=utf-8").
					JSON(`{
  "request_docs" : [ {
    "doc_token" : "fileToken",
    "doc_type" : "file"
  } ],
  "with_url" : true
}`).
					Reply(200).
					AddHeader(larkcore.HttpHeaderKeyLogId, "xyz").
					JSON(`{
  "code": 0,
  "msg": "success",
  "data": {
    "metas": [
      {
        "doc_token": "fileToken",
        "doc_type": "file",
        "title": "sampletitle",
        "owner_id": "ou_b13d41c02edc52ce66aaae67bf1abcef",
        "create_time": "1652066345",
        "latest_modify_user": "ou_b13d41c02edc52ce66aaae67bf1abcef",
        "latest_modify_time": "1652066345",
        "url": "https://sample.feishu.cn/docs/fileToken",
        "sec_label_name": "L2-内部"
      }
    ],
    "failed_list": []
  }
}`)
			},
			teardownMock: func(name string) {
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
			wantError: "",
			want: &DocumentNode{
				DocumentInfo: DocumentInfo{
					Name:             "sampletitle",
					Type:             "file",
					Token:            "fileToken",
					FileExtension:    "file",
					CanDownload:      true,
					DownloadDirectly: true,
					URL:              "",
					NodeToken:        "",
					SpaceID:          "",
					FilePath:         "",
				},
			},
		},
		{
			name:  "/docs",
			typ:   constant.DocTypeDoc,
			token: "token",
			setupMock: func(name string) {
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
			teardownMock: func(name string) {
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
			wantError: "logId: \x1b]8;;https://open.feishu.cn/search?q=xyz\x1b\\, error response: \n{\n  Code: 500,\n  Msg: \"something wrong\"\n}",
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock(tt.name)
			actual, err := s.client.QueryDriveDocuments(tt.typ, tt.token)
			tt.teardownMock(tt.name)
			if err != nil || tt.wantError != "" {
				s.Require().EqualError(err, tt.wantError, tt.name)
			}
			s.Equal(tt.want, actual, tt.name)
		})
	}
}

func (s *DocumentDriveTestSuite) Test_fetchDriveDescendant() {
	checkAuthenticated()
	tests := []struct {
		name         string
		dn           *DocumentNode
		hasChild     bool
		folderToken  string
		pageToken    string
		setupMock    func(name string, dn *DocumentNode, hasChild bool, folderToken, pageToken string)
		teardownMock func(name string)
		want         *DocumentNode
		wantError    string
	}{
		{
			name: "hasChild为false",
			dn: &DocumentNode{
				DocumentInfo: DocumentInfo{
					Token:            "FolderToken",
					Name:             "Name",
					Type:             constant.DocTypeFolder,
					URL:              "Url",
					FileExtension:    "",
					FilePath:         "",
					CanDownload:      false,
					DownloadDirectly: false,
				},
			},
			hasChild:    false,
			folderToken: "FolderToken",
			pageToken:   "PageToken",
			setupMock: func(name string, dn *DocumentNode, hasChild bool, folderToken, pageToken string) {
			},
			teardownMock: func(name string) {
			},
			want: &DocumentNode{
				DocumentInfo: DocumentInfo{
					Name:             "Name",
					Type:             constant.DocTypeFolder,
					Token:            "FolderToken",
					FileExtension:    "",
					CanDownload:      false,
					DownloadDirectly: false,
					URL:              "Url",
					NodeToken:        "",
					SpaceID:          "",
					FilePath:         "",
				},
			},
			wantError: "",
		},
		{
			name: "请求成功，没有递归",
			dn: &DocumentNode{
				DocumentInfo: DocumentInfo{
					Token:            "FolderToken",
					Name:             "Name",
					Type:             constant.DocTypeFolder,
					URL:              "Url",
					FileExtension:    "",
					FilePath:         "",
					CanDownload:      false,
					DownloadDirectly: false,
				},
			},
			hasChild:    true,
			folderToken: "FolderToken",
			pageToken:   "PageToken",
			setupMock: func(name string, dn *DocumentNode, hasChild bool, folderToken, pageToken string) {
				// 模拟 DriveList 的 API 响应
				gock.New("https://open.feishu.cn").
					Get("/open-apis/drive/v1/files").
					MatchParams(map[string]string{
						"folder_token": folderToken,
						"page_token":   pageToken,
						"order_by":     "EditedTime",
						"direction":    "DESC",
						"page_size":    "200",
					}).
					Reply(200).
					JSON(`{
    "code":0,
    "data":{
        "files":[],
        "has_more":false
    },
    "msg":"success"
}`)
			},
			teardownMock: func(name string) {
				for i, m := range gock.GetAll() {
					request := m.Request()
					s.T().Logf("Mock Http Request %d: %v", i+1, request.URLStruct)
				}
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
			want: &DocumentNode{
				DocumentInfo: DocumentInfo{
					Token:            "FolderToken",
					Name:             "Name",
					Type:             constant.DocTypeFolder,
					URL:              "Url",
					FileExtension:    "",
					CanDownload:      false,
					DownloadDirectly: false,
					NodeToken:        "",
					SpaceID:          "",
					FilePath:         "",
				},
			},
			wantError: "",
		},
		{
			name: "请求失败，一次递归",
			dn: &DocumentNode{
				DocumentInfo: DocumentInfo{
					Token:            "boxbc0dGSMu23m7QkC1bvabcef",
					Name:             "Name",
					Type:             constant.DocTypeFolder,
					URL:              "Url",
					FileExtension:    "",
					FilePath:         "",
					CanDownload:      false,
					DownloadDirectly: false,
				},
			},
			hasChild:    true,
			folderToken: "boxbc0dGSMu23m7QkC1bvabcef",
			pageToken:   "PageToken",
			setupMock: func(name string, dn *DocumentNode, hasChild bool, folderToken, pageToken string) {
				// 模拟 DriveList 的 API 响应
				gock.New("https://open.feishu.cn").
					Get("/open-apis/drive/v1/files").
					MatchParams(map[string]string{
						"folder_token": folderToken,
						"page_token":   pageToken,
						"order_by":     "EditedTime",
						"direction":    "DESC",
						"page_size":    "200",
					}).
					Reply(200).
					JSON(fmt.Sprintf(`{
    "code":0,
    "data":{
        "files":[
            {
                "name":"test folder",
                "parent_token":"%s",
                "token":"boxbc0dGSMu23m7QkC1bvabcef",
                "type":"folder",
                "created_time":"1679277808",
                "modified_time":"1679277808",
                "owner_id":"ou_20b31734443364ec8a1df89fdf325b44",
                "url":"https://feishu.cn/file/boxbc0dGSMu23m7QkC1bvabcef"
            }
        ],
        "has_more":false
    },
    "msg":"success"
}`, folderToken))
				// 因为第一个mock响应返回folder类型，产生递归调用，所以要mock第二次
				gock.New("https://open.feishu.cn").
					Get("/open-apis/drive/v1/files").
					MatchParams(map[string]string{
						"folder_token": "boxbc0dGSMu23m7QkC1bvabcef",
						"page_token":   "",
						"order_by":     "EditedTime",
						"direction":    "DESC",
						"page_size":    "200",
					}).
					Reply(400).
					AddHeader(larkcore.HttpHeaderKeyLogId, "xyz").
					JSON(`{"code":400,"msg":"Bad Request"}`)
			},
			teardownMock: func(name string) {
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
			want:      nil,
			wantError: "logId: \x1b]8;;https://open.feishu.cn/search?q=xyz\x1b\\, error response: \n{\n  Code: 400,\n  Msg: \"Bad Request\"\n}",
		},
		{
			name: "请求成功，两次递归",
			dn: &DocumentNode{
				DocumentInfo: DocumentInfo{
					Token:            "boxbc0dGSMu23m7QkC1bvabcef",
					Name:             "Name",
					Type:             constant.DocTypeFolder,
					URL:              "Url",
					FileExtension:    "",
					FilePath:         "",
					CanDownload:      false,
					DownloadDirectly: false,
				},
			},
			hasChild:    true,
			folderToken: "boxbc0dGSMu23m7QkC1bvabcef",
			pageToken:   "PageToken",
			setupMock: func(name string, dn *DocumentNode, hasChild bool, folderToken, pageToken string) {
				// 模拟 DriveList 的 API 响应
				gock.New("https://open.feishu.cn").
					Get("/open-apis/drive/v1/files").
					MatchParams(map[string]string{
						"folder_token": folderToken,
						"page_token":   pageToken,
						"order_by":     "EditedTime",
						"direction":    "DESC",
						"page_size":    "200",
					}).
					Reply(200).
					JSON(fmt.Sprintf(`{
    "code":0,
    "data":{
        "files":[
            {
                "name":"test folder",
                "parent_token":"%s",
                "token":"boxbc0dGSMu23m7QkC1bvabcef",
                "type":"folder",
                "created_time":"1679277808",
                "modified_time":"1679277808",
                "owner_id":"ou_20b31734443364ec8a1df89fdf325b44",
                "url":"https://feishu.cn/file/boxbc0dGSMu23m7QkC1bvabcef"
            }
        ],
        "next_page_token": "next_page_token",
        "has_more":true
    },
    "msg":"success"
}`, folderToken))
				// 因为第一个mock响应返回folder类型，产生递归调用，所以要mock第二次
				gock.New("https://open.feishu.cn").
					Get("/open-apis/drive/v1/files").
					MatchParams(map[string]string{
						"folder_token": "boxbc0dGSMu23m7QkC1bvabcef",
						"page_token":   "",
						"order_by":     "EditedTime",
						"direction":    "DESC",
						"page_size":    "200",
					}).
					Reply(200).
					JSON(`{
    "code":0,
    "data":{
        "files":[],
        "next_page_token": "",
        "has_more":false
    },
    "msg":"success"
}`)
				// 因为第一个mock响应has_more=true产生递归调用，所以要mock第三次
				// 第三次响应一个服务端的错误
				gock.New("https://open.feishu.cn").
					Get("/open-apis/drive/v1/files").
					MatchParams(map[string]string{
						"folder_token": folderToken,
						"page_token":   "next_page_token",
						"order_by":     "EditedTime",
						"direction":    "DESC",
						"page_size":    "200",
					}).
					Reply(200).
					JSON(`{
    "code":0,
    "data":{
        "files":[],
        "next_page_token": "",
        "has_more":false
    },
    "msg":"success"
}`)
			},
			teardownMock: func(name string) {
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
			want: &DocumentNode{
				DocumentInfo: DocumentInfo{
					Token:            "boxbc0dGSMu23m7QkC1bvabcef",
					Name:             "Name",
					Type:             constant.DocTypeFolder,
					URL:              "Url",
					FileExtension:    "",
					CanDownload:      false,
					DownloadDirectly: false,
					NodeToken:        "",
					SpaceID:          "",
					FilePath:         "",
				},
				Children: []*DocumentNode{
					{
						DocumentInfo: DocumentInfo{
							Name:             "test folder",
							Type:             "folder",
							Token:            "boxbc0dGSMu23m7QkC1bvabcef",
							FileExtension:    "folder",
							CanDownload:      false,
							DownloadDirectly: false,
							URL:              "https://feishu.cn/file/boxbc0dGSMu23m7QkC1bvabcef",
							NodeToken:        "",
							SpaceID:          "",
							FilePath:         "",
						},
					},
				},
			},
			wantError: "",
		},
		{
			name: "响应500失败",
			dn: &DocumentNode{
				DocumentInfo: DocumentInfo{
					Token:            "boxbc0dGSMu23m7QkC1bvabcef",
					Name:             "Name",
					Type:             constant.DocTypeFolder,
					URL:              "Url",
					FileExtension:    "",
					FilePath:         "",
					CanDownload:      false,
					DownloadDirectly: false,
				},
			},
			hasChild:    true,
			folderToken: "boxbc0dGSMu23m7QkC1bvabcef",
			pageToken:   "PageToken",
			setupMock: func(name string, dn *DocumentNode, hasChild bool, folderToken, pageToken string) {
				// 模拟 DriveList 的 API 响应
				gock.New("https://open.feishu.cn").
					Get("/open-apis/drive/v1/files").
					MatchParams(map[string]string{
						"folder_token": folderToken,
						"page_token":   pageToken,
						"order_by":     "EditedTime",
						"direction":    "DESC",
						"page_size":    "200",
					}).
					Reply(200).
					JSON(fmt.Sprintf(`{
    "code":0,
    "data":{
        "files":[
            {
                "name":"test folder",
                "parent_token":"%s",
                "token":"boxbc0dGSMu23m7QkC1bvabcef",
                "type":"folder",
                "created_time":"1679277808",
                "modified_time":"1679277808",
                "owner_id":"ou_20b31734443364ec8a1df89fdf325b44",
                "url":"https://feishu.cn/file/boxbc0dGSMu23m7QkC1bvabcef"
            }
        ],
        "next_page_token": "next_page_token",
        "has_more":true
    },
    "msg":"success"
}`, folderToken))
				// 因为第一个mock响应返回folder类型，产生递归调用，所以要mock第二次
				gock.New("https://open.feishu.cn").
					Get("/open-apis/drive/v1/files").
					MatchParams(map[string]string{
						"folder_token": "boxbc0dGSMu23m7QkC1bvabcef",
						"page_token":   "",
						"order_by":     "EditedTime",
						"direction":    "DESC",
						"page_size":    "200",
					}).
					Reply(200).
					JSON(fmt.Sprintf(`{
    "code":0,
    "data":{
        "files":[
            {
                "name":"test pdf.pdf",
                "parent_token":"%s",
                "token":"boxbc0dGSMu23m7QkC1bvabceg",
                "type":"file",
                "created_time":"1679277809",
                "modified_time":"1679277809",
                "owner_id":"ou_20b31734443364ec8a1df89fdf325b44",
                "url":"https://feishu.cn/file/boxbc0dGSMu23m7QkC1bvabceg"
            }
        ],
        "has_more":false
    },
    "msg":"success"
}`, folderToken))
				// 因为第一个mock响应has_more=true产生递归调用，所以要mock第三次
				// 第三次响应一个服务端的错误
				gock.New("https://open.feishu.cn").
					Get("/open-apis/drive/v1/files").
					MatchParams(map[string]string{
						"folder_token": folderToken,
						"page_token":   "next_page_token",
						"order_by":     "EditedTime",
						"direction":    "DESC",
						"page_size":    "200",
					}).
					Reply(500).
					JSON(`{"code":500,"msg":"failed"}`)
			},
			teardownMock: func(name string) {
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
			want:      nil,
			wantError: "logId: \x1b]8;;https://open.feishu.cn/search?q=\x1b\\, error response: \n{\n  Code: 500,\n  Msg: \"failed\"\n}",
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock(tt.name, tt.dn, tt.hasChild, tt.folderToken, tt.pageToken)
			err := s.client.fetchDriveDescendant(tt.dn, tt.hasChild, tt.folderToken, tt.pageToken)
			tt.teardownMock(tt.name)
			if err != nil || tt.wantError != "" {
				s.Require().Error(err, tt.name)
				s.IsType(oops.OopsError{}, err, tt.name)
				var actualError oops.OopsError
				yes := errors.As(err, &actualError)
				s.Require().True(yes, tt.name)
				s.Equal("", actualError.Code(), tt.name)
				s.Equal(tt.wantError, actualError.Error(), tt.name)
			} else {
				s.Require().NoError(err, tt.name)
				s.Equal(tt.want, tt.dn, tt.name)
			}
		})
	}
}

func (s *DocumentDriveTestSuite) TestClientImpl_fileToDocumentNode() {
	type args struct {
		file *larkdrive.File
		args *Args
	}
	tests := []struct {
		name string
		args args
		want *DocumentNode
	}{
		{
			name: "DocTypeShortcut",
			args: args{
				file: &larkdrive.File{
					Token:       larkcore.StringPtr("Token"),
					Name:        larkcore.StringPtr("Name"),
					Type:        larkcore.StringPtr("shortcut"),
					ParentToken: larkcore.StringPtr("ParentToken"),
					Url:         larkcore.StringPtr("Url"),
					ShortcutInfo: &larkdrive.ShortcutInfo{
						TargetType:  larkcore.StringPtr(string(constant.DocTypeDocx)),
						TargetToken: larkcore.StringPtr("TargetToken"),
					},
					CreatedTime:  larkcore.StringPtr("CreatedTime"),
					ModifiedTime: larkcore.StringPtr("ModifiedTime"),
					OwnerId:      larkcore.StringPtr("OwnerId"),
				},
				args: &Args{
					FileExtensions: map[constant.DocType]constant.FileExt{
						constant.DocTypeDocx: constant.FileExtPDF,
					},
				},
			},
			want: &DocumentNode{
				DocumentInfo: DocumentInfo{
					Token:            "TargetToken",
					Name:             "Name",
					Type:             constant.DocTypeDocx,
					URL:              "Url",
					FileExtension:    constant.FileExtPDF,
					FilePath:         "",
					CanDownload:      true,
					DownloadDirectly: false,
				},
			},
		},
		{
			name: "DocTypeNotShortcut",
			args: args{
				file: &larkdrive.File{
					Token:        larkcore.StringPtr("Token"),
					Name:         larkcore.StringPtr("Name"),
					Type:         larkcore.StringPtr(string(constant.DocTypeDoc)),
					ParentToken:  larkcore.StringPtr("ParentToken"),
					Url:          larkcore.StringPtr("Url"),
					ShortcutInfo: nil,
					CreatedTime:  larkcore.StringPtr("CreatedTime"),
					ModifiedTime: larkcore.StringPtr("ModifiedTime"),
					OwnerId:      larkcore.StringPtr("OwnerId"),
				},
				args: &Args{
					FileExtensions: map[constant.DocType]constant.FileExt{
						constant.DocTypeDoc: constant.FileExtPDF,
					},
				},
			},
			want: &DocumentNode{
				DocumentInfo: DocumentInfo{
					Token:            "Token",
					Name:             "Name",
					Type:             constant.DocTypeDoc,
					URL:              "Url",
					FileExtension:    constant.FileExtPDF,
					FilePath:         "",
					CanDownload:      true,
					DownloadDirectly: false,
				},
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := &ClientImpl{
				Client:      nil,
				Args:        tt.args.args,
				TaskCreator: nil,
			}
			actual := c.fileToDocumentNode(tt.args.file)
			s.Equal(tt.want, actual, tt.name)
		})
	}
}
