package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

// getVideoAspectRatio returns the aspect ratio of a video file by calling ffprobe
func GetVideoAspectRatio(filePath string) (string, error) {
	type streamInfo struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	}
	type ffprobeOutput struct {
		Streams []streamInfo `json:"streams"`
	}

	var buf bytes.Buffer
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		return "", err
	}

	var out ffprobeOutput
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		return "", err
	}
	if len(out.Streams) == 0 {
		return "", fmt.Errorf("no streams found")
	}
	width := out.Streams[0].Width
	height := out.Streams[0].Height
	if width == 0 || height == 0 {
		return "", fmt.Errorf("invalid width/height")
	}

	ratio := float64(width) / float64(height)
	switch {
	case ratio > 1.7 && ratio < 1.8:
		return "16:9", nil
	case ratio < 0.57 && ratio > 0.55:
		return "9:16", nil
	default:
		return "other", nil
	}
}

// ProcessVideoForFastStart takes a video file and returns a new video file with fast start enabled.
func ProcessVideoForFastStart(filePath string) (string, error) {
	outPath := filePath + ".processing"
	cmd := exec.Command(
		"ffmpeg",
		"-i", filePath,
		"-c", "copy",
		"-movflags", "faststart",
		"-f", "mp4",
		outPath,
	)
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return outPath, nil
}

// GeneratePresignedURL generates a presigned URL for a given bucket and key.
func GeneratePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s3Client)
	res, err := presignClient.PresignGetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expireTime))

	if err != nil {
		return "", err
	}

	return res.URL, nil
}

// DbVideoToSignedVideo takes a database video and returns a signed video URL.
func (cfg *apiConfig) DbVideoToSignedVideo(video database.Video) (database.Video, error) {
	if video.VideoURL == nil {
		return video, fmt.Errorf("video URL is nil")
	}
	parts := strings.Split(*video.VideoURL, ",")
	if len(parts) != 2 {
		return video, fmt.Errorf("invalid video URL")
	}
	bucket, key := parts[0], parts[1]

	presignedURL, err := GeneratePresignedURL(cfg.s3Client, bucket, key, 1*time.Hour)
	if err != nil {
		return video, err
	}

	video.VideoURL = &presignedURL
	return video, nil
}
