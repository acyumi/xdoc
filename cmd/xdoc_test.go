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
	"fmt"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/samber/oops"
	"github.com/savioxavier/termlink"
	"github.com/spf13/afero"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/suite"

	"github.com/acyumi/xdoc/component/app"
	"github.com/acyumi/xdoc/component/argument"
)

// 注册测试套件。
func TestXdocSuite(t *testing.T) {
	suite.Run(t, new(XdocTestSuite))
}

type XdocTestSuite struct {
	suite.Suite
	memFs      *afero.Afero
	executable func() (string, error)
}

func (s *XdocTestSuite) SetupSuite() {
	app.Fs = &afero.Afero{Fs: afero.NewMemMapFs()}
	s.memFs = app.Fs
	app.Executable = func() (string, error) {
		return "/tmp/xdoc.exe", nil
	}
	s.executable = app.Executable
}

func (s *XdocTestSuite) SetupTest() {
}

func (s *XdocTestSuite) TearDownTest() {
}

func (s *XdocTestSuite) TestExecute() {
	tests := []struct {
		name         string
		args         []string
		setupMock    func(name string, root *XdocCommand)
		teardownMock func(name string, root *XdocCommand)
		wantVerbose  bool
		wantError    string
		wantCode     string
		want         string
	}{
		{
			name:      "无参",
			args:      []string{},
			setupMock: func(name string, root *XdocCommand) {},
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

Flags:
      --config string        指定配置文件(默认使用./config.yaml), 
                             配置文件的参数可覆盖, 
                             优先级: 命令行参数 > 环境变量 > 配置文件 > 默认值
  -g, --generate-config      是否在程序目录生成config.yaml
  -h, --help                 help for xdoc
  -q, --quit-automatically   是否在程序跑完后自动退出
  -V, --verbose              是否显示详细日志
  -v, --version              version for xdoc
`,
		},
		{
			name:      "帮助",
			args:      []string{"help", "export"},
			setupMock: func(name string, root *XdocCommand) {},
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

Flags:
      --config string        指定配置文件(默认使用./config.yaml), 
                             配置文件的参数可覆盖, 
                             优先级: 命令行参数 > 环境变量 > 配置文件 > 默认值
  -g, --generate-config      是否在程序目录生成config.yaml
  -h, --help                 help for xdoc
  -q, --quit-automatically   是否在程序跑完后自动退出
  -V, --verbose              是否显示详细日志
  -v, --version              version for xdoc
`,
		},
		{
			name:      "生成config.yaml[正常]",
			args:      []string{"-g"},
			setupMock: func(name string, root *XdocCommand) {},
			wantError: "",
			want: fmt.Sprintf(`
[44;97m[44;97m OK [0m[0m [96m[96m配置文件已生成: [3;32mconfig.yaml (%s)[0m[96m[0m[0m

`, filepath.Clean("/tmp/config.yaml")),
		},
		{
			name: "生成config.yaml[指定环境变量]",
			args: []string{},
			setupMock: func(name string, root *XdocCommand) {
				t := s.T()
				t.Setenv("XDOC_VERBOSE", "true")
				t.Setenv("XDOC_GENERATE_CONFIG", "true")
			},
			teardownMock: func(name string, root *XdocCommand) {
			},
			wantVerbose: true,
			wantError:   "",
			want: fmt.Sprintf(`未找到配置文件, 将使用命令行参数，尝试使用命令行参数继续执行

[44;97m[44;97m OK [0m[0m [96m[96m配置文件已生成: [3;32mconfig.yaml (%s)[0m[96m[0m[0m

`, filepath.Clean("/tmp/config.yaml")),
		},
		{
			name: "生成config.yaml[取程序路径失败]",
			args: []string{"-g"},
			setupMock: func(name string, root *XdocCommand) {
				var count int
				app.Executable = func() (string, error) {
					count++
					if count == 1 {
						return "/tmp/xdoc.exe", nil
					}
					return "", errors.New("取程序路径失败")
				}
			},
			wantError: "获取程序所在目录失败: 取程序路径失败",
			want: `Usage:
  xdoc [flags]

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

Flags:
      --config string        指定配置文件(默认使用./config.yaml), 
                             配置文件的参数可覆盖, 
                             优先级: 命令行参数 > 环境变量 > 配置文件 > 默认值
  -g, --generate-config      是否在程序目录生成config.yaml
  -h, --help                 help for xdoc
  -q, --quit-automatically   是否在程序跑完后自动退出
  -V, --verbose              是否显示详细日志
  -v, --version              version for xdoc

`,
		},
		{
			name: "生成config.yaml[写文件失败]",
			args: []string{"--generate-config"},
			setupMock: func(name string, root *XdocCommand) {
				app.Fs = &afero.Afero{Fs: afero.NewReadOnlyFs(app.Fs)}
			},
			wantError: "operation not permitted",
			want: `Usage:
  xdoc [flags]

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

Flags:
      --config string        指定配置文件(默认使用./config.yaml), 
                             配置文件的参数可覆盖, 
                             优先级: 命令行参数 > 环境变量 > 配置文件 > 默认值
  -g, --generate-config      是否在程序目录生成config.yaml
  -h, --help                 help for xdoc
  -q, --quit-automatically   是否在程序跑完后自动退出
  -V, --verbose              是否显示详细日志
  -v, --version              version for xdoc

`,
		},
		{
			name: "指定环境变量，取程序路径失败",
			args: []string{},
			setupMock: func(name string, root *XdocCommand) {
				app.Executable = func() (string, error) {
					return "", errors.New("取程序路径失败")
				}
				s.T().Setenv("XDOC_VERBOSE", "true")
			},
			teardownMock: func(name string, root *XdocCommand) {
			},
			wantVerbose: false, // 赋值前报错了，所以指定了环境变量还是false
			wantError:   "加载配置文件失败: 获取程序所在目录失败: 取程序路径失败",
			want: `Usage:
  xdoc [flags]

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

Flags:
      --config string        指定配置文件(默认使用./config.yaml), 
                             配置文件的参数可覆盖, 
                             优先级: 命令行参数 > 环境变量 > 配置文件 > 默认值
  -g, --generate-config      是否在程序目录生成config.yaml
  -h, --help                 help for xdoc
  -q, --quit-automatically   是否在程序跑完后自动退出
  -V, --verbose              是否显示详细日志
  -v, --version              version for xdoc

`,
		},
		{
			name:        "指向不存在的配置文件",
			args:        []string{"--config", "/tmp/nonexistent.yaml", "-V"},
			setupMock:   func(name string, root *XdocCommand) {},
			wantVerbose: true,
			wantError:   "",
			want: `请检查配置文件权限，或者指定其他位置的配置文件，尝试使用命令行参数继续执行

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

Flags:
      --config string        指定配置文件(默认使用./config.yaml), 
                             配置文件的参数可覆盖, 
                             优先级: 命令行参数 > 环境变量 > 配置文件 > 默认值
  -g, --generate-config      是否在程序目录生成config.yaml
  -h, --help                 help for xdoc
  -q, --quit-automatically   是否在程序跑完后自动退出
  -V, --verbose              是否显示详细日志
  -v, --version              version for xdoc
`,
		},
		{
			name: "执行子命令",
			args: []string{"export"},
			setupMock: func(name string, root *XdocCommand) {
				children := root.children()
				s.Require().NotEmpty(children, name)
				// 初始化一个子命令
				firstChild := children[0]
				firstChild.init(root.vip, root.args)
				err := firstChild.bind()
				s.Require().NoError(err, name)
				root.AddCommand(firstChild.get())
			},
			want: `这是云文档批量导出、下载到本地的程序

Usage:
  xdoc export [flags]

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
`,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer func() {
				app.Executable = s.executable
				app.Fs = s.memFs
			}()
			vip := app.NewViper()
			args := &argument.Args{}
			root := &XdocCommand{}
			// 初始化以防root.get()为空
			root.init(vip, args)
			var buf bytes.Buffer
			cmd := root.get()
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			tt.setupMock(tt.name, root)
			err := root.bind()
			if err == nil {
				err = root.exec()
				s.Require().NoError(err, tt.name)
				// 这里不能传 nil，因为这会让 cobra 取了 IDE 的参数影响单测，如 Error: unknown shorthand flag: 't' in -testify.m
				if tt.args == nil {
					tt.args = []string{}
				}
				cmd.SetArgs(tt.args)
				err = cmd.Execute()
			}
			if tt.teardownMock != nil {
				tt.teardownMock(tt.name, root)
			}
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
			actual := buf.String()
			actual = cleanHyperlinks(actual)
			// fmt.Print(actual)
			s.Equal(tt.want, actual, tt.name)
		})
	}
}

