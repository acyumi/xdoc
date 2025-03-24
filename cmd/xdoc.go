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
