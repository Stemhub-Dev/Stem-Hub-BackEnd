package dto

type AsignarRolUsuarioRequest struct {
	CodRol int64 `json:"codRol" binding:"required"`
}
