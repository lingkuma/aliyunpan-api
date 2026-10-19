package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan-api/aliyunpan/apierror"
)

func printJSON(v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func printFileTable(list aliyunpan.FileList, showPath bool) {
	sorted := make(aliyunpan.FileList, 0, len(list))
	for _, f := range list {
		if f != nil {
			sorted = append(sorted, f)
		}
	}
	sort.SliceStable(sorted, func(i, j int) bool {
		return strings.ToLower(sorted[i].FileName) < strings.ToLower(sorted[j].FileName)
	})

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	if showPath {
		fmt.Fprintln(tw, "类型\t大小\t修改时间\t路径\t文件ID")
	} else {
		fmt.Fprintln(tw, "类型\t大小\t修改时间\t名称\t文件ID")
	}
	for _, f := range sorted {
		kind := "文件"
		size := humanSize(f.FileSize)
		if f.IsFolder() {
			kind = "目录"
			size = "-"
		}
		name := f.FileName
		if showPath {
			name = f.Path
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", kind, size, f.UpdatedAt, name, f.FileId)
	}
	tw.Flush()
}

func humanSize(n int64) string {
	if n <= 0 {
		return "0 B"
	}
	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	value := float64(n)
	idx := 0
	for value >= 1024 && idx < len(units)-1 {
		value /= 1024
		idx++
	}
	if idx == 0 {
		return fmt.Sprintf("%d %s", n, units[idx])
	}
	return fmt.Sprintf("%.1f %s", value, units[idx])
}

func wrapApiError(prefix string, e *apierror.ApiError) error {
	if e == nil {
		return nil
	}
	if prefix == "" {
		return fmt.Errorf("%s", e.Error())
	}
	return fmt.Errorf("%s: %s", prefix, e.Error())
}
