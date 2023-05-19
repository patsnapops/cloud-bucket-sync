package service

import (
	"cbs/config"
	"cbs/pkg/model"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/imroc/req"
	"github.com/patsnapops/noop/log"
)

type RequestService struct {
	Url string
}

func NewRequestService(config config.CliManager) model.RequestContract {
	config.Endpoint = strings.TrimSuffix(config.Endpoint, "/")
	config.ApiVersion = strings.TrimPrefix(config.ApiVersion, "/")
	if config.Endpoint == "" || config.ApiVersion == "" {
		log.Errorf(tea.Prettify(config))
	}
	return &RequestService{
		Url: fmt.Sprintf("%s/%s", config.Endpoint, config.ApiVersion),
	}
}

// get task
func (r *RequestService) TaskGet(taskID string) (*model.TaskResponse, error) {
	task := &model.TaskResponse{}
	data, err := doRequest(r.Url+"/task/"+taskID, "get", nil)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(data, &task)
	if task.ID == "" {
		return nil, fmt.Errorf("task not found")
	}
	return task, nil
}

// list task
func (r *RequestService) TaskList() ([]model.TaskResponse, error) {
	var tasks []model.TaskResponse
	data, err := doRequest(r.Url+"/task", "get", nil)
	if err != nil {
		return tasks, err
	}
	json.Unmarshal(data, &tasks)
	return tasks, nil
}

// create task
func (r *RequestService) TaskApply(args model.Task) (string, error) {
	var request req.Param
	taskID := ""
	data, err := json.Marshal(args)
	if err != nil {
		return taskID, err
	}
	err = json.Unmarshal(data, &request)
	if err != nil {
		return taskID, err
	}
	resP, err := doRequest(r.Url+"/task", "post", request)
	if err != nil {
		return taskID, err
	}
	json.Unmarshal(resP, &taskID)
	return taskID, nil
}

// exec task
func (r *RequestService) TaskExec(taskID string) (string, error) {
	input := req.Param{
		"id":     taskID,
		"action": "execute",
	}
	var recordID string
	resP, err := doRequest(r.Url+"/action", "post", input)
	if err != nil {
		return "", err
	}
	err = json.Unmarshal(resP, &recordID)
	if err != nil {
		fmt.Println(string(resP))
		return "", err
	}
	return recordID, nil
}

// ------------------ record ------------------

// success,failed,running,notallsuccess
func (r *RequestService) RecordUpdateStatus(recordID string, status model.Status) error {
	input := req.Param{
		"id":     recordID,
		"action": status,
	}
	_, err := doRequest(r.Url+"/action", "post", input)
	if err != nil {
		return err
	}
	return nil
}

func (r *RequestService) RecordUpdate(record *model.Record) error {
	// struct to map
	req := req.Param{}
	data, _ := json.Marshal(r)
	err := json.Unmarshal(data, &req)
	if err != nil {
		return err
	}
	_, err = doRequest(fmt.Sprintf(r.Url+"/record/%s", record.ID), "put", req)
	if err != nil {
		return err
	}
	return nil
}

func (r *RequestService) RecordQuery(input model.RecordInput) ([]model.Record, error) {
	var records []model.Record
	data, err := doRequest(r.Url+"/record", "get", nil)
	if err != nil {
		return records, err
	}
	err = json.Unmarshal(data, &records)
	if err != nil {
		return records, err
	}
	if len(records) == 0 {
		return records, fmt.Errorf("record not found")
	}
	return records, nil
}

func doRequest(url string, method string, param req.Param) ([]byte, error) {
	header := req.Header{
		"Accept":       "application/json",
		"Content-Type": "application/json;charset=UTF-8",
	}
	var r *req.Resp
	var err error
	switch method {
	case "get":
		r, err = req.Get(url, header, param)
	case "post":
		r, err = req.Post(url, header, req.BodyJSON(&param))
	case "put":
		r, err = req.Put(url, header, req.BodyJSON(&param))
	case "patch":
		r, err = req.Patch(url, header, req.BodyJSON(&param))
	case "delete":
		r, err = req.Delete(url, header, param)
	}
	if err != nil {
		return nil, err
	}
	switch r.Response().StatusCode {
	case 400:
		return r.Bytes(), fmt.Errorf("pop返回错误信息 %s", string(r.Bytes()))
	default:
		return r.Bytes(), err
	}
}
