package task

import (
	"reflect"
	"testing"
	"time"

	"github.com/bitxeno/atvloadly/internal/model"
)

func TestShouldUseRefreshMode(t *testing.T) {
	newApp := model.InstalledApp{}
	if shouldUseRefreshMode(newApp) {
		t.Fatal("new app install should not use refresh mode")
	}

	existingApp := model.InstalledApp{}
	existingApp.ID = 1
	if !shouldUseRefreshMode(existingApp) {
		t.Fatal("existing app refresh should use refresh mode")
	}
}

func TestSelectAutoRefreshApps(t *testing.T) {
	now := time.Now()
	refreshed := now.Add(-24 * time.Hour)
	dueSoon := now.Add(12 * time.Hour)
	notDue := now.Add(30 * 24 * time.Hour)

	app := func(id uint, mode model.SigningMode, expiration time.Time, refreshedError model.RefreshedError) model.InstalledApp {
		v := model.InstalledApp{
			Account:          "user@example.com",
			RefreshedDate:    &refreshed,
			ExpirationDate:   &expiration,
			RefreshedError:   refreshedError,
			SigningMode:      mode,
			BundleIdentifier: "com.example.app",
		}
		if mode == model.SigningModeExternalCertificate {
			v.Account = ""
			v.SigningIdentityID = 7
		}
		v.ID = id
		return v
	}

	tests := []struct {
		name         string
		apps         []model.InstalledApp
		wantRefresh  []uint
		wantExternal []uint
	}{
		{
			name:        "apple id apps due for refresh, including historical rows without mode",
			apps:        []model.InstalledApp{app(1, model.SigningModeAppleID, dueSoon, 0), app(2, "", dueSoon, 0)},
			wantRefresh: []uint{1, 2},
		},
		{
			name: "apps not due are ignored",
			apps: []model.InstalledApp{app(1, model.SigningModeAppleID, notDue, 0), app(2, model.SigningModeExternalCertificate, notDue, 0)},
		},
		{
			name: "invalid account is skipped",
			apps: []model.InstalledApp{app(1, model.SigningModeAppleID, dueSoon, model.RefreshedErrorInvalidAccount)},
		},
		{
			name:         "external certificate apps are never refreshed automatically",
			apps:         []model.InstalledApp{app(1, model.SigningModeExternalCertificate, dueSoon, 0), app(2, model.SigningModeAppleID, dueSoon, 0)},
			wantRefresh:  []uint{2},
			wantExternal: []uint{1},
		},
		{
			name:         "expired external app with a failed reinstall is still skipped",
			apps:         []model.InstalledApp{app(1, model.SigningModeExternalCertificate, now.Add(-time.Hour), model.RefreshedErrorTransport)},
			wantExternal: []uint{1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refresh, external := selectAutoRefreshApps(tt.apps, 1)
			if got := appIDs(refresh); !reflect.DeepEqual(got, tt.wantRefresh) {
				t.Fatalf("refresh = %v, want %v", got, tt.wantRefresh)
			}
			if got := appIDs(external); !reflect.DeepEqual(got, tt.wantExternal) {
				t.Fatalf("external = %v, want %v", got, tt.wantExternal)
			}
		})
	}
}

func appIDs(apps []model.InstalledApp) []uint {
	var ids []uint
	for _, v := range apps {
		ids = append(ids, v.ID)
	}
	return ids
}
