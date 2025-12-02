package service

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	StatusAvailable    = "available"
	StatusNotAvailable = "not available"
)

type Checker struct {
	client  *http.Client
	timeout time.Duration
}

func NewChecker(timeout time.Duration) *Checker {
	return &Checker{
		client: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
		timeout: timeout,
	}
}

// CheckLinks проверяет доступность ссылок параллельно
func (c *Checker) CheckLinks(ctx context.Context, links []string) map[string]string {
	results := make(map[string]string)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, link := range links {
		wg.Add(1)
		go func(l string) {
			defer wg.Done()

			status := StatusNotAvailable
			if c.isAvailable(ctx, l) {
				status = StatusAvailable
			}

			mu.Lock()
			results[l] = status
			mu.Unlock()
		}(link)
	}

	wg.Wait()
	return results
}

func (c *Checker) isAvailable(ctx context.Context, link string) bool {
	// Добавляем схему если отсутствует
	url := link
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return false
	}

	req.Header.Set("User-Agent", "LinkChecker/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		// Пробуем GET если HEAD не работает
		req, _ = http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		req.Header.Set("User-Agent", "LinkChecker/1.0")
		resp, err = c.client.Do(req)
		if err != nil {
			return false
		}
	}
	defer resp.Body.Close()

	// Считаем доступным если статус < 400
	return resp.StatusCode < 400
}
