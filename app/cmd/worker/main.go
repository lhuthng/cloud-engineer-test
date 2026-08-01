package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/cloud-engineer-test/internal/config"
	"github.com/cloud-engineer-test/internal/model"
	"github.com/cloud-engineer-test/internal/s3"
	"github.com/cloud-engineer-test/internal/store"
)

func main() {
	cfg := config.Load()
	if cfg.DatabaseURL == "" || cfg.Bucket == "" {
		log.Fatal("DATABASE_URL and MEDIA_BUCKET are required")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer st.Close()

	media, err := s3.New(ctx, cfg.Bucket, cfg.Region)
	if err != nil {
		log.Fatalf("s3: %v", err)
	}

	log.Println("worker started")
	for {
		select {
		case <-ctx.Done():
			log.Println("worker shutting down")
			return
		default:
		}

		job, err := st.ClaimNextJob(ctx)
		if err != nil {
			log.Printf("claim job: %v", err)
			time.Sleep(cfg.PollInterval)
			continue
		}
		if job == nil {
			time.Sleep(cfg.PollInterval)
			continue
		}

		if err := process(ctx, st, media, job); err != nil {
			log.Printf("job %s failed: %v", job.ID, err)
			_ = st.FailJob(ctx, job.ID, err.Error())
		}
	}
}

func process(ctx context.Context, st *store.Store, media *s3.Client, job *model.Job) error {
	sess, err := st.GetSession(ctx, job.SessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	dir, err := os.MkdirTemp("", "worker-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	inputExt := sess.Extension
	inputFile := filepath.Join(dir, "input."+inputExt)
	outExt, err := job.Operation.OutputExtension(job.Params, inputExt)
	if err != nil {
		return err
	}
	outputFile := filepath.Join(dir, "output."+outExt)

	log.Printf("job %s: downloading s3://%s/%s", job.ID, media.Bucket(), s3.Key(job.SessionID, job.InputVersion, inputExt))
	f, err := os.Create(inputFile)
	if err != nil {
		return err
	}
	if err := media.Download(ctx, s3.Key(job.SessionID, job.InputVersion, inputExt), f); err != nil {
		f.Close()
		return fmt.Errorf("download input: %w", err)
	}
	f.Close()

	log.Printf("job %s: running ffmpeg (%s)", job.ID, job.Operation)
	args, err := job.Operation.ToFFmpegArgs(job.Params, inputFile, outputFile)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg: %w: %s", err, out)
	}

	out, err := os.Open(outputFile)
	if err != nil {
		return fmt.Errorf("open output: %w", err)
	}
	defer out.Close()

	outKey := s3.Key(job.SessionID, job.OutputVersion, outExt)
	log.Printf("job %s: uploading s3://%s/%s", job.ID, media.Bucket(), outKey)
	if err := media.Upload(ctx, outKey, "video/"+outExt, out); err != nil {
		return fmt.Errorf("upload output: %w", err)
	}

	if err := st.CompleteJob(ctx, job.ID, job.SessionID, job.InputVersion, outExt); err != nil {
		return fmt.Errorf("complete job: %w", err)
	}
	log.Printf("job %s: done (v%d -> v%d)", job.ID, job.InputVersion, job.OutputVersion)
	return nil
}
