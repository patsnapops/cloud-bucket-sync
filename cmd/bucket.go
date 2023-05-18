package cmd

import (
	"cbs/config"
	"cbs/pkg/io"
	"cbs/pkg/model"
	"cbs/pkg/service"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/olekukonko/tablewriter"
	"github.com/patsnapops/noop/log"
	"github.com/spf13/cobra"
)

var (
	profile    string
	limit      int64
	recursive  bool
	include    string
	exclude    string
	timeBefore string
	timeAfter  string
	configPath string
	debug      bool
	queue      int64
	threadNum  int64
)
var (
	dryRun bool
	force  bool
)

func init() {
	bucketCmd.AddCommand(rmCmd)
	bucketCmd.AddCommand(lsCmd)

	bucketCmd.PersistentFlags().StringVarP(&profile, "profile", "p", "default", "profile name")
	bucketCmd.PersistentFlags().Int64VarP(&limit, "limit", "l", 0, "limit")
	bucketCmd.PersistentFlags().BoolVarP(&recursive, "recursive", "r", false, "recursive")
	bucketCmd.PersistentFlags().StringVarP(&include, "include", "i", "", "txt or txt,csv")
	bucketCmd.PersistentFlags().StringVarP(&exclude, "exclude", "e", "", "txt or txt,csv")
	bucketCmd.PersistentFlags().StringVarP(&timeBefore, "time-before", "b", "", "2023-03-01 00:00:00")
	bucketCmd.PersistentFlags().StringVarP(&timeAfter, "time-after", "a", "", "1992-03-01 00:00:00")
	bucketCmd.PersistentFlags().Int64VarP(&queue, "queue", "q", 0, "queue")

	rmCmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "dry run")
	rmCmd.Flags().BoolVarP(&force, "force", "f", false, "force")
	rmCmd.Flags().Int64VarP(&threadNum, "thread-num", "t", 1, "thread num")
}

