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
	"strings"
	"testing"

	"github.com/samber/oops"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"

	"github.com/acyumi/xdoc/component/app"
	"github.com/acyumi/xdoc/component/argument"
)

// 注册测试套件。
func TestExporterSuite(t *testing.T) {
	suite.Run(t, new(ExporterTestSuite))
}

type ExporterTestSuite struct {
	suite.Suite
	cachedOsArgs []string
}

func (s *ExporterTestSuite) SetupSuite() {
	app.Fs = &afero.Afero{Fs: afero.NewMemMapFs()}
	s.cachedOsArgs = os.Args
}

func (s *ExporterTestSuite) SetupTest() {}

func (s *ExporterTestSuite) TearDownTest() {

}

func (s *ExporterTestSuite) Test_exportCommand_Execute() {
	tests := []struct {
		name           string
		configFile     string
		configContent  []byte
		args           []string
		setupMock      func(name string, cmd *exportCommand, args []string)
		teardownMock   func(name string, cmd *exportCommand, args []string)
		wantConfigFile string
		wantError      string
		wantCode       string
	}{
		{
			name:          "纯命令参数执行，不支持--app-id",
			configFile:    "",
			configContent: nil,
			args: []string{
				"--config", "config.yaml",
				"--app-id", "xx",
			},
			wantConfigFile: "",
			wantError:      "unknown flag: --app-id",
		},
		{
			name:          "绑定匹配子命令feishu",
			configFile:    "",
			configContent: nil,
			args:          []string{"feishu"},
			setupMock: func(name string, cmd *exportCommand, args []string) {
				os.Args = []string{"xdoc", "export", "feishu"}
				cmd.subs = []command{
					NewMockCommand(s.T()),
				}
				children := cmd.children()
				firstChild := children[0]
				mc := firstChild.(*MockCommand)
				mc.EXPECT().get().Return(&cobra.Command{Use: commandNameFeishu}).Once()
				mc.EXPECT().exec().Return(nil).Once()
			},
			teardownMock: func(name string, cmd *exportCommand, args []string) {
				os.Args = s.cachedOsArgs
			},
			wantConfigFile: "",
			wantError:      "",
		},
		{
			name:          "执行测试子命令xxx",
			configFile:    "",
			configContent: nil,
			args:          []string{"xxx"},
			setupMock: func(name string, cmd *exportCommand, args []string) {
				os.Args = []string{"xdoc", "export", "xxx"}
				cmd.subs = []command{
					NewMockCommand(s.T()),
				}
				children := cmd.children()
				firstChild := children[0]
				mc := firstChild.(*MockCommand)
				mc.EXPECT().get().Return(&cobra.Command{Use: "xxx"}).Once()
				mc.EXPECT().exec().Return(errors.New("执行了xxx")).Once()
			},
			teardownMock: func(name string, cmd *exportCommand, args []string) {
				os.Args = s.cachedOsArgs
			},
			wantConfigFile: "",
			wantError:      "执行了xxx",
		},
		{
			name:          "测试未找到子命令xxx",
			configFile:    "",
			configContent: nil,
			args:          []string{"xxx"},
			setupMock: func(name string, cmd *exportCommand, args []string) {
				os.Args = []string{"xdoc", "export", "xxx"}
				cmd.subs = []command{
					NewMockCommand(s.T()),
				}
				children := cmd.children()
				firstChild := children[0]
				mc := firstChild.(*MockCommand)
				mc.EXPECT().get().Return(&cobra.Command{Use: "yyy"}).Once()
			},
			teardownMock: func(name string, cmd *exportCommand, args []string) {
				os.Args = s.cachedOsArgs
			},
			wantConfigFile: "",
			wantError:      "未找到export下的子命令: xxx\n",
			wantCode:       "InvalidArgument",
		},
		{
			name:       "指定有效配置文件，打开开关",
			configFile: "/tmp/test.yaml",
			configContent: []byte(`
verbose: true
generate-config: false
quit-automatically: true
export:
  list-only: true
  feishu:
    enabled: true
`),
			args: []string{"--config", "/tmp/test.yaml"},
			setupMock: func(name string, cmd *exportCommand, args []string) {
				cmd.subs = []command{
					NewMockCommand(s.T()),
				}
				children := cmd.children()
				firstChild := children[0]
				mc := firstChild.(*MockCommand)
				mc.EXPECT().get().Return(&cobra.Command{Use: commandNameFeishu}).Once()
				mc.EXPECT().exec().Return(errors.New("执行了feishu")).Once()
			},
			wantConfigFile: "/tmp/test.yaml",
			wantError:      "执行了feishu",
		},
		{
			name:       "指定有效配置文件，关闭开关",
			configFile: "/tmp/test.yaml",
			configContent: []byte(`
verbose: true
generate-config: false
quit-automatically: true
export:
  list-only: true
  feishu:
    enabled: false
`),
			args: []string{"--config", "/tmp/test.yaml"},
			setupMock: func(name string, cmd *exportCommand, args []string) {
			},
			wantConfigFile: "/tmp/test.yaml",
			wantError:      "",
		},
		{
			name:       "默认配置文件",
			configFile: "",
			configContent: []byte(`
verbose: true
generate-config: false
quit-automatically: true
export:
  list-only: true
  feishu:
    enabled: true
`),
			setupMock: func(name string, cmd *exportCommand, args []string) {
				cmd.subs = []command{
					NewMockCommand(s.T()),
				}
				children := cmd.children()
				firstChild := children[0]
				mc := firstChild.(*MockCommand)
				mc.EXPECT().get().Return(&cobra.Command{Use: commandNameFeishu}).Once()
				mc.EXPECT().exec().Return(errors.New("执行了feishu")).Once()
			},
			wantConfigFile: func() string {
				exePath, err := app.Executable()
				s.Require().NoError(err, "获取程序所在目录失败")
				exeDir := filepath.Dir(exePath)
				return filepath.Join(exeDir, flagNameConfig+".yaml")
			}(),
			wantError: "执行了feishu",
		},
		{
			name:           "配置文件类型不对",
			configFile:     "/tmp/ttt.exe",
			configContent:  []byte(`xxx`),
			args:           []string{"--config", "/tmp/ttt.exe"},
			wantConfigFile: "/tmp/ttt.exe",
			wantError:      "加载配置文件失败: Unsupported Config Type \"exe\"",
			wantCode:       "",
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
			export := &exportCommand{}
			if tt.setupMock != nil {
				tt.setupMock(tt.name, export, tt.args)
			}
			for _, cmd := range []command{root, export} {
				cmd.init(vip, args)
				err := cmd.bind()
				s.Require().NoError(err, tt.name)
				c := cmd.get()
				if root.get() != c {
					root.AddCommand(c)
				}
			}
			// 这里不能传 nil，因为这会让 cobra 取了 IDE 的参数影响单测，如 Error: unknown shorthand flag: 't' in -testify.m
			if tt.args == nil {
				tt.args = []string{}
			}
			// export.Execute() 内会递归到根命令再执行，跟执行 root.Execute() 的效果是一样的，但是参数要从 root 那里传递
			tt.args = append([]string{"export"}, tt.args...)
			root.SetArgs(tt.args)
			err := export.Execute()
			if tt.teardownMock != nil {
				tt.teardownMock(tt.name, export, tt.args)
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
			s.Equal(tt.wantConfigFile, tempFile, tt.name)
			yes, err := app.Fs.Exists(tempFile)
			s.Require().NoError(err, tt.name)
			if yes {
				err = app.Fs.Remove(tempFile)
				s.Require().NoError(err, tt.name)
			}
		})
	}
}

