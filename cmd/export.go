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
	"net/url"
	"os"
	"strings"

	"github.com/samber/oops"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/acyumi/xdoc/component/argument"
)

const (
	commandNameExport = "export" //

	flagNameListOnly = "list-only" // -f --list-only

	viperKeyFeishuEnabled = "export.feishu.enabled" //
)

type exportCommand struct {
	*cobra.Command
	vip        *viper.Viper
	args       *argument.Args
	subs       []command
	subCommand string
}

func (c *exportCommand) init(vip *viper.Viper, args *argument.Args) {
	c.Command = &cobra.Command{
		Use:   commandNameExport,
		Short: "云文档批量导出器",
		Long:  "这是云文档批量导出、下载到本地的程序",
		Example: `【使用默认config.yaml(需要设置相关enabled值为true)】
./xdoc export
【指定配置文件】
./xdoc export --config ./config.yaml
./xdoc export --config ./local.yaml
【指向下级命令】
./xdoc export feishu --help
./xdoc export feishu --config ./local.yaml
./xdoc export feishu --app-id cli_xxx --app-secret yyy --dir /tmp/docs --urls https://xxx.feishu.cn/wiki/123456789`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return c.exec()
		},
	}
	c.vip = vip
	c.args = args
}

func (c *exportCommand) bind() (err error) {
	persistentFlags := c.Command.PersistentFlags()
	persistentFlags.BoolP(flagNameListOnly, "l", false, "是否只列出云文档信息不进行导出下载")
	_ = c.vip.BindPFlag(commandNameExport+"."+flagNameListOnly, persistentFlags.Lookup(flagNameListOnly))
	osArgs := os.Args[1:]
	if len(osArgs) >= 2 {
		second := osArgs[1]
		if !strings.HasPrefix(second, "-") {
			c.subCommand = second
			// 如果是指定了children()中存在子命令，则不会执行export的runE函数，那就可以跳过了
			return nil
		}
	}
	// 执行到这里就代表未指定export下的子命令，后续有需求可添加相应逻辑供runE函数使用
	// 注意：遵循接口的设计，这里不要从c.vip中读取配置来使用，否则可有读取不到配置文件中的值
	return nil
}

func (c *exportCommand) get() *cobra.Command {
	return c.Command
}

func (c *exportCommand) children() []command {
	// 这里children()会被调用多次，所以需要缓存起来
	if len(c.subs) == 0 {
		c.subs = []command{
			&exportFeishuCommand{},
		}
	}
	return c.subs
}

func (c *exportCommand) exec() error {
	// export命令没有定义对应的flag参数，仅支持从配置文件或环境变量中取值
	// 从配置文件或环境变量中取值判断是否启用飞书导出功能
	if c.subCommand == "" && c.vip.GetBool(viperKeyFeishuEnabled) {
		// TODO 以后加入其他平台的云文档导出再继续判断，计划只同时支持打开一种开关
		c.subCommand = commandNameFeishu
	}
	if c.subCommand == "" {
		return pflag.ErrHelp
	}
	for _, child := range c.children() {
		if child.get().Name() == c.subCommand {
			return child.exec()
		}
	}
	return oops.Code("InvalidArgument").Errorf("未找到export下的子命令: %s\n", c.subCommand)
}

func analysisURL(docURL string) (host, typ, token string, err error) {
	// 文件夹 folder_token：https://sample.feishu.cn/drive/folder/cSJe2JgtFFBwRuTKAJK6baNGUn0
	// 文件 file_token：https://sample.feishu.cn/file/ndqUw1kpjnGNNaegyqDyoQDCLx1
	// 文档 doc_token：https://sample.feishu.cn/docs/2olt0Ts4Mds7j7iqzdwrqEUnO7q
	// 新版文档 document_id：https://sample.feishu.cn/docx/UXEAd6cRUoj5pexJZr0cdwaFnpd
	// 电子表格 spreadsheet_token：https://sample.feishu.cn/sheets/MRLOWBf6J47ZUjmwYRsN8utLEoY
	// 多维表格 app_token：https://sample.feishu.cn/base/Pc9OpwAV4nLdU7lTy71t6Kmmkoz
	// 知识空间 space_id：https://sample.feishu.cn/wiki/settings/7075377271827264924（需要知识库管理员在设置页面获取该地址）
	// 知识库节点 node_token：https://sample.feishu.cn/wiki/sZdeQp3m4nFGzwqR5vx4vZksMoe
	//
	// https://root:123456@baidu.com:443?dddd=oo&uuu=55#/adfadf/fade
	// temp.Scheme = "https"
	// temp.Opaque = ""
	// temp.User = {username:root,password:123456}
	// temp.Host = "baidu.com:443"
	// addressURL.Path = ""
	// addressURL.RawPath = ""
	// addressURL.OmitHost = false
	// addressURL.ForceQuery = false
	// addressURL.RawQuery = "dddd=oo&uuu=55"
	// addressURL.Fragment = "/adfadf/fade"
	// addressURL.RawFragment = ""
	URL, err := url.Parse(docURL)
	if err != nil {
		return host, typ, token, oops.Code("BadRequest").Wrapf(err, "解析url地址失败：%s", docURL)
	}
	if URL.Scheme != "http" && URL.Scheme != "https" {
		return host, typ, token, oops.Code("BadRequest").Errorf("url地址必须是http://或https://开头")
	}
	host = strings.Split(URL.Host, ":")[0]
	path := URL.Path
	split := strings.Split(path, "/")
	if len(split) < 3 {
		return host, typ, token, oops.Code("BadRequest").
			New("url地址的path部分至少包含两段才能解析出云文档类型和token，如:/docs/2olt0Ts4Mds7j7iqzdwrqEUnO7q")
	}
	typ = strings.Join(split[0:len(split)-1], "/")
	token = split[len(split)-1]
	return host, typ, token, nil
}
