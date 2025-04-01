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

package feishu

import (
	"context"
	"io"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkdrive "github.com/larksuite/oapi-sdk-go/v3/service/drive/v1"
	larkwiki "github.com/larksuite/oapi-sdk-go/v3/service/wiki/v2"

	"github.com/acyumi/xdoc/component/cloud"
	"github.com/acyumi/xdoc/component/progress"
)

type Client interface {
	cloud.Client

	// CreateTask 创建批量下载文件的任务
	CreateTask(docs []*DocumentNode, programConstructor func(progress.Stats) progress.IProgram) cloud.Task

	// DriveBatchQuery 【云盘】批量查询文件元信息
	DriveBatchQuery(ctx context.Context, req *larkdrive.BatchQueryMetaReq, options ...larkcore.RequestOptionFunc) (*larkdrive.BatchQueryMetaResp, error)
	// DriveList 【云盘】获取文件列表
	DriveList(ctx context.Context, req *larkdrive.ListFileReq, options ...larkcore.RequestOptionFunc) (*larkdrive.ListFileResp, error)
	// DriveDownload 【云盘】下载文件
	DriveDownload(ctx context.Context, req *larkdrive.DownloadFileReq, options ...larkcore.RequestOptionFunc) (*larkdrive.DownloadFileResp, error)

	// WikiGetNode 【知识库】获取节点信息
	WikiGetNode(ctx context.Context, req *larkwiki.GetNodeSpaceReq, options ...larkcore.RequestOptionFunc) (*larkwiki.GetNodeSpaceResp, error)
	// WikiGetSpace 【知识库】获取知识库信息
	WikiGetSpace(ctx context.Context, req *larkwiki.GetSpaceReq, options ...larkcore.RequestOptionFunc) (*larkwiki.GetSpaceResp, error)
	// WikiNodeList 【知识库】获取知识库节点列表
	WikiNodeList(ctx context.Context, req *larkwiki.ListSpaceNodeReq, options ...larkcore.RequestOptionFunc) (*larkwiki.ListSpaceNodeResp, error)

	// ExportCreate 【导出】创建导出任务
	ExportCreate(ctx context.Context, req *larkdrive.CreateExportTaskReq, options ...larkcore.RequestOptionFunc) (*larkdrive.CreateExportTaskResp, error)
	// ExportGet 【导出】获取导出任务信息，查看导出结果
	ExportGet(ctx context.Context, req *larkdrive.GetExportTaskReq, options ...larkcore.RequestOptionFunc) (*larkdrive.GetExportTaskResp, error)
	// ExportDownload 【导出】下载文件
	ExportDownload(ctx context.Context, req *larkdrive.DownloadExportTaskReq, options ...larkcore.RequestOptionFunc) (*larkdrive.DownloadExportTaskResp, error)
}

type IExporter interface {

	// doExport 创建导出任务。
	doExport(di *DocumentInfo) (string, error)

	// checkExport 查询导出任务结果。
	checkExport(di *DocumentInfo, ticket string) (*exportResult, progress.Status, error)

	// doDownloadExported 下载导出的文件。
	doDownloadExported(filePath, fileToken string) (io.Reader, error)

	// doDownloadDirectly 不需要经过导出操作，直接下载文件。
	doDownloadDirectly(filePath, fileToken string) (io.Reader, int64, error)
}
