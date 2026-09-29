package install

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func downloadFile(ctx context.Context, client *http.Client, url, dest string, label string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	tmp := dest + ".part"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "litedepth")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: HTTP %d", label, resp.StatusCode)
	}

	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	total := resp.ContentLength
	var n int64
	start := time.Now()
	buf := make([]byte, 256*1024)
	for {
		if err := ctx.Err(); err != nil {
			out.Close()
			_ = os.Remove(tmp)
			return err
		}
		nr, rerr := resp.Body.Read(buf)
		if nr > 0 {
			if _, werr := out.Write(buf[:nr]); werr != nil {
				out.Close()
				_ = os.Remove(tmp)
				return werr
			}
			n += int64(nr)
			if total > 0 && time.Since(start) > 2*time.Second {
				fmt.Printf("  %s  %d%%\n", label, n*100/total)
				start = time.Now()
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			out.Close()
			_ = os.Remove(tmp)
			return rerr
		}
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, dest); err != nil {
		return err
	}
	fmt.Printf("  %s  done (%d MB)\n", label, (n+1024*1024-1)/(1024*1024))
	return nil
}
