package model

// Permisos de ámbito PROYECTO.
//
// Las constantes centralizan las claves técnicas utilizadas por los services
// y evitan depender de strings escritos manualmente en distintos lugares.
//
// La existencia del permiso habilita la acción general. Algunas operaciones
// requieren además reglas de negocio adicionales, por ejemplo:
//   - ser propietario del proyecto;
//   - ser autor del comentario;
//   - pertenecer activamente al proyecto.
const (
	// Proyecto.
	PermisoVerProyecto      = "VER_PROYECTO"
	PermisoEditarProyecto   = "EDITAR_PROYECTO"
	PermisoEliminarProyecto = "ELIMINAR_PROYECTO"

	// Integrantes.
	PermisoVerIntegrantes       = "VER_INTEGRANTES"
	PermisoCambiarRolIntegrante = "CAMBIAR_ROL_INTEGRANTE"
	PermisoQuitarIntegrante     = "QUITAR_INTEGRANTE"

	// Invitaciones.
	PermisoInvitarIntegrante       = "INVITAR_INTEGRANTE"
	PermisoVerInvitacionesProyecto = "VER_INVITACIONES_PROYECTO"
	PermisoCancelarInvitacion      = "CANCELAR_INVITACION"

	// Canciones.
	PermisoVerCancion      = "VER_CANCION"
	PermisoCrearCancion    = "CREAR_CANCION"
	PermisoEditarCancion   = "EDITAR_CANCION"
	PermisoEliminarCancion = "ELIMINAR_CANCION"

	// Versiones.
	PermisoVerVersion   = "VER_VERSION"
	PermisoCrearVersion = "CREAR_VERSION"

	// Stems.
	PermisoVerStem        = "VER_STEM"
	PermisoVerStemsGlobal = "VER_STEMS_GLOBAL"
	PermisoCrearStem      = "CREAR_STEM"
	PermisoEditarStem     = "EDITAR_STEM"
	PermisoEliminarStem   = "ELIMINAR_STEM"
	PermisoSepararStemsIA = "SEPARAR_STEMS_IA"

	// Comentarios.
	PermisoVerComentarios              = "VER_COMENTARIOS"
	PermisoCrearComentario             = "CREAR_COMENTARIO"
	PermisoResponderComentario         = "RESPONDER_COMENTARIO"
	PermisoEditarComentarioPropio      = "EDITAR_COMENTARIO_PROPIO"
	PermisoEliminarComentarioPropio    = "ELIMINAR_COMENTARIO_PROPIO"
	PermisoEliminarCualquierComentario = "ELIMINAR_CUALQUIER_COMENTARIO"
	PermisoCambiarEstadoComentario     = "CAMBIAR_ESTADO_COMENTARIO"
	PermisoResumirComentariosIA        = "RESUMIR_COMENTARIOS_IA"

	// Reportes.
	PermisoVerReportes     = "VER_REPORTES"
	PermisoGenerarReporte  = "GENERAR_REPORTE"
	PermisoExportarReporte = "EXPORTAR_REPORTE"
)
