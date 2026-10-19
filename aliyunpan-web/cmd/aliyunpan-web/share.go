package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan-api/aliyunpan_web"
)

func (a *app) runShare(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("用法: share create <远程路径...> | share list | share cancel <分享ID...>")
	}
	switch args[0] {
	case "create":
		return a.shareCreate(args[1:])
	case "list", "ls":
		return a.shareList()
	case "cancel", "rm":
		return a.shareCancel(args[1:])
	default:
		return fmt.Errorf("未知的 share 子命令: %s（支持 create/list/cancel）", args[0])
	}
}

func (a *app) shareCreate(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("用法: share create <远程路径...> [--pwd 1234] [--expire \"2026-12-31 23:59:59\"]")
	}
	pwd := a.opts.pwd
	if pwd != "" && len([]rune(pwd)) != 4 {
		return fmt.Errorf("分享密码必须是4个字符")
	}
	expire := a.opts.expire
	if len(expire) == 10 {
		expire += " 23:59:59"
	}

	wc, err := a.client()
	if err != nil {
		return err
	}
	drive, err := a.drive()
	if err != nil {
		return err
	}
	targets, err := a.resolveEntities(wc, drive, args)
	if err != nil {
		return err
	}
	fileIDs := make([]string, 0, len(targets))
	for _, fi := range targets {
		fileIDs = append(fileIDs, fi.FileId)
	}
	res, apiErr := wc.ShareLinkCreate(aliyunpan.ShareCreateParam{
		DriveId:    drive,
		SharePwd:   pwd,
		Expiration: expire,
		FileIdList: fileIDs,
	})
	if apiErr != nil {
		return wrapApiError("创建分享失败", apiErr)
	}
	if a.opts.json {
		return printJSON(res)
	}
	fmt.Printf("分享名称: %s\n分享链接: %s\n分享ID: %s\n", res.ShareName, res.ShareUrl, res.ShareId)
	if res.SharePwd != "" {
		fmt.Printf("提取码: %s\n", res.SharePwd)
	}
	if res.Expiration != "" {
		fmt.Printf("过期时间: %s\n", res.Expiration)
	}
	return nil
}

func (a *app) shareList() error {
	ui, err := a.userInfo()
	if err != nil {
		return err
	}
	wc, err := a.client()
	if err != nil {
		return err
	}
	list, apiErr := wc.ShareLinkList(ui.UserId)
	if apiErr != nil {
		return wrapApiError("获取分享列表失败", apiErr)
	}
	if a.opts.json {
		return printJSON(list)
	}
	if len(list) == 0 {
		fmt.Println("没有分享记录")
		return nil
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "名称\t分享ID\t链接\t提取码\t创建时间\t转存数")
	for _, s := range list {
		if s == nil {
			continue
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%d\n", s.ShareName, s.ShareId, s.ShareUrl, s.SharePwd, s.CreatedAt, s.SaveCount)
	}
	tw.Flush()
	return nil
}

func (a *app) shareCancel(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("用法: share cancel <分享ID...>")
	}
	wc, err := a.client()
	if err != nil {
		return err
	}
	results, apiErr := wc.ShareLinkCancel(args)
	if apiErr != nil {
		return wrapApiError("取消分享失败", apiErr)
	}
	if a.opts.json {
		return printJSON(results)
	}
	for _, r := range results {
		status := "失败"
		if r.Success {
			status = "成功"
		}
		fmt.Printf("%s: %s\n", r.Id, status)
	}
	return nil
}

func (a *app) runRecycle(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("用法: recycle list | restore <文件ID...> | rm <文件ID...> | clear")
	}
	wc, err := a.client()
	if err != nil {
		return err
	}
	drive, err := a.drive()
	if err != nil {
		return err
	}

	switch args[0] {
	case "list", "ls":
		list, apiErr := wc.RecycleBinFileListGetAll(&aliyunpan_web.RecycleBinFileListParam{DriveId: drive})
		if apiErr != nil {
			return wrapApiError("获取回收站列表失败", apiErr)
		}
		if a.opts.json {
			return printJSON(list)
		}
		if len(list) == 0 {
			fmt.Println("回收站为空")
			return nil
		}
		printFileTable(list, false)
		return nil

	case "restore":
		ids := args[1:]
		if len(ids) == 0 {
			return fmt.Errorf("用法: recycle restore <文件ID...>")
		}
		params := make([]*aliyunpan.FileBatchActionParam, 0, len(ids))
		for _, id := range ids {
			params = append(params, &aliyunpan.FileBatchActionParam{DriveId: drive, FileId: id})
		}
		results, apiErr := wc.RecycleBinFileRestore(params)
		if apiErr != nil {
			return wrapApiError("还原失败", apiErr)
		}
		return a.printBatch(results)

	case "rm", "delete":
		if !a.opts.force {
			return fmt.Errorf("彻底删除不可恢复，请追加 --force 确认")
		}
		ids := args[1:]
		if len(ids) == 0 {
			return fmt.Errorf("用法: recycle rm --force <文件ID...>")
		}
		params := make([]*aliyunpan.FileBatchActionParam, 0, len(ids))
		for _, id := range ids {
			params = append(params, &aliyunpan.FileBatchActionParam{DriveId: drive, FileId: id})
		}
		results, apiErr := wc.RecycleBinFileDelete(params)
		if apiErr != nil {
			return wrapApiError("彻底删除失败", apiErr)
		}
		return a.printBatch(results)

	case "clear":
		if !a.opts.force {
			return fmt.Errorf("清空回收站不可恢复，请追加 --force 确认")
		}
		res, apiErr := wc.RecycleBinFileClear(&aliyunpan_web.RecycleBinFileClearParam{DriveId: drive})
		if apiErr != nil {
			return wrapApiError("清空回收站失败", apiErr)
		}
		if a.opts.json {
			return printJSON(res)
		}
		fmt.Println("回收站已清空")
		return nil

	default:
		return fmt.Errorf("未知的 recycle 子命令: %s（支持 list/restore/rm/clear）", args[0])
	}
}

func (a *app) printBatch(results []*aliyunpan.FileBatchActionResult) error {
	if a.opts.json {
		return printJSON(results)
	}
	for _, r := range results {
		status := "失败"
		if r.Success {
			status = "成功"
		}
		fmt.Printf("%s: %s\n", r.FileId, status)
	}
	return nil
}
