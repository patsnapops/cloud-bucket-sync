package io

import (
	"cbs/config"
	"cbs/pkg/model"
	"fmt"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/patsnapops/noop/log"
)

type bucketClient struct {
	configProfiles []config.Profile
	sessions       map[string]*session.Session
}

func NewBucketClient(config []config.Profile) model.BucketIo {
	sessions, err := newSession(config)
	if err != nil {
		log.Errorf("new session error %s", err)
	}
	return bucketClient{
		configProfiles: config,
		sessions:       sessions,
	}
}

func newSession(configProfiles []config.Profile) (map[string]*session.Session, error) {
	sessions := make(map[string]*session.Session)
	for _, param := range configProfiles {
		log.Debugf(tea.Prettify(param))
		s3_conf := &aws.Config{
			Credentials: credentials.NewStaticCredentials(param.AK, param.SK, ""),
			Region:      aws.String(param.Region),
			// DisableSSL:       aws.Bool(true),  //trail 需要关闭
			S3ForcePathStyle: aws.Bool(false), //virtual-host style方式，不要修改
			// MaxRetries:       aws.Int(0),
			Endpoint: aws.String(param.Endpoint),
		}
		// log.Debugf(s3_conf.Endpoint)
		sess, err := session.NewSession(s3_conf)
		if err != nil {
			return sessions, err
		}
		sessions[param.Name] = sess
	}
	return sessions, nil

}

func (c bucketClient) ListObjects(profile, bucketName, prefix string, input model.Input) ([]string, []model.Object, error) {
	log.Debugf("list objects %s/%s", bucketName, prefix)
	objects := make([]model.Object, 0)
	dirs := make([]string, 0)
	if sess, ok := c.sessions[profile]; ok {
		svc := s3.New(sess)
		s3Input := &s3.ListObjectsV2Input{
			Bucket: aws.String(bucketName),
			Prefix: aws.String(prefix),
		}
		log.Debugf(tea.Prettify(input))
		if input.Recursive {
			s3Input.Delimiter = aws.String("")
		} else {
			s3Input.Delimiter = aws.String("/")
		}

		for {
			resp, err := svc.ListObjectsV2(s3Input)
			if err != nil {
				return dirs, objects, err
			}
			for _, content := range resp.Contents {
				objects = append(objects, model.Object{
					Key:          *content.Key,
					Size:         *content.Size,
					ETag:         *content.ETag,
					StorageClass: *content.StorageClass,
					LastModified: *content.LastModified,
				})
			}
			for _, commonPrefix := range resp.CommonPrefixes {
				dirs = append(dirs, *commonPrefix.Prefix)
			}
			if len(objects) > int(input.Limit) {
				return dirs, objects[:input.Limit], nil
			}
			if !*resp.IsTruncated {
				return dirs, objects, nil
			}
			s3Input.ContinuationToken = resp.NextContinuationToken
		}
	}
	return dirs, objects, fmt.Errorf("profile %s not found,please check cli.yaml config.", profile)

}

func (c bucketClient) RmObject(profile, bucketName, prefix string) error {
	return nil
}
