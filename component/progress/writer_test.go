package progress

import (
	"os"
	"strings"
	"testing"

	"github.com/samber/oops"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/acyumi/xdoc/component/app"
)

func TestWriterSuite(t *testing.T) {
	suite.Run(t, new(WriterTestSuite))
}

type WriterTestSuite struct {
	suite.Suite
	writer      Writer
	mockProgram *MockProgram
	memFs       *afero.Afero
}

func (s *WriterTestSuite) SetupSuite() {
	s.memFs = &afero.Afero{Fs: afero.NewMemMapFs()}
	app.Fs = s.memFs
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

type mockFs struct {
	afero.Fs
	openFileHasErr bool
}

func (f *mockFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	if f.openFileHasErr {
		return nil, oops.New("创建文件失败")
	}
	file, err := f.Fs.OpenFile(name, flag, perm)
	return &mockFile{File: file}, err
}

type mockFile struct {
	afero.File
}

func (f *mockFile) Close() error {
	return oops.New("关闭文件失败")
}

func (s *WriterTestSuite) TestWriter() {
	tests := []struct {
		name      string
		content   string
		setupMock func(name string)
		wantErr   string
	}{
		{
			name:    "正常写文件",
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
		{
			name:    "创建目录失败",
			content: "hello world",
			setupMock: func(name string) {
				app.Fs = &afero.Afero{Fs: afero.NewReadOnlyFs(app.Fs)}
				s.writer = Writer{
					FileKey:  "fileKeyXyz",
					FilePath: "/usr/xxx/filePathXyz.txt",
					Program:  s.mockProgram,
					Total:    0,
					Wrote:    0,
					Walked:   0.2,
				}
			},
			wantErr: "operation not permitted",
		},
		{
			name:    "创建文件失败",
			content: "hello world",
			setupMock: func(name string) {
				app.Fs = &afero.Afero{Fs: &mockFs{Fs: afero.NewMemMapFs(), openFileHasErr: true}}
				s.writer = Writer{
					FileKey:  "fileKeyXyz",
					FilePath: "/opt/xxx/filePathXyz.txt",
					Program:  s.mockProgram,
					Total:    0,
					Wrote:    0,
					Walked:   0.2,
				}
			},
			wantErr: "创建文件失败",
		},
		{
			name:    "关闭文件失败",
			content: "hello world",
			setupMock: func(name string) {
				app.Fs = &afero.Afero{Fs: &mockFs{Fs: afero.NewMemMapFs(), openFileHasErr: false}}
				s.writer = Writer{
					FileKey:  "fileKeyXyz",
					FilePath: "/opt/xxx/filePathXyz.txt",
					Program:  s.mockProgram,
					Total:    0,
					Wrote:    0,
					Walked:   0.2,
				}
				s.writer.Total = int64(len("hello world"))
				fn := func(pg float64) bool {
					return pg >= 0.20 && pg <= 1.00
				}
				s.mockProgram.EXPECT().Update(s.writer.FilePath, mock.MatchedBy(fn), StatusDownloading,
					"total: %d, wrote: %d", s.writer.Total, mock.Anything).Maybe()
			},
			wantErr: "关闭文件失败",
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer func() {
				app.Fs = s.memFs
			}()
			tt.setupMock(tt.name)
			reader := strings.NewReader(tt.content)
			err := s.writer.WriteFile(reader)
			if err != nil || tt.wantErr != "" {
				s.Require().Error(err, tt.name)
				s.Require().EqualError(err, tt.wantErr, tt.name)
			} else {
				s.Require().NoError(err, tt.name)
				actual, err := app.Fs.ReadFile(s.writer.FilePath)
				s.Require().NoError(err, tt.name)
				s.Equal(tt.content, string(actual), tt.name)
			}
			if _, ok := app.Fs.Fs.(*afero.ReadOnlyFs); ok {
				return
			}
			yes, err := app.Fs.Exists(s.writer.FilePath)
			s.Require().NoError(err, tt.name)
			if !yes {
				return
			}
			err = app.Fs.Remove(s.writer.FilePath)
			s.Require().NoError(err, tt.name)
		})
	}
}
