package dialog

import (
	"context"
	"fmt"
	"time"

	"github.com/ChxisB/talon/deps/testing/pkg/catwalk"
	bubble "github.com/ChxisB/talon/deps/ui/terminal/v2"
	"github.com/ChxisB/talon/internal/config"
	"github.com/ChxisB/talon/internal/oauth/copilot"
	"github.com/ChxisB/talon/internal/ui/common"
)

func NewOAuthCopilot(
	com *common.Common,
	isOnboarding bool,
	provider catwalk.Provider,
	model config.SelectedModel,
	modelType config.SelectedModelType,
) (*OAuth, bubble.Cmd) {
	return newOAuth(com, isOnboarding, provider, model, modelType, &OAuthCopilot{})
}

type OAuthCopilot struct {
	deviceCode *copilot.DeviceCode
	cancelFunc func()
}

var _ OAuthProvider = (*OAuthCopilot)(nil)

func (m *OAuthCopilot) name() string {
	return "GitHub Copilot"
}

func (m *OAuthCopilot) initiateAuth() bubble.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	deviceCode, err := copilot.RequestDeviceCode(ctx)
	if err != nil {
		return ActionOAuthErrored{Error: fmt.Errorf("failed to initiate device auth: %w", err)}
	}

	m.deviceCode = deviceCode

	return ActionInitiateOAuth{
		DeviceCode:      deviceCode.DeviceCode,
		UserCode:        deviceCode.UserCode,
		VerificationURL: deviceCode.VerificationURI,
		ExpiresIn:       deviceCode.ExpiresIn,
		Interval:        deviceCode.Interval,
	}
}

func (m *OAuthCopilot) startPolling(deviceCode string, expiresIn int) bubble.Cmd {
	return func() bubble.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		m.cancelFunc = cancel

		token, err := copilot.PollForToken(ctx, m.deviceCode)
		if err != nil {
			if ctx.Err() != nil {
				return nil // cancelled, don't report error.
			}
			return ActionOAuthErrored{Error: err}
		}

		return ActionCompleteOAuth{Token: token}
	}
}

func (m *OAuthCopilot) stopPolling() bubble.Msg {
	if m.cancelFunc != nil {
		m.cancelFunc()
	}
	return nil
}
