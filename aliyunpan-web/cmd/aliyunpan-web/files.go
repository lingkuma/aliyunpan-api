package main

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan-api/aliyunpan/apierror"
	"github.com/tickstep/aliyunpan-api/aliyunpan_web"
)

func normalizeRemotePath(p string) string {
	p = strings.ReplaceAll(strings.TrimSpace(p), "\\", "/")
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return path.Clean(p)
}

func (a *app) resolveEntity(wc *aliyunpan_web.WebPanClient, drive, arg string) (*aliyunpan.FileEntity, error) {
	p := normalizeRemotePath(arg)
	if strings.ContainsAny(p, "*?[") {
		list, apiErr := wc.MatchPathByShellPattern(drive, p)
		if apiErr != nil {
			return nil, wrapApiError("搜索路径失败 "+p, apiErr)
		}
		if list == nil || len(*list) == 0 {
			return nil, fmt.Errorf("路径不存在: %s", p)
		}
		if len(*list) > 1 {
			return nil, fmt.Errorf("路径 %s 匹配到 %d 个文件，请使用更精确的路径", p, len(*list))
		}
		return (*list)[0], nil
	}
	fi, apiErr := wc.FileInfoByPath(drive, p)
	if apiErr != nil {
		return nil, wrapApiError("获取文件信息失败 "+p, apiErr)
	}
	if fi == nil {
		return nil, fmt.Errorf("路径不存在: %s", p)
	}
	return fi, nil
}

func (a *app) resolveEntities(wc *aliyunpan_web.WebPanClient, drive string, args []string) (aliyunpan.FileList, error) {
	result := aliyunpan.FileList{}
	seen := map[string]bool{}
	for _, arg := range args {
		p := normalizeRemotePath(arg)
		if strings.ContainsAny(p, "*?[") {
			list, apiErr := wc.MatchPathByShellPattern(drive, p)
			if apiErr != nil {
				return nil, wrapApiError("搜索路径失败 "+p, apiErr)
			}
			if list == nil || len(*list) == 0 {
				return nil, fmt.Errorf("路径不存在: %s", p)
			}
			for _, item := range *list {
				if item != nil && !seen[item.FileId] {
					seen[item.FileId] = true
					result = append(result, item)
				}
			}
			continue
		}
		fi, apiErr := wc.FileInfoByPath(drive, p)
		if apiErr != nil {
			return nil, wrapApiError("获取文件信息失败 "+p, apiErr)
		}
		if fi == nil {
			return nil, fmt.Errorf("路径不存在: %s", p)
		}
		if !seen[fi.FileId] {
			seen[fi.FileId] = true
			result = append(result, fi)
		}
	}
	return result, nil
}

func (a *app) runLs(args []string) error {
	target := "/"
	if len(args) > 0 {
		target = args[0]
	}
	wc, err := a.client()
	if err != nil {
		return err
	}
	drive, err := a.drive()
	if err != nil {
		return err
	}
	p := normalizeRemotePath(target)
	fi, apiErr := wc.FileInfoByPath(drive, p)
	if apiErr != nil {
		return wrapApiError("获取目录信息失败 "+p, apiErr)
	}

	var list aliyunpan.FileList
	switch {
	case fi.IsFile():
		list = aliyunpan.FileList{fi}
	case a.opts.all:
		rootID := fi.FileId
		wc.FilesDirectoriesRecurseList(drive, p, func(depth int, fdPath string, fd *aliyunpan.FileEntity, e *apierror.ApiError) bool {
			if e != nil {
				fmt.Fprintf(os.Stderr, "警告: %s: %s\n", fdPath, e.Error())
				return true
			}
			if fd != nil && fd.FileId != rootID {
				list = append(list, fd)
			}
			return true
		})
	default:
		fl, apiErr := wc.FileListGetAll(&aliyunpan.FileListParam{DriveId: drive, ParentFileId: fi.FileId}, 0)
		if apiErr != nil {
			return wrapApiError("获取目录列表失败", apiErr)
		}
		list = fl
	}

	if a.opts.json {
		return printJSON(list)
	}
	printFileTable(list, a.opts.all)
	return nil
}

func (a *app) runStat(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("用法: stat <远程路径>")
	}
	wc, err := a.client()
	if err != nil {
		return err
	}
	drive, err := a.drive()
	if err != nil {
		return err
	}
	fi, err := a.resolveEntity(wc, drive, args[0])
	if err != nil {
		return err
	}
	if a.opts.json {
		return printJSON(fi)
	}
	kind := "文件"
	if fi.IsFolder() {
		kind = "目录"
	}
	fmt.Printf("名称: %s\n类型: %s\n路径: %s\n文件ID: %s\n父目录ID: %s\n网盘ID: %s\n大小: %s\n创建时间: %s\n修改时间: %s\n",
		fi.FileName, kind, fi.Path, fi.FileId, fi.ParentFileId, fi.DriveId, humanSize(fi.FileSize), fi.CreatedAt, fi.UpdatedAt)
	return nil
}

