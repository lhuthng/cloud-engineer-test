package model

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type OperationType string

const (
	OperationConvert   OperationType = "convert"
	OperationResize    OperationType = "resize"
	OperationCompress  OperationType = "compress"
	OperationGrayscale OperationType = "grayscale"
)

func (o OperationType) Valid() bool {
	switch o {
	case OperationConvert, OperationResize, OperationCompress, OperationGrayscale:
		return true
	}
	return false
}

func (o OperationType) ToFFmpegArgs(params map[string]any, inputPath, outputPath string) ([]string, error) {
	switch o {
	case OperationConvert:
		return []string{"-y", "-i", inputPath, outputPath}, nil
	case OperationResize:
		width, okW := params["width"].(float64)
		height, okH := params["height"].(float64)
		if !okW || !okH || width <= 0 || height <= 0 {
			return nil, fmt.Errorf("resize requires numeric width and height")
		}
		scale := fmt.Sprintf("scale=%d:%d", int(width), int(height))
		return []string{"-y", "-i", inputPath, "-vf", scale, outputPath}, nil
	case OperationCompress:
		crf := 28.0
		if v, ok := params["crf"].(float64); ok && v > 0 {
			crf = v
		}
		return []string{"-y", "-i", inputPath, "-c:v", "libx264", "-crf", fmt.Sprintf("%d", int(crf)), "-preset", "medium", outputPath}, nil
	case OperationGrayscale:
		return []string{"-y", "-i", inputPath, "-vf", "format=gray", outputPath}, nil
	}
	return nil, fmt.Errorf("unsupported operation %q", o)
}

func (o OperationType) OutputExtension(params map[string]any, inputExt string) (string, error) {
	if o != OperationConvert {
		return inputExt, nil
	}
	f, ok := params["format"].(string)
	if !ok || f == "" {
		return "", fmt.Errorf("convert requires params.format")
	}
	return strings.TrimPrefix(strings.ToLower(f), "."), nil
}

type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusDone       JobStatus = "done"
	JobStatusFailed     JobStatus = "failed"
)

type Session struct {
	ID             string    `json:"id"`
	CurrentVersion int       `json:"current_version"`
	Extension      string    `json:"extension"`
	CreatedAt      time.Time `json:"created_at"`
}

type Job struct {
	ID            string         `json:"id"`
	SessionID     string         `json:"session_id"`
	Operation     OperationType  `json:"operation"`
	Params        map[string]any `json:"params"`
	InputVersion  int            `json:"input_version"`
	OutputVersion int            `json:"output_version"`
	Status        JobStatus      `json:"status"`
	Error         string         `json:"error,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
}

func (j *Job) ParamsJSON() []byte {
	b, err := json.Marshal(j.Params)
	if err != nil {
		return []byte("{}")
	}
	return b
}
