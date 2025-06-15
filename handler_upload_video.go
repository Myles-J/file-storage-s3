package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

const (
	uploadLimit          = 1 << 30  // 1GB
	multipartMemoryLimit = 32 << 20 // 32MB
	formFileKey          = "video"
	tempFileName         = "tubely-upload.mp4"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	videoMetadata, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Video not found", err)
		return
	}

	if videoMetadata.UserID != userID {
		respondWithError(w, http.StatusForbidden, "User does not have access to this video", nil)
		return
	}

	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, uploadLimit)

	// Parse multipart form with 32MB memory limit
	if err := r.ParseMultipartForm(multipartMemoryLimit); err != nil {
		if err == http.ErrNotMultipart {
			respondWithError(w, http.StatusBadRequest, "Request must be multipart", err)
			return
		}
		if err == http.ErrMissingBoundary {
			respondWithError(w, http.StatusBadRequest, "Missing multipart boundary", err)
			return
		}
		respondWithError(w, http.StatusBadRequest, "Couldn't parse multipart form", err)
		return
	}

	file, header, err := r.FormFile(formFileKey)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't get video file", err)
		return
	}
	defer file.Close()

	if header.Size > uploadLimit {
		respondWithError(w, http.StatusBadRequest, "File is too large. Maximum size is 1GB.", nil)
		return
	}

	// Read first 512 bytes for content type detection
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		respondWithError(w, http.StatusInternalServerError, "Couldn't read file for content type detection", err)
		return
	}

	// Detect actual content type
	mediaType := http.DetectContentType(buf[:n])
	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Invalid file type. Only MP4 is allowed.", nil)
		return
	}

	// Reset file position for later use
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't reset file position", err)
		return
	}

	tempFile, err := os.CreateTemp("", tempFileName)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create temp file", err)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	_, err = io.Copy(tempFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't copreturn res.Uy file", err)
		return
	}

	aspectRatio, err := GetVideoAspectRatio(tempFile.Name())
	log.Printf("Aspect ratio: %s", aspectRatio)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get video aspect ratio", err)
		return
	}

	filePrefixMap := map[string]string{
		"16:9":  "landscape",
		"9:16":  "portrait",
		"other": "square",
	}

	filePrefix := filePrefixMap[aspectRatio]
	if filePrefix == "" {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get video aspect ratio", err)
		return
	}

	// Verify file was copied correctly
	fileInfo, err := tempFile.Stat()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get file info", err)
		return
	}
	if fileInfo.Size() == 0 {
		respondWithError(w, http.StatusInternalServerError, "File is empty after copy", nil)
		return
	}
	log.Printf("Temp file size after copy: %d bytes", fileInfo.Size())

	_, err = tempFile.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't reset file pointer", err)
		return
	}

	// Get file info to verify content
	fileInfo, err = tempFile.Stat()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get file info", err)
		return
	}
	log.Printf("Temp file size: %d bytes", fileInfo.Size())

	key := fmt.Sprintf("/%s/%s.mp4", filePrefix, uuid.New().String())

	processedFilePath, err := ProcessVideoForFastStart(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't process video for fast start", err)
		return
	}
	defer os.Remove(processedFilePath)

	processedFile, err := os.Open(processedFilePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't open processed file", err)
		return
	}
	defer processedFile.Close()

	log.Printf("Uploading to S3 with key: %s", key)

	s3PutObjectInput := &s3.PutObjectInput{
		Bucket:       aws.String(cfg.s3Bucket),
		Key:          aws.String(key),
		Body:         processedFile,
		ContentType:  aws.String(mediaType),
		CacheControl: aws.String("public, max-age=31536000"), // 1 year
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if _, err = cfg.s3Client.PutObject(ctx, s3PutObjectInput); err != nil {
		log.Printf("S3 upload error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Couldn't upload video", err)
		return
	}

	// Store the bucket and key for later use when generating the presigned URL
	videoURL := fmt.Sprintf("%s,%s", cfg.s3Bucket, key)
	videoMetadata.VideoURL = &videoURL

	if err = cfg.db.UpdateVideo(videoMetadata); err != nil {
		deleteObjectInput := &s3.DeleteObjectInput{
			Bucket: aws.String(cfg.s3Bucket),
			Key:    aws.String(key),
		}

		_, _ = cfg.s3Client.DeleteObject(ctx, deleteObjectInput)
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video metadata", err)
		return
	}

	videoWithSignedURL, err := cfg.DbVideoToSignedVideo(videoMetadata)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get signed video URL", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoWithSignedURL)
}
