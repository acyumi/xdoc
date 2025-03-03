package progress

import (
	"bytes"
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/suite"

	"github.com/acyumi/doc-exporter/component/constant"
)

func TestProgressSuite(t *testing.T) {
	suite.Run(t, new(ProgressTestSuite))
}

type ProgressTestSuite struct {
	suite.Suite
}

func (s *ProgressTestSuite) SetupTest() {
}

func (s *ProgressTestSuite) TearDownTest() {
}

func (s *ProgressTestSuite) TestURLStyleRender() {
	result := URLStyleRender("https://go.dev")
	s.Equal("\x1b]8;;https://go.dev\x1b\\", result)
}

type SafeBuffer struct {
	buf bytes.Buffer
	mu  sync.RWMutex
}

func (b *SafeBuffer) Write(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *SafeBuffer) Read(p []byte) (n int, err error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.buf.Read(p)
}

func (b *SafeBuffer) String() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.buf.String()
}

func (s *ProgressTestSuite) Test_program_Add_Update() {
	// 创建模型
	m := newModel(nil)
	// 创建 BubbleTea 程序
	ctx := context.Background()
	context.WithTimeout(ctx, 200*time.Second)
	var buf SafeBuffer
	var in bytes.Buffer
	p := &program{
		Program: tea.NewProgram(m, tea.WithInput(&in), tea.WithOutput(&buf)),
	}
	p.Add("fileKeyXyz", "fileNameXyz")
	p.Update("fileKeyXyz", 0.2, StatusDownloading, "fileNameXyz")
	go func() {
		for {
			time.Sleep(time.Millisecond)
			if m.executed.Load() && strings.Contains(buf.String(), "20%") {
				p.Quit()
				return
			}
		}
	}()
	m1, err := p.Run()
	s.Require().NoError(err)
	expected := `
总数量: 1, 已下载: 0, 未下载: 1, 已失败: 0

███░░░░░░░░░░░░  20%: [d] fileNameXyz (fileNameXyz)                                                 
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                                                                                           
tips => 鼠标滚轮/↑/↓: 上下滚动视图 • q/esc/ctrl+c: 退出                
     => windows系统下，如果进度条没有色彩，请切换使用 PowerShell 或 cmd
                                                                       `
	view := m1.View()
	s.Equal(expected, view)
}

func (s *ProgressTestSuite) Test_model_Update() {
	tests := []struct {
		name            string
		msg             tea.Msg
		setupMock       func(m *model, msg tea.Msg)
		expected        string
		expectedCommand tea.Cmd
	}{
		{
			name: "【tea.KeyMsg】1",
			msg: tea.KeyMsg{
				Type: tea.KeyCtrlC,
			},
			setupMock: func(m *model, msg tea.Msg) {},
			expected: `
总数量: 0, 已下载: 0, 未下载: 0, 已失败: 0

                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                                                                                           
tips => 鼠标滚轮/↑/↓: 上下滚动视图 • q/esc/ctrl+c: 退出                
     => windows系统下，如果进度条没有色彩，请切换使用 PowerShell 或 cmd
                                                                       `,
			expectedCommand: tea.Quit,
		},
		{
			name: "【tea.KeyMsg】2",
			msg: tea.KeyMsg{
				Type: tea.KeyEnter,
			},
			setupMock: func(m *model, msg tea.Msg) {},
			expected: `
总数量: 0, 已下载: 0, 未下载: 0, 已失败: 0

                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                                                                                           
tips => 鼠标滚轮/↑/↓: 上下滚动视图 • q/esc/ctrl+c: 退出                
     => windows系统下，如果进度条没有色彩，请切换使用 PowerShell 或 cmd
                                                                       `,
			expectedCommand: nil,
		},
		{
			name: "【addMsg】",
			msg: addMsg{
				key:      "fileKeyXyz",
				fileName: "fileNameXyz",
			},
			setupMock: func(m *model, msg tea.Msg) {},
			expected: `
总数量: 1, 已下载: 0, 未下载: 1, 已失败: 0

░░░░░░░░░░░░░░░   0%: [a] fileNameXyz                                                               
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                                                                                           
tips => 鼠标滚轮/↑/↓: 上下滚动视图 • q/esc/ctrl+c: 退出                
     => windows系统下，如果进度条没有色彩，请切换使用 PowerShell 或 cmd
                                                                       `,
		},
		{
			name: "【updateMsg】1",
			msg: updateMsg{
				key:      "fileKeyXyz",
				msg:      "msgXyz",
				progress: 0.5,
				status:   StatusDownloading,
			},
			setupMock: func(m *model, msg tea.Msg) {},
			expected: `
总数量: 0, 已下载: 0, 未下载: 0, 已失败: 0

                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                                                                                           
tips => 鼠标滚轮/↑/↓: 上下滚动视图 • q/esc/ctrl+c: 退出                
     => windows系统下，如果进度条没有色彩，请切换使用 PowerShell 或 cmd
                                                                       `,
		},
		{
			name: "【updateMsg】2",
			msg: updateMsg{
				key:      "fileKeyXyz",
				msg:      "msgXyz",
				progress: 1.0,
				status:   StatusCompleted,
			},
			setupMock: func(m *model, msg tea.Msg) {
				m.handleAddMsg(addMsg{key: "fileKeyXyz", fileName: "msgXyz"})
			},
			expected: `
总数量: 1, 已下载: 1, 未下载: 0, 已失败: 0

███████████████ 100%: [c] msgXyz (msgXyz)                                                           
                                                                                                    
全部文件已经下载完成，请按 q 或 esc 或 ctrl+c 退出                                                  
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                                                                                           
tips => 鼠标滚轮/↑/↓: 上下滚动视图 • q/esc/ctrl+c: 退出                
     => windows系统下，如果进度条没有色彩，请切换使用 PowerShell 或 cmd
                                                                       `,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			m := newModel(nil)
			tt.setupMock(m, tt.msg)
			m1, cmd := m.Update(tt.msg)
			view := m1.View()
			s.Equal(tt.expected, view, tt.name)
			if tt.expectedCommand == nil && cmd == nil {
				return
			}
			if cmd != nil {
				s.Require().NotNil(tt.expectedCommand, tt.name)
			}
			s.Equal(tt.expectedCommand(), cmd(), tt.name)
		})
	}
}

