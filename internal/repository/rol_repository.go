package repository

import (
	"database/sql"
	"fmt"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
)

// defino un nuevo tipo de estructura que representa el repositorio de roles
type RolRepository struct {
	db *sql.DB
}

// db nombre de la variable, *sql.DB tipo de dato, asterisco es un puntero a la estructura sql.DB, que representa la conexión a la base de datos
// despues de los paréntesis aparece el tipo que devuelve la función, en este caso un puntero a la estructura RolRepository
func NewRolRepository(db *sql.DB) *RolRepository {
	//& significa que se está devolviendo la dirección de memoria de la estructura RolRepository, es decir, un puntero a la misma
	return &RolRepository{
		db: db,
	}
}

// r *RolRepository se llama receiver(Receptor), es una convención de Go para indicar que la función pertenece a la estructura RolRepository. Usamos r como nombre corto (repository)
// la funcion devuelve dos valores, colección (un slice o lista de roles) y errores por si ocurre problemas.
func (r *RolRepository) Listar() ([]model.Rol, error) {
	//:= es un operador de asignació corta en Go, que puede inferir el tipo de dato.
	query := `
	SELECT
		codrol,
		nombrerol,
		descripcionrol,
		ambitorol,
		fechahorabajarol
	FROM rol
	ORDER BY codrol
	`
	//recibe ambos valores que puede ser filas encontradas o posible error, si ocurre un error se devuelve nil y el error, si no se devuelve la colección.
	rows, err := r.db.Query(query)

	//nil = valor nulo.
	if err != nil {
		return nil, fmt.Errorf("Error al consultar roles: %W", err)
	}
	//defer significa ejecutar row.close() cuando la función termine.
	defer rows.Close()

	var roles []model.Rol

	//rows.Next() devuelve true si hay una fila siguiente, y false si no hay más filas.
	for rows.Next() {
		var rol model.Rol

		err := rows.Scan(
			&rol.CodRol,
			&rol.NombreRol,
			&rol.DescripcionRol,
			&rol.AmbitoRol,
			&rol.FechaHoraBajaRol,
		)

		if err != nil {
			return nil, fmt.Errorf("Error al leer rol: %W", err)
		}

		roles = append(roles, rol)

		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("Error al recorrer roles: %W", err)
		}

	}

	return roles, nil
}
