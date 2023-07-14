package model

type DingtalkIo interface {
	RobotSendText(content string) error

	// 创建钉钉审批流程
	CreateDingTalkProcess(task *Task) (processID string, err error)
}
