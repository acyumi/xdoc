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

package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/h2non/gock"
	"github.com/samber/oops"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"

	"github.com/acyumi/xdoc/component/app"
	"github.com/acyumi/xdoc/component/argument"
	"github.com/acyumi/xdoc/component/constant"
	"github.com/acyumi/xdoc/component/feishu"
)

// 注册测试套件。
func TestExportFeishuSuite(t *testing.T) {
	suite.Run(t, new(ExportFeishuTestSuite))
}

type ExportFeishuTestSuite struct {
	suite.Suite
	memFs   *afero.Afero
	TempDir string
}

func (s *ExportFeishuTestSuite) SetupSuite() {
	app.Fs = &afero.Afero{Fs: afero.NewMemMapFs()}
	s.memFs = app.Fs
}

func (s *ExportFeishuTestSuite) SetupTest() {
	s.TempDir = s.T().TempDir()
}

func (s *ExportFeishuTestSuite) TearDownTest() {

}

func (s *ExportFeishuTestSuite) TestExecute() {
	tests := []struct {
		name          string
		configFile    string
		configContent []byte
		args          []string
		setupMock     func(name string, cmd *exportFeishuCommand)
		teardownMock  func(name string, cmd *exportFeishuCommand)
		wantArgs      *feishu.Args
		wantError     string
		wantCode      string
	}{
		{
			name:          "纯命令参数执行",
			configFile:    "",
			configContent: nil,
			args: []string{
				"--app-id", "xx",
				"--app-secret", "yy",
				"--urls", "https://invalid.cn/docs/xxx",
				"--dir", "/tmp",
				// "--list-only", "true", 移到了export命令中
				"--quit-automatically", "true",
				"--ext", "docx=docx,doc=docx",
			},
			wantArgs: &feishu.Args{
				Args: &argument.Args{
					ConfigFile: func() string {
						exePath, err := app.Executable()
						s.Require().NoError(err, "获取程序所在目录失败")
						exeDir := filepath.Dir(exePath)
						return filepath.Join(exeDir, flagNameConfig+".yaml")
					}(),
					QuitAutomatically: true,
				},
				Enabled:   true,
				AppID:     "xx",
				AppSecret: "yy",
				DocURLs:   []string{"https://invalid.cn/docs/xxx"},
				SaveDir:   filepath.Clean("/tmp"),
				FileExtensions: map[constant.DocType]constant.FileExt{
					constant.DocTypeDocx: constant.FileExtDocx,
					constant.DocTypeDoc:  constant.FileExtDocx,
				},
				ListOnly: false,
			},
			wantError: "不支持的文档来源域名: invalid.cn",
		},
		{
			name:       "指定有效配置文件",
			configFile: filepath.Join(s.TempDir, "test.yaml"),
			configContent: []byte(`
quit-automatically: true
export:
  list-only: true
  feishu:
    app-id: "xx"
    app-secret: "yy"
    urls: ["https://invalid.cn/docs/xxx"]
    dir: "/tmp"
    file:
      extensions:
        docx: "docx"
        doc: "docx"
`),
			args: []string{"--config", filepath.Join(s.TempDir, "test.yaml")},
			wantArgs: &feishu.Args{
				Args: &argument.Args{
					ConfigFile:        filepath.Join(s.TempDir, "test.yaml"),
					QuitAutomatically: true,
				},
				Enabled:   true,
				AppID:     "xx",
				AppSecret: "yy",
				DocURLs:   []string{"https://invalid.cn/docs/xxx"},
				SaveDir:   filepath.Clean("/tmp"),
				FileExtensions: map[constant.DocType]constant.FileExt{
					constant.DocTypeDocx: constant.FileExtDocx,
					constant.DocTypeDoc:  constant.FileExtDocx,
				},
				ListOnly: true,
			},
			wantError: "不支持的文档来源域名: invalid.cn",
		},
		{
			name:       "默认配置文件",
			configFile: "",
			configContent: []byte(`
quit-automatically: true
export:
  list-only: true
  feishu:
    app-id: "xx"
    app-secret: "yy"
    urls: ["https://invalid.cn/docs/xxx"]
    dir: "/tmp"
    file:
      extensions:
        docx: "docx"
        doc: "docx"
`),
			wantArgs: &feishu.Args{
				Args: &argument.Args{
					ConfigFile: func() string {
						exePath, err := app.Executable()
						s.Require().NoError(err, "获取程序所在目录失败")
						exeDir := filepath.Dir(exePath)
						return filepath.Join(exeDir, flagNameConfig+".yaml")
					}(),
					QuitAutomatically: true,
				},
				Enabled:   true,
				AppID:     "xx",
				AppSecret: "yy",
				DocURLs:   []string{"https://invalid.cn/docs/xxx"},
				SaveDir:   filepath.Clean("/tmp"),
				FileExtensions: map[constant.DocType]constant.FileExt{
					constant.DocTypeDocx: constant.FileExtDocx,
					constant.DocTypeDoc:  constant.FileExtDocx,
				},
				ListOnly: true,
			},
			wantError: "不支持的文档来源域名: invalid.cn",
		},
		{
			name:       "不支持的schema",
			configFile: "",
			configContent: []byte(`
quit-automatically: true
export:
  list-only: true
  feishu:
    app-id: "xx"
    app-secret: "yy"
    urls: ["feishu://invalid.cn/docs/xxx"]
    dir: "/tmp"
    file:
      extensions:
        docx: "docx"
        doc: "docx"
`),
			wantArgs: &feishu.Args{
				Args: &argument.Args{
					ConfigFile: func() string {
						exePath, err := app.Executable()
						s.Require().NoError(err, "获取程序所在目录失败")
						exeDir := filepath.Dir(exePath)
						return filepath.Join(exeDir, flagNameConfig+".yaml")
					}(),
					QuitAutomatically: true,
				},
				Enabled:   true,
				AppID:     "xx",
				AppSecret: "yy",
				DocURLs:   []string{"feishu://invalid.cn/docs/xxx"},
				SaveDir:   filepath.Clean("/tmp"),
				FileExtensions: map[constant.DocType]constant.FileExt{
					constant.DocTypeDocx: constant.FileExtDocx,
					constant.DocTypeDoc:  constant.FileExtDocx,
				},
				ListOnly: true,
			},
			wantError: "url地址必须是http://或https://开头",
			wantCode:  "BadRequest",
		},
		{
			name:       "不支持的ext配置",
			configFile: "",
			configContent: []byte(`
quit-automatically: true
export:
  list-only: true
  feishu:
    app-id: "xx"
    app-secret: "yy"
    urls: ["https://invalid.cn/docs/xxx"]
    dir: "/tmp"
    file:
      extensions: 
        docx: "docx"
        doc: "docx"
`),
			args: []string{"--ext", "xxx"},
			wantArgs: &feishu.Args{
				Args: &argument.Args{
					ConfigFile:        "",
					QuitAutomatically: false,
				},
				Enabled:        false,
				AppID:          "",
				AppSecret:      "",
				DocURLs:        nil,
				SaveDir:        "",
				FileExtensions: nil,
				ListOnly:       false,
			},
			wantError: "invalid argument \"xxx\" for \"--ext\" flag: xxx must be formatted as key=value",
			wantCode:  "",
		},
		{
			name:       "配置文件不存在",
			configFile: "nonexistent.yaml",
			args:       []string{"--config", "nonexistent.yaml"},
			wantArgs: &feishu.Args{
				Args: &argument.Args{
					ConfigFile:        "nonexistent.yaml",
					QuitAutomatically: false,
				},
				Enabled:        true,
				AppID:          "",
				AppSecret:      "",
				DocURLs:        []string{},
				SaveDir:        ".",
				FileExtensions: map[constant.DocType]constant.FileExt{},
				ListOnly:       false,
			},
			wantError: "AppID: app-id是必需参数; AppSecret: app-secret是必需参数; DocURLs: urls是必需参数.",
			wantCode:  "InvalidArgument",
		},
		{
			name:          "配置文件类型不对",
			configFile:    filepath.Join(s.TempDir, "ttt.exe"),
			configContent: []byte(`xxx`),
			args:          []string{"--config", filepath.Join(s.TempDir, "ttt.exe")},
			wantArgs: &feishu.Args{
				Args: &argument.Args{
					ConfigFile:        filepath.Join(s.TempDir, "ttt.exe"),
					QuitAutomatically: false,
				},
				Enabled:        false,
				AppID:          "",
				AppSecret:      "",
				DocURLs:        nil,
				SaveDir:        "",
				FileExtensions: nil,
				ListOnly:       false,
			},
			wantError: "加载配置文件失败: Unsupported Config Type \"exe\"",
			wantCode:  "",
		},
		{
			name:       "文档地址不是同一域名",
			configFile: "/tmp/local.yaml",
			configContent: []byte(`
quit-automatically: true
export:
  list-only: true
  feishu:
    app-id: "xx"
    app-secret: "yy"
    urls:
      - "https://invalid.cn/docs/xxx"
      - "https://xxxyyy.feishu.cn/docs/xxx"
    dir: "/tmp"
    file:
      extensions: 
        docx: "docx"
        doc: "docx"
`),
			args: []string{"--config", "/tmp/local.yaml"},
			wantArgs: &feishu.Args{
				Args: &argument.Args{
					ConfigFile:        "/tmp/local.yaml",
					QuitAutomatically: true,
				},
				Enabled:   true,
				AppID:     "xx",
				AppSecret: "yy",
				DocURLs: []string{
					"https://invalid.cn/docs/xxx",
					"https://xxxyyy.feishu.cn/docs/xxx",
				},
				SaveDir:  filepath.Clean("/tmp"),
				ListOnly: true,
				FileExtensions: map[constant.DocType]constant.FileExt{
					"docx": "docx",
					"doc":  "docx",
				},
			},
			wantError: "文档地址不匹配, 请确保所有文档地址都是同一域名",
			wantCode:  "",
		},
		{
			name:       "执行下载报错",
			configFile: "/tmp/local.yaml",
			configContent: []byte(`
quit-automatically: true
export:
  list-only: true
  feishu:
    app-id: "xx"
    app-secret: "yy"
    urls:
      - "https://xxxyyy.feishu.cn/docs/xxx"
    dir: "/tmp"
    file:
      extensions: 
        docx: "docx"
        doc: "docx"
`),
			args: []string{"--config", "/tmp/local.yaml"},
			wantArgs: &feishu.Args{
				Args: &argument.Args{
					ConfigFile:        "/tmp/local.yaml",
					QuitAutomatically: true,
				},
				Enabled:   true,
				AppID:     "xx",
				AppSecret: "yy",
				DocURLs: []string{
					"https://xxxyyy.feishu.cn/docs/xxx",
				},
				SaveDir:  filepath.Clean("/tmp"),
				ListOnly: true,
				FileExtensions: map[constant.DocType]constant.FileExt{
					"docx": "docx",
					"doc":  "docx",
				},
			},
			setupMock: func(name string, cmd *exportFeishuCommand) {
				// 模拟获取 tenant_access_token 的响应
				gock.New("https://open.feishu.cn").
					Post("/open-apis/auth/v3/tenant_access_token/internal").
					Reply(500).
					JSON(`{"code":500,"msg":"模拟请求失败"}`)
			},
			teardownMock: func(name string, cmd *exportFeishuCommand) {
				gock.Off()
				s.True(gock.IsDone(), name)
			},
			wantError: "msg:模拟请求失败,code:500",
			wantCode:  "",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tempFile := tt.configFile
			if tempFile == "" && tt.configContent != nil {
				exePath, err := app.Executable()
				s.Require().NoError(err, "获取程序所在目录失败")
				exeDir := filepath.Dir(exePath)
				tempFile = filepath.Join(exeDir, flagNameConfig+".yaml")
			}
			if tempFile != "" && tt.configContent != nil {
				err := app.Fs.WriteFile(tempFile, tt.configContent, 0644)
				s.Require().NoError(err, "创建测试配置文件失败")
			}
			vip := app.NewViper()
			args := &argument.Args{}
			root := &XdocCommand{}
			sub := &exportFeishuCommand{}
			if tt.setupMock != nil {
				tt.setupMock(tt.name, sub)
			}
			for _, cmd := range []command{root, sub} {
				cmd.init(vip, args)
				err := cmd.bind()
				s.Require().NoError(err, tt.name)
				c := cmd.get()
				if root.get() != c {
					root.AddCommand(c)
				}
			}
			_ = sub.children()
			// 这里不能传 nil，因为这会让 cobra 取了 IDE 的参数影响单测，如 Error: unknown shorthand flag: 't' in -testify.m
			if tt.args == nil {
				tt.args = []string{}
			}
			// sub.Execute() 内会递归到根命令再执行，跟执行 root.Execute() 的效果是一样的，但是参数要从 root 那里传递
			tt.args = append([]string{"feishu"}, tt.args...)
			root.SetArgs(tt.args)
			err := sub.Execute()
			if tt.teardownMock != nil {
				tt.teardownMock(tt.name, sub)
			}
			if err != nil || tt.wantError != "" {
				s.Require().Error(err, tt.name)
				s.Require().EqualError(err, tt.wantError, tt.name)
				var oe oops.OopsError
				if ok := errors.As(err, &oe); ok {
					s.Equal(tt.wantCode, oe.Code(), tt.name)
				}
			} else {
				s.Require().NoError(err, tt.name)
			}
			tt.wantArgs.StartTime = sub.args.StartTime
			s.Equal(tt.wantArgs, sub.args, tt.name)
			yes, err := app.Fs.Exists(tempFile)
			s.Require().NoError(err, tt.name)
			if yes {
				err = app.Fs.Remove(tempFile)
				s.Require().NoError(err, tt.name)
			}
		})
	}
}

