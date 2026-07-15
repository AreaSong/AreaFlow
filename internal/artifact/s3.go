package artifact

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3API interface {
	PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	HeadBucket(context.Context, *s3.HeadBucketInput, ...func(*s3.Options)) (*s3.HeadBucketOutput, error)
}

type S3Backend struct {
	client   S3API
	bucket   string
	observer OperationObserver
}

type OperationObserver interface {
	ObserveArtifactOperation(operation string, err error, duration time.Duration)
}

var defaultS3 struct {
	sync.Mutex
	backend *S3Backend
	err     error
}

func (b *S3Backend) WithObserver(observer OperationObserver) *S3Backend {
	b.observer = observer
	return b
}

func NewS3Backend(client S3API, bucket string) (*S3Backend, error) {
	bucket = strings.TrimSpace(bucket)
	if client == nil || bucket == "" {
		return nil, fmt.Errorf("S3 client and bucket are required")
	}
	return &S3Backend{client: client, bucket: bucket}, nil
}

func DefaultS3Backend(ctx context.Context) (*S3Backend, error) {
	defaultS3.Lock()
	defer defaultS3.Unlock()
	if defaultS3.backend != nil || defaultS3.err != nil {
		return defaultS3.backend, defaultS3.err
	}
	region := strings.TrimSpace(os.Getenv("AREAFLOW_S3_REGION"))
	bucket := strings.TrimSpace(os.Getenv("AREAFLOW_S3_BUCKET"))
	if region == "" || bucket == "" {
		defaultS3.err = fmt.Errorf("AREAFLOW_S3_REGION and AREAFLOW_S3_BUCKET are required")
		return nil, defaultS3.err
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		defaultS3.err = fmt.Errorf("load AWS configuration: %w", err)
		return nil, defaultS3.err
	}
	endpoint := strings.TrimSpace(os.Getenv("AREAFLOW_S3_ENDPOINT"))
	usePathStyle := strings.EqualFold(strings.TrimSpace(os.Getenv("AREAFLOW_S3_USE_PATH_STYLE")), "true")
	client := s3.NewFromConfig(cfg, func(options *s3.Options) {
		options.UsePathStyle = usePathStyle
		if endpoint != "" {
			options.BaseEndpoint = aws.String(endpoint)
		}
	})
	defaultS3.backend, defaultS3.err = NewS3Backend(client, bucket)
	return defaultS3.backend, defaultS3.err
}

func (b *S3Backend) Put(ctx context.Context, key string, content []byte, contentType string) (result Stored, err error) {
	started := time.Now()
	defer func() { b.observe("put", err, started) }()
	key = strings.Trim(strings.TrimSpace(key), "/")
	if key == "" || strings.Contains(key, "../") {
		return Stored{}, fmt.Errorf("valid S3 artifact key is required")
	}
	digest := sha256Bytes(content)
	_, err = b.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(b.bucket), Key: aws.String(key), Body: bytes.NewReader(content),
		ContentType: aws.String(contentType), ChecksumSHA256: aws.String(base64.StdEncoding.EncodeToString(digest)),
		ServerSideEncryption: "AES256",
	})
	if err != nil {
		return Stored{}, fmt.Errorf("put S3 artifact: %w", err)
	}
	return stored("s3", "s3://"+b.bucket+"/"+key, content, contentType), nil
}

func (b *S3Backend) Get(ctx context.Context, uri string) (content []byte, err error) {
	started := time.Now()
	defer func() { b.observe("get", err, started) }()
	bucket, key, err := parseS3URI(uri)
	if err != nil {
		return nil, err
	}
	if bucket != b.bucket {
		return nil, fmt.Errorf("S3 artifact bucket %q is outside configured bucket", bucket)
	}
	output, err := b.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)})
	if err != nil {
		return nil, fmt.Errorf("get S3 artifact: %w", err)
	}
	defer output.Body.Close()
	content, err = io.ReadAll(output.Body)
	if err != nil {
		return nil, fmt.Errorf("read S3 artifact: %w", err)
	}
	return content, nil
}

func (b *S3Backend) Ping(ctx context.Context) (err error) {
	started := time.Now()
	defer func() { b.observe("head_bucket", err, started) }()
	if _, err = b.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(b.bucket)}); err != nil {
		return fmt.Errorf("head S3 artifact bucket: %w", err)
	}
	return nil
}

func (b *S3Backend) observe(operation string, err error, started time.Time) {
	if b.observer != nil {
		b.observer.ObserveArtifactOperation(operation, err, time.Since(started))
	}
}

func parseS3URI(raw string) (string, string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "s3" || parsed.Host == "" || strings.Trim(parsed.Path, "/") == "" {
		return "", "", fmt.Errorf("invalid S3 artifact URI")
	}
	return parsed.Host, strings.TrimPrefix(parsed.Path, "/"), nil
}
