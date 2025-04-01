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

	err = tc.DownloadDocuments(nil)
	s.Require().EqualError(err, "demo error")
}
