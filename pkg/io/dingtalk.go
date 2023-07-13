package io

import (
	"cbs/config"
	"cbs/pkg/model"
	"context"
	"strings"

	"github.com/alibabacloud-go/tea/tea"
	dt "github.com/patsnapops/go-dingtalk-sdk-wrapper"
	"github.com/patsnapops/noop/log"
)

type DingtalkService struct {
	Dt     *dt.DingTalkClient
	Config config.DingtalkConfig
}

func NewDingtalkClient(dtc *dt.DingTalkClient, config config.DingtalkConfig) model.DingtalkIo {
	if config.RobotToken == "" {
		log.Errorf("dingtalk robot token is empty,you may not receive any message!")
	}
	return &DingtalkService{
		Dt:     dtc,
		Config: config,
	}
}

// RobotSendText send text message to dingtalk robot
// accessToken 机器人的 token (https://oapi.dingtalk.com/robot/send?access_token=89ff497b0ccf42f4f73988bc4b5d9d2c63f04ae9561c747ead495ddb8c33a564)
// accessToken = 89ff497b0ccf42f4f73988bc4b5d9d2c63f04ae9561c747ead495ddb8c33a564
func (d *DingtalkService) RobotSendText(content string) error {
	// 发送通知
	err := d.Dt.RobotSvc().SendMessage(context.Background(), &dt.SendMessageRequest{
		AccessToken: d.Config.RobotToken,
		MessageContent: dt.MessageContent{
			MsgType: "text",
			Text: dt.TextBody{
				Content: content,
			},
		},
	})
	if strings.Contains(err.Error(), "ok") {
		return nil
	}
	return err
}

func (d *DingtalkService) CreateDingTalkProcess(task model.Task) (processID string, err error) {
	userid, departid, err := getUserInfoByName(task.Submitter)
	if err != nil {
		return "", err
	}
	input := &dt.CreateProcessInstanceInput{
		ProcessCode:      d.Config.ApproveInfo.ProcessCode,
		OriginatorUserID: userid,
		FormComponentValues: []dt.FormComponentValue{
			{
				Name:  "工单类型",
				Value: "auto",
			}, {
				Name:  "资源类型",
				Value: "s3Sync",
			}, {
				Name:  "其他信息",
				Value: tea.Prettify(task),
			},
		},
		DeptId: departid,
	}
	return d.Dt.Workflow.CreateProcessInstance(input)
}

func getUserInfoByName(name string) (string, string, error) {
	return "", "", nil
}
