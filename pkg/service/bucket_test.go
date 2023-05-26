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
)

var (
	profileFrom = "cn9554"
	profileTo   = "us0066"
	bucketFrom  = "ops-9554"
	bucketTo    = "zhoushoujiantest"
)

func init() {
	cliConfig := config.LoadCliConfig("../../config/")
	log.Default().WithLevel(log.DebugLevel).Init()
	log.Debugf("cliConfig: %v", cliConfig.Profiles)
	bucketIo = io.NewBucketClient(cliConfig.Profiles)
	requestC := service.NewRequestService(cliConfig.Manager)
	workerC = service.NewWorkerService(bucketIo, requestC, 100)
}

// test CopyObjectClientSide
func TestCopyObjectClientSide(t *testing.T) {
	err := bucketIo.CopyObjectClientSide(profileFrom, profileTo, bucketFrom, model.Object{
		Key: "p.patsnap.info/static/popAPI.png",
	}, bucketTo, "123/123.png")
	if err != nil {
		fmt.Println(err)
	}
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
	err = bucketIo.CopyObjectClientSide(profileFrom, profileTo, bucketFrom, object, bucketTo, "123/part-00082-2a050bd8-c083-4687-93f2-c1fc1fe6ae95-c000.json")
	if err != nil {
		fmt.Println(err)
	}
}

// test CopyObjectServerSide
func TestCopyObjectServerSide(t *testing.T) {
	err := bucketIo.CopyObjectServerSide(profileFrom, bucketFrom, model.Object{
		Key: "p.patsnap.info/static/popAPI.png",
	}, bucketFrom, "zhoushoujiantest/123.png")
	if err != nil {
		fmt.Println(err)
	}
}
