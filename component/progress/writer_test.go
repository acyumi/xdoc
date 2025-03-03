package progress

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

func TestWriterSuite(t *testing.T) {
	suite.Run(t, new(WriterTestSuite))
}

type WriterTestSuite struct {
	suite.Suite
	writer      Writer
	mockProgram *MockProgram
}

func (s *WriterTestSuite) SetupTest() {
	s.mockProgram = &MockProgram{}
	s.writer = Writer{
		FileKey:  "fileKeyXyz",
		FilePath: "/tmp/filePathXyz.txt",
		Program:  s.mockProgram,
		Total:    0,
		Wrote:    0,
		Walked:   0.2,
	}
}

func (s *WriterTestSuite) TearDownTest() {
}

func (s *WriterTestSuite) TestWriter() {
	tests := []struct {
		name      string
		content   string
		setupMock func(name string)
		wantErr   string
	}{
		{
			name:    "normal",
			content: "hello world",
			setupMock: func(name string) {
				s.writer.Total = int64(len("hello world"))
				fn := func(pg float64) bool {
					return pg >= 0.20 && pg <= 1.00
				}
				s.mockProgram.EXPECT().Update(s.writer.FilePath, mock.MatchedBy(fn), StatusDownloading,
					"total: %d, wrote: %d", s.writer.Total, mock.Anything).Maybe()
			},
			wantErr: "",
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock(tt.name)
			reader := strings.NewReader(tt.content)
			err := s.writer.WriteFile(reader)
			if err != nil || tt.wantErr != "" {
				s.Require().Error(err, tt.name)
			} else {
				s.Require().NoError(err, tt.name)
				actual, err := os.ReadFile(s.writer.FilePath)
				s.Require().NoError(err, tt.name)
				s.Equal(tt.content, string(actual), tt.name)
			}
			os.Remove(s.writer.FilePath)
		})
	}
}
