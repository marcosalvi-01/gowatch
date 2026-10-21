package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/marcosalvi-01/gowatch/internal/common"
	"github.com/marcosalvi-01/gowatch/internal/models"
	"golang.org/x/sync/singleflight"
)

const (
	defaultSeerrTimeout = 10 * time.Second
	seerrAPIKeyHeader   = "X-Api-Key" // #nosec G101 -- protocol header name, not an API key
	seerrAPIPath        = "/api/v1"
	seerrResponseLimit  = 1 << 20
	seerrErrorBodyLimit = 4 << 10
	seerrHTTPScheme     = "https"
)

var (
	ErrSeerrNotConfigured = errors.New("seerr integration is not configured")
	ErrSeerrInvalidConfig = errors.New("invalid Seerr configuration")
	ErrSeerrNotAllowed    = errors.New("user cannot request movies through Seerr")
	ErrSeerrUnauthorized  = errors.New("seerr API key was rejected")
	ErrSeerrForbidden     = errors.New("seerr API identity cannot perform this action")
	ErrSeerrRejected      = errors.New("seerr rejected the movie request")
	ErrSeerrUnavailable   = errors.New("seerr is unavailable")
)

type SeerrRequestResult struct {
	Status    *int
	State     SeerrMovieState
	Submitted bool
	Retried   bool
}

type SeerrMovieState string

const (
	SeerrMovieRequestable     SeerrMovieState = "requestable"
	SeerrMoviePendingApproval SeerrMovieState = "pending_approval"
	SeerrMovieProcessing      SeerrMovieState = "processing"
	SeerrMovieAvailable       SeerrMovieState = "available"
	SeerrMovieRetryable       SeerrMovieState = "retryable"
)

type seerrMovieDetails struct {
	MediaInfo *struct {
		Status   *int `json:"status"`
		Requests []struct {
			ID     int64 `json:"id"`
			Status *int  `json:"status"`
		} `json:"requests"`
	} `json:"mediaInfo"`
}

type SeerrService struct {
	client      *http.Client
	integration models.SeerrIntegration
	requests    singleflight.Group
}

func NewSeerrService(integration models.SeerrIntegration, client *http.Client) (*SeerrService, error) {
	baseURL := strings.TrimSpace(integration.BaseURL)
	apiKey := strings.TrimSpace(integration.APIKey)
	if (baseURL == "") != (apiKey == "") {
		return nil, fmt.Errorf("%w: Seerr base URL and API key must both be set", ErrSeerrInvalidConfig)
	}

	var configured models.SeerrIntegration
	if baseURL != "" {
		normalized, err := normalizeSeerrIntegration(baseURL, apiKey)
		if err != nil {
			return nil, err
		}
		configured = normalized
	}

	if client == nil {
		client = &http.Client{Timeout: defaultSeerrTimeout}
	}

	clientCopy := *client
	if clientCopy.Timeout <= 0 {
		clientCopy.Timeout = defaultSeerrTimeout
	}
	checkRedirect := clientCopy.CheckRedirect
	clientCopy.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		if len(via) > 0 {
			if request.Method != http.MethodGet {
				return http.ErrUseLastResponse
			}
			previous := via[0].URL
			if !strings.EqualFold(request.URL.Host, previous.Host) || (previous.Scheme == seerrHTTPScheme && request.URL.Scheme != seerrHTTPScheme) {
				return http.ErrUseLastResponse
			}
		}

		if checkRedirect != nil {
			return checkRedirect(request, via)
		}
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		return nil
	}

	return &SeerrService{client: &clientCopy, integration: configured}, nil
}

func (s *SeerrService) Configuration() models.SeerrIntegrationStatus {
	return models.SeerrIntegrationStatus{BaseURL: s.integration.BaseURL}
}

func (s *SeerrService) MovieState(ctx context.Context, movieID int64) (SeerrMovieState, error) {
	if err := checkSeerrRequestAccess(ctx); err != nil {
		return "", err
	}
	state, _, _, err := s.lookupMovieState(ctx, movieID)
	return state, err
}

func (s *SeerrService) lookupMovieState(ctx context.Context, movieID int64) (SeerrMovieState, int64, models.SeerrIntegration, error) {
	if movieID <= 0 {
		return "", 0, models.SeerrIntegration{}, fmt.Errorf("invalid TMDB movie ID %d", movieID)
	}

	if s.integration.BaseURL == "" {
		return "", 0, models.SeerrIntegration{}, ErrSeerrNotConfigured
	}
	integration := s.integration

	requestURL, err := seerrEndpoint(integration.BaseURL, fmt.Sprintf("movie/%d", movieID))
	if err != nil {
		return "", 0, models.SeerrIntegration{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return "", 0, models.SeerrIntegration{}, fmt.Errorf("create Seerr movie-state request: %w", err)
	}
	request.Header.Set(seerrAPIKeyHeader, integration.APIKey)
	request.Header.Set("Accept", "application/json")

	response, err := s.client.Do(request)
	if err != nil {
		return "", 0, models.SeerrIntegration{}, fmt.Errorf("get Seerr movie state: %w: %w", ErrSeerrUnavailable, err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		return "", 0, models.SeerrIntegration{}, seerrResponseError("get movie state", response)
	}

	var details seerrMovieDetails
	if err := json.NewDecoder(io.LimitReader(response.Body, seerrResponseLimit)).Decode(&details); err != nil {
		return "", 0, models.SeerrIntegration{}, fmt.Errorf("decode Seerr movie state: %w: %w", ErrSeerrUnavailable, err)
	}

	state, retryRequestID, err := classifySeerrMovieState(details)
	if err != nil {
		return "", 0, models.SeerrIntegration{}, err
	}
	return state, retryRequestID, integration, nil
}

func checkSeerrRequestAccess(ctx context.Context) error {
	user, err := common.GetUser(ctx)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrSeerrNotAllowed, err)
	}
	if user == nil {
		return ErrSeerrNotAllowed
	}
	return nil
}

