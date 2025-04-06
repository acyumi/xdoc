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
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/samber/oops"
	"github.com/savioxavier/termlink"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/acyumi/xdoc/component/app"
)

// 注册测试套件。
func TestRootSuite(t *testing.T) {
	suite.Run(t, new(RootTestSuite))
}

type RootTestSuite struct {
	suite.Suite
}

func (s *RootTestSuite) SetupSuite() {
	app.Fs = &afero.Afero{Fs: afero.NewMemMapFs()}
}

func (s *RootTestSuite) SetupTest() {
}

func (s *RootTestSuite) TearDownTest() {

}

func (s *RootTestSuite) TearDownSuite() {

}

func (s *RootTestSuite) TestExecute() {
	tests := []struct {
		name        string
		root        command
		args        []string
		setupMock   func(name string, root command)
		wantVerbose bool
		wantError   string
		wantCode    string
		want        string
	}{
		{
			name:      "无参",
			args:      []string{},
			setupMock: func(name string, root command) {},
			wantError: "",
			want: `
[104m[104m                                       [0m[0m
[104m[104m[97m[97m        嗯? 导出你的云文档吧...        [0m[104m[0m[104m[0m[0m
[104m[104m                                       [0m[0m
[92m[0m
[92m    ██╗  ██╗██████╗  ██████╗  ██████╗[0m
[92m    ╚██╗██╔╝██╔══██╗██╔═══██╗██╔════╝[0m
[92m     ╚███╔╝ ██║  ██║██║   ██║██║     [0m
[92m     ██╔██╗ ██║  ██║██║   ██║██║     [0m
[92m    ██╔╝ ██╗██████╔╝╚██████╔╝╚██████╗[0m
[92m    ╚═╝  ╚═╝╚═════╝  ╚═════╝  ╚═════╝[0m
[92m[0m
[44;97m[44;97m Go [0m[0m [96m[96mFind more information at: [3;32mgithub.com/acyumi/xdoc (https://github.com/acyumi/xdoc)[0m[96m[0m[0m

Usage:
  xdoc [flags]
  xdoc [command]

Examples:
【在程序目录生成config.yaml】
./xdoc -g
./xdoc --generate-config
【使用程序目录的config.yaml导出文档(需要设置相关enabled值为true)】
./xdoc export
【指定配置文件导出文档(需要设置相关enabled值为true)】
./xdoc export --config ./local.yaml
【指定命令行参数执行飞书导出】
./xdoc export feishu --help
./xdoc export feishu --app-id cli_xxx --app-secret yyy --dir /tmp/docs --urls url1,url2...

Available Commands:
  export      云文档批量导出器
  help        Help about any command

Flags:
      --config string        指定配置文件(默认使用./config.yaml), 
                             配置文件的参数可覆盖, 
                             优先级: 命令行参数 > 环境变量 > 配置文件 > 默认值
  -g, --generate-config      是否在程序目录生成config.yaml
  -h, --help                 help for xdoc
  -q, --quit-automatically   是否在程序跑完后自动退出
  -V, --verbose              是否显示详细日志
  -v, --version              version for xdoc

Use "xdoc [command] --help" for more information about a command.
`,
		},
		{
			name:      "帮助",
			args:      []string{"help", "export"},
			setupMock: func(name string, root command) {},
			wantError: "",
			want: `这是云文档批量导出、下载到本地的程序

Usage:
  xdoc export [flags]
  xdoc export [command]

Examples:
【使用默认config.yaml(需要设置相关enabled值为true)】
./xdoc export
【指定配置文件】
./xdoc export --config ./config.yaml
./xdoc export --config ./local.yaml
【指向下级命令】
./xdoc export feishu --help
./xdoc export feishu --config ./local.yaml
./xdoc export feishu --app-id cli_xxx --app-secret yyy --dir /tmp/docs --urls https://xxx.feishu.cn/wiki/123456789

Available Commands:
  feishu      飞书云文档批量导出器

Flags:
  -h, --help        help for export
  -l, --list-only   是否只列出云文档信息不进行导出下载

Global Flags:
      --config string        指定配置文件(默认使用./config.yaml), 
                             配置文件的参数可覆盖, 
                             优先级: 命令行参数 > 环境变量 > 配置文件 > 默认值
  -g, --generate-config      是否在程序目录生成config.yaml
  -q, --quit-automatically   是否在程序跑完后自动退出
  -V, --verbose              是否显示详细日志

Use "xdoc export [command] --help" for more information about a command.
`,
		},
		{
			name:      "无效子命令",
			args:      []string{"xxx"},
			setupMock: func(name string, root command) {},
			wantError: `unknown command "xxx" for "xdoc"`,
			want:      "",
		},
		{
			name: "子命令bind报错",
			root: func() command {
				return NewMockCommand(s.T())
			}(),
			args: []string{"yyy"},
			setupMock: func(name string, root command) {
				mc, _ := root.(*MockCommand)
				mc.EXPECT().init(mock.Anything, mock.Anything).Return().Maybe()
				mc.EXPECT().bind().Return(nil).Once()
				xdoc := &XdocCommand{}
				// 初始化以防root.get()为空
				xdoc.init(nil, nil)
				mc.EXPECT().get().Return(xdoc.get()).Maybe()
				mc.EXPECT().children().Return([]command{mc}).Once()
				mc.EXPECT().init(mock.Anything, mock.Anything).Return().Maybe()
				mc.EXPECT().bind().Return(errors.New("bind error")).Once()
			},
			wantError: `bind error`,
			want:      "",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			root := tt.root
			if root == nil {
				root = &XdocCommand{}
				// 初始化以防root.get()为空
				root.init(nil, nil)
			}
			tt.setupMock(tt.name, root)
			var buf bytes.Buffer
			root.get().SetOut(&buf)
			root.get().SetErr(&buf)
			// 这里不能传 nil，因为这会让 cobra 取了 IDE 的参数影响单测，如 Error: unknown shorthand flag: 't' in -testify.m
			if tt.args == nil {
				tt.args = []string{}
			}
			root.get().SetArgs(tt.args)
			args, err := Execute(root)
			if err != nil || tt.wantError != "" {
				s.Require().Error(err, tt.name)
				s.IsType(oops.OopsError{}, err, tt.name)
				var actualError oops.OopsError
				yes := errors.As(err, &actualError)
				s.Require().True(yes, tt.name)
				s.Equal(tt.wantCode, actualError.Code(), tt.name)
				s.Equal(tt.wantError, actualError.Error(), tt.name)
			} else {
				s.Require().NoError(err, tt.name)
			}
			s.Equal(tt.wantVerbose, args.Verbose, tt.name)
			// 不同终端打印出来的效果会有一点差别
			actual := buf.String()
			if termlink.SupportsHyperlinks() {
				actual = strings.ReplaceAll(buf.String(),
					`]8;;https://github.com/acyumi/xdoc[3;32mgithub.com/acyumi/xdoc]8;;`,
					`[3;32mgithub.com/acyumi/xdoc (https://github.com/acyumi/xdoc)`)
				// 辅助获取单测输出
				// if tt.name == "无参" {
				// 	err = app.Fs.WriteFile("/tmp/test_help.txt", buf.Bytes(), 0644)
				// 	s.Require().NoError(err, tt.name)
				// }
			}
			s.Equal(tt.want, actual, tt.name)
		})
	}
}
