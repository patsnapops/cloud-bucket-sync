package cmd

import (
	"cbs/config"
	"cbs/pkg/io"
	"cbs/pkg/model"
	"cbs/pkg/service"
	"fmt"
	"os"
	"strings"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	profile *string
	input   model.Input
)

func init() {
	bucketCmd.AddCommand(rmCmd)
	bucketCmd.AddCommand(lsCmd)
	profile = pflag.StringP("profile", "p", "default", "profile name")
	limit := pflag.Int64P("limit", "l", 1000, "limit")
	recursive := pflag.BoolP("recursive", "r", false, "recursive")
	include := pflag.StringP("include", "i", "", "include")
	exclude := pflag.StringP("exclude", "e", "", "exclude")
	timeBefore := pflag.StringP("time-before", "b", "", "time before 2023-03-01 00:00:00")
	timeAfter := pflag.StringP("time-after", "a", "", "time after 1992-03-01 00:00:00")
	pflag.Parse()
	input = model.NewInput(tea.BoolValue(recursive), strings.Split(tea.StringValue(include), ","), strings.Split(tea.StringValue(exclude), ","), tea.StringValue(timeBefore), tea.StringValue(timeAfter), tea.Int64Value(limit))
	// fmt.Println(tea.Prettify(input))
}

var bucketCmd = &cobra.Command{
	Use:     "bucket",
	Aliases: []string{"b"},
	Long:    "bucket for cbs.you can cp,upload,download,rm buckets or objects by this command.",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var lsCmd = &cobra.Command{
	Use:  "ls",
	Long: "ls bucket or object",
	Run: func(cmd *cobra.Command, args []string) {
		switch len(args) {
		case 1:
			cliConfig := config.LoadCliConfig(tea.StringValue(configPath))
			bucketService := service.NewBucketService(io.NewBucketClient(cliConfig.Profiles))
			bucketName, prefix := parseBucketAndPrefix(args[0])
			dirs, objects, err := bucketService.ListObjects(tea.StringValue(profile), bucketName, prefix, input)
			if err != nil {
				panic(err)
			}
			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Key", "Type", "Last Modified", "Size", "ETag"})
			table.SetBorder(false)
			table.SetAlignment(tablewriter.ALIGN_RIGHT)
			var totalSize int64
			for _, dir := range dirs {
				table.Append([]string{dir, "dir", "", "", ""})
			}
			for _, object := range objects {
				table.Append([]string{object.Key, "", object.LastModified.UTC().Format("2006-01-02 15:04:05"), FormatSize(object.Size), object.ETag})
				totalSize += object.Size
			}
			table.SetFooter([]string{"", "", "Total Objects: ", FormatSize(totalSize), fmt.Sprintf("%d", len(objects))})
			table.Render()
			if len(objects) > 0 {
			}

		default:
			cmd.Help()
		}
	},
}

var rmCmd = &cobra.Command{
	Use:  "rm",
	Long: "rm bucket or object",
	Run: func(cmd *cobra.Command, args []string) {
		// cliConfig := config.LoadCliConfig(*configPath)
		// fmt.Println(tea.Prettify(cliConfig))
		switch len(args) {
		case 1:
			cliConfig := config.LoadCliConfig(tea.StringValue(configPath))
			bucketService := service.NewBucketService(io.NewBucketClient(cliConfig.Profiles))
			bucketName, prefix := parseBucketAndPrefix(args[0])
			err := bucketService.RmObject(tea.StringValue(profile), bucketName, prefix, input)
			if err != nil {
				panic(err)
			}
		default:
			cmd.Help()
		}
	},
}

// turn s3://bucket/prefix to bucket and prefix
func parseBucketAndPrefix(s3Path string) (bucket, prefix string) {
	bucket = strings.TrimPrefix(s3Path, "s3://")
	bucket = strings.Split(bucket, "/")[0]
	prefix = strings.TrimPrefix(s3Path, "s3://"+bucket+"/")
	return
}

func FormatSize(b int64) string {
	if b < 1024 {
		return fmt.Sprintf("%d B", b)
	} else if b < 1024*1024 {
		return fmt.Sprintf("%.2f KB", float64(b)/1024)
	} else if b < 1024*1024*1024 {
		return fmt.Sprintf("%.2f MB", float64(b)/(1024*1024))
	} else if b < 1024*1024*1024*1024 {
		return fmt.Sprintf("%.2f GB", float64(b)/(1024*1024*1024))
	} else {
		return fmt.Sprintf("%.2f TB", float64(b)/(1024*1024*1024*1024))
	}
}
