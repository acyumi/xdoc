package progress

import (
	"fmt"
	"runtime"
	"sync/atomic"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cast"

	"github.com/acyumi/xdoc/component/constant"
)

const (
	viewportWith   = 100 // 滚动视图宽度
	viewportHeight = 20  // 滚动视图高度
	progressWidth  = 20  // 进度条宽度

	StatusAdded       Status = "a"
	StatusExporting   Status = "x"
	StatusExported    Status = "p"
	StatusWaiting     Status = "w"
	StatusDownloading Status = "d"
	StatusCompleted   Status = "c"
	StatusFailed      Status = "f"
	StatusInterrupted Status = "i"
)

var (
	TipsStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	GreenStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	OrangeStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	BlueUnderlineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Underline(true)
	URLStyleRender     = func(url string) string {
		styledURL := BlueUnderlineStyle.Render(url)
		// 包裹OSC 8转义码以支持可点击链接
		// \x1b]8;;<URL>\x1b\\ 开始链接，\x1b]8;;\x1b\\ 结束链接
		return fmt.Sprintf("\x1b]8;;%s\x1b\\", styledURL)
	}
)

type (
	program struct {
		*tea.Program
	}

	// model 下载进度模型。
	model struct {
		executed   *atomic.Bool
		pm         map[string]*progressStatus // 进度map，用于通过key取出指定的进度进行更新
		progresses []*progressStatus          // 进度列表，用于在滚动视图中按顺序渲染进度条
		viewport   viewport.Model             // 滚动视图
		stats      Stats                      // 获取统计信息函数，用于在顶部显示
	}

	// Status 文件导出和下载状态。
	Status string

	// progressStatus 下载进度及状态。
	progressStatus struct {
		fileName string          // 文件的名称
		status   Status          // 文件的下载状态
		progress *progress.Model // 文件的进度条
		msg      string          // 自定义消息，用于每一行下载记录的最右侧
	}

	// Stats 文件数量统计函数。
	Stats func(total, downloaded, failed int) string

	// addMsg 自定义消息类型，用于动态增加文件。
	addMsg struct {
		key      string // 自定义文件唯一标识
		fileName string // 文件名
	}

	// updateMsg 自定义消息类型，用于更新进度。
	updateMsg struct {
		key      string  // 自定义文件唯一标识
		progress float64 // 下载进度 0.0->1.0
		// 状态，根据需要自定义，下面是示例
		// 正常下载：a added -> x exporting -> p exported -> w waiting -> d downloading -> c completed
		// 下载失败：a added -> x exporting -> p exported -> w waiting -> d downloading -> f failed
		// 导出失败：a added -> x exporting -> f failed
		status Status
		msg    string // 自定义消息，用于每一行下载记录的最右侧
	}
)

func NewProgram(stats Stats) IProgram {
	// 创建模型
	m := newModel(stats)
	// 创建 BubbleTea 程序
	return &program{
		Program: tea.NewProgram(m),
	}
}

// Add 添加新的待下载文件。
func (p *program) Add(key, fileName string) {
	go p.Send(addMsg{key: key, fileName: fileName})
}

// Update 更新文件下载进度。
func (p *program) Update(key string, progress float64, status Status, msgFormat ...any) {
	var msg string
	if len(msgFormat) > 0 {
		msg = fmt.Sprintf(cast.ToString(msgFormat[0]), msgFormat[1:]...)
	}
	go p.Send(updateMsg{key: key, progress: progress, status: status, msg: msg})
}

func newModel(stats Stats) *model {
	m := &model{
		executed:   &atomic.Bool{},
		pm:         make(map[string]*progressStatus),
		progresses: make([]*progressStatus, 0),
		viewport:   viewport.New(viewportWith, viewportHeight), // 初始化滚动视图
		stats:      stats,
	}
	return m
}

// Init 初始化模型。
func (m *model) Init() tea.Cmd {
	return nil
}

