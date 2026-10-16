package mangaplus

import (
	"context"
	"time"

	"github.com/luevano/libmangal"
)

// Account: The subscription of the device secret, or the free device (none with the web api).
func (p *plus) Account(ctx context.Context) (libmangal.Account, bool, error) {
	account := libmangal.Account{Service: providerInfo.Name, URL: website, Status: "free device"}
	if p.client.WebAPI() {
		account.Status = "web api (no device)"
		return account, true, nil
	}
	if p.options.Secret == "" {
		return account, true, nil
	}

	sub, err := p.client.User.SubscriptionContext(ctx)
	if err != nil {
		return account, false, err
	}
	account.Status = sub.GetPlanType()
	if account.Status == "" {
		account.Status = "no subscription"
	}
	if next := sub.GetNextPaymentDate(); next > 0 {
		account.Expires = time.Unix(next, 0)
	}
	if sub.GetIsFreeTrial() {
		account.Details = append(account.Details, "free trial")
	}
	if sub.GetIsPendingDowngrade() {
		account.Details = append(account.Details, "pending downgrade")
	}
	// empty for accounts that never commented
	if profile, err := p.client.User.ProfileContext(ctx); err == nil {
		account.User = profile.GetUserName()
	}
	return account, true, nil
}
