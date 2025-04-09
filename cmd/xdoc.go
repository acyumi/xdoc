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
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/pterm/pterm"
	"github.com/samber/oops"
	"github.com/savioxavier/termlink"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/acyumi/xdoc/component/app"
	"github.com/acyumi/xdoc/component/argument"
)

const (
	commandNameXdoc = "xdoc"

	flagNameConfig            = "config"             //    --config
	flagNameGenerateConfig    = "generate-config"    // -g --generate-config
	flagNameQuitAutomatically = "quit-automatically" // -q --quit-automatically
	flagNameVerbose           = "verbose"            // -V --verbose

	viperKeyGotConfigFile = "gotConfigFile"
)

var (
	version = "dev"     // 编译时注入的版本号
	gitRev  = "unknown" // 编译时注入的Git revision
	builtBy = "unknown" // 编译时注入的Git revision
	builtAt = "unknown" // 编译时注入的Git revision

	//go:embed config-template.yaml
	// 本来想将 config-template.yaml 放到根目录的，但是go的语法不支持 ../config-template.yaml。
	// https://go.googlesource.com/proposal/+/master/design/draft-embed.md#go_embed-directives
	// https://github.com/golang/go/issues/46056
	configTemplate string
)

type XdocCommand struct {
	*cobra.Command
	vip  *viper.Viper
	args *argument.Args
}

func (c *XdocCommand) init(vip *viper.Viper, args *argument.Args) {
	if c.Command == nil {
		c.Command = &cobra.Command{
			Use:   commandNameXdoc,
			Short: "执行云文档的相关操作(如:导出)",
			Long:  logo(),
			Example: `【在程序目录生成config.yaml】
./xdoc -g
./xdoc --generate-config
【使用程序目录的config.yaml导出文档(需要设置相关enabled值为true)】
./xdoc export
【指定配置文件导出文档(需要设置相关enabled值为true)】
./xdoc export --config ./local.yaml
【指定命令行参数执行飞书导出】
./xdoc export feishu --help
./xdoc export feishu --app-id cli_xxx --app-secret yyy --dir /tmp/docs --urls url1,url2...`,
			Version:           version,
			DisableAutoGenTag: true,
			CompletionOptions: cobra.CompletionOptions{
				HiddenDefaultCmd: true,
			},
			PersistentPreRunE: c.PersistentPreRunE,
			RunE:              doNothing,
			SilenceErrors:     true, // 禁用默认的错误输出，转为自己打印，子命令会遵循这个配置
		}
		c.SetVersionTemplate(fmt.Sprintf(`program: xdoc
version: {{.Version}}
git: %s
built by: %s
built at: %s`, gitRev, builtBy, builtAt))
		c.SetOut(os.Stdout) // 子命令如果不覆盖，则会递归到根命令取到这个配置
	}
	c.vip = vip
	c.args = args
}

func logo() string {
	header := pterm.DefaultHeader.WithMargin(8).
		WithBackgroundStyle(pterm.NewStyle(pterm.BgLightBlue)).
		WithTextStyle(pterm.NewStyle(pterm.FgLightWhite)).
		Sprint("嗯? 导出你的云文档吧...")
	logo := pterm.FgLightGreen.Sprint(`
    ██╗  ██╗██████╗  ██████╗  ██████╗
    ╚██╗██╔╝██╔══██╗██╔═══██╗██╔════╝
     ╚███╔╝ ██║  ██║██║   ██║██║     
     ██╔██╗ ██║  ██║██║   ██║██║     
    ██╔╝ ██╗██████╔╝╚██████╔╝╚██████╗
    ╚═╝  ╚═╝╚═════╝  ╚═════╝  ╚═════╝
`)
	tips := tips("Go", "Find more information at:", "github.com/acyumi/xdoc", "https://github.com/acyumi/xdoc")
	return fmt.Sprintf("\n%s%s\n%s\n", header, logo, tips)
}

func tips(prefix, middle, text, url string) string {
	pterm.Info.Prefix = pterm.Prefix{
		Text:  prefix,
		Style: pterm.NewStyle(pterm.BgBlue, pterm.FgLightWhite),
	}
	link := termlink.ColorLink(text, url, "italic green")
	return pterm.Info.Sprintf("%s %s", middle, link)
}

func (c *XdocCommand) bind() error {
	// 添加 --config 参数
	persistentFlags := c.PersistentFlags()
	// 先尝试从命令行中读取 --config 参数，取不到再取默认路径上的配置文件具体看下面的loadConfig函数
	persistentFlags.String(flagNameConfig, "", `指定配置文件(默认使用./config.yaml), 
配置文件的参数可覆盖, 
优先级: 命令行参数 > 环境变量 > 配置文件 > 默认值`)
	persistentFlags.BoolP(flagNameGenerateConfig, "g", false, "是否在程序目录生成config.yaml")
	persistentFlags.BoolP(flagNameQuitAutomatically, "q", false, "是否在程序跑完后自动退出")
	persistentFlags.BoolP(flagNameVerbose, "V", false, "是否显示详细日志")
	// 绑定 Viper
	// 反复测试发现目前版本的BindPFlags正常使用下不会报错，所以这里直接吃掉错误
	_ = c.vip.BindPFlags(persistentFlags)
	// 设置环境变量前缀
	c.vip.SetEnvPrefix(strings.ToUpper(commandNameXdoc))
	// 自动绑定环境变量（将点替换为下划线，如 "export.feishu.app-id" -> "XDOC_EXPORT_FEISHU_APP_ID"）
	c.vip.AutomaticEnv()
	c.vip.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	// 这里只做绑定，不读取配置，因为此时可能还未设置环境变量、命令行参数，放到PersistentPreRunE中读取
	return nil
}

