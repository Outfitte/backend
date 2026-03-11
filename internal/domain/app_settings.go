package domain

// AppSettings holds application-wide configuration that is persisted at runtime.
//
// AppSettings is intentionally excluded from the generic StorageProvider pattern:
// it is a singleton (there is always exactly one instance) and therefore does not
// implement ports.Entity.  It must only be read and written via ports.SingletonStore.
type AppSettings struct {
	RegistrationEnabled bool
}
