package httpclient

import (
	"encoding/json"
	"errors"
	"maps"
	"net/url"
	"strings"

	app "github.com/ach968/x-twt-cli/internal/app"
)

func copyMap(source map[string]any) map[string]any {
	result := make(map[string]any, len(source))
	maps.Copy(result, source)
	return result
}

func prepareRequest(contract app.OperationContract, authentication app.AuthenticationState, overrides map[string]any) (app.PreparedRequest, error) {
	requestURL := &url.URL{Scheme: "https", Host: contract.Host, Path: contract.Path}
	variables := copyMap(contract.Variables)
	maps.Copy(variables, overrides)
	cookies := make([]string, 0, len(authentication.Cookies))
	headers := map[string]string{
		"authorization":         authentication.Authorization,
		"x-twitter-active-user": "yes",
		"x-twitter-auth-type":   "OAuth2Session",
	}
	for _, cookie := range authentication.Cookies {
		cookies = append(cookies, cookie.Name+"="+cookie.Value)
		if cookie.Name == "ct0" {
			headers["x-csrf-token"] = cookie.Value
		}
	}
	headers["cookie"] = strings.Join(cookies, "; ")

	prepared := app.PreparedRequest{Method: contract.Method, URL: requestURL.String(), Headers: headers}
	switch contract.Encoding {
	case "query":
		encodedVariables, err := json.Marshal(variables)
		if err != nil {
			return app.PreparedRequest{}, err
		}
		encodedFeatures, err := json.Marshal(contract.Features)
		if err != nil {
			return app.PreparedRequest{}, err
		}
		encodedToggles, err := json.Marshal(contract.FieldToggles)
		if err != nil {
			return app.PreparedRequest{}, err
		}
		query := requestURL.Query()
		query.Set("variables", string(encodedVariables))
		query.Set("features", string(encodedFeatures))
		query.Set("fieldToggles", string(encodedToggles))
		requestURL.RawQuery = query.Encode()
		prepared.URL = requestURL.String()
	case "json":
		body := copyMap(contract.Body)
		body["variables"] = variables
		if _, present := body["features"]; present {
			body["features"] = contract.Features
		}
		if _, present := body["fieldToggles"]; present {
			body["fieldToggles"] = contract.FieldToggles
		}
		encodedBody, err := json.Marshal(body)
		if err != nil {
			return app.PreparedRequest{}, err
		}
		prepared.Body = string(encodedBody)
		prepared.Headers["content-type"] = "application/json"
	default:
		return app.PreparedRequest{}, errors.New("unsupported operation contract encoding")
	}
	return prepared, nil
}
