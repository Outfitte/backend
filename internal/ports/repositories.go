package ports

// Repositories bundles all repository interfaces into a single struct.
// It is the single dependency that run.go passes to service constructors
// and is populated by the adapter factory.
type Repositories struct {
	Items       ItemRepository
	Users       UserRepository
	Sessions    SessionRepository
	Locations   LocationRepository
	WearLogs    WearLogRepository
	AppSettings AppSettingsRepository
}
