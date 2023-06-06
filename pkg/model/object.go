package model

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"hash"
	"hash/crc64"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/patsnapops/noop/log"
)

const (
	// MaxPartSize - maximum part size 5GiB for a single multipart upload
	// operation.
	MaxPartSize = 1024 * 1024 * 1024 * 5

	MaxPartSizeForThread = 1024 * 1024 * 64 // 64MB

	// AbsMinPartSize - absolute minimum part size (5 MiB) below which
	// a part in a multipart upload may not be uploaded.
	AbsMinPartSize = 1024 * 1024 * 5
	// MinPartSize - minimum part size 5MiB for a single multipart upload
	// operation.
	MinPartSize = 1024 * 1024 * 16
	// MaxPartsCount - maximum number of parts for a single multipart upload
	// operation.
	MaxPartsCount = 10000
	// MaxMultipartPutObjectSize - maximum size 5TiB of object for
	// Multipart operation.
	MaxMultipartPutObjectSize = 1024 * 1024 * 1024 * 1024 * 5
	// MaxSinglePutObjectSize - maximum size 5GiB of object per PUT
	// operation.
	MaxSinglePutObjectSize = 1024 * 1024 * 1024 * 5
)

type Object struct {
	Key          string
	Size         int64
	ETag         string
	StorageClass string
	LastModified time.Time
}

type ChanObjects struct {
	Key          *string
	Size         *int64
	ETag         *string
	LastModified *time.Time
	StorageClass *string
	// Owner        *Owner
	// ChecksumAlgorithm []*string
}

// type Owner struct {
// 	DisplayName *string
// 	ID          *string
// }

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

func ToInt64(s string) (int64, error) {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid int64: %s", s)
	}
	return i, nil
}

// 依据Etag获取源的分片数
// 7c33fc2d3a6e1a92e5eaa20bc9bf030a-49
func GetPartsCount(etag string) (int64, error) {
	// 如不不包含'-'，则为单个分片
	// 处理掉双引号
	etag = strings.Replace(etag, "\"", "", -1)
	log.Debugf("GetPartsCount: %s", etag)
	if !strings.Contains(etag, "-") {
		return 1, nil
	} else {
		return ToInt64(strings.Split(etag, "-")[1])
	}
}

// 如果有源文件带上的etag分片数，返回分片数，否则根据源文件大小计算分片数
func PartsRequired(size int64) int64 {
	var MaxPartSize int
	if size < MinPartSize*9999 {
		MaxPartSize = MinPartSize
	} else if size < MinPartSize*4*9999 {
		MaxPartSize = MinPartSize * 4
	} else {
		MaxPartSize = MaxMultipartPutObjectSize / (MaxPartsCount - 1)
	}

	r := size / int64(MaxPartSize)
	if size%int64(MaxPartSize) > 0 {
		r++
	}
	return r
}

// calculateEvenSplits - computes splits for a source and returns
// start and end index slices. Splits happen evenly to be sure that no
// part is less than 5MiB, as that could fail the multipart request if
// it is not the last part.
func CalculateEvenSplits(size int64) (startIndex, endIndex []int64) {
	var start int64
	if size == 0 {
		return
	}
	reqParts := PartsRequired(size)
	startIndex = make([]int64, reqParts)
	endIndex = make([]int64, reqParts)
	if start == -1 {
		start = 0
	}
	quot, rem := size/reqParts, size%reqParts
	nextStart := start
	for j := int64(0); j < reqParts; j++ {
		curPartSize := quot
		if j < rem {
			curPartSize++
		}

		cStart := nextStart
		cEnd := cStart + curPartSize - 1
		nextStart = cEnd + 1

		startIndex[j], endIndex[j] = cStart, cEnd
	}
	return
}

// 类似CalculateEvenSplits 通过指定分片数量计算分片
// 放弃使用，改为 CalculateEvenSplitsByPartSize
// func CalculateEvenSplitsByParts(size int64, parts int64) (startIndex, endIndex []int64) {
// 	var start int64
// 	if size == 0 {
// 		return
// 	}
// 	reqParts := parts
// 	startIndex = make([]int64, reqParts)
// 	endIndex = make([]int64, reqParts)
// 	if start == -1 {
// 		start = 0
// 	}
// 	quot, rem := size/reqParts, size%reqParts
// 	nextStart := start
// 	for j := int64(0); j < reqParts; j++ {
// 		curPartSize := quot
// 		if j < rem {
// 			curPartSize++
// 		}

// 		cStart := nextStart
// 		cEnd := cStart + curPartSize - 1
// 		nextStart = cEnd + 1

// 		startIndex[j], endIndex[j] = cStart, cEnd
// 	}
// 	return
// }

// 类似CalculateEvenSplits 通过指定分片大小计算分片，这样能确保源和目标的etag一致
func CalculateEvenSplitsByPartSize(size int64, partSize int64) (startIndex, endIndex []int64) {
	var start int64
	if size == 0 {
		return
	}
	reqParts := size / partSize
	if size%partSize > 0 {
		reqParts++
	}
	startIndex = make([]int64, reqParts)
	endIndex = make([]int64, reqParts)
	if start == -1 {
		start = 0
	}
	nextStart := start
	for j := int64(0); j < reqParts; j++ {
		curPartSize := partSize

		cStart := nextStart
		cEnd := cStart + curPartSize - 1
		if cEnd >= size {
			cEnd = size - 1
		}
		nextStart = cEnd + 1
		// log.Debugf("%d %d %d", cEnd-cStart, cStart, cEnd)
		startIndex[j], endIndex[j] = cStart, cEnd
	}
	return
}

// tencent use crc64 ,aws use md5
func CalculateHashForLocalFile(path string, hashType string) (h string, b string) {
	f, err := os.Open(path)
	if err != nil {
		return "", ""
	}
	defer f.Close()
	_, _ = f.Seek(0, 0)

	switch hashType {
	case "crc64":
		ecma := crc64.New(crc64.MakeTable(crc64.ECMA))
		w, _ := ecma.(hash.Hash)
		if _, err := io.Copy(w, f); err != nil {
			log.Fatal(err.Error())
			os.Exit(1)
		}

		res := ecma.Sum64()
		h = fmt.Sprintf("%d", res)
	case "md5":
		m := md5.New()
		w, _ := m.(hash.Hash)
		if _, err := io.Copy(w, f); err != nil {
			log.Fatal(err.Error())
			os.Exit(1)
		}

		res := m.Sum(nil)
		h = fmt.Sprintf("%x", res)
		b = base64.StdEncoding.EncodeToString(res)
	default:
		return "", ""
	}
	return fmt.Sprintf("\"%s\"", h), b
}
