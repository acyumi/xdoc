package progress

import (
	"errors"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/acyumi/xdoc/component/app"
)

func TestDemoSuite(t *testing.T) {
	suite.Run(t, new(DemoTestSuite))
}

type DemoTestSuite struct {
	suite.Suite
	mockProgram *MockProgram
}

func (s *DemoTestSuite) SetupTest() {
	s.mockProgram = NewMockProgram(s.T())
}

func (s *DemoTestSuite) TearDownTest() {
}

func (s *DemoTestSuite) TestDemo() {
	app.Sleep = func(duration time.Duration) { /* 单测时不需要睡眠等待 */ }
	c := NewTestClient(nil)
	args := c.GetArgs()
	s.Nil(args)
	err := c.Validate()
	s.Require().NoError(err)

	var tc TestClient
	tc.p = s.mockProgram

	// mock
	s.mockProgram.EXPECT().Add(mock.Anything, mock.Anything).Maybe()
	s.mockProgram.EXPECT().Update(mock.Anything, mock.Anything, StatusDownloading, mock.Anything).Maybe()
	s.mockProgram.EXPECT().Run().RunAndReturn(func() (tea.Model, error) {
		time.Sleep(500 * time.Millisecond)
		return nil, errors.New("demo error")
	})

	err = tc.DownloadDocuments("", "")
	s.Require().EqualError(err, "demo error")
}