func (s *ExportFeishuTestSuite) Test_exec() {
	tests := []struct {
		name         string
		cmd          *exportFeishuCommand
		setupMock    func(name string, cmd *exportFeishuCommand)
		teardownMock func(name string, cmd *exportFeishuCommand)
		wantArgs     *feishu.Args
		wantError    string
		wantCode     string
	}{
		{
			name: "不支持的ext配置",
			cmd:  &exportFeishuCommand{},
			setupMock: func(name string, cmd *exportFeishuCommand) {
				cmd.vip.Set(commandNameExport+"."+flagNameListOnly, true)
				cmd.vip.Set(viperKeyFeishuEnabled, true)
				cmd.vip.Set(getFlagName(flagNameAppID), "xxx")
				cmd.vip.Set(getFlagName(flagNameAppSecret), "yyy")
				cmd.vip.Set(getFlagName(flagNameURLs), []string{"https://xxx.feishu.cn/docs/xxx"})
				cmd.vip.Set(getFlagName(flagNameDir), "/tmp")
				var zzz = "zzz"
				cmd.Flag(flagNameExt).Value = newStringValue(zzz, &zzz)
			},
			wantArgs: &feishu.Args{
				Args: &argument.Args{
					ConfigFile:        "",
					QuitAutomatically: false,
				},
				Enabled:        true,
				AppID:          "xxx",
				AppSecret:      "yyy",
				DocURLs:        []string{"https://xxx.feishu.cn/docs/xxx"},
				SaveDir:        filepath.Clean("/tmp"),
				FileExtensions: map[constant.DocType]constant.FileExt{},
				ListOnly:       true,
			},
			wantError: "trying to get stringToString value of flag of type string",
			wantCode:  "",
		},
		{
			name: "模拟文件系统Exists失败",
			cmd:  &exportFeishuCommand{},
			setupMock: func(name string, cmd *exportFeishuCommand) {
				cmd.vip.Set(commandNameExport+"."+flagNameListOnly, true)
				cmd.vip.Set(viperKeyFeishuEnabled, true)
				cmd.vip.Set(getFlagName(flagNameAppID), "xxx")
				cmd.vip.Set(getFlagName(flagNameAppSecret), "yyy")
				cmd.vip.Set(getFlagName(flagNameURLs), []string{"https://xxx.feishu.cn/docs/xxx"})
				cmd.vip.Set(getFlagName(flagNameDir), "/tmp")
				app.Fs = &afero.Afero{Fs: mockFs{}}
			},
			teardownMock: func(name string, cmd *exportFeishuCommand) {
				app.Fs = s.memFs
			},
			wantArgs: &feishu.Args{
				Args: &argument.Args{
					ConfigFile:        "",
					QuitAutomatically: false,
				},
				Enabled:        true,
				AppID:          "xxx",
				AppSecret:      "yyy",
				DocURLs:        []string{"https://xxx.feishu.cn/docs/xxx"},
				SaveDir:        filepath.Clean("/tmp"),
				FileExtensions: map[constant.DocType]constant.FileExt{},
				ListOnly:       true,
			},
			wantError: "模拟Stat执行失败",
			wantCode:  "",
		},
		{
			name: "silence.test",
			cmd:  &exportFeishuCommand{},
			setupMock: func(name string, cmd *exportFeishuCommand) {
				cmd.vip.Set(commandNameExport+"."+flagNameListOnly, true)
				cmd.vip.Set(viperKeyFeishuEnabled, true)
				cmd.vip.Set(getFlagName(flagNameAppID), "xxx")
				cmd.vip.Set(getFlagName(flagNameAppSecret), "yyy")
				cmd.vip.Set(getFlagName(flagNameURLs), []string{"https://silence.test/docs/xxx"})
				cmd.vip.Set(getFlagName(flagNameDir), "/tmp")
			},
			teardownMock: func(name string, cmd *exportFeishuCommand) {
				app.Fs = s.memFs
			},
			wantArgs: &feishu.Args{
				Args: &argument.Args{
					ConfigFile:        "",
					QuitAutomatically: false,
				},
				Enabled:        true,
				AppID:          "xxx",
				AppSecret:      "yyy",
				DocURLs:        []string{"https://silence.test/docs/xxx"},
				SaveDir:        filepath.Clean("/tmp"),
				FileExtensions: map[constant.DocType]constant.FileExt{},
				ListOnly:       true,
			},
			wantError: "",
			wantCode:  "",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			vip := app.NewViper()
			args := &argument.Args{}
			tt.cmd.init(vip, args)
			err := tt.cmd.bind()
			s.Require().NoError(err, tt.name)
			if tt.setupMock != nil {
				tt.setupMock(tt.name, tt.cmd)
			}
			err = tt.cmd.exec()
			if tt.teardownMock != nil {
				tt.teardownMock(tt.name, tt.cmd)
			}
			if err != nil || tt.wantError != "" {
				s.Require().Error(err, tt.name)
				s.Require().EqualError(err, tt.wantError, tt.name)
				var oe oops.OopsError
				if ok := errors.As(err, &oe); ok {
					s.Equal(tt.wantCode, oe.Code(), tt.name)
				}
			} else {
				s.Require().NoError(err, tt.name)
			}
			tt.wantArgs.StartTime = tt.cmd.args.StartTime
			s.Equal(tt.wantArgs, tt.cmd.args, tt.name)
		})
	}
}

