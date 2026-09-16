package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"jade-gamble/backend/store"
)

const maxBodyBytes = 64 * 1024

// readJSON enforces strict JSON bodies: decode twice, second must be EOF.
func readJSON(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("trailing data after JSON body")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// seedStr serialises the render seed as an exact decimal string.
// JS numbers lose uint64 precision (that made every stone look alike);
// the client hashes this string (FNV-1a) into its drawing RNG.
func seedStr(seed uint64) string {
	return strconv.FormatUint(seed, 10)
}

type apiFunc func(w http.ResponseWriter, r *http.Request) error

// handler wraps apiFunc with error mapping.
func (a *API) handler(fn apiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			switch {
			case errors.Is(err, store.ErrInsufficient),
				strings.Contains(err.Error(), "筹码不足"):
				writeErr(w, http.StatusBadRequest, "筹码不足")
			case errors.Is(err, store.ErrNotFound),
				strings.Contains(err.Error(), "not found"):
				writeErr(w, http.StatusNotFound, "找不到这笔记录")
			case errors.Is(err, store.ErrNotOwned),
				strings.Contains(err.Error(), "不在你的仓库"):
				writeErr(w, http.StatusForbidden, "石头不在你的仓库")
			default:
				writeErr(w, http.StatusBadRequest, err.Error())
			}
		}
	}
}

// userID resolves the session cookie to a user ID.
func (a *API) userID(r *http.Request) (int, error) {
	c, err := r.Cookie("session")
	if err != nil {
		return 0, errors.New("未登录")
	}
	uid, err := a.Store.GetSession(c.Value)
	if err != nil {
		return 0, errors.New("session 无效")
	}
	return uid, nil
}

func (a *API) mustUser(w http.ResponseWriter, r *http.Request) (int, bool) {
	uid, err := a.userID(r)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, err.Error())
		return 0, false
	}
	return uid, true
}
