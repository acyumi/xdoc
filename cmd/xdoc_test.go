package cmd

import (
	"bytes"
	"errors"
	"testing"

	"github.com/samber/oops"
	"github.com/savioxavier/termlink"
	"github.com/stretchr/testify/suite"

	"github.com/acyumi/xdoc/component/app"
)

// 注册测试套件。
func TestXdocSuite(t *testing.T) {
	suite.Run(t, new(XdocTestSuite))
}

type XdocTestSuite struct {
	suite.Suite
	TempDir string
}

func (s *XdocTestSuite) SetupTest() {
	s.TempDir = s.T().TempDir()
}

func (s *XdocTestSuite) TearDownTest() {

}

func (s *XdocTestSuite) TestExecute() {
	oc := export
	defer func() {
		export = oc
	}()
	tests := []struct {
		name          string
		configFile    string
		configContent []byte
		args          []string
		wantAppID     string
		wantError     string
		wantCode      string
		want          string
		want1         string
	}{
		{
			name:      "无参",
			args:      []string{},
			wantAppID: "",
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
  xdoc [command]

Available Commands:
  export      飞书云文档批量导出器
  help        Help about any command

Flags:
  -h, --help      help for xdoc
  -v, --version   version for xdoc

Use "xdoc [command] --help" for more information about a command.
`,
			want1: `
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
[44;97m[44;97m Go [0m[0m [96m[96mFind more information at: ]8;;https://github.com/acyumi/xdoc[3;32mgithub.com/acyumi/xdoc]8;;[0m[96m[0m[0m

Usage:
  xdoc [command]

Available Commands:
  export      飞书云文档批量导出器
  help        Help about any command

Flags:
  -h, --help      help for xdoc
  -v, --version   version for xdoc

Use "xdoc [command] --help" for more information about a command.
`,
		},
		{
			name:      "帮助",
			args:      []string{"help", "export"},
			wantAppID: "",
			wantError: "",
			want: `这是飞书云文档批量导出、下载到本地的程序

Usage:
  xdoc export [flags]

Flags:
      --app-id string        飞书应用ID
      --app-secret string    飞书应用密钥
      --config string        指定配置文件(默认使用./config.yaml), 配置文件的参数会被命令行参数覆盖
      --dir string           文档存放目录(本地)
      --ext stringToString   文档扩展名映射, 用于指定文档下载后的文件类型, 对应配置文件file.extensions(如 docx=docx,doc=pdf) (default [])
  -h, --help                 help for export
  -l, --list-only            是否只列出云文档信息不进行导出下载
  -q, --quit-automatically   是否在下载完成后自动退出程序
      --urls strings         文档地址, 如 https://sample.feishu.cn/wiki/MP4PwXweMi2FydkkG0ScNwBdnLz
  -V, --verbose              是否显示详细日志
`,
			want1: `这是飞书云文档批量导出、下载到本地的程序

Usage:
  xdoc export [flags]

Flags:
      --app-id string        飞书应用ID
      --app-secret string    飞书应用密钥
      --config string        指定配置文件(默认使用./config.yaml), 配置文件的参数会被命令行参数覆盖
      --dir string           文档存放目录(本地)
      --ext stringToString   文档扩展名映射, 用于指定文档下载后的文件类型, 对应配置文件file.extensions(如 docx=docx,doc=pdf) (default [])
  -h, --help                 help for export
  -l, --list-only            是否只列出云文档信息不进行导出下载
  -q, --quit-automatically   是否在下载完成后自动退出程序
      --urls strings         文档地址, 如 https://sample.feishu.cn/wiki/MP4PwXweMi2FydkkG0ScNwBdnLz
  -V, --verbose              是否显示详细日志
`,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			args := GetArgs()
			root = rootCommand()
			var buf bytes.Buffer
			root.SetOut(&buf)
			root.SetErr(&buf)
			export = exportCommand(vip, args)
			root.AddCommand(export)
			// 这里不能传 nil，因为这会让 cobra 取了 IDE 的参数影响单测，如 Error: unknown shorthand flag: 't' in -testify.m
			if tt.args == nil {
				tt.args = []string{}
			}
			root.SetArgs(tt.args)
			err := Execute()
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
			s.Equal(tt.wantAppID, args.AppID, tt.name)
			// 不同终端打印出来的效果会有一点差别
			if termlink.SupportsHyperlinks() {
				s.Equal(tt.want1, buf.String(), tt.name)
				// 辅助获取单测输出
				if tt.name == "无参" {
					err = app.Fs.WriteFile("/tmp/test_help.txt", buf.Bytes(), 0644)
					s.Require().NoError(err, tt.name)
				}
				return
			}
			s.Equal(tt.want, buf.String(), tt.name)
		})
	}
}
