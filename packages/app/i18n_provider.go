package app

import (
	"github.com/krewire/krewire/packages/i18n"
)

// I18nProvider returns an app.Provider that registers *i18n.Bundle and *i18n.Translator
// singletons into the application container. If an i18n.Config is registered in
// the container, it is applied to the bundle automatically.
//
// The adapter lives in app rather than in i18n itself: i18n is a runtime primitive
// and must not depend on the application container, ensuring a strict pyramid dependency
// chain and preventing circular dependencies.
func I18nProvider(bundle *i18n.Bundle) Provider {
	return ProviderFunc(func(c *Container) error {
		if err := Singleton[*i18n.Bundle](c, func() *i18n.Bundle {
			if cfg, err := Resolve[i18n.Config](c); err == nil {
				i18n.WithConfig(cfg)(bundle)
			} else if cfgPtr, err := Resolve[*i18n.Config](c); err == nil && cfgPtr != nil {
				i18n.WithConfig(*cfgPtr)(bundle)
			}
			return bundle
		}); err != nil {
			return err
		}

		return Singleton[*i18n.Translator](c, func() *i18n.Translator {
			b, err := Resolve[*i18n.Bundle](c)
			if err != nil || b == nil {
				b = bundle
			}
			if b == nil {
				return i18n.NewTranslator(nil, "en")
			}
			return b.ForLocale(b.DefaultLocale())
		})
	})
}

// I18nProviderWithConfig returns an app.Provider that explicitly binds an i18n.Config,
// *i18n.Bundle, and *i18n.Translator into the application container.
func I18nProviderWithConfig(bundle *i18n.Bundle, cfg i18n.Config) Provider {
	return ProviderFunc(func(c *Container) error {
		if err := Singleton[i18n.Config](c, func() i18n.Config {
			return cfg
		}); err != nil {
			return err
		}
		if bundle != nil {
			i18n.WithConfig(cfg)(bundle)
		}
		return I18nProvider(bundle).Register(c)
	})
}
