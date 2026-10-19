package main

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/library-go/requester/rio"
)

func (a *app) runURL(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("用法: url <远程路径>")
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
	if fi.IsFolder() {
		return fmt.Errorf("目录不支持获取下载直链: %s", fi.Path)
	}
	d := fi.DriveId
	if d == "" {
		d = drive
	}
	res, apiErr := wc.GetFileDownloadUrl(&aliyunpan.GetFileDownloadUrlParam{DriveId: d, FileId: fi.FileId})
	if apiErr != nil {
		return wrapApiError("获取下载直链失败", apiErr)
	}
	if a.opts.json {
		return printJSON(res)
	}
	fmt.Printf("文件: %s\n大小: %s\n过期时间: %s\n下载地址: %s\n", fi.Path, humanSize(fi.FileSize), res.Expiration, res.Url)
	return nil
}

func (a *app) runDownload(args []string) error {
	if len(args) == 0 || len(args) > 2 {
		return fmt.Errorf("用法: dl <远程路径> [本地路径]，可用 --out 指定本地路径")
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
	if fi.IsFolder() {
		return fmt.Errorf("暂不支持直接下载目录，请先使用 ls --all 查看文件: %s", fi.Path)
	}

	local := ""
	if len(args) == 2 {
		local = args[1]
	} else if a.opts.out != "" {
		local = a.opts.out
	} else {
		local = safeLocalName(fi.FileName)
	}
	if st, statErr := os.Stat(local); statErr == nil && st.IsDir() {
		local = filepath.Join(local, safeLocalName(fi.FileName))
	}
	if _, statErr := os.Stat(local); statErr == nil && !a.opts.force {
		return fmt.Errorf("本地文件已存在: %s（使用 --force 覆盖）", local)
	}
	if dir := filepath.Dir(local); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建本地目录失败: %v", err)
		}
	}

	d := fi.DriveId
	if d == "" {
		d = drive
	}
	urlRes, apiErr := wc.GetFileDownloadUrl(&aliyunpan.GetFileDownloadUrlParam{DriveId: d, FileId: fi.FileId})
	if apiErr != nil {
		return wrapApiError("获取下载直链失败", apiErr)
	}
	if strings.HasPrefix(urlRes.Url, aliyunpan.IllegalDownloadUrlPrefix) {
		return fmt.Errorf("该文件下载地址被阿里云盘屏蔽，无法下载: %s", fi.Path)
	}

	f, err := os.OpenFile(local, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("创建本地文件失败: %v", err)
	}
	defer f.Close()

	fmt.Fprintf(os.Stderr, "开始下载 %s (%s)\n", fi.Path, humanSize(fi.FileSize))
	if apiErr := wc.DownloadFileDataAndSave(urlRes.Url, aliyunpan.FileDownloadRange{}, f); apiErr != nil {
		return wrapApiError("下载失败", apiErr)
	}
	st, _ := f.Stat()
	var size int64
	if st != nil {
		size = st.Size()
	}
	if a.opts.json {
		return printJSON(map[string]interface{}{
			"remotePath": fi.Path,
			"localPath":  local,
			"size":       size,
		})
	}
	fmt.Printf("下载完成: %s (%s)\n", local, humanSize(size))
	return nil
}