func (s *ExporterTestSuite) Test_exportCommand_exec() {
	tests := []struct {
		name         string
		cmd          *exportCommand
		setupMock    func(name string, cmd *exportCommand)
		teardownMock func(name string, cmd *exportCommand)
		wantError    string
		wantCode     string
	}{
		{
			name: "subCommand为空",
			cmd: &exportCommand{
				vip:        app.NewViper(),
				subCommand: "",
			},
			setupMock:    func(name string, cmd *exportCommand) {},
			teardownMock: func(name string, cmd *exportCommand) {},
			wantError:    "pflag: help requested",
			wantCode:     "",
		},
		{
			name: "未找到export下的子命令",
			cmd: &exportCommand{
				vip:        app.NewViper(),
				subCommand: "xxx",
			},
			setupMock: func(name string, cmd *exportCommand) {
				children := cmd.children()
				firstChild := children[0]
				firstChild.init(nil, nil)
			},
			teardownMock: func(name string, cmd *exportCommand) {},
			wantError:    "未找到export下的子命令: xxx\n",
			wantCode:     "InvalidArgument",
		},
		{
			name: "执行export下的子命令",
			cmd: &exportCommand{
				subCommand: "xxx",
				subs: []command{
					NewMockCommand(s.T()),
				},
			},
			setupMock: func(name string, cmd *exportCommand) {
				children := cmd.children()
				firstChild := children[0]
				mc := firstChild.(*MockCommand)
				mc.EXPECT().get().Return(&cobra.Command{Use: "xxx"}).Once()
				mc.EXPECT().exec().Return(errors.New("执行了xxx")).Once()
			},
			teardownMock: func(name string, cmd *exportCommand) {},
			wantError:    "执行了xxx",
			wantCode:     "",
		},
		{
			name: "通过环境变量指定export下的feishu子命令",
			cmd: &exportCommand{
				vip: app.NewViper(),
				subs: []command{
					NewMockCommand(s.T()),
				},
			},
			setupMock: func(name string, cmd *exportCommand) {
				cmd.vip.SetEnvPrefix("XDOC")
				cmd.vip.AutomaticEnv()
				cmd.vip.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
				s.T().Setenv("XDOC_EXPORT_FEISHU_ENABLED", "true")
				children := cmd.children()
				firstChild := children[0]
				mc := firstChild.(*MockCommand)
				mc.EXPECT().get().Return(&cobra.Command{Use: commandNameFeishu}).Once()
				mc.EXPECT().exec().Return(errors.New("执行了feishu")).Once()
			},
			teardownMock: func(name string, cmd *exportCommand) {
			},
			wantError: "执行了feishu",
			wantCode:  "",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock(tt.name, tt.cmd)
			err := tt.cmd.exec()
			tt.teardownMock(tt.name, tt.cmd)
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
		})
	}
}

