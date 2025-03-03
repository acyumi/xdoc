package feishu

import (
	"fmt"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v5"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/stretchr/testify/suite"
)

const (
	Success = "success"
)

// 注册测试套件。
func TestRetryClientSuite(t *testing.T) {
	suite.Run(t, new(RetryClientSuite))
}

// 定义测试套件结构体。
type RetryClientSuite struct {
	suite.Suite
}

func (s *RetryClientSuite) SetupSuite() {
	initBackOff = func(ebo *backoff.ExponentialBackOff) {
		ebo.InitialInterval = 10 * time.Millisecond
		ebo.Multiplier = 2.0
		ebo.MaxInterval = 100 * time.Millisecond
		ebo.RandomizationFactor = 0.2
	}
}

func (s *RetryClientSuite) TearDownSuite() {
	initBackOff = initExponentialBackOff
}

// 实现测试方法。
func (s *RetryClientSuite) TestSendWithRetry() {
	maxAttemptCount = 3
	// 新增结构体类型测试分组
	type ResponseWithCountCodeError struct {
		larkcore.CodeError
		Count int
	}
	type ResponseWithCountCodeError1 struct {
		CodeError larkcore.CodeError
		Count     int
	}
	type ResponseWithoutCodeError struct {
		Data string
	}
	type ResponseWithWrongCodeError struct {
		CodeError string
	}
	type args[R any] struct {
		operation func(count int) (R, error)
	}
	type testCase[R any] struct {
		name     string
		args     args[R]
		wantResp R
		wantErr  string
	}
	tests := []testCase[any]{
		{
			name: `attempt[1]:success`,
			args: args[any]{
				operation: func(count int) (any, error) {
					if count == 1 {
						return Success, nil
					}
					return "", nil
				},
			},
			wantResp: Success,
			wantErr:  ``,
		},
		{
			name: `attempt[1]:(response 99991400, failed) -> attempt[2]:(response 0, success)`,
			args: args[any]{
				operation: func(count int) (any, error) {
					if count == 2 {
						return ResponseWithCountCodeError{
							Count:     count,
							CodeError: larkcore.CodeError{Code: 0, Msg: Success},
						}, nil
					}
					return ResponseWithCountCodeError{
						Count:     count,
						CodeError: larkcore.CodeError{Code: 99991400, Msg: "触发限流"},
					}, nil
				},
			},
			wantResp: ResponseWithCountCodeError{
				Count:     2,
				CodeError: larkcore.CodeError{Code: 0, Msg: Success},
			},
			wantErr: ``,
		},
		{
			name: `attempt[1]:(response 11232, failed), attempt[2]:(response 0, success)`,
			args: args[any]{
				operation: func(count int) (any, error) {
					if count == 2 {
						return ResponseWithCountCodeError{
							Count:     count,
							CodeError: larkcore.CodeError{Code: 0, Msg: Success},
						}, nil
					}
					return ResponseWithCountCodeError{
						Count:     count,
						CodeError: larkcore.CodeError{Code: 11232, Msg: "触发限流"},
					}, nil
				},
			},
			wantResp: ResponseWithCountCodeError{
				Count:     2,
				CodeError: larkcore.CodeError{Code: 0, Msg: Success},
			},
			wantErr: ``,
		},
		{
			name: "max attempts:(response 99991400, failed)",
			args: args[any]{
				operation: func(count int) (any, error) {
					return ResponseWithCountCodeError{
						Count:     count,
						CodeError: larkcore.CodeError{Code: 99991400, Msg: "触发限流"},
					}, nil
				},
			},
			wantResp: ResponseWithCountCodeError{
				Count:     maxAttemptCount,
				CodeError: larkcore.CodeError{Code: 99991400, Msg: "触发限流"},
			},
			wantErr: ``,
		},
		{
			name: "max attempts:(response 11232, failed)",
			args: args[any]{
				operation: func(count int) (any, error) {
					return ResponseWithCountCodeError{
						Count:     count,
						CodeError: larkcore.CodeError{Code: 11232, Msg: "触发限流"},
					}, nil
				},
			},
			wantResp: ResponseWithCountCodeError{
				Count:     maxAttemptCount,
				CodeError: larkcore.CodeError{Code: 11232, Msg: "触发限流"},
			},
			wantErr: ``,
		},
		{
			name: "max attempts:(response1 99991400, failed)",
			args: args[any]{
				operation: func(count int) (any, error) {
					return ResponseWithCountCodeError1{
						Count:     count,
						CodeError: larkcore.CodeError{Code: 99991400, Msg: "触发限流"},
					}, nil
				},
			},
			wantResp: ResponseWithCountCodeError1{
				Count:     maxAttemptCount,
				CodeError: larkcore.CodeError{Code: 99991400, Msg: "触发限流"},
			},
			wantErr: ``,
		},
		{
			name: "attempt[1]:(error 500, failed) -> no retry",
			args: args[any]{
				operation: func(count int) (any, error) {
					return "", &larkcore.CodeError{Code: 500}
				},
			},
			wantResp: "",
			wantErr:  `msg:,code:500`,
		},
		{
			name: "attempt[1]:(error 99991400, failed) -> no retry",
			args: args[any]{
				operation: func(count int) (any, error) {
					if count < maxAttemptCount {
						return "", &larkcore.CodeError{Code: 99991400, Msg: "触发限流"}
					}
					return Success, nil
				},
			},
			wantResp: "",
			wantErr:  `msg:触发限流,code:99991400`,
		},
		{
			name: "attempt[1]:(error 11232, failed) -> no retry",
			args: args[any]{
				operation: func(count int) (any, error) {
					if count < maxAttemptCount {
						return "", &larkcore.CodeError{Code: 11232, Msg: "触发限流"}
					}
					return Success, nil
				},
			},
			wantResp: "",
			wantErr:  `msg:触发限流,code:11232`,
		},
		{
			name: "struct response without CodeError, no retry",
			args: args[any]{
				operation: func(count int) (any, error) {
					return ResponseWithoutCodeError{Data: Success}, nil
				},
			},
			wantResp: ResponseWithoutCodeError{Data: Success},
			wantErr:  "",
		},
		{
			name: "struct response with invalid CodeError type, no retry",
			args: args[any]{
				operation: func(count int) (any, error) {
					return ResponseWithWrongCodeError{CodeError: "invalid_type"}, nil
				},
			},
			wantResp: ResponseWithWrongCodeError{CodeError: "invalid_type"},
			wantErr:  "",
		},
		{
			name: "slice response, no retry",
			args: args[any]{
				operation: func(count int) (any, error) {
					return []string{"xxx"}, nil
				},
			},
			wantResp: []string{"xxx"},
			wantErr:  "",
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			gotResp, err := SendWithRetry(tt.args.operation)
			if err != nil {
				// 这里如果err是oops.OopsError，则通过%+v格式化输出可以得到更多的错误细节信息
				actualErr := fmt.Sprintf("%+v", err)
				s.Equal(tt.wantErr, actualErr, tt.name)
			}
			s.Equal(tt.wantResp, gotResp, tt.name)
		})
	}
}
