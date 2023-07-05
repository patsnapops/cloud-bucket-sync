package cmd

import (
	"cbs/pkg/model"
	"strings"

	"github.com/spf13/cobra"
)

var (
	expiresIn int64 //签名时效
	profile   string
)

func init() {
	presignCmd.Flags().Int64VarP(&expiresIn, "expiresIn", "e", 3600, "签名时效，单位秒")
	presignCmd.Flags().StringVarP(&profile, "profile", "p", "default", "指定profile")
	rootCmd.AddCommand(presignCmd)
}

var presignCmd = &cobra.Command{
	Use:     "presign",
	Aliases: []string{"ps"},
	Short:   "Presign a URL for an S3 object",
	Long:    `You know, for signing URLs`,
	Run: func(cmd *cobra.Command, args []string) {
		initConfig()
		switch len(args) {
		case 1:
			s3Url := args[0]
			if !isS3Url(s3Url) {
				panic("invalid s3 url")
			}
			presign(s3Url)
		default:
			cmd.Help()
		}
	},
}

func presign(s3Url string) {
	bucket, key := model.ParseBucketAndPrefix(s3Url)
	url, err := bucketIo.Presign(profile, bucket, key, expiresIn)
	if err != nil {
		panic(err)
	}
	println(url)
}

func isS3Url(s3Url string) bool {
	return strings.HasPrefix(s3Url, "s3://")
}
