package cmd

import (
	"bufio"
	"cbs/pkg/model"
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
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
	queue      int64
	threadNum  int64
)
var (
	dryRun    bool
	force     bool
	file      string //支持指定txt csv的类型。
	dir       string //支持指定目录，然后处理目录下的所有的txt csv文件。
	errorFile string //错误文件汇总文本
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
	// 支持--file参数，可以从文件中读取bucket对象
	rmCmd.Flags().StringVarP(&file, "file", "", "", "object file path,file must be key per line.")
	// 支持--dir参数，可以从目录中读取bucket对象
	rmCmd.Flags().StringVarP(&dir, "dir", "", "", "must be end with / support *.txt,*.csv")
	// 支持--error-file参数，可以将错误的对象写入到文件中
	rmCmd.Flags().StringVarP(&errorFile, "error-file", "", ".cbs_rm_error.txt", "error file path")
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
		initApp()
		input := model.NewInput(recursive, include, exclude, timeBefore, timeAfter, limit)
		log.Debugf(tea.Prettify(input))
		switch len(args) {
		case 1:
			bucketName, prefix := parseBucketAndPrefix(args[0])
			timeStart := time.Now()
			if queue != 0 {
				objectsChan := make(chan model.ChanObject, queue)
				// 放弃table的展示，因为他不能体现chan的特性，会等到所有结果出来一起打印。
				var totalSize int64
				var totalObjects int64
				go bucketC.ListObjectsWithChan(profile, bucketName, prefix, input, objectsChan)
				for objectChan := range objectsChan {
					if objectChan.Error != nil {
						panic(objectChan.Error)
					}
					if objectChan.Obj != nil {
						totalSize += objectChan.Obj.Size
						if totalObjects > limit && limit != 0 {
							break
						}
						fmt.Printf("%s\t%s\t%22s\t%10s\t%34s\n",
							objectChan.Obj.Key, "", objectChan.Obj.LastModified.UTC().Format("2006-01-02 15:04:05"),
							FormatSize(objectChan.Obj.Size), objectChan.Obj.ETag)
					}
					if objectChan.Dir != nil {
						fmt.Printf("%s\t%s\t%s\t%s\t%s\n", *objectChan.Dir, "dir", "", "", "")
					}
					totalObjects++
				}
				fmt.Printf("\nTotal: %d, Size: %s, Cost: %s\n", totalObjects, FormatSize(totalSize), time.Since(timeStart))
			} else {
				dirs, objects, err := bucketC.ListObjects(profile, bucketName, prefix, input)
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
		initApp()
		// check 冲突
		log.Debugf("file: %s, dir: %s, timeAfter: %s, timeBefore: %s", file, dir, timeAfter, timeBefore)
		if (file != "" || dir != "") && (timeAfter != "" || timeBefore != "") {
			panic("not support time filter when use file or dir!")
		}
		input := model.NewInput(recursive, include, exclude, timeBefore, timeAfter, limit)

		f, err := os.OpenFile(errorFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			log.Panicf("open error file %s failed: %s", errorFile, err.Error())
		}
		defer f.Close()
		switch len(args) {
		case 1:
			timeStart := time.Now()
			bucketName, prefix := parseBucketAndPrefix(args[0])
			// 默认使用chan方式删除
			if queue == 0 {
				queue = 1000
			}
			objectsChan := make(chan model.ChanObject, queue)
			// 放弃table的展示，因为他不能体现chan的特性，会等到所有结果出来一起打印。
			if file != "" {
				if strings.HasSuffix(file, ".csv") {
					go readObjectsFromCsv(file, input, objectsChan, true)
				} else {
					go readObjectsFromTxt(file, input, objectsChan, true)
				}
			} else {
				if dir != "" {
					go readObjectsFromDir(dir, input, objectsChan)
				} else {
					go bucketC.ListObjectsWithChan(profile, bucketName, prefix, input, objectsChan)
				}
			}
			// 实现多线程删除，当force为true时，不需要等待确认
			if !force {
				// 默认使用1个线程 否则在等待确认时候会不知道删的哪个。
				threadNum = 1
			}
			deleteChan := make(chan int, threadNum)
			var totalSize int64
			var totalObjects int64
			for objectChan := range objectsChan {
				if objectChan.Error != nil {
					panic(objectChan.Error)
				}
				if objectChan.Obj != nil {
					totalSize += objectChan.Obj.Size
					if totalObjects >= limit && limit != 0 {
						break
					}
					deleteChan <- 1
					go deleteObject(bucketC, bucketName, objectChan.Obj.Key, dryRun, force, deleteChan, f)
				}
				totalObjects++
			}
			for {
				if len(deleteChan) == 0 {
					break
				}
			}
			defer func() {
				fmt.Printf("\nTotal Objects: %d, Total Size: %s, Cost Time: %s\n", totalObjects, FormatSize(totalSize), time.Since(timeStart))
			}()
		default:
			cmd.Help()
		}
	},
}

func deleteObject(bucketC model.BucketContract, bucketName, key string, dryRun bool, force bool, deleteChan chan int, f *os.File) {
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
	err := bucketC.RmObject(profile, bucketName, key)
	if err != nil {
		fmt.Printf("delete %s/%s failed: %s\n", bucketName, key, err.Error())
		f.WriteString(key + " " + err.Error() + "\n")
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

// 在文件中读取bucket 放到objectsChan中
// 支持处理.txt结尾的文件，每个key一行
func readObjectsFromTxt(file string, input model.Input, objectsChan chan model.ChanObject, closeChan bool) {
	f, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		key := scanner.Text()
		// 处理空格
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		obj := model.Object{
			Key:  key,
			Size: 0,
		}
		if model.ListObjectsWithFilter(obj, input) {
			log.Debugf("read key: %s", key)
			objectsChan <- model.ChanObject{
				Obj: &obj,
			}
		}

	}
	if closeChan {
		close(objectsChan)
	}
}

// 在文件中读取bucket 放到objectsChan中
// 支持处理.txt结尾的文件，每个key一行
func readObjectsFromCsv(file string, input model.Input, objectsChan chan model.ChanObject, closeChan bool) {
	f, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	csvReader := csv.NewReader(f)
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		key := record[1]
		// 处理空格
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		obj := model.Object{
			Key:  key,
			Size: 0,
		}
		log.Debugf("read key: %s", key)
		if model.ListObjectsWithFilter(obj, input) {
			objectsChan <- model.ChanObject{
				Obj: &obj,
			}
		}
	}
	if closeChan {
		close(objectsChan)
	}
}

// 支持整个dir的文件遍历objects 放到objectsChan中
func readObjectsFromDir(dir string, input model.Input, objectsChan chan model.ChanObject) {
	// 读取目录下的所有文件，只处理.txt .csv结尾的文件
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if strings.HasSuffix(file.Name(), ".txt") {
			readObjectsFromTxt(dir+"/"+file.Name(), input, objectsChan, false)
		} else if strings.HasSuffix(file.Name(), ".csv") {
			readObjectsFromCsv(dir+"/"+file.Name(), input, objectsChan, false)
		} else {
			continue
		}
	}
	close(objectsChan)
}
