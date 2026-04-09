// Package s3 implements the destination.Writer interface backed by Amazon S3.
package s3

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Dest is an S3-backed destination.
type Dest struct {
	client *s3.Client
	bucket string
	prefix string // optional key prefix, e.g. "backups/server01"
}

// New creates a new S3 destination.
// Credentials are loaded from the environment (AWS_ACCESS_KEY_ID /
// AWS_SECRET_ACCESS_KEY), ~/.aws/credentials, or the instance metadata service.
func New(bucket, region, prefix string) (*Dest, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("s3: load aws config: %w", err)
	}
	return &Dest{
		client: s3.NewFromConfig(cfg),
		bucket: bucket,
		prefix: strings.TrimRight(prefix, "/"),
	}, nil
}

func (d *Dest) key(name string) string {
	if d.prefix == "" {
		return name
	}
	return d.prefix + "/" + name
}

// Write opens a buffered writer; the object is uploaded to S3 on Close.
func (d *Dest) Write(name string) (io.WriteCloser, error) {
	tmp, err := os.CreateTemp("", "bsmc_s3_*.tmp")
	if err != nil {
		return nil, fmt.Errorf("s3: create temp: %w", err)
	}
	return &s3Writer{dest: d, name: name, tmp: tmp}, nil
}

type s3Writer struct {
	dest *Dest
	name string
	tmp  *os.File
}

func (w *s3Writer) Write(p []byte) (int, error) { return w.tmp.Write(p) }

func (w *s3Writer) Close() error {
	if _, err := w.tmp.Seek(0, io.SeekStart); err != nil {
		_ = w.tmp.Close()
		_ = os.Remove(w.tmp.Name())
		return fmt.Errorf("s3: seek: %w", err)
	}

	stat, _ := w.tmp.Stat()
	var size int64
	if stat != nil {
		size = stat.Size()
	}

	_, err := w.dest.client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:        aws.String(w.dest.bucket),
		Key:           aws.String(w.dest.key(w.name)),
		Body:          w.tmp,
		ContentLength: aws.Int64(size),
		StorageClass:  types.StorageClassStandardIa,
	})

	_ = w.tmp.Close()
	_ = os.Remove(w.tmp.Name())

	if err != nil {
		return fmt.Errorf("s3: put object %q: %w", w.name, err)
	}
	return nil
}

// Read fetches an object from S3 for restore.
func (d *Dest) Read(name string) (io.ReadCloser, error) {
	out, err := d.client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(d.bucket),
		Key:    aws.String(d.key(name)),
	})
	if err != nil {
		return nil, fmt.Errorf("s3: get object %q: %w", name, err)
	}
	return out.Body, nil
}

// Delete removes an object from S3.
func (d *Dest) Delete(name string) error {
	_, err := d.client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(d.bucket),
		Key:    aws.String(d.key(name)),
	})
	if err != nil {
		return fmt.Errorf("s3: delete %q: %w", name, err)
	}
	return nil
}

// List returns all keys under the given prefix.
func (d *Dest) List(prefix string) ([]string, error) {
	fullPrefix := d.key(prefix)
	var keys []string
	paginator := s3.NewListObjectsV2Paginator(d.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(d.bucket),
		Prefix: aws.String(fullPrefix),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.Background())
		if err != nil {
			return nil, fmt.Errorf("s3: list %q: %w", prefix, err)
		}
		for _, obj := range page.Contents {
			k := aws.ToString(obj.Key)
			// Strip our prefix so callers see relative keys
			if d.prefix != "" {
				k = strings.TrimPrefix(k, d.prefix+"/")
			}
			keys = append(keys, k)
		}
	}
	return keys, nil
}

func (d *Dest) Close() error { return nil }