func classifySeerrMovieState(details seerrMovieDetails) (SeerrMovieState, int64, error) {
	if details.MediaInfo == nil {
		return SeerrMovieRequestable, 0, nil
	}
	if details.MediaInfo.Status == nil {
		return "", 0, fmt.Errorf("%w: Seerr returned a movie without an availability status", ErrSeerrUnavailable)
	}

	mediaStatus := *details.MediaInfo.Status
	if mediaStatus < 1 || mediaStatus > 6 {
		return "", 0, fmt.Errorf("%w: unknown Seerr media status %d", ErrSeerrUnavailable, mediaStatus)
	}

	pendingApproval := false
	approved := false
	completed := false
	latestRequestID := int64(0)
	latestRequestStatus := 0
	failedRequestID := int64(0)
	for _, request := range details.MediaInfo.Requests {
		if request.Status == nil {
			return "", 0, fmt.Errorf("%w: Seerr returned a request without a status", ErrSeerrUnavailable)
		}
		if request.ID > latestRequestID {
			latestRequestID = request.ID
			latestRequestStatus = *request.Status
		}
		switch *request.Status {
		case 1:
			pendingApproval = true
		case 2:
			approved = true
		case 3:
		case 4:
			if request.ID <= 0 {
				return "", 0, fmt.Errorf("%w: Seerr returned a failed request without an ID", ErrSeerrUnavailable)
			}
			if request.ID > failedRequestID {
				failedRequestID = request.ID
			}
		case 5:
			completed = true
		default:
			return "", 0, fmt.Errorf("%w: unknown Seerr request status %d", ErrSeerrUnavailable, *request.Status)
		}
	}

	if mediaStatus == 5 {
		return SeerrMovieAvailable, 0, nil
	}
	if mediaStatus == 2 || mediaStatus == 3 || mediaStatus == 4 || approved {
		return SeerrMovieProcessing, 0, nil
	}
	if pendingApproval {
		return SeerrMoviePendingApproval, 0, nil
	}
	if failedRequestID > 0 && failedRequestID == latestRequestID && latestRequestStatus == 4 {
		return SeerrMovieRetryable, failedRequestID, nil
	}
	if completed && mediaStatus != 6 {
		return SeerrMovieAvailable, 0, nil
	}
	return SeerrMovieRequestable, 0, nil
}

func (s *SeerrService) RequestMovie(ctx context.Context, movieID int64) (*SeerrRequestResult, error) {
	if err := checkSeerrRequestAccess(ctx); err != nil {
		return nil, err
	}
	if movieID <= 0 {
		return nil, fmt.Errorf("invalid TMDB movie ID %d", movieID)
	}

	result, err, _ := s.requests.Do(fmt.Sprintf("%d", movieID), func() (any, error) {
		state, retryRequestID, integration, err := s.lookupMovieState(ctx, movieID)
		if err != nil {
			return nil, err
		}
		if state == SeerrMovieRetryable {
			return s.retryMovieRequest(ctx, retryRequestID, integration)
		}
		if state != SeerrMovieRequestable {
			return &SeerrRequestResult{State: state}, nil
		}
		return s.submitMovieRequest(ctx, movieID, integration)
	})
	if err != nil {
		return nil, err
	}
	requestResult, ok := result.(*SeerrRequestResult)
	if !ok {
		return nil, fmt.Errorf("unexpected Seerr request result type %T", result)
	}
	return requestResult, nil
}

func (s *SeerrService) retryMovieRequest(ctx context.Context, requestID int64, integration models.SeerrIntegration) (*SeerrRequestResult, error) {
	requestURL, err := seerrEndpoint(integration.BaseURL, fmt.Sprintf("request/%d/retry", requestID))
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create Seerr retry request: %w", err)
	}
	request.Header.Set(seerrAPIKeyHeader, integration.APIKey)
	request.Header.Set("Accept", "application/json")

	response, err := s.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("retry Seerr movie request: %w: %w", ErrSeerrUnavailable, err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		return nil, seerrResponseError("retry movie request", response)
	}

	status := decodeSeerrRequestStatus(response.Body)
	return &SeerrRequestResult{Status: status, Submitted: true, Retried: true}, nil
}

