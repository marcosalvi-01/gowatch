package server

import "testing"

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr string
	}{
		{name: "both missing", wantErr: "SEERR_BASE_URL is required"},
		{
			name:    "missing base URL",
			config:  Config{SeerrAPIKey: "api-key"},
			wantErr: "SEERR_BASE_URL is required",
		},
		{
			name:    "missing API key",
			config:  Config{SeerrBaseURL: "https://seerr.example"},
			wantErr: "SEERR_API_KEY is required",
		},
		{
			name:   "configured",
			config: Config{SeerrBaseURL: "https://seerr.example", SeerrAPIKey: "api-key"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Config.Validate() error = %v, want nil", err)
				}
				return
			}
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("Config.Validate() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}
