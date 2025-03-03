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

	"github.com/acyumi/doc-exporter/component/argument"
	"github.com/acyumi/doc-exporter/component/cloud"
	"github.com/acyumi/doc-exporter/component/constant"
)

func TestDocumentDriveSuite(t *testing.T) {
	suite.Run(t, new(DocumentDriveTestSuite))
}

type DocumentDriveTestSuite struct {
	suite.Suite
	client   *ClientImpl
	mockTask *mockTask
}

func (s *DocumentDriveTestSuite) SetupSuite() {
	initBackOff = testInitBackOff
	cleanSleep()
}

func (s *DocumentDriveTestSuite) SetupTest() {
	s.client = NewClient(&argument.Args{
		AppID:     "cli_xxx",
		AppSecret: "xxx",
		StartTime: time.Now(),
	}).(*ClientImpl)
	s.mockTask = &mockTask{}
	s.client.TaskCreator = func(args *argument.Args, docs *DocumentNode) cloud.Task {
		return s.mockTask
	}
}

func (s *DocumentDriveTestSuite) TearDownTest() {
	s.mockTask.AssertExpectations(s.T())
}

func (s *DocumentDriveTestSuite) TearDownSuite() {
	initBackOff = initExponentialBackOff
}

func (s *DocumentDriveTestSuite) TestDownloadDriveDocuments() {
	tests := []struct {
		name      string
		typ       constant.DocType
		token     string
		setupMock func(mt *mockTask, name string)
		wantError string
		want      func(mt *mockTask, name string)
	}{
		{
			name:  "请求报错",
			typ:   constant.DocTypeFolder,
			token: "token",
			setupMock: func(mt *mockTask, name string) {
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
			wantError: "logId: \x1b]8;;https://open.feishu.cn/search?q=xyz\x1b\\, error response: \n{\n  Code: 500,\n  Msg: \"something wrong\"\n}",
			want: func(mt *mockTask, name string) {
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
		},
		{
			name:  "服务端响应有failed_list",
			typ:   constant.DocTypeFile,
			token: "fileToken",
			setupMock: func(mt *mockTask, name string) {
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
			wantError: "获取文件夹元数据失败: code: 970005, token: boxcnrHpsg1QDqXAAAyachabcef",
			want: func(mt *mockTask, name string) {
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
		},
		{
			name:  "请求成功，读到文件列表",
			typ:   constant.DocTypeFolder,
			token: "folderToken",
			setupMock: func(mt *mockTask, name string) {
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
				mt.On("Validate").Return(nil)
				mt.On("Run").Return(nil)
				mt.On("Close").Return()
			},
			wantError: "",
			want: func(mt *mockTask, name string) {
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
		},
		{
			name:  "读到文件列表失败",
			typ:   constant.DocTypeFolder,
			token: "folderToken",
			setupMock: func(mt *mockTask, name string) {
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
			wantError: "logId: \x1b]8;;https://open.feishu.cn/search?q=xyz\x1b\\, error response: \n{\n  Code: 400,\n  Msg: \"Failed\"\n}",
			want: func(mt *mockTask, name string) {
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
		},
		{
			name:  "非目录，请求成功",
			typ:   constant.DocTypeFile,
			token: "fileToken",
			setupMock: func(mt *mockTask, name string) {
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
				mt.On("Validate").Return(nil)
				mt.On("Run").Return(nil)
				mt.On("Close").Return()
			},
			wantError: "",
			want: func(mt *mockTask, name string) {
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
		},
		{
			name:  "/docs",
			typ:   constant.DocTypeDoc,
			token: "token",
			setupMock: func(mt *mockTask, name string) {
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
			wantError: "logId: \x1b]8;;https://open.feishu.cn/search?q=xyz\x1b\\, error response: \n{\n  Code: 500,\n  Msg: \"something wrong\"\n}",
			want: func(mt *mockTask, name string) {
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock(s.mockTask, tt.name)
			err := s.client.DownloadDriveDocuments(tt.typ, tt.token)
			if err != nil || tt.wantError != "" {
				s.Require().EqualError(err, tt.wantError, tt.name)
			}
		})
	}
}

func (s *DocumentDriveTestSuite) Test_fetchDriveDescendant() {
	checkAuthenticated()
	tests := []struct {
		name         string
		di           *DocumentNode
		hasChild     bool
		folderToken  string
		pageToken    string
		setupMock    func(name string, di *DocumentNode, hasChild bool, folderToken, pageToken string)
		teardownMock func(name string)
		want         *DocumentNode
		wantError    string
	}{
		{
			name: "hasChild为false",
			di: &DocumentNode{
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
			setupMock: func(name string, di *DocumentNode, hasChild bool, folderToken, pageToken string) {
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
			di: &DocumentNode{
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
			setupMock: func(name string, di *DocumentNode, hasChild bool, folderToken, pageToken string) {
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
			di: &DocumentNode{
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
			setupMock: func(name string, di *DocumentNode, hasChild bool, folderToken, pageToken string) {
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
			di: &DocumentNode{
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
			setupMock: func(name string, di *DocumentNode, hasChild bool, folderToken, pageToken string) {
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
			di: &DocumentNode{
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
			setupMock: func(name string, di *DocumentNode, hasChild bool, folderToken, pageToken string) {
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
			tt.setupMock(tt.name, tt.di, tt.hasChild, tt.folderToken, tt.pageToken)
			err := s.client.fetchDriveDescendant(tt.di, tt.hasChild, tt.folderToken, tt.pageToken)
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
				s.Equal(tt.want, tt.di, tt.name)
			}
		})
	}
}

func (s *DocumentDriveTestSuite) TestClientImpl_fileToDocumentNode() {
	type args struct {
		file *larkdrive.File
		args *argument.Args
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
				args: &argument.Args{
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
				args: &argument.Args{
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
