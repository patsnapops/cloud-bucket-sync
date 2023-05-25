package model

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"hash"
	"hash/crc64"
	"io"
	"log"
	"os"
	"time"
)

const (
	// MaxPartSize - maximum part size 5GiB for a single multipart upload
	// operation.
	MaxPartSize = 1024 * 1024 * 1024 * 5
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

// partsRequired is maximum parts possible with
// max part size of ceiling(MaxMultipartPutObjectSize / (MaxPartsCount - 1))
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

// tencent use crc64 ,aws use md5
func CalculateHash(path string, hashType string) (h string, b string) {
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
	return h, b
}
