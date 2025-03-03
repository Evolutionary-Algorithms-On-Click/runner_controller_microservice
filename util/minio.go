package util

import (
	"context"
	"fmt"
	"os"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func UploadFile(ctx context.Context, runID string, fileName string, extension string) error {
	var logger = NewLogger()

	endpoint := os.Getenv("MINIO_ENDPOINT")
	accessKeyID := os.Getenv("MINIO_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("MINIO_SECRET_KEY")
	bucketName := "code"

	// Initialize minio client object.
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: false,
	})
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create minio client: %v", err))
		return err
	}

	// Create a bucket called code if it doesn't exist.
	err = minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
	if err != nil {
		// Check to see if we already own this bucket (which happens if you run this twice)
		exists, errBucketExists := minioClient.BucketExists(ctx, bucketName)
		if errBucketExists == nil && exists {
			logger.Info(fmt.Sprintf("We already own bucket: %s\n", bucketName))
		} else {
			logger.Error(fmt.Sprintf("Failed to create bucket %s: %v", bucketName, err))
			return err
		}
	} else {
		logger.Info(fmt.Sprintf("Successfully created %s\n", bucketName))
	}

	// Set bucket policy to public.
	policy := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":["*"]},"Action":["s3:GetObject"],"Resource":["arn:aws:s3:::` + bucketName + `/*"]},{"Effect":"Allow","Principal":{"AWS":["*"]},"Action":["s3:GetBucketLocation"],"Resource":["arn:aws:s3:::` + bucketName + `"]}]}`
	err = minioClient.SetBucketPolicy(ctx, bucketName, policy)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to set bucket policy: %v", err))
		return err
	}
	logger.Info(fmt.Sprintf("Successfully set bucket policy for %s\n", bucketName))

	// Upload the file.
	objectName := fmt.Sprintf("%s/%s.%s", runID, fileName, extension)
	filePath := fmt.Sprintf("%s/%s.%s", fileName, runID, extension)
	info, err := minioClient.FPutObject(ctx, bucketName, objectName, filePath, minio.PutObjectOptions{})
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to upload %s: %v", filePath, err))
		return err
	}

	logger.Info(fmt.Sprintf("Successfully uploaded %s of size %d\n", objectName, info.Size))
	logger.Info(fmt.Sprintf("Download link: http://%s/%s/%s", endpoint, bucketName, objectName))

	return nil
}
