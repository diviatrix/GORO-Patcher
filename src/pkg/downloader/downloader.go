package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type ProgressFunc func(downloaded, total int64, speed float64)

type Downloader struct {
	client   *http.Client
	maxRetry int
}

func New(maxRetry int) *Downloader {
	return &Downloader{
		client: &http.Client{
			Timeout: 0,
		},
		maxRetry: maxRetry,
	}
}

func (d *Downloader) FetchBytes(ctx context.Context, url string) ([]byte, error) {
	// Support local file URLs
	if len(url) > 7 && url[:7] == "file://" {
		path := url[7:]
		return os.ReadFile(path)
	}

	var lastErr error

	for attempt := 0; attempt <= d.maxRetry; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}

		resp, err := d.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
			continue
		}

		return data, nil
	}

	return nil, fmt.Errorf("fetch %s failed after %d attempts: %w", url, d.maxRetry+1, lastErr)
}

func (d *Downloader) Fetch(ctx context.Context, url, dest string, progress ProgressFunc) error {
	// Support local file paths
	if len(url) > 0 && url[0] == '/' {
		return copyFile(url, dest, progress)
	}

	var offset int64
	if info, err := os.Stat(dest); err == nil {
		offset = info.Size()
	}

	var lastErr error

	for attempt := 0; attempt <= d.maxRetry; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ctx.Err()
			}
			if info, err := os.Stat(dest); err == nil {
				offset = info.Size()
			}
		}

		err := d.fetchAttempt(ctx, url, dest, offset, progress)
		if err == nil {
			return nil
		}
		lastErr = err

		if ctx.Err() != nil {
			return ctx.Err()
		}
	}

	return fmt.Errorf("fetch %s failed after %d attempts: %w", url, d.maxRetry+1, lastErr)
}

func (d *Downloader) fetchAttempt(ctx context.Context, url, dest string, offset int64, progress ProgressFunc) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if offset > 0 && resp.StatusCode == http.StatusPartialContent {
		// Resume from offset
	} else if offset > 0 && resp.StatusCode == http.StatusOK {
		offset = 0
	} else if offset > 0 && resp.StatusCode == http.StatusRequestedRangeNotSatisfiable {
		// File already complete
		return nil
	} else if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	total := resp.ContentLength
	if total > 0 {
		total += offset
	}

	var flag int
	if offset > 0 {
		flag = os.O_WRONLY | os.O_APPEND
	} else {
		flag = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}

	f, err := os.OpenFile(dest, flag, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	downloaded := offset
	start := time.Now()
	buf := make([]byte, 64*1024)

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := f.Write(buf[:n])
			if writeErr != nil {
				return writeErr
			}
			downloaded += int64(n)

			if progress != nil {
				elapsed := time.Since(start).Seconds()
				speed := float64(downloaded-offset) / elapsed / 1024 / 1024
				progress(downloaded, total, speed)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}

	return nil
}

func copyFile(src, dst string, progress ProgressFunc) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("stat source: %w", err)
	}
	totalSize := srcInfo.Size()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create dest: %w", err)
	}
	defer dstFile.Close()

	buf := make([]byte, 64*1024)
	var copied int64
	start := time.Now()

	for {
		n, readErr := srcFile.Read(buf)
		if n > 0 {
			_, writeErr := dstFile.Write(buf[:n])
			if writeErr != nil {
				return writeErr
			}
			copied += int64(n)

			if progress != nil {
				elapsed := time.Since(start).Seconds()
				speed := float64(copied) / elapsed / 1024 / 1024
				progress(copied, totalSize, speed)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}

	return nil
}
