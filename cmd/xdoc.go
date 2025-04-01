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
	"fmt"

	"github.com/pterm/pterm"
	"github.com/savioxavier/termlink"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/acyumi/xdoc/component/argument"
)

var (
	vip  = viper.New()
	args = &argument.Args{}
	root = rootCommand()
)

func Execute() error {
	return root.Execute()
}

func GetArgs() *argument.Args {
	return args
}

func rootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "xdoc",
		Short:             "执行云文档的相关操作(如:导出)",
		Long:              logo(),
		Version:           "0.0.1",
		DisableAutoGenTag: true,
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
	}
	return cmd
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
	pterm.Info.Prefix = pterm.Prefix{
		Text:  "Go",
		Style: pterm.NewStyle(pterm.BgBlue, pterm.FgLightWhite),
	}
	link := termlink.ColorLink("github.com/acyumi/xdoc", "https://github.com/acyumi/xdoc", "italic green")
	url := pterm.Info.Sprintf("Find more information at: %s", link)
	return fmt.Sprintf("\n%s%s\n%s\n", header, logo, url)
}
