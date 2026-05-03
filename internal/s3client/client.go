package s3client

import (
	"context"
	"io"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Client struct {
	api *s3.Client
}

func New(ctx context.Context) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &Client{api: s3.NewFromConfig(cfg)}, nil
}

func (c *Client) Download(ctx context.Context, bucket, key string, w io.Writer) error {
	out, err := c.api.GetObject(ctx, &s3.GetObjectInput{Bucket: &bucket, Key: &key})
	if err != nil {
		return err
	}
	defer out.Body.Close()
	_, err = io.Copy(w, out.Body)
	return err
}
