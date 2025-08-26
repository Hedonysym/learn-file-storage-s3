package main

import (
	"bytes"
	"fmt"
	"os/exec"
)

func processVideoForFastStart(filePath string) (string, error) {
	newPath := filePath + ".processing"
	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", newPath)
	output := bytes.Buffer{}
	cmd.Stdout = &output

	if err := cmd.Run(); err != nil {
		fmt.Printf("error: %v", err)
		return "", err
	}

	return newPath, nil
}