var bucketCmd = &cobra.Command{
	Use:     "bucket",
	Aliases: []string{"b"},
	Long:    "you can cp,upload,download,rm buckets or objects by this command.",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var lsCmd = &cobra.Command{
	Use:  "ls",
	Long: "ls bucket or object with s3_url must start with s3://",
	Run: func(cmd *cobra.Command, args []string) {
		input := model.NewInput(recursive, include, exclude, timeBefore, timeAfter, limit)
		if debug {
			log.Default().WithLevel(log.DebugLevel).WithFilename("cbs.log").Init()
		} else {
			log.Default().WithLevel(log.InfoLevel).WithFilename("cbs.log").Init()
		}
		log.Debugf(tea.Prettify(input))
		switch len(args) {
		case 1:
			cliConfig := config.LoadCliConfig(configPath)
			bucketService := service.NewBucketService(io.NewBucketClient(cliConfig.Profiles))
			bucketName, prefix := parseBucketAndPrefix(args[0])
			timeStart := time.Now()
			if queue != 0 {
				objectsChan := make(chan model.ChanObject, queue)
				// 放弃table的展示，因为他不能体现chan的特性，会等到所有结果出来一起打印。
				var totalSize int64
				var totalObjects int64
				go bucketService.ListObjectsWithChan(profile, bucketName, prefix, input, objectsChan)
				for objectChan := range objectsChan {
					if objectChan.Error != nil {
						panic(objectChan.Error)
					}
					if objectChan.Obj != nil {
						totalSize += objectChan.Obj.Size
						if objectChan.Count > limit && limit != 0 {
							break
						}
						fmt.Printf("%s\t%s\t%22s\t%10s\t%34s\n",
							objectChan.Obj.Key, "", objectChan.Obj.LastModified.UTC().Format("2006-01-02 15:04:05"),
							FormatSize(objectChan.Obj.Size), objectChan.Obj.ETag)
					}
					if objectChan.Dir != nil {
						fmt.Printf("%s\t%s\t%s\t%s\t%s\n", *objectChan.Dir, "dir", "", "", "")
					}
					totalObjects = objectChan.Count
				}
				fmt.Printf("\nTotal: %d, Size: %s, Cost: %s\n", totalObjects, FormatSize(totalSize), time.Since(timeStart))
			} else {
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
				log.Debugf("list objects cost %s", time.Since(timeStart))
			}
		default:
			cmd.Help()
		}
	},
}

var rmCmd = &cobra.Command{
	Use:  "rm",
	Long: "rm bucket or object with s3_url must start with s3://\nrm default use --queue 1000 reduce memory usage and loading time.",
	Run: func(cmd *cobra.Command, args []string) {
		input := model.NewInput(recursive, include, exclude, timeBefore, timeAfter, limit)
		if debug {
			log.Default().WithLevel(log.DebugLevel).WithFilename("cbs.log").Init()
		} else {
			log.Default().WithLevel(log.InfoLevel).WithFilename("cbs.log").Init()
		}

		switch len(args) {
		case 1:
			timeStart := time.Now()
			cliConfig := config.LoadCliConfig(configPath)
			bucketService := service.NewBucketService(io.NewBucketClient(cliConfig.Profiles))
			bucketName, prefix := parseBucketAndPrefix(args[0])
			// 默认使用chan方式删除
			if queue == 0 {
				queue = 1000
			}
			objectsChan := make(chan model.ChanObject, queue)
			// 放弃table的展示，因为他不能体现chan的特性，会等到所有结果出来一起打印。
			var totalSize int64
			var totalObjects int64
			go bucketService.ListObjectsWithChan(profile, bucketName, prefix, input, objectsChan)
			// 实现多线程删除，当force为true时，不需要等待确认
			if !force {
				// 默认使用1个线程 否则在等待确认时候会不知道删的哪个。
				threadNum = 1
			}
			deleteChan := make(chan int, threadNum)
			for objectChan := range objectsChan {
				if objectChan.Error != nil {
					panic(objectChan.Error)
				}
				if objectChan.Obj != nil {
					totalSize += objectChan.Obj.Size
					if objectChan.Count > limit && limit != 0 {
						break
					}
					deleteChan <- 1
					go deleteObject(bucketService, bucketName, objectChan.Obj.Key, dryRun, force, deleteChan)
				}

				totalObjects = objectChan.Count
			}
			for {
				if len(deleteChan) == 0 {
					break
				}
			}
			fmt.Printf("\nTotal Objects: %d, Total Size: %s, Cost Time: %s\n", totalObjects, FormatSize(totalSize), time.Since(timeStart))
		default:
			cmd.Help()
		}
	},
}

func deleteObject(bucketService model.BucketContract, bucketName, key string, dryRun bool, force bool, deleteChan chan int) {
	defer func() {
		<-deleteChan
	}()
	if dryRun {
		fmt.Printf("dry run delete %s/%s\n", bucketName, key)
		return
	}
	if !force {
		fmt.Printf("delete %s/%s? [y/n]: ", bucketName, key)
		var confirm string
		fmt.Scanln(&confirm)
		if confirm == "y" || confirm == "Y" || confirm == "yes" || confirm == "YES" {
		} else {
			return
		}
	}
	err := bucketService.RmObject(profile, bucketName, key)
	if err != nil {
		fmt.Printf("delete %s/%s failed: %s\n", bucketName, key, err.Error())
		// TODO: remove panic for service
		panic(err)
	} else {
		fmt.Printf("delete %s/%s success\n", bucketName, key)
	}
}

// turn s3://bucket/prefix to bucket and prefix
func parseBucketAndPrefix(s3Path string) (bucket, prefix string) {
	bucket = strings.TrimPrefix(s3Path, "s3://")
	bucketS := strings.Split(bucket, "/")
	bucket = bucketS[0]
	if len(bucketS) > 1 {
		prefix = strings.Join(bucketS[1:], "/")
	} else {
		prefix = ""
	}
	log.Debugf("bucket: %s, prefix: %s", bucket, prefix)
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