func (s *ProgressTestSuite) Test_model_handleAddMsg() {
	m := newModel(nil)
	cmd := m.Init()
	s.Nil(cmd)
	m1, cmd := m.handleAddMsg(addMsg{key: "fileKey", fileName: "fileNameXyz"})
	s.Nil(cmd)
	view := m1.View()
	expected := `
总数量: 1, 已下载: 0, 未下载: 1, 已失败: 0

░░░░░░░░░░░░░░░   0%: [a] fileNameXyz                                                               
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                                                                                           
tips => 鼠标滚轮/↑/↓: 上下滚动视图 • q/esc/ctrl+c: 退出                
     => windows系统下，如果进度条没有色彩，请切换使用 PowerShell 或 cmd
                                                                       `
	if runtime.GOOS == constant.Windows {
		s.Equal(expected, view)
		return
	}
	expected = strings.Replace(expected, "     => windows系统下，如果进度条没有色彩，请切换使用 PowerShell 或 cmd", "", 1)
	s.Equal(expected, view)
}

func (s *ProgressTestSuite) Test_model_view() {
	m := newModel(nil)
	cmd := m.Init()
	s.Nil(cmd)
	m1, cmd := m.handleAddMsg(addMsg{key: "fileKeyXyz", fileName: "fileNameXyz"})
	s.Nil(cmd)
	view := m1.View()
	expected := `
总数量: 1, 已下载: 0, 未下载: 1, 已失败: 0

░░░░░░░░░░░░░░░   0%: [a] fileNameXyz                                                               
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                                                                                           
tips => 鼠标滚轮/↑/↓: 上下滚动视图 • q/esc/ctrl+c: 退出                
     => windows系统下，如果进度条没有色彩，请切换使用 PowerShell 或 cmd
                                                                       `
	if runtime.GOOS == constant.Windows {
		s.Equal(expected, view)
		return
	}
	expected = strings.Replace(expected, "     => windows系统下，如果进度条没有色彩，请切换使用 PowerShell 或 cmd", "", 1)
	s.Equal(expected, view)
}

func (s *ProgressTestSuite) Test_model_renderStats() {
	m := newModel(nil)
	_, cmd := m.handleAddMsg(addMsg{key: "fileKeyXyz", fileName: "fileNameXyz"})
	s.Nil(cmd)
	view := m.renderStats()
	s.Equal("总数量: 1, 已下载: 0, 未下载: 1, 已失败: 0", view)

	p := m.progresses[0]
	p.progress.SetPercent(1.00)
	p.msg = "msgXyz"
	p.status = StatusCompleted
	view = m.renderStats()
	s.Equal("总数量: 1, 已下载: 1, 未下载: 0, 已失败: 0", view)

	p.status = StatusFailed
	view = m.renderStats()
	s.Equal("总数量: 1, 已下载: 0, 未下载: 0, 已失败: 1", view)

	m.stats = func(total, downloaded, failed int) string {
		return fmt.Sprintf("【Stats】总数量: %d, 已下载: %d, 未下载: %d, 已失败: %d", total, downloaded, total-downloaded-failed, failed)
	}
	view = m.renderStats()
	s.Equal("【Stats】总数量: 1, 已下载: 0, 未下载: 0, 已失败: 1", view)
}

func (s *ProgressTestSuite) Test_model_helpView() {
	m := newModel(nil)
	view := m.helpView()
	if runtime.GOOS == constant.Windows {
		s.Equal("                                                                       \n"+
			"tips => 鼠标滚轮/↑/↓: 上下滚动视图 • q/esc/ctrl+c: 退出                \n"+
			"     => windows系统下，如果进度条没有色彩，请切换使用 PowerShell 或 cmd\n"+
			"                                                                       ", view)
		return
	}
	s.Equal("                                                       \n"+
		"tips => 鼠标滚轮/↑/↓: 上下滚动视图 • q/esc/ctrl+c: 退出\n"+
		"                                                       ", view)
}

func (s *ProgressTestSuite) Test_model_renderContent() {
	m := newModel(nil)
	_, cmd := m.handleAddMsg(addMsg{key: "fileKeyXyz", fileName: "fileNameXyz"})
	s.Nil(cmd)
	view := m.renderContent()
	s.Equal("░░░░░░░░░░░░░░░   0%: [a] fileNameXyz\n", view)

	p := m.progresses[0]
	p.progress.SetPercent(1.00)
	p.msg = "msgXyz"
	p.status = StatusCompleted
	view = m.renderContent()
	s.Equal("███████████████ 100%: [c] fileNameXyz (msgXyz)\n\n全部文件已经下载完成，请按 q 或 esc 或 ctrl+c 退出", view)
}
