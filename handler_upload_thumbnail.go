package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

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

	videoMeta, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to get video metadata", err)
		return
	}
	if videoMeta.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "User does not own this video", fmt.Errorf("user %s does not own video %s", userID, videoID))
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	// "thumbnail" should match the HTML form input name
	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid media type", err)
		return
	}
	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "Invalid media type", fmt.Errorf("media type must be image/jpeg or image/png"))
		return
	}

	mediaTypeExtension := ""
	switch mediaType {
	case "image/jpeg":
		mediaTypeExtension = ".jpg"
	case "image/png":
		mediaTypeExtension = ".png"
	default:
		respondWithError(w, http.StatusBadRequest, "Invalid media type", fmt.Errorf("media type must be image/jpeg or image/png"))
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to read file", err)
		return
	}

	thumbnailFile := filepath.Join(cfg.assetsRoot, videoID.String()+mediaTypeExtension)
	f, err := os.Create(thumbnailFile)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to create thumbnail file", err)
		return
	}
	defer f.Close()
	_, err = f.Write(data)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to write thumbnail file", err)
		return
	}

	thumbnailURL := "http://docker:8091/assets/" + videoID.String() + mediaTypeExtension
	videoMeta.ThumbnailURL = &thumbnailURL
	err = cfg.db.UpdateVideo(videoMeta)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to update video with thumbnail", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoMeta)
}
