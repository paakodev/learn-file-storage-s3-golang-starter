package main

import (
	"context"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s3Client)
	obj, err := presignClient.PresignGetObject(context.Background(), &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}, s3.WithPresignExpires(expireTime))
	if err != nil {
		return "", err
	}
	return obj.URL, nil
}

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	if video.VideoURL != nil {
		videoInfo := strings.SplitN(*video.VideoURL, ",", 2)
		if len(videoInfo) == 2 {
			signedURL, err := generatePresignedURL(cfg.s3Client, videoInfo[0], videoInfo[1], 15*time.Minute)
			if err != nil {
				return video, err
			}
			video.VideoURL = &signedURL
		}
	}
	return video, nil
}
