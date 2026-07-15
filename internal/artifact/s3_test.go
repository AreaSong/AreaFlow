package artifact

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type fakeS3Client struct {
	put     *s3.PutObjectInput
	get     []byte
	headErr error
}

type fakeOperationObserver struct{ operations []string }

func (f *fakeOperationObserver) ObserveArtifactOperation(operation string, _ error, _ time.Duration) {
	f.operations = append(f.operations, operation)
}

func (f *fakeS3Client) PutObject(_ context.Context, input *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	f.put = input
	return &s3.PutObjectOutput{}, nil
}

func (f *fakeS3Client) GetObject(_ context.Context, _ *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return &s3.GetObjectOutput{Body: io.NopCloser(bytes.NewReader(f.get))}, nil
}

func (f *fakeS3Client) HeadBucket(_ context.Context, _ *s3.HeadBucketInput, _ ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
	if f.headErr != nil {
		return nil, f.headErr
	}
	return &s3.HeadBucketOutput{}, nil
}

func TestS3BackendPutAndGet(t *testing.T) {
	client := &fakeS3Client{get: []byte("stored")}
	backend, err := NewS3Backend(client, "areaflow")
	if err != nil {
		t.Fatal(err)
	}
	observer := &fakeOperationObserver{}
	backend.WithObserver(observer)
	storedArtifact, err := backend.Put(context.Background(), "project/report.json", []byte("stored"), "application/json")
	if err != nil {
		t.Fatal(err)
	}
	if storedArtifact.Backend != "s3" || storedArtifact.URI != "s3://areaflow/project/report.json" || storedArtifact.SHA256 == "" {
		t.Fatalf("unexpected stored artifact: %+v", storedArtifact)
	}
	if client.put == nil || client.put.ChecksumSHA256 == nil || client.put.ServerSideEncryption != "AES256" {
		t.Fatalf("S3 put lacks checksum or encryption: %+v", client.put)
	}
	if err := backend.Ping(context.Background()); err != nil {
		t.Fatalf("S3 ping failed: %v", err)
	}
	content, err := backend.Get(context.Background(), storedArtifact.URI)
	if err != nil || string(content) != "stored" {
		t.Fatalf("S3 get = %q, %v", content, err)
	}
	if strings.Join(observer.operations, ",") != "put,head_bucket,get" {
		t.Fatalf("observed operations = %v", observer.operations)
	}
}

func TestS3BackendRejectsOtherBucket(t *testing.T) {
	backend, _ := NewS3Backend(&fakeS3Client{}, "areaflow")
	if _, err := backend.Get(context.Background(), "s3://other/key"); err == nil {
		t.Fatal("cross-bucket read must fail")
	}
}

func TestS3BackendMinIOSmoke(t *testing.T) {
	if os.Getenv("AREAFLOW_S3_SMOKE") != "1" {
		t.Skip("set AREAFLOW_S3_SMOKE=1 to run the MinIO smoke")
	}
	ctx := context.Background()
	endpoint := os.Getenv("AREAFLOW_S3_ENDPOINT")
	region := os.Getenv("AREAFLOW_S3_REGION")
	bucket := os.Getenv("AREAFLOW_S3_BUCKET")
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		t.Fatal(err)
	}
	client := s3.NewFromConfig(cfg, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(endpoint)
		options.UsePathStyle = true
	})
	if _, err := client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucket)}); err != nil {
		t.Fatal(err)
	}
	backend, err := NewS3Backend(client, bucket)
	if err != nil {
		t.Fatal(err)
	}
	storedArtifact, err := backend.Put(ctx, "smoke/check.txt", []byte("areaflow-s3-smoke"), "text/plain")
	if err != nil {
		t.Fatal(err)
	}
	content, err := backend.Get(ctx, storedArtifact.URI)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "areaflow-s3-smoke" || storedArtifact.SHA256 == "" {
		t.Fatalf("unexpected S3 smoke result: %+v %q", storedArtifact, content)
	}
}
