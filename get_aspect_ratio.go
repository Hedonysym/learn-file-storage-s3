package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
)

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", "-loglevel", "debug", filePath)
	output := bytes.Buffer{}
	cmd.Stdout = &output

	if err := cmd.Run(); err != nil {
		fmt.Printf("error: %v, filepath; %v\n", err, filePath)
		return "", err
	}
	type Stream struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	}

	type VideoInfo struct {
		Streams []Stream `json:"streams"`
	}

	var videoInfo VideoInfo
	err := json.Unmarshal(output.Bytes(), &videoInfo)
	if err != nil {
		return "", err
	}

	width := videoInfo.Streams[0].Width
	height := videoInfo.Streams[0].Height

	fmt.Printf("width: %d, height: %d, ratio: %f\n", width, height, float64(16)/9)
	if ratioComp(width, height, float64(16)/9) {
		return "landscape", nil
	} else if ratioComp(width, height, float64(9)/16) {
		return "portrait", nil
	} else {
		return "other", nil
	}
}

var epsilon = 2.0

func ratioCalc(w, h int) float64 {
	return math.Round((float64(w/h) / epsilon)) * epsilon
}

func ratioComp(w, h int, ratio float64) bool {
	inputRatio := ratioCalc(w, h)
	roundedRatio := math.Round(ratio/epsilon) * epsilon
	return math.Abs(inputRatio-roundedRatio) < epsilon
}