func (a *app) runUpload(args []string) error {
	if len(args) == 0 || len(args) > 2 {
		return fmt.Errorf("用法: upload <本地文件> [远程目录]")
	}
	local := args[0]
	remoteDir := "/"
	if len(args) == 2 {
		remoteDir = args[1]
	}

	f, err := os.Open(local)
	if err != nil {
		return fmt.Errorf("打开本地文件失败: %v", err)
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return fmt.Errorf("读取本地文件信息失败: %v", err)
	}
	if st.IsDir() {
		return fmt.Errorf("暂不支持上传目录: %s", local)
	}

	wc, err := a.client()
	if err != nil {
		return err
	}
	drive, err := a.drive()
	if err != nil {
		return err
	}
	dirPath := normalizeRemotePath(remoteDir)
	target, err := a.resolveEntity(wc, drive, dirPath)
	if err != nil {
		return err
	}
	if !target.IsFolder() {
		return fmt.Errorf("远程目标不是目录: %s", dirPath)
	}

	hasher := sha1.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return fmt.Errorf("计算文件SHA1失败: %v", err)
	}
	contentHash := strings.ToUpper(hex.EncodeToString(hasher.Sum(nil)))

	partSize := uploadPartSize(st.Size())
	var proofCode string
	if st.Size() > 0 {
		proofCode = aliyunpan.CalcProofCode(wc.GetAccessToken(), rio.NewFileReaderAtLen64(f), st.Size())
	}

	param := &aliyunpan.CreateFileUploadParam{
		Name:            st.Name(),
		DriveId:         drive,
		ParentFileId:    target.FileId,
		Size:            st.Size(),
		ContentHash:     contentHash,
		ContentHashName: "sha1",
		ProofCode:       proofCode,
		ProofVersion:    "v1",
		CheckNameMode:   "auto_rename",
		BlockSize:       partSize,
	}
	res, apiErr := wc.CreateUploadFile(param)
	if apiErr != nil {
		return wrapApiError("创建上传任务失败", apiErr)
	}
	if res.RapidUpload {
		if a.opts.json {
			return printJSON(res)
		}
		fmt.Printf("秒传成功: %s (file_id=%s)\n", res.FileName, res.FileId)
		return nil
	}

	totalParts := len(res.PartInfoList)
	fmt.Fprintf(os.Stderr, "开始上传 %s (%s)，分片数 %d\n", st.Name(), humanSize(st.Size()), totalParts)
	for idx, part := range res.PartInfoList {
		if part.UploadURL == "" {
			continue
		}
		offset := int64(part.PartNumber-1) * partSize
		chunkSize := partSize
		if offset+chunkSize > st.Size() {
			chunkSize = st.Size() - offset
		}
		if chunkSize <= 0 {
			continue
		}
		if a.opts.verbose {
			fmt.Fprintf(os.Stderr, "上传分片 %d/%d\n", idx+1, totalParts)
		}
		chunk := &aliyunpan.FileUploadChunkData{
			Reader:    io.NewSectionReader(f, offset, chunkSize),
			ChunkSize: chunkSize,
		}
		if apiErr := wc.UploadDataChunk(part.UploadURL, chunk); apiErr != nil {
			return fmt.Errorf("上传分片 %d/%d 失败: %s", idx+1, totalParts, apiErr.Error())
		}
	}

	complete, apiErr := wc.CompleteUploadFile(&aliyunpan.CompleteUploadFileParam{
		DriveId:  drive,
		FileId:   res.FileId,
		UploadId: res.UploadId,
	})
	if apiErr != nil {
		return wrapApiError("完成上传失败", apiErr)
	}
	if a.opts.json {
		return printJSON(complete)
	}
	fmt.Printf("上传完成: %s (file_id=%s, 大小=%s)\n", complete.Name, complete.FileId, humanSize(complete.Size))
	return nil
}

func uploadPartSize(size int64) int64 {
	const minPart = int64(4 * 1024 * 1024)
	const maxParts = int64(10000)
	partSize := minPart
	if size > 0 && size/partSize >= maxParts {
		need := size/maxParts + 1
		const mb = int64(1024 * 1024)
		partSize = ((need + mb - 1) / mb) * mb
	}
	return partSize
}

func safeLocalName(name string) string {
	replacer := strings.NewReplacer(
		"\\", "_", "/", "_", ":", "_", "*", "_",
		"?", "_", "\"", "_", "<", "_", ">", "_", "|", "_",
	)
	name = strings.TrimSpace(replacer.Replace(name))
	if name == "" {
		return "download"
	}
	return name
}
