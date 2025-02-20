package feishu

import (
	"context"
	"time"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkwiki "github.com/larksuite/oapi-sdk-go/v3/service/wiki/v2"
	"github.com/samber/oops"

	"acyumi.com/feishu-doc-exporter/component/argument"
)

func DownloadWikiDocuments(client *lark.Client, token string) error {
	// 创建请求对象
	req := larkwiki.NewGetNodeSpaceReqBuilder().Token(token).ObjType(`wiki`).Build()
	// 发起请求
	resp, err := SendWithRetry(func(_ int) (*larkwiki.GetNodeSpaceResp, error) {
		return client.Wiki.V2.Space.GetNode(context.Background(), req)
	})
	// 处理错误
	if err != nil {
		return oops.Wrap(err)
	}
	if !resp.Success() {
		return oops.Errorf("logId: %s, error response: \n%s", resp.RequestId(), larkcore.Prettify(resp.CodeError))
	}

	node := resp.Data.Node
	di := wikiNodeToDocumentNode(node)

	hasChild := larkcore.BoolValue(node.HasChild)
	err = fetchWikiDescendant(client, di, hasChild, di.SpaceID, di.NodeToken, "")
	if err != nil {
		return oops.Wrap(err)
	}

	return doExportAndDownload(client, di)
}

func DownloadWikiSpaceDocuments(client *lark.Client, spaceID string) error {
	req := larkwiki.NewGetSpaceReqBuilder().SpaceId(spaceID).Lang(`zh`).Build()
	// 发起请求
	resp, err := SendWithRetry(func(_ int) (*larkwiki.GetSpaceResp, error) {
		return client.Wiki.V2.Space.Get(context.Background(), req)
	})
	// 处理错误
	if err != nil {
		return oops.Wrap(err)
	}
	if !resp.Success() {
		return oops.Errorf("logId: %s, error response: \n%s", resp.RequestId(), larkcore.Prettify(resp.CodeError))
	}
	name := larkcore.StringValue(resp.Data.Space.Name)
	var di = &DocumentNode{DocumentInfo: DocumentInfo{Name: name, SpaceID: spaceID, Token: spaceID, Type: "folder"}}
	err = fetchWikiDescendant(client, di, true, di.SpaceID, di.NodeToken, "")
	if err != nil {
		return oops.Wrap(err)
	}
	return doExportAndDownload(client, di)
}

func fetchWikiDescendant(client *lark.Client, di *DocumentNode, hasChild bool,
	spaceID, parentNodeToken, pageToken string) error {
	if !hasChild {
		return nil
	}
	timeout := time.Minute
	if time.Since(argument.StartTime) > time.Minute {
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
		return client.Wiki.V2.SpaceNode.List(context.Background(), req)
	})
	// 处理错误
	if err != nil {
		return oops.Wrap(err)
	}
	if !resp.Success() {
		return oops.Errorf("logId: %s, error response: \n%s", resp.RequestId(), larkcore.Prettify(resp.CodeError))
	}

	// 查到的子节点添加到doc中，然后再递归查询子节点
	for _, node := range resp.Data.Items {
		// 先判断文档类型，看是否可以下载
		child := wikiNodeToDocumentNode(node)
		di.Children = append(di.Children, child)
		// 然后再判断有没有子节点
		hasChild := larkcore.BoolValue(node.HasChild)
		err = fetchWikiDescendant(client, child, hasChild, spaceID, child.NodeToken, "")
		if err == nil {
			continue
		}
		return oops.Wrap(err)
	}

	if larkcore.BoolValue(resp.Data.HasMore) {
		pageToken = larkcore.StringValue(resp.Data.PageToken)
		err = fetchWikiDescendant(client, di, true, spaceID, parentNodeToken, pageToken)
		if err != nil {
			return oops.Wrap(err)
		}
	}

	return nil
}

func wikiNodeToDocumentNode(node *larkwiki.Node) *DocumentNode {
	var di = &DocumentNode{}
	di.Name = larkcore.StringValue(node.Title)
	di.Name = cleanName(di.Name)
	di.Type = larkcore.StringValue(node.ObjType)
	di.Token = larkcore.StringValue(node.ObjToken)
	setFileExtension(di)
	// 取节点token
	di.NodeToken = larkcore.StringValue(node.NodeToken)
	// 取wiki的知识空间ID
	di.SpaceID = larkcore.StringValue(node.SpaceId)
	return di
}
