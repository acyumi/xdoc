package feishu

import (
	"context"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkdrive "github.com/larksuite/oapi-sdk-go/v3/service/drive/v1"
	"github.com/samber/oops"

	"github.com/acyumi/xdoc/component/constant"
)

func (c *ClientImpl) QueryDriveDocuments(typ constant.DocType, token string) (*DocumentNode, error) {
	// 调用【获取文件夹元数据】接口
	// https://open.feishu.cn/document/server-docs/docs/drive-v1/folder/get-folder-meta
	// 创建请求对象
	req := larkdrive.NewBatchQueryMetaReqBuilder().
		MetaRequest(larkdrive.NewMetaRequestBuilder().
			RequestDocs([]*larkdrive.RequestDoc{
				larkdrive.NewRequestDocBuilder().
					DocToken(token).
					DocType(string(typ)).
					Build(),
			}).
			WithUrl(true).
			Build()).
		Build()
	// 发起请求
	resp, err := SendWithRetry(func(_ int) (*larkdrive.BatchQueryMetaResp, error) {
		return c.DriveBatchQuery(context.Background(), req)
	})
	// 处理错误
	if err != nil {
		return nil, oops.Wrap(err)
	}
	failedList := resp.Data.FailedList
	if len(failedList) > 0 {
		failed := failedList[0]
		code := larkcore.IntValue(failed.Code)
		token := larkcore.StringValue(failed.Token)
		return nil, oops.Errorf("获取文件夹元数据失败: code: %d, token: %s", code, token)
	}
	meta := resp.Data.Metas[0]
	dn := &DocumentNode{
		DocumentInfo: DocumentInfo{
			Name:  larkcore.StringValue(meta.Title),
			Type:  typ,
			Token: token,
		},
	}
	if typ == constant.DocTypeFolder {
		err = c.fetchDriveDescendant(dn, true, token, "")
		if err != nil {
			return nil, oops.Wrap(err)
		}
	} else {
		setFileExtension(dn, c.Args)
	}

	return dn, nil
}

func (c *ClientImpl) fetchDriveDescendant(dn *DocumentNode, hasChild bool, folderToken, pageToken string) error {
	if !hasChild {
		return nil
	}
	// 调用【获取文件夹中的文件清单】接口
	// https://open.feishu.cn/document/server-docs/docs/drive-v1/folder/list
	// 创建请求对象
	req := larkdrive.NewListFileReqBuilder().
		FolderToken(folderToken).
		PageToken(pageToken).
		OrderBy(`EditedTime`).
		Direction(`DESC`).
		PageSize(200).
		Build()
	// 发起请求
	resp, err := SendWithRetry(func(_ int) (*larkdrive.ListFileResp, error) {
		return c.DriveList(context.Background(), req)
	})
	// 处理错误
	if err != nil {
		return oops.Wrap(err)
	}

	// 查到的子节点添加到doc中，然后再递归查询子节点
	for _, file := range resp.Data.Files {
		child := c.fileToDocumentNode(file)
		dn.Children = append(dn.Children, child)
		// 如果类型是文件夹那就递归遍历
		if larkcore.StringValue(file.Type) != string(constant.DocTypeFolder) {
			continue
		}
		err = c.fetchDriveDescendant(child, true, larkcore.StringValue(file.Token), "")
		if err != nil {
			return oops.Wrap(err)
		}
	}

	if larkcore.BoolValue(resp.Data.HasMore) {
		pageToken = larkcore.StringValue(resp.Data.NextPageToken)
		err = c.fetchDriveDescendant(dn, true, folderToken, pageToken)
		if err != nil {
			return oops.Wrap(err)
		}
	}

	return nil
}

func (c *ClientImpl) fileToDocumentNode(file *larkdrive.File) *DocumentNode {
	var dn = &DocumentNode{}
	// 先判断文件夹类型，看是否可以下载，然后再判断有没有子节点
	dn.Name = larkcore.StringValue(file.Name)
	dn.Name = cleanName(dn.Name)
	dn.URL = larkcore.StringValue(file.Url)
	// 如果是快捷方式，则获取快捷方式的目标文件
	dn.Type = constant.DocType(larkcore.StringValue(file.Type))
	if dn.Type == constant.DocTypeShortcut {
		dn.Type = constant.DocType(larkcore.StringValue(file.ShortcutInfo.TargetType))
		dn.Token = larkcore.StringValue(file.ShortcutInfo.TargetToken)
	} else {
		dn.Token = larkcore.StringValue(file.Token)
	}
	setFileExtension(dn, c.Args)
	return dn
}
