package cmd

import (
	"cbs/config"
	"cbs/pkg/io"
	"cbs/pkg/model"
	"cbs/pkg/service"
	"fmt"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/patsnapops/noop/log"
	"github.com/spf13/cobra"
)

var (
	profile    string
	limit      int64
	recursive  bool
	include    []string
	exclude    []string
	timeBefore string
	timeAfter  string
	configPath string
	debug      bool
)

func init() {
	bucketCmd.AddCommand(rmCmd)
	bucketCmd.AddCommand(lsCmd)
	bucketCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable debug mode")
	bucketCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "~/.cbs/", "config file dir,default is ~/.cbs/")

	bucketCmd.PersistentFlags().StringVarP(&profile, "profile", "p", "default", "profile name")
	bucketCmd.PersistentFlags().Int64VarP(&limit, "limit", "l", 1000, "limit")
	bucketCmd.PersistentFlags().BoolVarP(&recursive, "recursive", "r", false, "recursive")
	bucketCmd.PersistentFlags().StringArrayVarP(&include, "include", "i", []string{}, "'[aaa,sss]'")
	bucketCmd.PersistentFlags().StringArrayVarP(&exclude, "exclude", "e", []string{}, "'[aaa,sss]'")
	bucketCmd.PersistentFlags().StringVarP(&timeBefore, "time-before", "b", "", "time before 2023-03-01 00:00:00")
	bucketCmd.PersistentFlags().StringVarP(&timeAfter, "time-after", "a", "", "time after 1992-03-01 00:00:00")
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
		input := model.NewInput(recursive, include, exclude, timeBefore, timeAfter, limit)
		if debug == true {
			log.Default().WithLevel(log.DebugLevel).WithFilename("cbs.log").Init()
		} else {
			log.Default().WithLevel(log.InfoLevel).WithFilename("cbs.log").Init()
		}

		switch len(args) {
		case 1:
			cliConfig := config.LoadCliConfig(configPath)
			bucketService := service.NewBucketService(io.NewBucketClient(cliConfig.Profiles))
			bucketName, prefix := parseBucketAndPrefix(args[0])
			dirs, objects, err := bucketService.ListObjects(profile, bucketName, prefix, input)
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
		input := model.NewInput(recursive, include, exclude, timeBefore, timeAfter, limit)
		if debug == true {
			log.Default().WithLevel(log.DebugLevel).WithFilename("cbs.log").Init()
		} else {
			log.Default().WithLevel(log.InfoLevel).WithFilename("cbs.log").Init()
		}

		switch len(args) {
		case 1:
			cliConfig := config.LoadCliConfig(configPath)
			bucketService := service.NewBucketService(io.NewBucketClient(cliConfig.Profiles))
			bucketName, prefix := parseBucketAndPrefix(args[0])
			err := bucketService.RmObject(profile, bucketName, prefix, input)
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
