package progress

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/samber/oops"
	"github.com/spf13/cast"
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
	TipsStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	GreenStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	OrangeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
)

type (
	Program struct {
		*tea.Program
		addChannel    chan addMsg
		updateChannel chan updateMsg
	}

	// Model 下载进度模型
	Model struct {
		pm         map[string]*progressStatus // 进度map，用于通过key取出指定的进度进行更新
		progresses []*progressStatus          // 进度列表，用于在滚动视图中按顺序渲染进度条
		viewport   viewport.Model             // 滚动视图
		stats      stats                      // 获取统计信息函数，用于在顶部显示
	}

	// Status 文件导出和下载状态
	Status string

	// progressStatus 下载进度及状态
	progressStatus struct {
		fileName string          // 文件的名称
		status   Status          // 文件的下载状态
		progress *progress.Model // 文件的进度条
		msg      string          // 自定义消息，用于每一行下载记录的最右侧
	}

	// stats 文件数量统计函数
	stats func(total, downloaded, failed int) string

	// addMsg 自定义消息类型，用于动态增加文件
	addMsg struct {
		key      string // 自定义文件唯一标识
		fileName string // 文件名
	}

	// updateMsg 自定义消息类型，用于更新进度
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

	Writer struct {
		FileKey  string   // 文件key
		FilePath string   // 文件写入路径
		Program  *Program // 程序，显示进度
		Total    int64    // 文件总大小
		Wrote    int64    // 文件已写盘大小
		Walked   float64  // 文件写盘前进度条已走过的占比，如 0.2
	}
)

func NewProgram(fileNames []string, stats stats) *Program {
	// 创建模型
	m := NewModel(fileNames, stats)
	// 创建 BubbleTea 程序
	p := &Program{
		Program:       tea.NewProgram(m),
		addChannel:    make(chan addMsg),
		updateChannel: make(chan updateMsg),
	}
	return p
}

// Add 添加新的待下载文件
func (p *Program) Add(key, fileName string) {
	go p.Send(addMsg{key: key, fileName: fileName})
}

// Update 更新文件下载进度
func (p *Program) Update(key string, progress float64, status Status, msgFormat ...any) {
	var msg string
	if len(msgFormat) > 0 {
		msg = fmt.Sprintf(cast.ToString(msgFormat[0]), msgFormat[1:]...)
	}
	go p.Send(updateMsg{key: key, progress: progress, status: status, msg: msg})
}

func NewModel(fileNames []string, stats stats) *Model {
	m := &Model{
		pm:         make(map[string]*progressStatus),
		progresses: make([]*progressStatus, len(fileNames)),
		viewport:   viewport.New(viewportWith, viewportHeight), // 初始化滚动视图
		stats:      stats,
	}
	for i := range fileNames {
		pm := progress.New(
			progress.WithDefaultGradient(),    // 使用默认渐变颜色
			progress.WithWidth(progressWidth), // 设置进度条宽度
		)
		m.progresses[i] = &progressStatus{fileName: fileNames[i], status: StatusAdded, progress: &pm}
	}
	return m
}

// Init 初始化模型
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update 更新逻辑
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

	// 处理滚动视图的更新
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View 渲染界面
func (m *Model) View() string {
	// TODO 添加大标题，如："Acyumi Feishu Doc Exporter"
	// 更新统计信息
	statsInfo := m.renderStats()
	return fmt.Sprintf("\n%s\n\n%s%s", statsInfo, m.viewport.View(), m.helpView())
}

// 渲染统计信息
func (m *Model) renderStats() string {
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

func (m *Model) helpView() string {
	tips := "\ntips => 鼠标滚轮/↑/↓: 上下滚动视图 • q/esc/ctrl+c: 退出"
	// 如果是windows系统，则显示提示信息
	if runtime.GOOS == "windows" {
		tips += "\n     => windows系统下，如果进度条没有色彩，请切换使用 PowerShell 或 cmd"
	}
	tips += "\n"
	return TipsStyle.Render(tips)
}

// 渲染内容
func (m *Model) renderContent() (view string) {
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

func (pw *Writer) WriteFile(reader io.Reader) error {
	// 创建目录
	dirPath := filepath.Dir(pw.FilePath)
	err := os.MkdirAll(dirPath, 0o755)
	if err != nil {
		return oops.Wrap(err)
	}
	// 保存文件
	file, err := os.OpenFile(pw.FilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return oops.Wrap(err)
	}
	// 将数据写入文件，同时更新进度
	teeReader := io.TeeReader(reader, pw)
	_, err = io.Copy(file, teeReader)
	if er := file.Close(); er != nil && err == nil {
		err = er
	}
	return oops.Wrap(err)
}

// Write 实现了 io.Writer 接口
func (pw *Writer) Write(p []byte) (int, error) {
	n := len(p)
	pw.Wrote += int64(n)
	pg := pw.Progress()
	// 更新进度
	pw.Program.Update(pw.FileKey, pg, StatusDownloading, "total: %d, wrote: %d", pw.Total, pw.Wrote)
	return n, nil
}

func (pw *Writer) Progress() float64 {
	return pw.Walked + float64(pw.Wrote)/float64(pw.Total)*(1.0-pw.Walked)
}
