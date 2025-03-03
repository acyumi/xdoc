package feishu

import (
	"context"
	"time"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkwiki "github.com/larksuite/oapi-sdk-go/v3/service/wiki/v2"
	"github.com/samber/oops"

	"github.com/acyumi/doc-exporter/component/constant"
	"github.com/acyumi/doc-exporter/component/progress"
)

func (c *ClientImpl) DownloadWikiDocuments(token string) error {
	// 创建请求对象
	req := larkwiki.NewGetNodeSpaceReqBuilder().Token(token).ObjType(`wiki`).Build()
	// 发起请求
	resp, err := SendWithRetry(func(_ int) (*larkwiki.GetNodeSpaceResp, error) {
		return c.WikiGetNode(context.Background(), req)
	})
	// 处理错误
	if err != nil {
		return oops.Wrap(err)
	}

	node := resp.Data.Node
	di := c.wikiNodeToDocumentNode(node)

	hasChild := larkcore.BoolValue(node.HasChild)
	err = c.fetchWikiDescendant(di, hasChild, di.SpaceID, di.NodeToken, "")
	if err != nil {
		return oops.Wrap(err)
	}

	task := c.CreateTask(di, progress.NewProgram)
	return doExportAndDownload(task)
}

func (c *ClientImpl) DownloadWikiSpaceDocuments(spaceID string) error {
	req := larkwiki.NewGetSpaceReqBuilder().SpaceId(spaceID).Lang(`zh`).Build()
	// 发起请求
	resp, err := SendWithRetry(func(_ int) (*larkwiki.GetSpaceResp, error) {
		return c.WikiGetSpace(context.Background(), req)
	})
	// 处理错误
	if err != nil {
		return oops.Wrap(err)
	}
	name := larkcore.StringValue(resp.Data.Space.Name)
	var di = &DocumentNode{DocumentInfo: DocumentInfo{Name: name, SpaceID: spaceID, Token: spaceID, Type: constant.DocTypeFolder}}
	err = c.fetchWikiDescendant(di, true, di.SpaceID, di.NodeToken, "")
	if err != nil {
		return oops.Wrap(err)
	}

	task := c.CreateTask(di, progress.NewProgram)
	return doExportAndDownload(task)
}

func (c *ClientImpl) fetchWikiDescendant(di *DocumentNode, hasChild bool,
	spaceID, parentNodeToken, pageToken string) error {
	if !hasChild {
		return nil
	}
	timeout := time.Minute
	if time.Since(c.Args.StartTime) > time.Minute {
		return oops.Errorf("获取知识库文档信息超时: %s", timeout.String())
	}
	// 调用【获取知识空间子节点列表】接口
	// https://open.feishu.cn/document/server-docs/docs/wiki-v2/space-node/list
	// 创建请求对象
	req := larkwiki.NewListSpaceNodeReqBuilder().
		SpaceId(spaceID).
		PageSize(50).
		PageToken(pageToken).
		ParentNodeToken(parentNodeToken).
		Build()
	// 发起请求
	resp, err := SendWithRetry(func(_ int) (*larkwiki.ListSpaceNodeResp, error) {
		return c.WikiNodeList(context.Background(), req)
	})
	// 处理错误
	if err != nil {
		return oops.Wrap(err)
	}

	// 查到的子节点添加到doc中，然后再递归查询子节点
	for _, node := range resp.Data.Items {
		// 先判断文档类型，看是否可以下载
		child := c.wikiNodeToDocumentNode(node)
		di.Children = append(di.Children, child)
		// 然后再判断有没有子节点
		hasChild := larkcore.BoolValue(node.HasChild)
		err = c.fetchWikiDescendant(child, hasChild, spaceID, child.NodeToken, "")
		if err == nil {
			continue
		}
		return oops.Wrap(err)
	}

	if larkcore.BoolValue(resp.Data.HasMore) {
		pageToken = larkcore.StringValue(resp.Data.PageToken)
		err = c.fetchWikiDescendant(di, true, spaceID, parentNodeToken, pageToken)
		if err != nil {
			return oops.Wrap(err)
		}
	}

	return nil
}

func (c *ClientImpl) wikiNodeToDocumentNode(node *larkwiki.Node) *DocumentNode {
	var di = &DocumentNode{}
	di.Name = larkcore.StringValue(node.Title)
	di.Name = cleanName(di.Name)
	di.Type = constant.DocType(larkcore.StringValue(node.ObjType))
	di.Token = larkcore.StringValue(node.ObjToken)
	setFileExtension(di, c.Args)
	// 取节点token
	di.NodeToken = larkcore.StringValue(node.NodeToken)
	// 取wiki的知识空间ID
	di.SpaceID = larkcore.StringValue(node.SpaceId)
	return di
}
