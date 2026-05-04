package s3client

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/example/aws-large-file-downloader/internal/download"
)

type Client struct{ api *s3.Client }

func New(ctx context.Context) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &Client{api: s3.NewFromConfig(cfg)}, nil
}

func (c *Client) HeadObject(ctx context.Context, bucket, key string) (download.ObjectInfo, error) {
	out, err := c.api.HeadObject(ctx, &s3.HeadObjectInput{Bucket: &bucket, Key: &key})
	if err != nil {
		return download.ObjectInfo{}, err
	}
	size := int64(0)
	if out.ContentLength != nil {
		size = *out.ContentLength
	}
	etag := ""
	if out.ETag != nil {
		etag = *out.ETag
	}
	return download.ObjectInfo{Size: size, ETag: etag}, nil
}

func (c *Client) DownloadRange(ctx context.Context, bucket, key string, start, end int64, w io.Writer) error {
	rangeHeader := fmt.Sprintf("bytes=%d-%d", start, end)
	out, err := c.api.GetObject(ctx, &s3.GetObjectInput{Bucket: &bucket, Key: &key, Range: &rangeHeader})
	if err != nil {
		return err
	}
	defer out.Body.Close()
	_, err = io.Copy(w, out.Body)
	return err
}
