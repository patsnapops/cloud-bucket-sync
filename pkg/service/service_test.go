package service_test

import (
	"cbs/config"
	"cbs/pkg/io"
	"cbs/pkg/model"
	"cbs/pkg/service"
	"fmt"
	"testing"

	"github.com/patsnapops/noop/log"
)

var (
	bucketIo model.BucketIo
	workerC  model.WorkerContract
	requestC model.RequestContract
	mangerIo model.ManagerIo
	mana     model.ManagerContract
)

var (
	profileFrom = "cn9554"
	profileTo   = "us0066"
	bucketFrom  = "ops-9554"
	bucketTo    = "zhoushoujiantest"
)

func init() {
	cliConfig := config.LoadCliConfig("../../config/")
	managerConfig := config.LoadManagerConfig("../../config/")
	log.Default().WithLevel(log.DebugLevel).Init()
	bucketIo = io.NewBucketClient(cliConfig.Profiles)
	mangerIo = io.NewManagerClient(config.InitDB(*managerConfig, true))
	requestC = service.NewRequestService(cliConfig.Manager)
	workerC = service.NewWorkerService(bucketIo, requestC, 1)
	client := config.InitDt(*managerConfig, true)
	dingtalkC := io.NewDingtalkClient(client, managerConfig.Dingtalk)
	mana = service.NewManagerService(mangerIo, dingtalkC, true)
}

// request test
func TestRequest(t *testing.T) {
	recordId := "cc5e3618-6e6d-4fc5-a0be-84228a2ee52f"
	fmt.Println(requestC.RecordUpdateStatus(recordId, model.TaskSuccess))
}

// test CopyObjectClientSide
func TestCopyObjectClientSide(t *testing.T) {
	_, err := bucketIo.CopyObjectClientSide(profileFrom, profileTo, bucketFrom, model.Object{
		Key: "p.patsnap.info/static/popAPI.png",
	}, bucketTo, "123/123.png")
	if err != nil {
		fmt.Println(err)
	}
}

// head object test
func TestHeadObject(t *testing.T) {
	// object, err := bucketIo.HeadObject("default", bucketFrom, "redis-test-0")
	object, err := bucketIo.HeadObject("proxy", "ops-9554", "bin/123cbs")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(object)
}

// test GetObjectPart
func TestGetObjectPart(t *testing.T) {
	fmt.Println(bucketIo.GetSourceSplit("proxy", "ops-9554", "zhoushoujiantest/cbs", 3))
	fmt.Println(bucketIo.GetSourceSplit("proxy", "ops-9554",
		"zhoushoujiantest/1cbs", 3))
	fmt.Println(bucketIo.GetSourceSplit("proxy", "ops-9554",
		"zhoushoujiantest/2cbs", 3))
	fmt.Println(bucketIo.GetSourceSplit("proxy", "ops-9554",
		"zhoushoujiantest/3cbs", 3))
	fmt.Println(bucketIo.GetSourceSplit("proxy", "ops-9554",
		"zhoushoujiantest/5cbs", 3))
	fmt.Println(bucketIo.GetSourceSplit("proxy", "ops-9554", "bin/cbs", 3))
}

// test getPartSizeByPartNu
func TestGetPartSizeByPartNu(t *testing.T) {
	fmt.Println(bucketIo.GetPartSizeByPartNu("proxy", "search-cn-northwest-1", "eks/s-search-insightsfee-solr/20001010/index/_7cy_Lucene70_0.dvd", 1424))
}

// test getSourceSplit
func TestGetSourceSplit(t *testing.T) {
	fmt.Println(bucketIo.GetSourceSplit("proxy", "zhoushoujiantest", "1685198194.mkv", 7))
}

// test CopyObjectClientSide 大文件
// 必须带上准确的对象大小
func TestCopyObjectClientSideBig(t *testing.T) {
	bucketFrom = "lifescience-data-us-east-1"
	profileFrom = "us7478"
	// lessThan5G := "tidb-data/20230522/part-00082-2a050bd8-c083-4687-93f2-c1fc1fe6ae95-c000.json"
	moreThan5G := "tidb-data/20230522/part-00088-2a050bd8-c083-4687-93f2-c1fc1fe6ae95-c000.json"
	object, err := bucketIo.HeadObject(profileFrom, bucketFrom, moreThan5G)
	if err != nil {
		fmt.Println(err)
	}
	log.Debugf("size: %v", object.Size)
	_, err = bucketIo.CopyObjectClientSide(profileFrom, profileTo, bucketFrom, object, bucketTo, "123/part-00082-2a050bd8-c083-4687-93f2-c1fc1fe6ae95-c000.json")
	if err != nil {
		fmt.Println(err)
	}
}

// test CopyObjectServerSide
func TestCopyObjectServerSide(t *testing.T) {
	_, err := bucketIo.CopyObjectServerSide(profileFrom, bucketFrom, model.Object{
		Key: "p.patsnap.info/static/popAPI.png",
	}, bucketFrom, "zhoushoujiantest/123.png")
	if err != nil {
		fmt.Println(err)
	}
}

// test CalculateEvenSplitsByPartSize
func TestCalculateEvenSplitsByPartSize(t *testing.T) {
	model.CalculateEvenSplitsByPartSize(108152993, 17179870)
}

func TestCreateTaskWithApprove(t *testing.T) {
	task := &model.Task{
		Submitter:     "zhoushoujian",
		SourceProfile: "cn9554",
		TargetProfile: "us0066",
		SourceUrl:     "s3://ops-9554/zhoushoujiantest/1cbs",
		TargetUrl:     "s3://zhoushoujiantest/1cbs",
		WorkerTag:     "debug-cn",
	}
	taskID, err := mana.CreateTask(task)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(taskID)
}

func TestUpdateTask(t *testing.T) {
	fmt.Println(mangerIo.UpdateTask(&model.Task{
		Id:                 "d5391542-8306-4424-b5d4-379014c1b472",
		DingtalkInstanceId: "JGnMNypzRvSdUILJVkGwbA07561689316494",
		// IsServerSide:       tea.Bool(true),
		// IsSilence:          tea.Bool(true),
	}))
}
