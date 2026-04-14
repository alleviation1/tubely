package main

import (
	"fmt"
	"net/http"
	"io"
	"os"
	"path/filepath"
	"log"
	"strings"
	"mime"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

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

	const maxMemory = 10 << 20
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to parse multipart form", err)
		return
	}

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to form file", err)
		return
	}

	defer file.Close()

	media := header.Header.Get("Content-Type")
	if media == "" {
		respondWithError(w, http.StatusBadRequest, "Missing Content-Type for thumbnail", nil)
		return
	}

	mimeType, _, err := mime.ParseMediaType(media)
	if err != nil || (mimeType != "image/jpeg" && mimeType != "image/png") {
		respondWithError(w, http.StatusBadRequest, "Invalid mime type", nil)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't find video", err)
		return
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Not authorized to update this video", nil)
		return
	}

	fileType := strings.Split(media, "/")[1]

	path := filepath.Join(cfg.assetsRoot, "/", videoIDString)
	path += "." + fileType
	newFile, err := os.Create(path)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to make file", err)
		return
	}

	if _, err := io.Copy(newFile, file); err != nil {
		log.Fatal(err)
		return
	}

	filePath := fmt.Sprintf("http://localhost:%s/assets/%s.%s", cfg.port, videoIDString, fileType)
	video.ThumbnailURL = &filePath

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}