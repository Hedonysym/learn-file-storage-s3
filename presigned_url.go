package main

import (
	"context"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	client := s3.NewPresignClient(s3Client)

	params := s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}
	req, err := client.PresignGetObject(context.TODO(), &params, s3.WithPresignExpires(expireTime))
	if err != nil {
		return "", err
	}

	return req.URL, nil
}

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	if video.VideoURL == nil {
		return video, nil
	}

	split := strings.Split(*video.VideoURL, ",")
	if len(split) < 2 {
		return video, nil
	}

	newUrl, err := generatePresignedURL(cfg.s3Client, split[0], split[1], 5*time.Minute)
	if err != nil {
		return video, err
	}

	video.VideoURL = &newUrl
	return video, nil
}