var (
	hyperlinksRegex  = regexp.MustCompile(`]8;;(.+)(\[(\d{1,2};)*\d{1,2}m)?(.+)]8;;\[0m`)
	normalLinksRegex = regexp.MustCompile(`(\[(\d{1,2};)*\d{1,2}m)(.+) \(.+\)\[0m`)
)

// cleanHyperlinks 如果终端支持超链接，则替换为普通格式再进行比较。
func cleanHyperlinks(str string) string {
	// 不同终端打印出来的效果会有一点差别
	if termlink.SupportsHyperlinks() {
		str = hyperlinksRegex.ReplaceAllString(str, `$2$4 ($1)[0m`)
	}
	return str
}

func (s *XdocTestSuite) Test_regex_replace() {
	str1 := "你好\x1b]8;;https://xxx.feishu.cn\x07\u001B[30;33;32mxxx.feishu\x1b]8;;\x07\u001b[0m你好"
	result := hyperlinksRegex.ReplaceAllString(str1, "666")
	s.Equal("你好666你好", result)

	str2 := `你好]8;;https://xxx.feishu.cn[30;33;32mxxx.feishu]8;;[0m你好`
	result = hyperlinksRegex.ReplaceAllString(str2, "777")
	s.Equal("你好777你好", result)

	str3 := `你好]8;;https://feishu.cn[32mfeishu]8;;[0m你好`
	result = hyperlinksRegex.ReplaceAllString(str3, "888")
	s.Equal("你好888你好", result)

	str4 := `你好]8;;https://feishu.cn[32mfeishu]8;;[0m你好`
	result = hyperlinksRegex.ReplaceAllString(str4, `[3;32m$4 ($1)`)
	s.Equal(`你好[3;32mfeishu (https://feishu.cn)你好`, result)
	result = hyperlinksRegex.ReplaceAllString(str4, `$2$4 ($1)`)
	s.Equal(`你好[32mfeishu (https://feishu.cn)你好`, result)

	str5 := `你好[3;32mhttps://feishu.cn (feishu)[0m你好`
	result = normalLinksRegex.ReplaceAllString(str5, `999`)
	s.Equal(`你好999你好`, result)
}

