package downloader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type ProgressFunc func(downloaded, total int64, speed float64)

var errRangeUnsatisfiable = errors.New("range not satisfiable")

const maxFetchBytes int64 = 32 << 20

func readLimited(r io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("response exceeds %d byte limit", limit)
	}
	return data, nil
}

type Downloader struct {
	client   *http.Client
	maxRetry int
}

func New(maxRetry int) *Downloader {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = 30 * time.Second
	transport.IdleConnTimeout = 90 * time.Second

	return &Downloader{
		client: &http.Client{
			Transport:     transport,
			CheckRedirect: checkRedirect,
			Timeout:       0,
		},
		maxRetry: maxRetry,
	}
}

func checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return errors.New("stopped after 10 redirects")
	}
	return validateURL(req.URL.String())
}

func (d *Downloader) FetchBytes(ctx context.Context, url string) ([]byte, error) {

	if err := validateURL(url); err != nil {
		return nil, err
	}

	if len(url) > 7 && url[:7] == "file://" {
		path := url[7:]
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		return readLimited(f, maxFetchBytes)
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

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
			continue
		}

		data, err := readLimited(resp.Body, maxFetchBytes)
		resp.Body.Close()

		if err != nil {
			lastErr = err
			continue
		}

		return data, nil
	}

	return nil, fmt.Errorf("fetch %s failed after %d attempts: %w", url, d.maxRetry+1, lastErr)
}

func (d *Downloader) Fetch(ctx context.Context, url, dest string, progress ProgressFunc) error {

	if err := validateURL(url); err != nil {
		return err
	}

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
	for {
		start := offset
		err := d.writeResource(ctx, url, dest, offset, progress)
		if err == nil {
			return nil
		}
		if errors.Is(err, errRangeUnsatisfiable) {
			if terr := os.Truncate(dest, 0); terr != nil && !os.IsNotExist(terr) {
				return terr
			}
			offset = 0
			continue
		}
		if start > 0 {
			if terr := os.Truncate(dest, start); terr != nil && !os.IsNotExist(terr) {
				return terr
			}
		}
		return err
	}
}

func (d *Downloader) writeResource(ctx context.Context, url, dest string, offset int64, progress ProgressFunc) error {
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
		// append resumed bytes
	} else if resp.StatusCode == http.StatusOK {
		if offset > 0 {
			offset = 0
		}
	} else if offset > 0 && resp.StatusCode == http.StatusRequestedRangeNotSatisfiable {
		return errRangeUnsatisfiable
	} else {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	total := resp.ContentLength
	if total > 0 {
		total += offset
	}

	flag := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	if offset > 0 {
		flag = os.O_WRONLY | os.O_APPEND
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
			if _, writeErr := f.Write(buf[:n]); writeErr != nil {
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

	if total > 0 && downloaded != total {
		return fmt.Errorf("truncated transfer: got %d of %d bytes", downloaded, total)
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
