package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
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

	fmt.Println("uploading video", videoID, "by user", userID)

	video, err := cfg.db.GetVideo(videoID)
	if err != nil || video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Invalid id", err)
		return
	}

	maxMemory := 10 << 30

	err = r.ParseMultipartForm(int64(maxMemory))
	if err != nil {
		respondWithError(w, 400, err.Error(), err)
		return
	}

	file, fileHeader, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, 400, err.Error(), err)
		return
	}

	defer file.Close()

	fileType, _, err := mime.ParseMediaType(fileHeader.Header.Get("Content-Type"))
	if fileType != "video/mp4" || err != nil {
		respondWithError(w, 401, "invalid file type", err)
		return
	}

	temp, err := os.CreateTemp("", "tubely-upload.mp4")
	defer os.Remove(temp.Name())
	defer temp.Close()
	if err != nil {
		respondWithError(w, 400, "something happened: "+err.Error(), err)
		return
	}

	_, err = io.Copy(temp, file)
	if err != nil {
		respondWithError(w, 400, "something happened: "+err.Error(), err)
		return
	}

	_, err = temp.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, 400, "something happened: "+err.Error(), err)
		return
	}

	ratio, err := getVideoAspectRatio(temp.Name())
	if err != nil {
		respondWithError(w, 400, "something happened: "+err.Error(), err)
		return
	}

	processedPath, err := processVideoForFastStart(temp.Name())
	if err != nil {
		respondWithError(w, 400, fmt.Sprintf("shit: %v", err.Error()), err)
		return
	}

	processed, err := os.OpenFile(processedPath, os.O_RDONLY, 0666)
	defer os.Remove(processedPath)
	defer processed.Close()
	if err != nil {
		respondWithError(w, 400, fmt.Sprintf("shit: %v", err.Error()), err)
		return
	}

	randId := [32]byte{}
	_, err = rand.Read(randId[:])
	if err != nil {
		respondWithError(w, 400, "something happened: "+err.Error(), err)
		return
	}
	base64Id := base64.RawURLEncoding.EncodeToString(randId[:])
	final := ratio + "/" + base64Id + ".mp4"
	bucketParams := s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &final,
		Body:        processed,
		ContentType: &fileType,
	}

	_, err = cfg.s3Client.PutObject(r.Context(), &bucketParams)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error uploading file to S3", err)
		return
	}

	url := fmt.Sprintf("%v,%v", cfg.s3Bucket, final)

	video.VideoURL = &url

	signedVideo, err := cfg.dbVideoToSignedVideo(video)
	if err != nil {
		respondWithError(w, 400, "something happened: "+err.Error(), err)
		return
	}

	err = cfg.db.UpdateVideo(signedVideo)
	if err != nil {
		respondWithError(w, 400, "database update error", err)
		return
	}
}
