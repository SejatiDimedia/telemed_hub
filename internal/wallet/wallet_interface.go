package wallet

import (
	"github.com/timurdianradhasejati/telemed_hub/internal/wallet/service"
)

// WalletService is a type alias to expose the interface in the parent package
// without causing circular dependencies in the subpackages.
type WalletService = service.WalletService
