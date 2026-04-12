package captcha

import (
	"sync"
	"time"

	"github.com/mojocn/base64Captcha"
)

var (
	captchaOnce sync.Once
	store       base64Captcha.Store
	driver      base64Captcha.Driver
)

func ensure() {
	captchaOnce.Do(func() {
		// For multiple app instances behind a load balancer, implement base64Captcha.Store
		// with a shared backend (e.g. Redis) and assign it here instead of NewMemoryStore.
		store = base64Captcha.NewMemoryStore(4096, 5*time.Minute)
		driver = base64Captcha.NewDriverDigit(40, 120, 4, 0.7, 52)
	})
}

// GenerateCaptcha returns captcha id, a data-uri PNG for <img src="...">, and error.
func GenerateCaptcha() (string, string, error) {
	ensure()
	c := base64Captcha.NewCaptcha(driver, store)
	id, b64s, _, err := c.Generate()
	if err != nil {
		return "", "", err
	}
	return id, "data:image/png;base64," + b64s, nil
}

// VerifyCaptcha verifies user input and removes the captcha on success.
func VerifyCaptcha(id, code string) bool {
	if id == "" || code == "" {
		return false
	}
	ensure()
	return store.Verify(id, code, true)
}
