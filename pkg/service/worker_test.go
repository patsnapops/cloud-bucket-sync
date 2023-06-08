package service_test

import (
	"cbs/pkg/model"
	"testing"

	"github.com/alibabacloud-go/tea/tea"
)

var (
	serverSideTask = model.Task{
		Name:          "serverSideTask",
		IsServerSide:  tea.Bool(true),
		SourceProfile: "cn9554",
		TargetProfile: "cn9554",
		SourceUrl:     "s3://ops-9554/zhoushoujiantest/",
		TargetUrl:     "s3://ops-9554/cbs/serverSideTask/",
		Include:       "",
		Exclude:       "",
		TimeBefore:    "",
		TimeAfter:     "",
		SyncMode:      "syncOnce",
		WorkerTag:     "aws-cn",
	}
)

// Test sync once
func TestSyncOnce(t *testing.T) {
	workerC.SyncOnce(serverSideTask, model.Record{})
}
