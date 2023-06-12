package model

type DingtalkIo interface {
	RobotSendText(content string) error
}
