package repository

// PermissionRepository handles memory access permissions.
// Full implementation in Chunk 5 (Social Layer).
type PermissionRepository interface {
	// Placeholder for future implementation
}

type pgxPermissionRepo struct {
	pool interface{} // Will use *pgxpool.Pool
}

// NewPermissionRepository creates a stub permission repository.
// TODO: implement full permission checks in Chunk 5.
func NewPermissionRepository(pool interface{}) PermissionRepository {
	return &pgxPermissionRepo{pool: pool}
}