type mockFs struct {
	afero.Fs
}

func (m mockFs) Stat(_ string) (os.FileInfo, error) {
	return nil, errors.New("模拟Stat执行失败")
}

// 测试newCloudClient。
func (s *ExportFeishuTestSuite) Test_doExport() {
	tests := []struct {
		name      string
		host      string
		args      *feishu.Args
		wantError string
		wantCode  string
	}{
		{
			name: "飞书客户端",
			host: "sample.feishu.cn",
			args: &feishu.Args{
				Args: &argument.Args{
					StartTime: time.Now(),
				},
				AppID:     "1111",
				AppSecret: "2222",
			},
			wantError: "Client: Args: DocURLs: urls是必需参数; SaveDir: dir是必需参数..; Docs: cannot be blank.",
			wantCode:  "InvalidArgument",
		},
		{
			name: "进度条UI测试客户端",
			host: "progress.test",
			args: &feishu.Args{
				AppID:     "1111",
				AppSecret: "2222",
			},
			wantError: "文档源为空",
		},
		{
			name:      "不支持",
			host:      "invalid.com",
			wantError: "不支持的文档来源域名: invalid.com",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := doExport(tt.args, tt.host, nil)
			if tt.wantError != "" || err != nil {
				s.Require().Error(err, tt.name)
				s.IsType(oops.OopsError{}, err, tt.name)
				var actualError oops.OopsError
				yes := errors.As(err, &actualError)
				s.Require().True(yes, tt.name)
				s.Equal(tt.wantCode, actualError.Code(), tt.name)
				s.Equal(tt.wantError, actualError.Error(), tt.name)
				return
			}
			s.Require().NoError(err, tt.name)
		})
	}
}