func (s *SeerrService) submitMovieRequest(ctx context.Context, movieID int64, integration models.SeerrIntegration) (*SeerrRequestResult, error) {
	body := struct {
		MediaType string `json:"mediaType"`
		MediaID   int64  `json:"mediaId"`
	}{
		MediaType: "movie",
		MediaID:   movieID,
	}

	var requestBody strings.Builder
	if err := json.NewEncoder(&requestBody).Encode(body); err != nil {
		return nil, fmt.Errorf("encode Seerr movie request: %w", err)
	}

	requestURL, err := seerrEndpoint(integration.BaseURL, "request")
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, strings.NewReader(requestBody.String()))
	if err != nil {
		return nil, fmt.Errorf("create Seerr movie request: %w", err)
	}
	request.Header.Set(seerrAPIKeyHeader, integration.APIKey)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")

	response, err := s.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("submit Seerr movie request: %w: %w", ErrSeerrUnavailable, err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusCreated {
		return nil, seerrResponseError("submit movie request", response)
	}

	status := decodeSeerrRequestStatus(response.Body)
	return &SeerrRequestResult{Status: status, Submitted: true}, nil
}

// decodeSeerrRequestStatus is best-effort; a malformed response does not undo an accepted request.
func decodeSeerrRequestStatus(body io.Reader) *int {
	var result struct {
		Status *int `json:"status"`
	}
	if err := json.NewDecoder(io.LimitReader(body, seerrResponseLimit)).Decode(&result); err != nil && !errors.Is(err, io.EOF) {
		return nil
	}
	return result.Status
}

func normalizeSeerrIntegration(baseURL, apiKey string) (models.SeerrIntegration, error) {
	baseURL = strings.TrimSpace(baseURL)
	apiKey = strings.TrimSpace(apiKey)
	parsedURL, err := url.Parse(baseURL)
	if err != nil || !parsedURL.IsAbs() || parsedURL.Hostname() == "" {
		return models.SeerrIntegration{}, fmt.Errorf("%w: enter a valid Seerr base URL", ErrSeerrInvalidConfig)
	}
	parsedURL.Scheme = strings.ToLower(parsedURL.Scheme)
	if parsedURL.Scheme != "http" && parsedURL.Scheme != seerrHTTPScheme {
		return models.SeerrIntegration{}, fmt.Errorf("%w: URL must use http or https", ErrSeerrInvalidConfig)
	}
	if parsedURL.User != nil || parsedURL.RawQuery != "" || parsedURL.Fragment != "" || parsedURL.Opaque != "" {
		return models.SeerrIntegration{}, fmt.Errorf("%w: URL cannot contain credentials, query, or fragment", ErrSeerrInvalidConfig)
	}
	if apiKey == "" || strings.ContainsAny(apiKey, "\r\n") {
		return models.SeerrIntegration{}, fmt.Errorf("%w: enter a valid API key", ErrSeerrInvalidConfig)
	}

	parsedURL.Path = strings.TrimRight(parsedURL.Path, "/")
	parsedURL.RawPath = ""
	parsedURL.Path = strings.TrimSuffix(parsedURL.Path, seerrAPIPath)
	parsedURL.Path = strings.TrimRight(parsedURL.Path, "/")
	return models.SeerrIntegration{BaseURL: strings.TrimRight(parsedURL.String(), "/"), APIKey: apiKey}, nil
}

func seerrEndpoint(baseURL, endpoint string) (string, error) {
	apiURL, err := url.JoinPath(baseURL, seerrAPIPath, endpoint)
	if err != nil {
		return "", fmt.Errorf("build Seerr API URL: %w", err)
	}
	return apiURL, nil
}

func seerrResponseError(action string, response *http.Response) error {
	body, readErr := io.ReadAll(io.LimitReader(response.Body, seerrErrorBodyLimit+1))
	bodyText := strings.TrimSpace(string(body))
	if len(body) > seerrErrorBodyLimit {
		bodyText = strings.TrimSpace(strings.ToValidUTF8(string(body[:seerrErrorBodyLimit]), "�")) + "… (truncated)"
	}
	if readErr != nil {
		if bodyText != "" {
			bodyText += "; "
		}
		bodyText += "failed to read response body: " + readErr.Error()
	}

	var responseErr error
	switch response.StatusCode {
	case http.StatusUnauthorized:
		responseErr = fmt.Errorf("%s: %w", action, ErrSeerrUnauthorized)
	case http.StatusForbidden:
		responseErr = fmt.Errorf("%s: %w", action, ErrSeerrForbidden)
	}
	if responseErr == nil && response.StatusCode >= http.StatusBadRequest && response.StatusCode < http.StatusInternalServerError {
		responseErr = fmt.Errorf("%s: %w (HTTP %d)", action, ErrSeerrRejected, response.StatusCode)
	}
	if responseErr == nil {
		responseErr = fmt.Errorf("%s: %w (HTTP %d)", action, ErrSeerrUnavailable, response.StatusCode)
	}
	if bodyText != "" {
		return fmt.Errorf("%w: response body: %s", responseErr, bodyText)
	}
	return responseErr
}
