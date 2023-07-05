package cmd

import (
	"bufio"
	"bytes"
	"cbs/pkg/model"
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/olekukonko/tablewriter"
	"github.com/patsnapops/noop/log"
	"github.com/spf13/cobra"
)

const (
	// 上传
	upload = iota
	// 下载
	download
	// 桶到桶
	bucketToBucket
)

var (
	profileFrom string
	profileTo   string
	limit       int64
	recursive   bool // 是否递归
	include     string
	exclude     string
	timeBefore  string
	timeAfter   string
	queue       int64
	threadNum   int64
)
var (
	dryRun    bool
	force     bool
	file      string //支持指定txt csv的类型。
	dir       string //支持指定目录，然后处理目录下的所有的txt csv文件。
	errorFile string //错误文件汇总文本
)
var (
	// sync
	isServerSide bool // 是否使用服务端同步
)

func init() {
	bucketCmd.AddCommand(rmCmd)
	bucketCmd.AddCommand(lsCmd)
	bucketCmd.AddCommand(syncCmd)

	bucketCmd.PersistentFlags().StringVarP(&profileFrom, "profile_from", "p", "default", "profile name")
	bucketCmd.PersistentFlags().StringVarP(&profileTo, "profile_to", "", "default", "profile name")
	bucketCmd.PersistentFlags().Int64VarP(&limit, "limit", "l", 0, "limit")
	bucketCmd.PersistentFlags().BoolVarP(&recursive, "recursive", "r", false, "recursive")
	bucketCmd.PersistentFlags().StringVarP(&include, "include", "i", "", "string1 or string1,string2")
	bucketCmd.PersistentFlags().StringVarP(&exclude, "exclude", "e", "", "string1 or string1,string2")
	bucketCmd.PersistentFlags().StringVarP(&timeBefore, "time-before", "b", "", "\"2023-03-01 00:00:00\"")
	bucketCmd.PersistentFlags().StringVarP(&timeAfter, "time-after", "a", "", "\"1992-03-01 00:00:00\"")
	bucketCmd.PersistentFlags().Int64VarP(&queue, "queue", "q", 0, "queue")
	bucketCmd.PersistentFlags().Int64VarP(&threadNum, "thread-num", "t", 100, "thread num,every thread will sync a object")

	syncCmd.Flags().BoolVarP(&force, "force", "f", false, "force")
	syncCmd.Flags().BoolVarP(&isServerSide, "server-side", "s", false, "default use local network.")

	rmCmd.Flags().BoolVarP(&force, "force", "f", false, "force")
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

var syncCmd = &cobra.Command{
	Use:  "sync",
	Long: "sync bucket or object with s3_url must start with s3:// or cos://",
	Run: func(cmd *cobra.Command, args []string) {
		initConfig()
		input := model.SyncInput{
			Input:  model.NewInput(recursive, include, exclude, timeBefore, timeAfter, limit),
			Force:  force,
			DryRun: dryRun,
		}

		log.Debugf(tea.Prettify(input))
		switch len(args) {
		case 2:
			syncMode := getSyncMode(args[0], args[1])
			switch syncMode {
			case upload:
				syncLocalToBucket(args[0], args[1], input)
			case download:
				syncBucketToLocal(args[0], args[1], input)
			case bucketToBucket:
				syncBucketToBucket(args[0], args[1], input)
			default:
				cmd.Printf("s3_url must start with s3://")
			}
		default:
			cmd.Help()
		}
	},
}

// 兼容 cos 和 s3
func getSyncMode(sourceUrl, targetUrl string) int {
	if strings.HasPrefix(sourceUrl, "cos://") {
		sourceUrl = strings.Replace(sourceUrl, "cos://", "s3://", 1)
	}
	if strings.HasPrefix(targetUrl, "cos://") {
		targetUrl = strings.Replace(targetUrl, "cos://", "s3://", 1)
	}
	if strings.HasPrefix(sourceUrl, "s3://") && strings.HasPrefix(targetUrl, "s3://") {
		return bucketToBucket
	} else if strings.HasPrefix(sourceUrl, "s3://") && !strings.HasPrefix(targetUrl, "s3://") {
		return download
	} else if !strings.HasPrefix(sourceUrl, "s3://") && strings.HasPrefix(targetUrl, "s3://") {
		return upload
	}
	return -1
}

var lsCmd = &cobra.Command{
	Use:  "ls",
	Long: "ls bucket or object with s3_url must start with s3://",
	Run: func(cmd *cobra.Command, args []string) {
		initConfig()
		input := model.NewInput(recursive, include, exclude, timeBefore, timeAfter, limit)
		log.Debugf(tea.Prettify(input))
		switch len(args) {
		case 1:
			bucketName, prefix := model.ParseBucketAndPrefix(args[0])
			timeStart := time.Now()
			if queue != 0 {
				objectsChan := make(chan *model.ChanObject, queue)
				// 放弃table的展示，因为他不能体现chan的特性，会等到所有结果出来一起打印。
				var totalSize int64
				var totalObjects int64
				go bucketIo.ListObjectsWithChan(profileFrom, bucketName, prefix, input, objectsChan)
				for objectChan := range objectsChan {
					if objectChan.Obj != nil {
						totalSize += objectChan.Obj.Size
						if totalObjects > limit && limit != 0 {
							break
						}
						fmt.Printf("%s\t%s\t%22s\t%10s\t%34s\t%10s\n",
							objectChan.Obj.Key, "", objectChan.Obj.LastModified.UTC().Format("2006-01-02 15:04:05"),
							model.FormatSize(objectChan.Obj.Size), objectChan.Obj.ETag, objectChan.Obj.StorageClass)
					}
					if objectChan.Dir != nil {
						fmt.Printf("%s\t%s\t%s\t%s\t%s\t%s\n", *objectChan.Dir, "dir", "", "", "", "")
					}
					totalObjects++
				}
				fmt.Printf("\nTotal: %d, Size: %s, Cost: %s\n", totalObjects, model.FormatSize(totalSize), time.Since(timeStart))
			} else {
				dirs, objects, err := bucketIo.ListObjects(profileFrom, bucketName, prefix, input)
				if err != nil {
					panic(err)
				}
				table := tablewriter.NewWriter(os.Stdout)
				table.SetHeader([]string{"Key", "Type", "Last Modified", "Size", "ETag", "Class"})
				table.SetBorder(false)
				table.SetAlignment(tablewriter.ALIGN_RIGHT)
				var totalSize int64
				for _, dir := range dirs {
					table.Append([]string{dir, "dir", "", "", "", ""})
				}
				for _, object := range objects {
					table.Append([]string{object.Key, "", object.LastModified.UTC().Format("2006-01-02 15:04:05"), model.FormatSize(object.Size), object.ETag, object.StorageClass})
					totalSize += object.Size
				}
				table.SetFooter([]string{"", "", "", "Total Objects: ", model.FormatSize(totalSize), fmt.Sprintf("%d", len(objects))})
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
		initConfig()
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
			bucketName, prefix := model.ParseBucketAndPrefix(args[0])
			// 默认使用chan方式删除
			if queue == 0 {
				queue = 1000
			}
			objectsChan := make(chan *model.ChanObject, queue)
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
					go bucketIo.ListObjectsWithChan(profileFrom, bucketName, prefix, input, objectsChan)
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
				if objectChan.Obj != nil {
					totalSize += objectChan.Obj.Size
					if totalObjects >= limit && limit != 0 {
						break
					}
					deleteChan <- 1
					go deleteObject(profileFrom, bucketName, objectChan.Obj.Key, dryRun, force, deleteChan, f)
				}
				totalObjects++
			}
			for {
				if len(deleteChan) == 0 {
					break
				}
			}
			defer func() {
				fmt.Printf("\nTotal Objects: %d, Total Size: %s, Cost Time: %s\n", totalObjects, model.FormatSize(totalSize), time.Since(timeStart))
			}()
		default:
			cmd.Help()
		}
	},
}

func syncBucketToBucket(sourceUrl, targetUrl string, input model.SyncInput) {
	log.Warnf("use this mode ,you can use --dry-run --server-side --force")
	// sync bucket to bucket
	srcBucketName, srcPrefix := model.ParseBucketAndPrefix(sourceUrl)
	dstBucketName, dstPrefix := model.ParseBucketAndPrefix(targetUrl)
	// 获取源所有的key
	if queue == 0 {
		queue = 1000
	}
	objectsChan := make(chan *model.ChanObject, queue)
	go bucketIo.ListObjectsWithChan(profileFrom, srcBucketName, srcPrefix, input.Input, objectsChan)
	threadChan := make(chan int, threadNum)
	for object := range objectsChan {
		log.Debugf("%d", len(threadChan))
		log.Debugf("source object:%s", tea.Prettify(object))
		if object.Obj == nil {
			continue
		}
		targetKey := model.GetTargetKey(object.Obj.Key, srcPrefix, dstPrefix)
		if srcBucketName == dstBucketName && object.Obj.Key == targetKey {
			log.Infof("skip same bucket same key:%s", object.Obj.Key)
			continue
		}
		log.Debugf("%s => %s", object.Obj.Key, targetKey)
		if input.DryRun {
			log.Infof("%s => %s", object.Obj.Key, targetKey)
			log.Infof("dry run object:%s", tea.Prettify(object))
			continue
		}
		threadChan <- 1
		go func(object *model.ChanObject, targetKey string) {
			if !isServerSide {
				log.Infof("copy object client side")
				isSameEtag, err := bucketIo.CopyObjectClientSide(profileFrom, profileTo, srcBucketName, *object.Obj, dstBucketName, targetKey)
				if err != nil {
					log.Errorf("copy object error:%s", err.Error())
				}
				if isSameEtag {
					log.Infof("same Etag ,skip copy")
				}
			} else {
				log.Infof("copy object server side")
				isSameEtag, err := bucketIo.CopyObjectServerSide(profileFrom, srcBucketName, *object.Obj, dstBucketName, targetKey)
				if err != nil {
					log.Errorf("copy object error:%s", err.Error())
				}
				if isSameEtag {
					log.Infof("same Etag ,skip copy")
				}
			}
			<-threadChan
		}(object, targetKey)
	}
	for {
		if len(threadChan) == 0 {
			break
		}
		time.Sleep(1 * time.Second)
	}
}

func syncLocalToBucket(localPath, targetUrl string, input model.SyncInput) {
	startTime := time.Now()
	// sync local to bucket
	targetBucket, prefix := model.ParseBucketAndPrefix(targetUrl)
	// 获取本地路径的所有文件
	total := 0
	size := int64(0)
	if queue == 0 {
		queue = 1000
	}
	objectChan := make(chan *model.LocalFile, queue)
	go model.ListObjectsWithChanLocalRecursive(
		localPath, recursive, input, objectChan)
	threadChan := make(chan int, threadNum)
	for object := range objectChan {
		total++
		targetKey := model.GetTargetKey(object.Key, localPath, prefix)
		log.Debugf("%s => %s", object.Key, targetKey)
		if input.DryRun {
			log.Infof("%s => %s/%s", object.Key, targetBucket, targetKey)
			log.Debugf("dry run object:%s", tea.Prettify(object))
			continue
		}
		threadChan <- 1
		go func(object *model.LocalFile) {
			isSameEtag, err := bucketIo.CopyObjectLocalToRemote(profileTo, *object, targetBucket, targetKey)
			if err != nil {
				log.Errorf("copy object error:%s", err.Error())
			}
			if isSameEtag {
				log.Infof("skip same Etag:%s", object.Key)
			} else {
				size += object.Size
			}
			<-threadChan
		}(object)
	}
	for {
		if len(threadChan) == 0 {
			break
		}
	}
	log.Infof("sync local to bucket success total:%d size(ignore skip):%s cost:%s", total, model.FormatSize(size), time.Since(startTime).String())
}

func syncBucketToLocal(sourceUrl, targetPath string, input model.SyncInput) {
	// sync bucket to local
	bucketName, prefix := model.ParseBucketAndPrefix(sourceUrl)
	// 获取源所有的key
	if queue == 0 {
		queue = 1000
	}
	objectsChan := make(chan *model.ChanObject, queue)
	go bucketIo.ListObjectsWithChan(profileFrom, bucketName, prefix, input.Input, objectsChan)
	threadChan := make(chan int, threadNum)
	for object := range objectsChan {
		if object.Obj == nil {
			continue
		}
		targetPath := model.GetTargetKey(object.Obj.Key, prefix, targetPath)
		if input.DryRun {
			log.Infof("%s => %s", object.Obj.Key, targetPath)
			log.Debugf("dry run object:%s", tea.Prettify(object))
			continue
		}
		if !input.Force {
			// 没有覆盖要去检查目标文件的hash
			hash, base := model.CalculateHashForLocalFile(targetPath, "md5")
			log.Debugf("%s %s %s", object.Obj.ETag, hash, base)
			if strings.Contains(object.Obj.ETag, hash) && hash != "" {
				log.Infof("same etag for %s, skip.", targetPath)
				continue
			}
		}
		log.Debugf("%s => %s", object.Obj.Key, targetPath)
		threadChan <- 1
		go func(object *model.ChanObject) {
			defer func() {
				<-threadChan
			}()
			body, err := bucketIo.GetObject(profileFrom, bucketName, object.Obj.Key)
			if err != nil {
				log.Debugf("%s %s/%s", profileFrom, bucketName, object.Obj.Key)
				log.Errorf("download failed:%s", err.Error())
				return
			}
			// body 写入文件
			err = writeToFile(targetPath, &body)
			if err != nil {
				panic(err)
			}
			log.Infof("download success: %s", targetPath)
		}(object)
		for {
			if len(threadChan) == 0 {
				break
			}
			time.Sleep(1 * time.Second)
		}
	}
}

// byte数据写入到 targetkey 本地文件
func writeToFile(targetKey string, body *[]byte) error {
	// 创建目录
	err := os.MkdirAll(filepath.Dir(targetKey), 0755)
	if err != nil {
		return err
	}
	// 创建文件
	f, err := os.Create(targetKey)
	if err != nil {
		return err
	}
	defer f.Close()
	// 写入文件
	_, err = io.Copy(f, bytes.NewReader(*body))
	if err != nil {
		return err
	}
	return nil
}

func deleteObject(profile, bucketName, key string, dryRun bool, force bool, deleteChan chan int, f *os.File) {
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
	err := bucketIo.RmObject(profile, bucketName, key)
	if err != nil {
		fmt.Printf("delete %s/%s failed: %s\n", bucketName, key, err.Error())
		f.WriteString(key + " " + err.Error() + "\n")
	} else {
		fmt.Printf("delete %s/%s success\n", bucketName, key)
	}
}

// 在文件中读取bucket 放到objectsChan中
// 支持处理.txt结尾的文件，每个key一行
func readObjectsFromTxt(file string, input model.Input, objectsChan chan *model.ChanObject, closeChan bool) {
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
			objectsChan <- &model.ChanObject{
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
func readObjectsFromCsv(file string, input model.Input, objectsChan chan *model.ChanObject, closeChan bool) {
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
			objectsChan <- &model.ChanObject{
				Obj: &obj,
			}
		}
	}
	if closeChan {
		close(objectsChan)
	}
}

// 支持整个dir的文件遍历objects 放到objectsChan中
func readObjectsFromDir(dir string, input model.Input, objectsChan chan *model.ChanObject) {
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
