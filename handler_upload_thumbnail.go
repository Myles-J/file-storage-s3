package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

const maxMemory = 10 << 20 // 10MB

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
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

	log.Printf("uploading thumbnail for video %s by user %s", videoID, userID)

	if err := r.ParseMultipartForm(maxMemory); err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse multipart form", err)
		return
	}

	file, header, err := r.FormFile("thumbnail")

	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't get thumbnail file", err)
		return
	}
	defer file.Close()

	fileExt := strings.ToLower(strings.TrimPrefix(filepath.Ext(header.Filename), "."))

	if fileExt == "" {
		respondWithError(w, http.StatusBadRequest, "Invalid file extension", nil)
		return
	}

	allowedMimeTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
	}

	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid file extension.", err)
		return
	}

	if !allowedMimeTypes[mediaType] {
		respondWithError(w, http.StatusBadRequest, "Invalid file extension. Only JPEG and PNG are allowed.", nil)
		return
	}

	cryptoKey := make([]byte, 32)
	if _, err := rand.Read(cryptoKey); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't generate crypto key", err)
		return
	}

	encodedKey := base64.RawURLEncoding.EncodeToString(cryptoKey)
	thumbnailName := encodedKey + "." + fileExt
	thumbnailPath := filepath.Join(cfg.assetsRoot, thumbnailName)

	thumbnailFile, err := os.Create(thumbnailPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create thumbnail file", err)
		return
	}
	defer thumbnailFile.Close()

	if _, err = io.Copy(thumbnailFile, file); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't copy thumbnail file", err)
		return
	}

	thumbnailURL := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, thumbnailName)

	videoMetadata.ThumbnailURL = &thumbnailURL

	if err = cfg.db.UpdateVideo(videoMetadata); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video metadata", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoMetadata)
}
