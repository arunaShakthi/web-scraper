package auth

import (
	"errors"
	"net/http"
	"strings"
)

// Getapi key extract the apikey from the request header
// Getuser extract the user from the api key
func GetApiKey(headers http.Header) (string, error) {
	val := headers.Get("Authorization")
	if val == "" {
		return "", errors.New("No auth info found")
	}

	vals := strings.Split(val, " ")
	if len(vals) != 2 || vals[0] != "ApiKey" {
		return "", errors.New("Malfor auth header")
	}
	return vals[1], nil
}