func (s *ExporterTestSuite) Test_analysisURL() {
	// 文件夹 folder_token： https://sample.feishu.cn/drive/folder/cSJe2JgtFFBwRuTKAJK6baNGUn0
	// 文件 file_token：https://sample.feishu.cn/file/ndqUw1kpjnGNNaegyqDyoQDCLx1
	// 文档 doc_token：https://sample.feishu.cn/docs/2olt0Ts4Mds7j7iqzdwrqEUnO7q
	// 新版文档 document_id：https://sample.feishu.cn/docx/UXEAd6cRUoj5pexJZr0cdwaFnpd
	// 电子表格 spreadsheet_token：https://sample.feishu.cn/sheets/MRLOWBf6J47ZUjmwYRsN8utLEoY
	// 多维表格 app_token：https://sample.feishu.cn/base/Pc9OpwAV4nLdU7lTy71t6Kmmkoz
	// 知识空间 space_id：https://sample.feishu.cn/wiki/settings/7075377271827264924（需要知识库管理员在设置页面获取该地址）
	// 知识库节点 node_token：https://sample.feishu.cn/wiki/sZdeQp3m4nFGzwqR5vx4vZksMoe
	type args struct {
		docURL string
	}
	tests := []struct {
		name      string
		args      args
		wantHost  string
		wantType  string
		wantToken string
		wantError string
		wantCode  string
	}{
		{
			name: "url非法0",
			args: args{
				docURL: "+%%",
			},
			wantHost:  "",
			wantType:  "",
			wantToken: "",
			wantError: `解析url地址失败：+%%: parse "+%%": invalid URL escape "%%"`,
			wantCode:  "BadRequest",
		},
		{
			name: "url非法1",
			args: args{
				docURL: "https://sample.feishu.cn",
			},
			wantHost:  "sample.feishu.cn",
			wantType:  "",
			wantToken: "",
			wantError: "url地址的path部分至少包含两段才能解析出云文档类型和token，如:/docs/2olt0Ts4Mds7j7iqzdwrqEUnO7q",
			wantCode:  "BadRequest",
		},
		{
			name: "url非法2",
			args: args{
				docURL: "https://sample.feishu.cn/",
			},
			wantHost:  "sample.feishu.cn",
			wantType:  "",
			wantToken: "",
			wantError: "url地址的path部分至少包含两段才能解析出云文档类型和token，如:/docs/2olt0Ts4Mds7j7iqzdwrqEUnO7q",
			wantCode:  "BadRequest",
		},
		{
			name: "url非法3",
			args: args{
				docURL: "https://sample.feishu.cn/cSJe2JgtFFBwRuTKAJK6baNGUn0",
			},
			wantHost:  "sample.feishu.cn",
			wantType:  "",
			wantToken: "",
			wantError: "url地址的path部分至少包含两段才能解析出云文档类型和token，如:/docs/2olt0Ts4Mds7j7iqzdwrqEUnO7q",
			wantCode:  "BadRequest",
		},
		{
			name: "url非法4",
			args: args{
				docURL: "xxx://sample.feishu.cn/cSJe2JgtFFBwRuTKAJK6baNGUn0",
			},
			wantHost:  "",
			wantType:  "",
			wantToken: "",
			wantError: "url地址必须是http://或https://开头",
			wantCode:  "BadRequest",
		},
		{
			name: "文件夹",
			args: args{
				docURL: "https://sample.feishu.cn/drive/folder/cSJe2JgtFFBwRuTKAJK6baNGUn0",
			},
			wantHost:  "sample.feishu.cn",
			wantType:  "/drive/folder",
			wantToken: "cSJe2JgtFFBwRuTKAJK6baNGUn0",
			wantError: "",
		},
		{
			name: "文件",
			args: args{
				docURL: "https://sample.feishu.cn/file/ndqUw1kpjnGNNaegyqDyoQDCLx1",
			},
			wantHost:  "sample.feishu.cn",
			wantType:  "/file",
			wantToken: "ndqUw1kpjnGNNaegyqDyoQDCLx1",
			wantError: "",
		},
		{
			name: "文档",
			args: args{
				docURL: "https://sample.feishu.cn/docs/2olt0Ts4Mds7j7iqzdwrqEUnO7q",
			},
			wantHost:  "sample.feishu.cn",
			wantType:  "/docs",
			wantToken: "2olt0Ts4Mds7j7iqzdwrqEUnO7q",
			wantError: "",
		},
		{
			name: "新版文档",
			args: args{
				docURL: "https://sample.feishu.cn/docx/UXEAd6cRUoj5pexJZr0cdwaFnpd",
			},
			wantHost:  "sample.feishu.cn",
			wantType:  "/docx",
			wantToken: "UXEAd6cRUoj5pexJZr0cdwaFnpd",
			wantError: "",
		},
		{
			name: "电子表格",
			args: args{
				docURL: "https://sample.feishu.cn/sheets/MRLOWBf6J47ZUjmwYRsN8utLEoY",
			},
			wantHost:  "sample.feishu.cn",
			wantType:  "/sheets",
			wantToken: "MRLOWBf6J47ZUjmwYRsN8utLEoY",
			wantError: "",
		},
		{
			name: "多维表格",
			args: args{
				docURL: "https://sample.feishu.cn/base/Pc9OpwAV4nLdU7lTy71t6Kmmkoz",
			},
			wantHost:  "sample.feishu.cn",
			wantType:  "/base",
			wantToken: "Pc9OpwAV4nLdU7lTy71t6Kmmkoz",
			wantError: "",
		},
		{
			name: "知识空间",
			args: args{
				docURL: "https://sample.feishu.cn/wiki/settings/7075377271827264924",
			},
			wantHost:  "sample.feishu.cn",
			wantType:  "/wiki/settings",
			wantToken: "7075377271827264924",
			wantError: "",
		},
		{
			name: "知识库节点",
			args: args{
				docURL: "https://sample.feishu.cn/wiki/sZdeQp3m4nFGzwqR5vx4vZksMoe",
			},
			wantHost:  "sample.feishu.cn",
			wantType:  "/wiki",
			wantToken: "sZdeQp3m4nFGzwqR5vx4vZksMoe",
			wantError: "",
		},
	}
	// logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	// logger := slog.Default()
	for _, tt := range tests {
		s.Run(tt.name, func() {
			gotHost, gotType, gotToken, err := analysisURL(tt.args.docURL)
			if err != nil || tt.wantError != "" {
				msg := err.Error()
				s.Equal(tt.wantError, msg)
				oopsError, ok := oops.AsOops(err)
				if ok {
					s.Equal(tt.wantCode, oopsError.Code())
				}
				// 报错时打印日志方便排查
				// fmt.Printf("%+v", err)
				// logger.Error(msg, slog.Any("error", err))
			}
			s.Equal(tt.wantHost, gotHost)
			s.Equal(tt.wantType, gotType)
			s.Equal(tt.wantToken, gotToken)
		})
	}
}
