//go:build windows

package config

// NormalizePath agrega el prefijo de ruta larga de Windows si la ruta supera 260 caracteres.
// MAX_PATH en Windows es 260 chars; el prefijo \\?\ desactiva esa limitación
// para operaciones del kernel (CreateFile, etc.), pero no para todas las APIs Win32.
func NormalizePath(p string) string {
	const maxPath = 260
	if len(p) > maxPath {
		// No agregar el prefijo si ya lo tiene
		if len(p) >= 4 && p[:4] == `\\?\` {
			return p
		}
		// Convertir barras forward a backward antes de agregar el prefijo
		normalized := `\\?\` + p
		return normalized
	}
	return p
}
