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
	"testing"
	"time"

	"github.com/h2non/gock"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkwiki "github.com/larksuite/oapi-sdk-go/v3/service/wiki/v2"
	"github.com/samber/oops"
	"github.com/stretchr/testify/suite"

	"github.com/acyumi/xdoc/component/argument"
	"github.com/acyumi/xdoc/component/constant"
)

func TestDocumentWikiSuite(t *testing.T) {
	suite.Run(t, new(DocumentWikiTestSuite))
}

type DocumentWikiTestSuite struct {
	suite.Suite
	client *ClientImpl
}

func (s *DocumentWikiTestSuite) SetupSuite() {
	initBackOff = testInitBackOff
	cleanSleep()
}

func (s *DocumentWikiTestSuite) SetupTest() {
	s.client = NewClient(&argument.Args{
		AppID:     "cli_xxx",
		AppSecret: "xxx",
		StartTime: time.Now(),
	}).(*ClientImpl)
}

func (s *DocumentWikiTestSuite) TearDownTest() {
}

func (s *DocumentWikiTestSuite) TearDownSuite() {
	initBackOff = initExponentialBackOff
}

func (s *DocumentWikiTestSuite) TestQueryWikiDocuments() {
	tests := []struct {
		name         string
		token        string
		setupMock    func(name, token string)
		teardownMock func(name, token string)
		wantError    string
		want         *DocumentNode
	}{
		{
			name:  "请求报错",
			token: "token",
			setupMock: func(name, token string) {
				gock.New("https://open.feishu.cn").
					Get("/open-apis/wiki/v2/spaces/get_node").
					MatchParam("obj_type", "wiki").
					MatchParam("token", token).
					Reply(500).
					AddHeader(larkcore.HttpHeaderKeyLogId, "xyz").
					JSON(`{"code": 500,"msg": "something wrong"}`)
			},
			teardownMock: func(name, token string) {
				for i, m := range gock.GetAll() {
					request := m.Request()
					s.T().Logf("Mock Http Request %d: %v", i+1, request.URLStruct)
				}
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
			wantError: "logId: \x1b]8;;https://open.feishu.cn/search?q=xyz\x1b\\, error response: \n{\n  Code: 500,\n  Msg: \"something wrong\"\n}",
		},
		{
			name:  "请求成功，读到文件列表",
			token: "token",
			setupMock: func(name, token string) {
				gock.New("https://open.feishu.cn").
					Get("/open-apis/wiki/v2/spaces/get_node").
					MatchParam("obj_type", "wiki").
					MatchParam("token", token).
					Reply(200).
					AddHeader(larkcore.HttpHeaderKeyLogId, "xyz").
					JSON(`{
    "code": 0,
    "msg": "success",
    "data": {
        "node": {
            "space_id": "6946843325487912356",
            "node_token": "wikcnKQ1k3pxxxxxx8Vabcef",
            "obj_token": "doccnzAaODxxxxxxWabcdef",
            "obj_type": "doc",
            "parent_node_token": "wikcnKQ1k3pxxxxxx8Vabcef",
            "node_type": "origin",
            "origin_node_token": "wikcnKQ1k3pxxxxxx8Vabcef",
            "origin_space_id": "6946843325487912356",
            "has_child": true,
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
				// 模拟 WikiNodeList 的 API 响应
				gock.New("https://open.feishu.cn").
					Get("/open-apis/wiki/v2/spaces/6946843325487912356/nodes").
					PathParam("spaces", "6946843325487912356").
					MatchParams(map[string]string{
						"parent_node_token": "wikcnKQ1k3pxxxxxx8Vabcef",
						"page_token":        "",
						"page_size":         "50",
					}).
					Reply(200).
					JSON(`{
    "code": 0,
    "msg": "success",
    "data": {
        "items": [
            {
                "space_id": "6946843325487912356",
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
}`)
			},
			teardownMock: func(name, token string) {
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
			wantError: "",
			want: &DocumentNode{
				DocumentInfo: DocumentInfo{
					Name:             "标题",
					Type:             "doc",
					Token:            "doccnzAaODxxxxxxWabcdef",
					FileExtension:    "docx",
					CanDownload:      true,
					DownloadDirectly: false,
					URL:              "",
					NodeToken:        "wikcnKQ1k3pxxxxxx8Vabcef",
					SpaceID:          "6946843325487912356",
					FilePath:         "",
				},
				Children: []*DocumentNode{
					{
						DocumentInfo: DocumentInfo{
							Name:             "标题",
							Type:             "doc",
							Token:            "doccnzAaODxxxxxxWabcdeg",
							FileExtension:    "docx",
							CanDownload:      true,
							DownloadDirectly: false,
							URL:              "",
							NodeToken:        "wikcnKQ1k3pxxxxxx8Vabceg",
							SpaceID:          "6946843325487912356",
							FilePath:         "",
						},
					},
				},
			},
		},
		{
			name:  "读到文件列表失败",
			token: "token",
			setupMock: func(name, token string) {
				gock.New("https://open.feishu.cn").
					Get("/open-apis/wiki/v2/spaces/get_node").
					MatchParam("obj_type", "wiki").
					MatchParam("token", token).
					Reply(200).
					AddHeader(larkcore.HttpHeaderKeyLogId, "xyz").
					JSON(`{
    "code": 0,
    "msg": "success",
    "data": {
        "node": {
            "space_id": "6946843325487912356",
            "node_token": "wikcnKQ1k3pxxxxxx8Vabcef",
            "obj_token": "doccnzAaODxxxxxxWabcdef",
            "obj_type": "doc",
            "parent_node_token": "wikcnKQ1k3pxxxxxx8Vabcef",
            "node_type": "origin",
            "origin_node_token": "wikcnKQ1k3pxxxxxx8Vabcef",
            "origin_space_id": "6946843325487912356",
            "has_child": true,
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
				// 模拟 WikiNodeList 的 API 响应
				gock.New("https://open.feishu.cn").
					Get("/open-apis/wiki/v2/spaces/6946843325487912356/nodes").
					PathParam("spaces", "6946843325487912356").
					MatchParams(map[string]string{
						"parent_node_token": "wikcnKQ1k3pxxxxxx8Vabcef",
						"page_token":        "",
						"page_size":         "50",
					}).
					Reply(400).
					AddHeader(larkcore.HttpHeaderKeyLogId, "xyz").
					JSON(`{"code":400,"msg":"Failed"}`)
			},
			teardownMock: func(name, token string) {
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
			wantError: "logId: \x1b]8;;https://open.feishu.cn/search?q=xyz\x1b\\, error response: \n{\n  Code: 400,\n  Msg: \"Failed\"\n}",
		},
		{
			name:  "无子集，请求成功",
			token: "token",
			setupMock: func(name, token string) {
				gock.New("https://open.feishu.cn").
					Get("/open-apis/wiki/v2/spaces/get_node").
					MatchParam("obj_type", "wiki").
					MatchParam("token", token).
					Reply(200).
					AddHeader(larkcore.HttpHeaderKeyLogId, "xyz").
					JSON(`{
    "code": 0,
    "msg": "success",
    "data": {
        "node": {
            "space_id": "6946843325487912356",
            "node_token": "wikcnKQ1k3pxxxxxx8Vabcef",
            "obj_token": "doccnzAaODxxxxxxWabcdef",
            "obj_type": "doc",
            "parent_node_token": "wikcnKQ1k3pxxxxxx8Vabcef",
            "node_type": "origin",
            "origin_node_token": "wikcnKQ1k3pxxxxxx8Vabcef",
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
			},
			teardownMock: func(name, token string) {
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
			wantError: "",
			want: &DocumentNode{
				DocumentInfo: DocumentInfo{
					Name:             "标题",
					Type:             "doc",
					Token:            "doccnzAaODxxxxxxWabcdef",
					FileExtension:    "docx",
					CanDownload:      true,
					DownloadDirectly: false,
					URL:              "",
					NodeToken:        "wikcnKQ1k3pxxxxxx8Vabcef",
					SpaceID:          "6946843325487912356",
					FilePath:         "",
				},
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock(tt.name, tt.token)
			actual, err := s.client.QueryWikiDocuments(tt.token)
			tt.teardownMock(tt.name, tt.token)
			if err != nil || tt.wantError != "" {
				s.Require().EqualError(err, tt.wantError, tt.name)
			}
			s.Equal(tt.want, actual, tt.name)
		})
	}
}

func (s *DocumentWikiTestSuite) TestQueryWikiSpaceDocuments() {
	tests := []struct {
		name         string
		spaceID      string
		setupMock    func(name, spaceID string)
		teardownMock func(name, spaceID string)
		wantError    string
		want         *DocumentNode
	}{
		{
			name:    "请求报错",
			spaceID: "6946843325487912356",
			setupMock: func(name, spaceID string) {
				gock.New("https://open.feishu.cn").
					Get("/open-apis/wiki/v2/spaces/"+spaceID).
					PathParam("spaces", spaceID).
					Reply(500).
					AddHeader(larkcore.HttpHeaderKeyLogId, "xyz").
					JSON(`{"code": 500,"msg": "something wrong"}`)
			},
			teardownMock: func(name, token string) {
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
			wantError: "logId: \x1b]8;;https://open.feishu.cn/search?q=xyz\x1b\\, error response: \n{\n  Code: 500,\n  Msg: \"something wrong\"\n}",
		},
		{
			name:    "请求成功，读到文件列表",
			spaceID: "6946843325487912366",
			setupMock: func(name, spaceID string) {
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
					Get("/open-apis/wiki/v2/spaces/6946843325487912366/nodes").
					PathParam("spaces", "6946843325487912366").
					MatchParams(map[string]string{
						"parent_node_token": "",
						"page_token":        "",
						"page_size":         "50",
					}).
					Reply(200).
					JSON(`{
    "code": 0,
    "msg": "success",
    "data": {
        "items": [
            {
                "space_id": "6946843325487912366",
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
}`)
			},
			teardownMock: func(name, spaceID string) {
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
			wantError: "",
			want: &DocumentNode{
				DocumentInfo: DocumentInfo{
					Name:             "知识空间",
					Type:             "folder",
					Token:            "6946843325487912366",
					FileExtension:    "",
					CanDownload:      false,
					DownloadDirectly: false,
					URL:              "",
					NodeToken:        "",
					SpaceID:          "6946843325487912366",
					FilePath:         "",
				},
				Children: []*DocumentNode{
					{
						DocumentInfo: DocumentInfo{
							Name:             "标题",
							Type:             "doc",
							Token:            "doccnzAaODxxxxxxWabcdeg",
							FileExtension:    "docx",
							CanDownload:      true,
							DownloadDirectly: false,
							URL:              "",
							NodeToken:        "wikcnKQ1k3pxxxxxx8Vabceg",
							SpaceID:          "6946843325487912366",
							FilePath:         "",
						},
					},
				},
			},
		},
		{
			name:    "读到文件列表失败",
			spaceID: "6946843325487912356",
			setupMock: func(name, spaceID string) {
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
            "space_id": "6946843325487912356"
        }
    }
}`)
				// 模拟 WikiNodeList 的 API 响应
				gock.New("https://open.feishu.cn").
					Get("/open-apis/wiki/v2/spaces/6946843325487912356/nodes").
					PathParam("spaces", "6946843325487912356").
					MatchParams(map[string]string{
						"parent_node_token": "",
						"page_token":        "",
						"page_size":         "50",
					}).
					Reply(400).
					AddHeader(larkcore.HttpHeaderKeyLogId, "xyz").
					JSON(`{"code":400,"msg":"Failed"}`)
			},
			teardownMock: func(name, spaceID string) {
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
			wantError: "logId: \x1b]8;;https://open.feishu.cn/search?q=xyz\x1b\\, error response: \n{\n  Code: 400,\n  Msg: \"Failed\"\n}",
		},
		{
			name:    "无子集，请求成功",
			spaceID: "6946843325487912356",
			setupMock: func(name, spaceID string) {
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
            "space_id": "6946843325487912356"
        }
    }
}`)
				// 模拟 WikiNodeList 的 API 响应
				gock.New("https://open.feishu.cn").
					Get("/open-apis/wiki/v2/spaces/6946843325487912356/nodes").
					PathParam("spaces", "6946843325487912356").
					MatchParams(map[string]string{
						"parent_node_token": "",
						"page_token":        "",
						"page_size":         "50",
					}).
					Reply(200).
					JSON(`{
    "code": 0,
    "msg": "success",
    "data": {
        "items": [],
        "page_token": "",
        "has_more": false
    }
}`)
			},
			teardownMock: func(name, spaceID string) {
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
			wantError: "",
			want: &DocumentNode{
				DocumentInfo: DocumentInfo{
					Name:             "知识空间",
					Type:             "folder",
					Token:            "6946843325487912356",
					FileExtension:    "",
					CanDownload:      false,
					DownloadDirectly: false,
					URL:              "",
					NodeToken:        "",
					SpaceID:          "6946843325487912356",
					FilePath:         "",
				},
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock(tt.name, tt.spaceID)
			actual, err := s.client.QueryWikiSpaceDocuments(tt.spaceID)
			tt.teardownMock(tt.name, tt.spaceID)
			if err != nil || tt.wantError != "" {
				s.Require().EqualError(err, tt.wantError, tt.name)
			}
			s.Equal(tt.want, actual, tt.name)
		})
	}
}

func (s *DocumentWikiTestSuite) Test_fetchWikiDescendant() {
	tests := []struct {
		name            string
		dn              *DocumentNode
		hasChild        bool
		spaceID         string
		parentNodeToken string
		pageToken       string
		setupMock       func(name string, dn *DocumentNode, hasChild bool, spaceID, parentNodeToken, pageToken string)
		teardownMock    func(name string)
		want            *DocumentNode
		wantError       string
	}{
		{
			name: "hasChild为false",
			dn: &DocumentNode{
				DocumentInfo: DocumentInfo{
					Token:            "Token",
					Name:             "Name",
					Type:             constant.DocTypeDocx,
					URL:              "Url",
					FileExtension:    "",
					FilePath:         "",
					CanDownload:      true,
					DownloadDirectly: false,
				},
			},
			hasChild:        false,
			spaceID:         "space_id",
			parentNodeToken: "Token",
			pageToken:       "PageToken",
			setupMock: func(name string, dn *DocumentNode, hasChild bool, spaceID, parentNodeToken, pageToken string) {
			},
			teardownMock: func(name string) {
			},
			want: &DocumentNode{
				DocumentInfo: DocumentInfo{
					Name:             "Name",
					Type:             constant.DocTypeDocx,
					Token:            "Token",
					FileExtension:    "",
					CanDownload:      true,
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
			name: "获取超时",
			dn: &DocumentNode{
				DocumentInfo: DocumentInfo{
					Token:            "Token",
					Name:             "Name",
					Type:             constant.DocTypeDocx,
					URL:              "Url",
					FileExtension:    "",
					FilePath:         "",
					CanDownload:      true,
					DownloadDirectly: false,
				},
			},
			hasChild:        true,
			spaceID:         "space_id",
			parentNodeToken: "Token",
			pageToken:       "PageToken",
			setupMock: func(name string, dn *DocumentNode, hasChild bool, spaceID, parentNodeToken, pageToken string) {
				s.client.Args.StartTime = time.Now().Add(-time.Hour)
			},
			teardownMock: func(name string) {
				s.client.Args.StartTime = time.Now()
			},
			want: &DocumentNode{
				DocumentInfo: DocumentInfo{
					Name:             "Name",
					Type:             constant.DocTypeDocx,
					Token:            "Token",
					FileExtension:    "",
					CanDownload:      true,
					DownloadDirectly: false,
					URL:              "Url",
					NodeToken:        "",
					SpaceID:          "",
					FilePath:         "",
				},
			},
			wantError: "获取知识库文档信息超时: 1m0s",
		},
		{
			name: "请求成功，没有递归",
			dn: &DocumentNode{
				DocumentInfo: DocumentInfo{
					Token:            "Token",
					Name:             "Name",
					Type:             constant.DocTypeDocx,
					URL:              "Url",
					FileExtension:    "",
					FilePath:         "",
					CanDownload:      true,
					DownloadDirectly: false,
				},
			},
			hasChild:        true,
			spaceID:         "space_id",
			parentNodeToken: "Token",
			pageToken:       "PageToken",
			setupMock: func(name string, dn *DocumentNode, hasChild bool, spaceID, parentNodeToken, pageToken string) {
				// 模拟 WikiNodeList 的 API 响应
				gock.New("https://open.feishu.cn").
					Get("/open-apis/wiki/v2/spaces/space_id/nodes").
					PathParam("spaces", spaceID).
					MatchParams(map[string]string{
						"parent_node_token": parentNodeToken,
						"page_token":        pageToken,
						"page_size":         "50",
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
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
			want: &DocumentNode{
				DocumentInfo: DocumentInfo{
					Token:            "Token",
					Name:             "Name",
					Type:             constant.DocTypeDocx,
					URL:              "Url",
					FileExtension:    "",
					CanDownload:      true,
					DownloadDirectly: false,
					NodeToken:        "",
					SpaceID:          "",
					FilePath:         "",
				},
			},
			wantError: "",
		},
		{
			name: "请求成功，一次递归，但文件无子集，不发起第二次请求",
			dn: &DocumentNode{
				DocumentInfo: DocumentInfo{
					Token:            "boxbc0dGSMu23m7QkC1bvabcef",
					Name:             "Name",
					Type:             constant.DocTypeDocx,
					URL:              "Url",
					FileExtension:    "",
					FilePath:         "",
					CanDownload:      true,
					DownloadDirectly: false,
				},
			},
			hasChild:        true,
			spaceID:         "space_id",
			parentNodeToken: "boxbc0dGSMu23m7QkC1bvabcef",
			pageToken:       "PageToken",
			setupMock: func(name string, dn *DocumentNode, hasChild bool, spaceID, parentNodeToken, pageToken string) {
				// 模拟 WikiNodeList 的 API 响应
				gock.New("https://open.feishu.cn").
					Get("/open-apis/wiki/v2/spaces/space_id/nodes").
					PathParam("spaces", spaceID).
					MatchParams(map[string]string{
						"parent_node_token": parentNodeToken,
						"page_token":        pageToken,
						"page_size":         "50",
					}).
					Reply(200).
					JSON(`{
    "code": 0,
    "msg": "success",
    "data": {
        "items": [
            {
                "space_id": "6946843325487912356",
                "node_token": "wikcnKQ1k3pxxxxxx8Vabcef",
                "obj_token": "doccnzAaODxxxxxxWabcdef",
                "obj_type": "doc",
                "parent_node_token": "wikcnKQ1k3pxxxxxx8Vabcef",
                "node_type": "origin",
                "origin_node_token": "wikcnKQ1k3pxxxxxx8Vabcef",
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
        "has_more": false
    }
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
					Type:             constant.DocTypeDocx,
					URL:              "Url",
					FileExtension:    "",
					FilePath:         "",
					CanDownload:      true,
					DownloadDirectly: false,
				},
				Children: []*DocumentNode{
					{
						DocumentInfo: DocumentInfo{
							Token:            "doccnzAaODxxxxxxWabcdef",
							Name:             "标题",
							Type:             constant.DocTypeDoc,
							NodeToken:        "wikcnKQ1k3pxxxxxx8Vabcef",
							URL:              "",
							FileExtension:    "docx",
							FilePath:         "",
							SpaceID:          "6946843325487912356",
							CanDownload:      true,
							DownloadDirectly: false,
						},
					},
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
					Type:             constant.DocTypeDocx,
					URL:              "Url",
					FileExtension:    "",
					FilePath:         "",
					CanDownload:      true,
					DownloadDirectly: false,
				},
			},
			hasChild:        true,
			spaceID:         "space_id",
			parentNodeToken: "boxbc0dGSMu23m7QkC1bvabcef",
			pageToken:       "PageToken",
			setupMock: func(name string, dn *DocumentNode, hasChild bool, spaceID, parentNodeToken, pageToken string) {
				// 模拟 WikiNodeList 的 API 响应
				gock.New("https://open.feishu.cn").
					Get("/open-apis/wiki/v2/spaces/space_id/nodes").
					PathParam("spaces", spaceID).
					MatchParams(map[string]string{
						"parent_node_token": parentNodeToken,
						"page_token":        pageToken,
						"page_size":         "50",
					}).
					Reply(200).
					JSON(`{
    "code": 0,
    "msg": "success",
    "data": {
        "items": [
            {
                "space_id": "6946843325487912356",
                "node_token": "wikcnKQ1k3pxxxxxx8Vabcef",
                "obj_token": "doccnzAaODxxxxxxWabcdef",
                "obj_type": "doc",
                "parent_node_token": "wikcnKQ1k3pxxxxxx8Vabcef",
                "node_type": "origin",
                "origin_node_token": "wikcnKQ1k3pxxxxxx8Vabcef",
                "origin_space_id": "6946843325487912356",
                "has_child": true,
                "title": "标题",
                "obj_create_time": "1642402428",
                "obj_edit_time": "1642402428",
                "node_create_time": "1642402428",
                "creator": "ou_xxxxx",
                "owner": "ou_xxxxx",
                "node_creator": "ou_xxxxx"
            }
        ],
        "has_more": false
    }
}`)
				// 因为第一个mock响应返回了文件列表，产生递归调用，所以要mock第二次
				gock.New("https://open.feishu.cn").
					Get("/open-apis/wiki/v2/spaces/space_id/nodes").
					PathParam("spaces", spaceID).
					MatchParams(map[string]string{
						"parent_node_token": "wikcnKQ1k3pxxxxxx8Vabcef",
						"page_token":        "",
						"page_size":         "50",
					}).
					Reply(400).
					AddHeader(larkcore.HttpHeaderKeyLogId, "xyz").
					JSON(`{"code":400,"msg":"Bad Request"}`)
			},
			teardownMock: func(name string) {
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
			want: &DocumentNode{
				DocumentInfo: DocumentInfo{
					Token:            "boxbc0dGSMu23m7QkC1bvabcef",
					Name:             "Name",
					Type:             constant.DocTypeDocx,
					URL:              "Url",
					FileExtension:    "",
					FilePath:         "",
					CanDownload:      true,
					DownloadDirectly: false,
				},
				Children: []*DocumentNode{
					{
						DocumentInfo: DocumentInfo{
							Token:            "doccnzAaODxxxxxxWabcdef",
							Name:             "标题",
							Type:             constant.DocTypeDoc,
							NodeToken:        "wikcnKQ1k3pxxxxxx8Vabcef",
							URL:              "",
							FileExtension:    "docx",
							FilePath:         "",
							SpaceID:          "6946843325487912356",
							CanDownload:      true,
							DownloadDirectly: false,
						},
					},
				},
			},
			wantError: "logId: \x1b]8;;https://open.feishu.cn/search?q=xyz\x1b\\, error response: \n{\n  Code: 400,\n  Msg: \"Bad Request\"\n}",
		},
		{
			name: "请求成功，两次递归",
			dn: &DocumentNode{
				DocumentInfo: DocumentInfo{
					Token:            "boxbc0dGSMu23m7QkC1bvabcef",
					Name:             "Name",
					Type:             constant.DocTypeDocx,
					URL:              "Url",
					FileExtension:    "",
					FilePath:         "",
					CanDownload:      false,
					DownloadDirectly: false,
				},
			},
			hasChild:        true,
			spaceID:         "space_id",
			parentNodeToken: "boxbc0dGSMu23m7QkC1bvabcef",
			pageToken:       "PageToken",
			setupMock: func(name string, dn *DocumentNode, hasChild bool, spaceID, parentNodeToken, pageToken string) {
				// 模拟 WikiNodeList 的 API 响应
				gock.New("https://open.feishu.cn").
					Get("/open-apis/wiki/v2/spaces/space_id/nodes").
					PathParam("spaces", spaceID).
					MatchParams(map[string]string{
						"parent_node_token": parentNodeToken,
						"page_token":        pageToken,
						"page_size":         "50",
					}).
					Reply(200).
					JSON(`{
    "code": 0,
    "msg": "success",
    "data": {
        "items": [
            {
                "space_id": "6946843325487912356",
                "node_token": "wikcnKQ1k3pxxxxxx8Vabcef",
                "obj_token": "doccnzAaODxxxxxxWabcdef",
                "obj_type": "doc",
                "parent_node_token": "wikcnKQ1k3pxxxxxx8Vabcef",
                "node_type": "origin",
                "origin_node_token": "wikcnKQ1k3pxxxxxx8Vabcef",
                "origin_space_id": "6946843325487912356",
                "has_child": true,
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
				// 因为第一个mock响应返回了文件列表，产生递归调用，所以要mock第二次
				gock.New("https://open.feishu.cn").
					Get("/open-apis/wiki/v2/spaces/space_id/nodes").
					PathParam("spaces", spaceID).
					MatchParams(map[string]string{
						"parent_node_token": "wikcnKQ1k3pxxxxxx8Vabcef",
						"page_token":        "",
						"page_size":         "50",
					}).
					Reply(200).
					JSON(`{
   "code":0,
   "data":{
       "files":[],
       "page_token": "",
       "has_more":false
   },
   "msg":"success"
}`)
				// 因为第一个mock响应has_more=true产生递归调用，所以要mock第三次
				// 第三次响应一个服务端的错误
				gock.New("https://open.feishu.cn").
					Get("/open-apis/wiki/v2/spaces/space_id/nodes").
					PathParam("spaces", spaceID).
					MatchParams(map[string]string{
						"parent_node_token": parentNodeToken,
						"page_token":        "6946843325487906839",
						"page_size":         "50",
					}).
					Reply(200).
					JSON(`{
   "code":0,
   "data":{
       "items": [],
       "page_token": "",
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
					Type:             constant.DocTypeDocx,
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
							Name:             "标题",
							Type:             "doc",
							Token:            "doccnzAaODxxxxxxWabcdef",
							FileExtension:    "docx",
							CanDownload:      true,
							DownloadDirectly: false,
							URL:              "",
							NodeToken:        "wikcnKQ1k3pxxxxxx8Vabcef",
							SpaceID:          "6946843325487912356",
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
					Type:             constant.DocTypeDocx,
					URL:              "Url",
					FileExtension:    "",
					FilePath:         "",
					CanDownload:      false,
					DownloadDirectly: false,
				},
			},
			hasChild:        true,
			spaceID:         "space_id",
			parentNodeToken: "boxbc0dGSMu23m7QkC1bvabcef",
			pageToken:       "PageToken",
			setupMock: func(name string, dn *DocumentNode, hasChild bool, spaceID, parentNodeToken, pageToken string) {
				// 模拟 WikiNodeList 的 API 响应
				gock.New("https://open.feishu.cn").
					Get("/open-apis/wiki/v2/spaces/space_id/nodes").
					PathParam("spaces", spaceID).
					MatchParams(map[string]string{
						"parent_node_token": parentNodeToken,
						"page_token":        pageToken,
						"page_size":         "50",
					}).
					Reply(200).
					JSON(`{
    "code": 0,
    "msg": "success",
    "data": {
        "items": [
            {
                "space_id": "6946843325487912356",
                "node_token": "wikcnKQ1k3pxxxxxx8Vabcef",
                "obj_token": "doccnzAaODxxxxxxWabcdef",
                "obj_type": "doc",
                "parent_node_token": "wikcnKQ1k3pxxxxxx8Vabcef",
                "node_type": "origin",
                "origin_node_token": "wikcnKQ1k3pxxxxxx8Vabcef",
                "origin_space_id": "6946843325487912356",
                "has_child": true,
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
				// 因为第一个mock响应返回了文件列表，产生递归调用，所以要mock第二次
				gock.New("https://open.feishu.cn").
					Get("/open-apis/wiki/v2/spaces/space_id/nodes").
					PathParam("spaces", spaceID).
					MatchParams(map[string]string{
						"parent_node_token": "wikcnKQ1k3pxxxxxx8Vabcef",
						"page_token":        "",
						"page_size":         "50",
					}).
					Reply(200).
					JSON(`{
   "code":0,
   "data":{
       "files":[],
       "page_token": "",
       "has_more":false
   },
   "msg":"success"
}`)
				// 因为第一个mock响应has_more=true产生递归调用，所以要mock第三次
				// 第三次响应一个服务端的错误
				gock.New("https://open.feishu.cn").
					Get("/open-apis/wiki/v2/spaces/space_id/nodes").
					PathParam("spaces", spaceID).
					MatchParams(map[string]string{
						"parent_node_token": parentNodeToken,
						"page_token":        "6946843325487906839",
						"page_size":         "50",
					}).
					Reply(500).
					JSON(`{"code":500,"msg":"failed"}`)
			},
			teardownMock: func(name string) {
				defer gock.Off()
				s.True(gock.IsDone(), name)
			},
			want: &DocumentNode{
				DocumentInfo: DocumentInfo{
					Token:            "boxbc0dGSMu23m7QkC1bvabcef",
					Name:             "Name",
					Type:             constant.DocTypeDocx,
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
							Name:             "标题",
							Type:             "doc",
							Token:            "doccnzAaODxxxxxxWabcdef",
							FileExtension:    "docx",
							CanDownload:      true,
							DownloadDirectly: false,
							URL:              "",
							NodeToken:        "wikcnKQ1k3pxxxxxx8Vabcef",
							SpaceID:          "6946843325487912356",
							FilePath:         "",
						},
					},
				},
			},
			wantError: "logId: \x1b]8;;https://open.feishu.cn/search?q=\x1b\\, error response: \n{\n  Code: 500,\n  Msg: \"failed\"\n}",
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock(tt.name, tt.dn, tt.hasChild, tt.spaceID, tt.parentNodeToken, tt.pageToken)
			err := s.client.fetchWikiDescendant(tt.dn, tt.hasChild, tt.spaceID, tt.parentNodeToken, tt.pageToken)
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
			}
			s.Equal(tt.want, tt.dn, tt.name)
		})
	}
}

func (s *DocumentWikiTestSuite) TestClientImpl_wikiNodeToDocumentNode() {
	type args struct {
		node *larkwiki.Node
		args *argument.Args
	}
	tests := []struct {
		name string
		args args
		want *DocumentNode
	}{
		{
			name: "转换OK",
			args: args{
				node: &larkwiki.Node{
					SpaceId:         larkcore.StringPtr("SpaceId"),
					NodeToken:       larkcore.StringPtr("NodeToken"),
					ObjToken:        larkcore.StringPtr("ObjToken"),
					ObjType:         larkcore.StringPtr(string(constant.DocTypeDocx)),
					ParentNodeToken: larkcore.StringPtr("ParentNodeToken"),
					NodeType:        larkcore.StringPtr("NodeType"),
					OriginNodeToken: larkcore.StringPtr("OriginNodeToken"),
					OriginSpaceId:   larkcore.StringPtr("OriginSpaceId"),
					HasChild:        larkcore.BoolPtr(false),
					Title:           larkcore.StringPtr("Title"),
					ObjCreateTime:   larkcore.StringPtr("ObjCreateTime"),
					ObjEditTime:     larkcore.StringPtr("ObjEditTime"),
					NodeCreateTime:  larkcore.StringPtr("NodeCreateTime"),
					Creator:         larkcore.StringPtr("Creator"),
					Owner:           larkcore.StringPtr("Owner"),
					NodeCreator:     larkcore.StringPtr("NodeCreator"),
				},
				args: &argument.Args{
					FileExtensions: map[constant.DocType]constant.FileExt{
						constant.DocTypeDocx: constant.FileExtPDF,
					},
				},
			},
			want: &DocumentNode{
				DocumentInfo: DocumentInfo{
					Token:            "ObjToken",
					Name:             "Title",
					Type:             constant.DocTypeDocx,
					URL:              "",
					FileExtension:    constant.FileExtPDF,
					FilePath:         "",
					CanDownload:      true,
					DownloadDirectly: false,
					NodeToken:        "NodeToken",
					SpaceID:          "SpaceId",
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
			actual := c.wikiNodeToDocumentNode(tt.args.node)
			s.Equal(tt.want, actual, tt.name)
		})
	}
}