// 测试配置文件加载。
func (s *XdocTestSuite) Test_loadConfigFromFile() {
	tests := []struct {
		name          string
		configFile    string
		configContent []byte
		setupMock     func(name string, args *argument.Args)
		wantConfig    string
		wantAppID     string
		wantError     string
		wantCode      string
	}{
		{
			name:       "指定有效配置文件",
			configFile: filepath.Clean("/tmp/test.yaml"),
			configContent: []byte(`
export:
  feishu:
    app-id: config_app
    file:
      extensions:
        doc: pdf
`),
			setupMock:  func(name string, args *argument.Args) {},
			wantConfig: filepath.Clean("/tmp/test.yaml"),
			wantAppID:  "config_app",
		},
		{
			name:       "默认配置文件",
			configFile: "",
			configContent: []byte(`
export:
  feishu:
    app-id: default_app
`),
			setupMock:  func(name string, args *argument.Args) {},
			wantConfig: filepath.Clean("/tmp/config.yaml"),
			wantAppID:  "default_app",
		},
		{
			name:       "配置文件不存在",
			configFile: "nonexistent.yaml",
			setupMock:  func(name string, args *argument.Args) {},
			wantConfig: "nonexistent.yaml",
			wantAppID:  "",
			wantError:  "请检查配置文件权限，或者指定其他位置的配置文件，尝试使用命令行参数继续执行",
			wantCode:   "continue",
		},
		{
			name:          "配置文件类型不对",
			configFile:    filepath.Clean("/tmp/ttt.exe"),
			configContent: []byte(`xxx`),
			setupMock:     func(name string, args *argument.Args) {},
			wantConfig:    filepath.Clean("/tmp/ttt.exe"),
			wantAppID:     "",
			wantError:     "Unsupported Config Type \"exe\"",
			wantCode:      "",
		},
		{
			name:          "获取程序所在目录失败",
			configFile:    "",
			configContent: nil,
			setupMock: func(name string, args *argument.Args) {
				app.Executable = func() (string, error) {
					return "", errors.New("取程序路径失败")
				}
			},
			wantConfig: "",
			wantAppID:  "",
			wantError:  "获取程序所在目录失败: 取程序路径失败",
			wantCode:   "",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer func() {
				app.Executable = s.executable
				app.Fs = s.memFs
			}()
			configFile := tt.configFile
			if tt.configFile == "" && tt.configContent != nil {
				exePath, err := app.Executable()
				s.Require().NoError(err, "获取程序所在目录失败")
				exeDir := filepath.Dir(exePath)
				configFile = filepath.Join(exeDir, flagNameConfig+".yaml")
			}
			if configFile != "" && tt.configContent != nil {
				err := app.Fs.WriteFile(configFile, tt.configContent, 0644)
				s.Require().NoError(err, "创建测试配置文件失败")
			}
			vip := app.NewViper()
			err := vip.BindPFlag(flagNameConfig, &pflag.Flag{Name: flagNameConfig, Value: newStringValue(configFile, &configFile)})
			s.Require().NoError(err, tt.name)
			args := &argument.Args{ConfigFile: tt.configFile}
			tt.setupMock(tt.name, args)
			err = loadConfigFromFile(vip, args)
			if err != nil || tt.wantError != "" {
				s.Require().Error(err, tt.name)
				var oe oops.OopsError
				ok := errors.As(err, &oe)
				s.True(ok, tt.name)
				s.Require().EqualError(err, tt.wantError, tt.name)
				s.Equal(tt.wantCode, oe.Code(), tt.name)
			} else {
				s.Require().NoError(err, tt.name)
			}
			s.Equal(tt.wantConfig, args.ConfigFile, tt.name)
			s.Equal(tt.wantAppID, vip.GetString(viperKeyPrefix+flagNameAppID), tt.name)
			yes, err := app.Fs.Exists(args.ConfigFile)
			s.Require().NoError(err, tt.name)
			if yes {
				err = app.Fs.Remove(args.ConfigFile)
				s.Require().NoError(err, tt.name)
			}
		})
	}
}

type stringValue string

func newStringValue(val string, p *string) *stringValue {
	*p = val
	return (*stringValue)(p)
}

func (s *stringValue) Set(val string) error {
	*s = stringValue(val)
	return nil
}

func (s *stringValue) Type() string {
	return "string"
}

func (s *stringValue) String() string {
	return string(*s)
}