func (c *XdocCommand) get() *cobra.Command {
	return c.Command
}

func (c *XdocCommand) children() []command {
	// 这里children()只在初始化时调用一次，所以可以不缓存起来
	return []command{
		&exportCommand{},
	}
}

func (c *XdocCommand) exec() error {
	return nil
}

// PersistentPreRunE 实现全部命令支持生成配置文件。
func (c *XdocCommand) PersistentPreRunE(cmd *cobra.Command, _ []string) error {
	err := c.loadConfig()
	if err != nil {
		return oops.Wrap(err)
	}
	// 最后再按优先级从viper中取出配置值
	c.args.Verbose = c.vip.GetBool(flagNameVerbose)
	c.args.GenerateConfig = c.vip.GetBool(flagNameGenerateConfig)
	c.args.QuitAutomatically = c.vip.GetBool(flagNameQuitAutomatically)
	if !c.args.GenerateConfig {
		// 如果是根命令，则打印 logo 和 帮助信息
		if c.Command == cmd {
			return pflag.ErrHelp
		}
		// 如果不是，则返回nil，继续子命令的正常执行
		return nil
	}
	// 生成配置文件
	exePath, err := app.Executable()
	if err != nil {
		return oops.Wrapf(err, "获取程序所在目录失败")
	}
	exeDir := filepath.Dir(exePath)
	configPath := filepath.Join(exeDir, "config.yaml")
	// TODO 检查配置文件是否存在，提示是否覆盖
	// TODO 支持配合--output来指定目录和文件名存放
	err = app.Fs.WriteFile(configPath, []byte(configTemplate), 0644)
	if err != nil {
		return oops.Wrap(err)
	}
	msg := tips("OK", "配置文件已生成:", "config.yaml", configPath)
	app.Fprintf(c.Command.OutOrStdout(), "\n%s\n\n", msg)
	// 覆盖子命令的执行操作，仅生成配置文件，让程序直接结束
	cmd.RunE = doNothing
	return nil
}

// doNothing 忽略命令执行。
func doNothing(_ *cobra.Command, _ []string) error {
	return nil
}

func (c *XdocCommand) loadConfig() (err error) {
	err = loadConfigFromFile(c.vip, c.args)
	if err != nil {
		var oe oops.OopsError
		if ok := errors.As(err, &oe); !ok || oe.Code() != "continue" {
			return oops.Wrapf(err, "加载配置文件失败")
		}
		// 不从c.args中取值，尝试从c.vip中取
		if c.vip.GetBool(flagNameVerbose) {
			app.Fprintln(c.Command.OutOrStdout(), oe.Error())
		}
	}
	return nil
}

func loadConfigFromFile(vip *viper.Viper, args *argument.Args) error {
	// 如果指定了 --config 参数，则使用指定的配置文件
	configFile := vip.GetString(flagNameConfig)
	if configFile != "" {
		vip.SetConfigFile(configFile)
		args.ConfigFile = configFile
	} else {
		// 默认从程序所在目录读取配置文件
		exePath, err := app.Executable()
		if err != nil {
			return oops.Wrapf(err, "获取程序所在目录失败")
		}
		exeDir := filepath.Dir(exePath)
		vip.AddConfigPath(exeDir)         // 添加程序所在目录为配置文件搜索路径
		vip.SetConfigName(flagNameConfig) // 配置文件名称（不带扩展名）
		vip.SetConfigType("yaml")         // 配置文件类型
		args.ConfigFile = filepath.Join(exeDir, flagNameConfig+".yaml")
	}
	// 读取配置文件
	if err := vip.ReadInConfig(); err != nil {
		var cfnfe viper.ConfigFileNotFoundError
		if ok := errors.As(err, &cfnfe); ok {
			// 配置文件不存在，忽略错误
			return oops.Code("continue").New("未找到配置文件, 将使用命令行参数，尝试使用命令行参数继续执行")
		}
		pathErr := &fs.PathError{}
		if ok := errors.As(err, &pathErr); ok {
			return oops.Code("continue").New("请检查配置文件权限，或者指定其他位置的配置文件，尝试使用命令行参数继续执行")
		}
		// 其他错误
		return oops.Wrap(err)
	}
	vip.Set(viperKeyGotConfigFile, true)
	return nil
}
