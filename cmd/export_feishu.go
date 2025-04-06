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
	"path/filepath"
	"strings"
	"time"

	"github.com/pterm/pterm"
	"github.com/samber/lo"
	"github.com/samber/oops"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/acyumi/xdoc/component/app"
	"github.com/acyumi/xdoc/component/argument"
	"github.com/acyumi/xdoc/component/cloud"
	"github.com/acyumi/xdoc/component/feishu"
	"github.com/acyumi/xdoc/component/progress"
)

const (
	commandNameFeishu = "feishu"

	flagNameAppID          = "app-id"          //    --app-id
	flagNameAppSecret      = "app-secret"      //    --app-secret
	flagNameURLs           = "urls"            //    --urls
	flagNameDir            = "dir"             //    --dir
	flagNameExt            = "ext"             //    --ext
	flagNameFileExtensions = "file.extensions" //    --ext

	viperKeyPrefix = "export.feishu."
)

type exportFeishuCommand struct {
	*cobra.Command
	vip  *viper.Viper
	args *feishu.Args
}

func (c *exportFeishuCommand) init(vip *viper.Viper, args *argument.Args) {
	c.Command = &cobra.Command{
		Use:   commandNameFeishu,
		Short: "飞书云文档批量导出器",
		Long:  "这是飞书云文档批量导出、下载到本地的程序",
		Example: `【使用默认config.yaml】
./xdoc export feishu
【指定配置文件】
./xdoc export feishu --config ./config.yaml
./xdoc export feishu --config ./local.yaml
【指定命令行参数】
./xdoc export feishu --help
./xdoc export feishu --app-id cli_xxx --app-secret yyy --dir /tmp/docs --urls url1,url2...
./xdoc export feishu --app-id cli_xxx --app-secret yyy --dir /tmp/docs --urls https://xxx.feishu.cn/wiki/123456789`,
		RunE: func(_ *cobra.Command, _ []string) error {
			// 执行到当前命令了，那就把开关设置为打开
			c.vip.Set(viperKeyFeishuEnabled, true)
			return c.exec()
		},
	}
	c.vip = vip
	c.args = &feishu.Args{Args: args}
}

func (c *exportFeishuCommand) bind() (err error) {
	// 添加命令行参数
	flags := c.Command.Flags()
	flags.String(flagNameAppID, "", "飞书应用ID")
	flags.String(flagNameAppSecret, "", "飞书应用密钥")
	flags.StringSlice(flagNameURLs, []string{}, "文档地址, 如 https://sample.feishu.cn/wiki/MP4PwXweMi2FydkkG0ScNwBdnLz")
	flags.String(flagNameDir, "", "文档存放目录(本地)")
	flags.StringToString(flagNameExt, map[string]string{}, `文档扩展名映射, 用于指定文档下载后的文件类型, 如 docx=docx,doc=pdf
对应配置文件参数 export.feishu.file.extensions`)

	// 绑定 Viper
	flags.VisitAll(func(flag *pflag.Flag) {
		switch flag.Name {
		case flagNameExt:
			// 不绑定，因为要实现 --ext 参数【局部覆盖】fileExtensions中的key的效果
			return
		default:
			_ = c.vip.BindPFlag(viperKeyPrefix+flag.Name, flag)
		}
	})
	return nil
}

func (c *exportFeishuCommand) get() *cobra.Command {
	return c.Command
}

func (c *exportFeishuCommand) children() []command {
	return []command{}
}

