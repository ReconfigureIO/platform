package storage

// Interface for anything storagey
type Service interface {
	Upload()
	Download()
}
