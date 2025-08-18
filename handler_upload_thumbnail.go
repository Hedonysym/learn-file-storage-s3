package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	maxMemory := 10 << 20

	err = r.ParseMultipartForm(int64(maxMemory))
	if err != nil {
		respondWithError(w, 400, err.Error(), err)
	}

	file, fileHeader, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, 400, err.Error(), err)
	}
	fileType, _, err := mime.ParseMediaType(fileHeader.Header.Get("Content-Type"))
	if fileType != "image/jpeg" && fileType != "image/png" || err != nil {
		respondWithError(w, 401, "invalid file type", err)
		return
	}

	mediaType, ok := strings.CutPrefix(fileType, "image/")
	if !ok {
		respondWithError(w, 401, "invalid file type", err)
		return
	}

	randId := [32]byte{}
	_, err = rand.Read(randId[:])
	if err != nil {
		respondWithError(w, 400, "something happened: "+err.Error(), err)
		return
	}
	base64Id := base64.RawURLEncoding.EncodeToString(randId[:])

	thumbPath := filepath.Join(cfg.assetsRoot, base64Id+"."+mediaType)
	thumbFile, err := os.Create(thumbPath)
	if err != nil {
		respondWithError(w, 400, "something happened: "+err.Error(), err)
		return
	}

	_, err = io.Copy(thumbFile, file)
	if err != nil {
		respondWithError(w, 400, "operation failed: "+err.Error(), err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid id", err)
		return
	}

	thumbUrl := "http://localhost:8091/" + thumbPath
	video.ThumbnailURL = &thumbUrl

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, 400, err.Error(), err)
	}

	respondWithJSON(w, http.StatusOK, video)
}