func (c *exportFeishuCommand) exec() (err error) {
	out := c.OutOrStdout()
	args := c.args
	err = setArgs(c.Command, c.vip, args)
	if err != nil {
		return oops.Wrap(err)
	}
	args.StartTime = time.Now()
	defer func() {
		duration := time.Since(args.StartTime)
		app.Fprintln(out, "----------------------------------------------")
		app.Fprintf(out, "完成飞书云文档操作, 总耗时: %s\n", duration.String())
		if err != nil {
			return
		}
		err, _ = recover().(error)
	}()
	// 判断配置文件是否存在
	yes, err := app.Fs.Exists(args.ConfigFile)
	if err != nil {
		return oops.Wrap(err)
	}
	configFileLog := args.ConfigFile
	if !yes {
		configFileLog += pterm.LightYellow("(不存在)")
	}
	app.Fprintln(out, "----------------------------------------------")
	app.Fprintf(out, " ConfigFile: %s\n", configFileLog)
	app.Fprintf(out, " Verbose: %v\n", args.Verbose)
	app.Fprintf(out, " AppID: %s\n", args.Desensitize(args.AppID))
	app.Fprintf(out, " AppSecret: %s\n", args.Desensitize(args.AppSecret))
	app.Fprintf(out, " DocURLs: %v\n", func() string {
		ds := args.DesensitizeSlice(args.DocURLs...)
		urls, _ := app.MarshalIndent(ds, "", "  ")
		return strings.ReplaceAll(string(urls), "\n", "\n ")
	}())
	app.Fprintf(out, " SaveDir: %s\n", args.SaveDir)
	app.Fprintf(out, " FileExtensions: %v\n", args.FileExtensions)
	app.Fprintf(out, " ListOnly: %v\n", args.ListOnly)
	app.Fprintf(out, " QuitAutomatically: %v\n", args.QuitAutomatically)
	app.Fprintln(out, "----------------------------------------------")
	if err = args.Validate(); err != nil {
		return oops.Wrap(err)
	}
	// 先通过文档地址获取文件类型和token
	var gotHost string
	var docSources []*cloud.DocumentSource
	for _, docURL := range args.DocURLs {
		host, typ, token, err := analysisURL(docURL)
		if err != nil {
			return oops.Wrap(err)
		}
		if gotHost == "" {
			gotHost = host
		} else if gotHost != host {
			return oops.Errorf("文档地址不匹配, 请确保所有文档地址都是同一域名")
		}
		docSources = append(docSources, &cloud.DocumentSource{Type: typ, Token: token})
	}
	err = doExport(args, gotHost, docSources)
	return oops.Wrap(err)
}

func setArgs(cmd *cobra.Command, vip *viper.Viper, args *feishu.Args) error {
	// 从 Viper 中读取配置
	args.ListOnly = vip.GetBool(commandNameExport + "." + flagNameListOnly)
	args.Enabled = vip.GetBool(viperKeyFeishuEnabled)
	args.AppID = vip.GetString(getFlagName(flagNameAppID))
	args.AppSecret = vip.GetString(getFlagName(flagNameAppSecret))
	args.DocURLs = vip.GetStringSlice(getFlagName(flagNameURLs))
	args.SaveDir = vip.GetString(getFlagName(flagNameDir))
	args.SaveDir = filepath.Clean(args.SaveDir)
	args.SetFileExtensions(vip.GetStringMapString(getFlagName(flagNameFileExtensions)))
	overrides, err := cmd.Flags().GetStringToString(flagNameExt)
	if err != nil {
		return oops.Wrap(err)
	}
	args.SetFileExtensions(overrides)
	// 去重
	args.DocURLs = lo.Uniq[string](args.DocURLs)
	return nil
}

func getFlagName(name string) string {
	return viperKeyPrefix + name
}

func doExport(args *feishu.Args, host string, docSources []*cloud.DocumentSource) error {
	switch {
	case strings.HasSuffix(host, "feishu.cn"):
		// 创建 飞书客户端
		client := feishu.NewClient(args)
		// 下载文档
		return client.DownloadDocuments(docSources)
	case host == "progress.test":
		// 创建 进度条测试客户端
		client := progress.NewTestClient()
		return client.DownloadDocuments(docSources)
	case host == "silence.test":
		return nil
	default:
		return oops.Errorf("不支持的文档来源域名: %s", host)
	}
}
