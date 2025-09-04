package service

import (
	"net/http"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/haatos/simple-ci/internal"
	"github.com/haatos/simple-ci/internal/settings"
	"github.com/labstack/echo/v4"
)

type CookieService struct {
	s *securecookie.SecureCookie
}

func NewCookieService(hashKey, blockKey []byte) *CookieService {
	return &CookieService{
		s: securecookie.New(hashKey, blockKey),
	}
}

func (cs *CookieService) GetSessionID(c echo.Context) (string, error) {
	cookie, err := c.Cookie(internal.SessionCookie)
	if err != nil {
		return "", err
	}
	values := make(map[string]string)
	if err := cs.s.Decode(internal.SessionCookie, cookie.Value, &values); err != nil {
		return "", err
	}
	return values["session_id"], nil
}

func (cs *CookieService) SetSessionCookie(c echo.Context, sessionID string) error {
	return cs.setCookie(
		c,
		internal.SessionCookie,
		map[string]string{"session_id": sessionID},
		"/",
		settings.Settings.Domain != "localhost",
		true,
		time.Now().UTC().Add(settings.Settings.SessionExpires),
		settings.Settings.Domain,
	)
}

func (cs *CookieService) RemoveSessionCookie(c echo.Context) {
	cookie := &http.Cookie{
		Name:     internal.SessionCookie,
		Value:    "",
		Path:     "/",
		Secure:   settings.Settings.Domain != "localhost",
		HttpOnly: true,
		Expires:  time.Now().UTC(),
		Domain:   settings.Settings.Domain,
	}
	c.SetCookie(cookie)
}

func (cs *CookieService) setCookie(
	c echo.Context,
	name string,
	values map[string]string,
	path string,
	secure, httpOnly bool,
	expires time.Time,
	domain string,
) error {
	encoded, err := cs.s.Encode(name, values)
	if err != nil {
		return err
	}
	cookie := &http.Cookie{
		Name:     name,
		Value:    encoded,
		Path:     path,
		Secure:   secure,
		HttpOnly: httpOnly,
		Expires:  expires,
		Domain:   domain,
	}
	c.SetCookie(cookie)
	return nil
}
