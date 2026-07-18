package public

import "fixture/internal/secret"

type Options struct { // want `exported type "Options" depends on internal package\(s\): fixture/internal/secret`
	Store secret.Store
}

type PublicAlias = string