func (a *app) runSearch(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("用法: search <通配符路径>，例如 \"/照片/2026*\"")
	}
	wc, err := a.client()
	if err != nil {
		return err
	}
	drive, err := a.drive()
	if err != nil {
		return err
	}
	pattern := normalizeRemotePath(args[0])
	list, apiErr := wc.MatchPathByShellPattern(drive, pattern)
	if apiErr != nil {
		return wrapApiError("搜索失败", apiErr)
	}
	if a.opts.json {
		if list == nil {
			return printJSON(aliyunpan.FileList{})
		}
		return printJSON(*list)
	}
	if list == nil || len(*list) == 0 {
		fmt.Println("没有匹配的文件")
		return nil
	}
	printFileTable(*list, true)
	return nil
}

func (a *app) runMkdir(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("用法: mkdir <远程目录>")
	}
	wc, err := a.client()
	if err != nil {
		return err
	}
	drive, err := a.drive()
	if err != nil {
		return err
	}
	p := normalizeRemotePath(args[0])
	if p == "/" {
		return fmt.Errorf("根目录已存在")
	}
	res, apiErr := wc.MkdirByFullPath(drive, p)
	if apiErr != nil {
		return wrapApiError("创建目录失败", apiErr)
	}
	if a.opts.json {
		return printJSON(res)
	}
	fmt.Printf("已创建目录: %s (file_id=%s)\n", p, res.FileId)
	return nil
}

func (a *app) runRm(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("用法: rm <远程路径...>")
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
	params := make([]*aliyunpan.FileBatchActionParam, 0, len(targets))
	for _, fi := range targets {
		if fi.FileId == aliyunpan.DefaultRootParentFileId || fi.FileId == "" {
			return fmt.Errorf("不能删除根目录")
		}
		d := fi.DriveId
		if d == "" {
			d = drive
		}
		params = append(params, &aliyunpan.FileBatchActionParam{DriveId: d, FileId: fi.FileId})
	}
	results, apiErr := wc.FileDelete(params)
	if apiErr != nil {
		return wrapApiError("删除失败", apiErr)
	}
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

func (a *app) runMv(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("用法: mv <源路径...> <目标目录>")
	}
	wc, err := a.client()
	if err != nil {
		return err
	}
	drive, err := a.drive()
	if err != nil {
		return err
	}
	dstArg := args[len(args)-1]
	srcArgs := args[:len(args)-1]
	dst, err := a.resolveEntity(wc, drive, dstArg)
	if err != nil {
		return err
	}
	if !dst.IsFolder() {
		return fmt.Errorf("目标不是目录: %s", dstArg)
	}
	targets, err := a.resolveEntities(wc, drive, srcArgs)
	if err != nil {
		return err
	}
	params := make([]*aliyunpan.FileMoveParam, 0, len(targets))
	for _, fi := range targets {
		if fi.FileId == dst.FileId {
			return fmt.Errorf("源目录和目标目录相同: %s", fi.Path)
		}
		d := fi.DriveId
		if d == "" {
			d = drive
		}
		toDrive := dst.DriveId
		if toDrive == "" {
			toDrive = drive
		}
		params = append(params, &aliyunpan.FileMoveParam{
			DriveId:        d,
			FileId:         fi.FileId,
			ToDriveId:      toDrive,
			ToParentFileId: dst.FileId,
		})
	}
	results, apiErr := wc.FileMove(params)
	if apiErr != nil {
		return wrapApiError("移动失败", apiErr)
	}
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

func (a *app) runRename(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("用法: rename <远程路径> <新名称>")
	}
	wc, err := a.client()
	if err != nil {
		return err
	}
	drive, err := a.drive()
	if err != nil {
		return err
	}
	fi, err := a.resolveEntity(wc, drive, args[0])
	if err != nil {
		return err
	}
	newName := strings.TrimSpace(args[1])
	if newName == "" {
		return fmt.Errorf("新名称不能为空")
	}
	ok, apiErr := wc.FileRename(drive, fi.FileId, newName)
	if apiErr != nil {
		return wrapApiError("重命名失败", apiErr)
	}
	if a.opts.json {
		return printJSON(map[string]interface{}{"fileId": fi.FileId, "success": ok})
	}
	fmt.Printf("已重命名: %s -> %s\n", fi.FileName, newName)
	return nil
}

func (a *app) runStar(args []string, starred bool) error {
	if len(args) == 0 {
		if starred {
			return fmt.Errorf("用法: star <远程路径...>")
		}
		return fmt.Errorf("用法: unstar <远程路径...>")
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
	params := make([]*aliyunpan.FileBatchActionParam, 0, len(targets))
	for _, fi := range targets {
		d := fi.DriveId
		if d == "" {
			d = drive
		}
		params = append(params, &aliyunpan.FileBatchActionParam{DriveId: d, FileId: fi.FileId})
	}
	var results []*aliyunpan.FileBatchActionResult
	var apiErr *apierror.ApiError
	if starred {
		results, apiErr = wc.FileStarred(params)
	} else {
		results, apiErr = wc.FileUnstarred(params)
	}
	if apiErr != nil {
		return wrapApiError("收藏操作失败", apiErr)
	}
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