// Update 更新逻辑。
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// 按 q 或 ctrl+c 或 Esc 退出
		if msg.String() == "q" || msg.Type == tea.KeyCtrlC || msg.Type == tea.KeyEsc {
			fmt.Println(TipsStyle.Render("\n=> ") + OrangeStyle.Render("已退出") + TipsStyle.Render(" <=\n"))
			return m, tea.Quit
		}
	case updateMsg:
		// 更新指定文件的下载进度
		mp, ok := m.pm[msg.key]
		if !ok {
			return m, nil
		}
		mp.status = msg.status
		mp.msg = msg.msg
		mp.progress.SetPercent(msg.progress)
		if msg.progress >= 1.0 {
			mp.status = StatusCompleted
			mp.progress.PercentageStyle = GreenStyle
		}
		// 更新视图内容
		m.viewport.SetContent(m.renderContent())
		return m, nil
	case addMsg:
		return m.handleAddMsg(msg)
	}

	// 处理滚动视图的更新
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *model) handleAddMsg(msg addMsg) (tea.Model, tea.Cmd) {
	// 动态增加文件
	pm := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(progressWidth),
	)
	p := &progressStatus{fileName: msg.fileName, status: StatusAdded, progress: &pm}
	m.pm[msg.key] = p
	m.progresses = append(m.progresses, p)
	// 更新视图内容
	m.viewport.SetContent(m.renderContent())
	// 检查是否在底部，如果是，则在添加文件时滚动到底部
	if !m.viewport.PastBottom() && (m.viewport.ScrollPercent() > 0.9 ||
		m.viewport.YOffset+m.viewport.VisibleLineCount()+2 >= m.viewport.TotalLineCount()) {
		m.viewport.GotoBottom()
	}
	return m, nil
}

// View 渲染界面。
func (m *model) View() string {
	defer m.executed.Store(true)
	// TODO 添加大标题，如："Acyumi Feishu Doc Exporter"
	// 更新统计信息
	statsInfo := m.renderStats()
	return fmt.Sprintf("\n%s\n\n%s%s", statsInfo, m.viewport.View(), m.helpView())
}

// renderStats 渲染统计信息。
func (m *model) renderStats() string {
	total := len(m.progresses)
	downloaded := 0
	failed := 0
	for _, p := range m.progresses {
		if p.status == StatusCompleted {
			downloaded++
		} else if p.status == StatusFailed {
			failed++
		}
	}
	if m.stats != nil {
		return m.stats(total, downloaded, failed)
	}
	remaining := total - downloaded - failed
	statsInfo := fmt.Sprintf("总数量: %d, 已下载: %d, 未下载: %d, 已失败: %d", total, downloaded, remaining, failed)
	return TipsStyle.Render(statsInfo)
}

func (m *model) helpView() string {
	var tips string
	// 如果是windows系统，则显示提示信息
	if runtime.GOOS == constant.Windows {
		tips += "     => windows系统下，如果进度条没有色彩，请切换使用 PowerShell 或 cmd\n"
	}
	tips = "\ntips => 鼠标滚轮/↑/↓: 上下滚动视图 • q/esc/ctrl+c: 退出\n" + tips
	return TipsStyle.Render(tips)
}

// renderContent 渲染内容。
func (m *model) renderContent() (view string) {
	// 显示每个文件的下载进度（进度条）
	for _, p := range m.progresses {
		progressBlock := p.progress.ViewAs(p.progress.Percent())
		if p.msg == "" {
			view += fmt.Sprintf("%s: [%s] %s\n", progressBlock, p.status, p.fileName)
			continue
		}
		view += fmt.Sprintf("%s: [%s] %s (%s)\n", progressBlock, p.status, p.fileName, p.msg)
	}

	// 如果全部文件下载完成，显示提示信息
	total := len(m.progresses)
	completed := 0
	for _, p := range m.progresses {
		if p.status == StatusCompleted || p.status == StatusFailed {
			completed++
		}
	}
	if completed == total {
		view += "\n" + GreenStyle.Render("全部文件已经下载完成，请按 q 或 esc 或 ctrl+c 退出")
	}
	return view
}
