package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
	// Base de zonas horarias embebida: la imagen alpine no trae tzdata.
	_ "time/tzdata"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/database"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/handler"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/mailer"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/middleware"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/mlservice"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/router"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/storage"
	"github.com/joho/godotenv"
)

const zonaHorariaPorDefecto = "America/Argentina/Buenos_Aires"

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Println("No se encontró archivo .env, se usarán variables de entorno del sistema")
	}

	configurarZonaHoraria()

	db, err := database.NewPostgresConnection()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	log.Println("Conexión con PostgreSQL establecida correctamente")

	audioStorage, err := storage.NewMinioAudioStorage()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Conexión con MinIO establecida correctamente")

	//Rol
	rolRepository := repository.NewRolRepository(db)
	rolService := service.NewRolService(rolRepository)
	rolHandler := handler.NewRolHandler(rolService)

	// Género musical
	generoMusicalRepository := repository.NewGeneroMusicalRepository(db)
	generoMusicalService := service.NewGeneroMusicalService(generoMusicalRepository)
	generoMusicalHandler := handler.NewGeneroMusicalHandler(generoMusicalService)

	// Tipo de proyecto
	tipoProyectoRepository := repository.NewTipoProyectoRepository(db)
	tipoProyectoService := service.NewTipoProyectoService(tipoProyectoRepository)
	tipoProyectoHandler := handler.NewTipoProyectoHandler(tipoProyectoService)

	// Estado de proyecto
	estadoProyectoRepository :=
		repository.NewEstadoProyectoRepository(db)

	estadoProyectoService :=
		service.NewEstadoProyectoService(
			estadoProyectoRepository,
		)

	estadoProyectoHandler :=
		handler.NewEstadoProyectoHandler(
			estadoProyectoService,
		)

	// Usuario y UsuarioRol
	usuarioRolRepository := repository.NewUsuarioRolRepository(db)
	usuarioRolService := service.NewUsuarioRolService(usuarioRolRepository)
	usuarioRepository := repository.NewUsuarioRepository(db)
	usuarioService := service.NewUsuarioService(usuarioRepository)
	usuarioHandler := handler.NewUsuarioHandler(usuarioService, usuarioRolService)

	supabaseURL := os.Getenv("SUPABASE_URL")
	supabaseAudience := os.Getenv("SUPABASE_AUDIENCE")
	// Opcional: solo necesaria en desarrollo local dockerizado, cuando el
	// backend no puede alcanzar SUPABASE_URL por red (ver supabase/README.md).
	supabaseJWKSBaseURL := os.Getenv("SUPABASE_JWKS_BASE_URL")

	authMiddleware, err := middleware.NewAuthMiddleware(
		usuarioService,
		supabaseURL,
		supabaseAudience,
		supabaseJWKSBaseURL,
	)

	if err != nil {
		log.Fatalf("error al configurar autenticación: %v", err)
	}

	// Permiso
	permisoRepository := repository.NewPermisoRepository(db)
	permisoService := service.NewPermisoService(permisoRepository)
	permisoMiddleware := middleware.NewPermisoMiddleware(permisoService)
	permisoHandler := handler.NewPermisoHandler(permisoService)

	// RolPermiso
	rolPermisoRepository := repository.NewRolPermisoRepository(db)
	rolPermisoService := service.NewRolPermisoService(rolPermisoRepository)
	rolPermisoHandler := handler.NewRolPermisoHandler(rolPermisoService)

	// UsuarioAdministracion
	usuarioAdministracionRepository := repository.NewUsuarioAdministracionRepository(db)
	usuarioAdministracionService := service.NewUsuarioAdministracionService(usuarioAdministracionRepository)
	usuarioAdministracionHandler := handler.NewUsuarioAdministracionHandler(usuarioAdministracionService)

	//Integrante
	integranteRepository := repository.NewIntegranteRepository(db)
	integranteService := service.NewIntegranteService(integranteRepository, audioStorage)
	integranteHandler := handler.NewIntegranteHandler(integranteService, usuarioRolService)

	//Proyecto
	proyectoRepository := repository.NewProyectoRepository(db)
	proyectoService := service.NewProyectoService(
		proyectoRepository,
		integranteRepository,
		audioStorage,
	)
	proyectoHandler := handler.NewProyectoHandler(proyectoService)

	//Mailer (SMTP directo, separado del auth.email.smtp de GoTrue)
	smtpMailer, err := mailer.NewSMTPMailer(
		os.Getenv("SMTP_HOST"),
		os.Getenv("SMTP_PORT"),
		os.Getenv("SMTP_USER"),
		os.Getenv("SMTP_PASSWORD"),
		os.Getenv("SMTP_REMITENTE"),
	)

	if err != nil {
		log.Fatalf("error al configurar el mailer: %v", err)
	}

	//Invitación a proyecto
	diasVencimientoInvitacion, err := strconv.Atoi(os.Getenv("INVITACION_DIAS_VENCIMIENTO"))
	if err != nil {
		diasVencimientoInvitacion = 7
	}

	invitacionProyectoRepository := repository.NewInvitacionProyectoRepository(db)
	invitacionProyectoService := service.NewInvitacionProyectoService(
		invitacionProyectoRepository,
		proyectoRepository,
		integranteRepository,
		usuarioRepository,
		rolRepository,
		smtpMailer,
		os.Getenv("FRONTEND_BASE_URL"),
		diasVencimientoInvitacion,
	)
	invitacionProyectoHandler := handler.NewInvitacionProyectoHandler(invitacionProyectoService)

	//Canción
	cancionRepository := repository.NewCancionRepository(db)
	cancionService := service.NewCancionService(
		cancionRepository,
		proyectoRepository,
		integranteRepository,
		audioStorage,
	)
	cancionHandler := handler.NewCancionHandler(cancionService)

	//Stem
	service.ConfigurarTamanosMaximos(
		leerMegabytes("TAMANO_MAXIMO_CANCION_MB", 100),
		leerMegabytes("TAMANO_MAXIMO_STEM_MB", 100),
	)
	stemRepository := repository.NewStemRepository(db)
	stemService := service.NewStemService(
		stemRepository,
		cancionRepository,
		proyectoRepository,
		integranteRepository,
		audioStorage,
	)
	//Microservicio de IA (separación de stems y resumen de comentarios)
	mlCliente, err := mlservice.NewHTTPCliente(
		os.Getenv("ML_SERVICE_URL"),
		leerSegundos("ML_SEPARACION_TIMEOUT_SEGUNDOS", 330),
		leerSegundos("ML_RESUMEN_TIMEOUT_SEGUNDOS", 60),
	)

	if err != nil {
		log.Fatalf("error al configurar el servicio de IA: %v", err)
	}

	separacionStemRepository := repository.NewSeparacionStemRepository(db)
	separacionStemService := service.NewSeparacionStemService(
		separacionStemRepository,
		stemRepository,
		cancionRepository,
		proyectoRepository,
		integranteRepository,
		audioStorage,
		mlCliente,
	)

	if err := separacionStemService.FallarInterrumpidas(); err != nil {
		log.Fatalf("error al cerrar separaciones de stems interrumpidas: %v", err)
	}

	stemHandler := handler.NewStemHandler(stemService, separacionStemService)

	//Comentario
	comentarioRepository := repository.NewComentarioRepository(db)
	comentarioService := service.NewComentarioService(
		comentarioRepository,
		proyectoRepository,
		cancionRepository,
		integranteRepository,
		mlCliente,
	)
	comentarioHandler := handler.NewComentarioHandler(comentarioService)

	// Reportes
	reporteRepository := repository.NewReporteRepository(db)
	reporteService := service.NewReporteService(
		reporteRepository,
		proyectoRepository,
		integranteRepository,
	)
	reporteHandler := handler.NewReporteHandler(reporteService)

	// Tablero
	tableroRepository := repository.NewTableroRepository(db)
	tableroService := service.NewTableroService(
		tableroRepository,
		proyectoRepository,
		integranteRepository,
	)
	tableroHandler := handler.NewTableroHandler(tableroService)

	corsAllowedOrigins := strings.Split(os.Getenv("CORS_ALLOWED_ORIGINS"), ",")

	r := router.NewRouter(
		db,
		rolHandler,
		generoMusicalHandler,
		tipoProyectoHandler,
		estadoProyectoHandler,
		usuarioHandler,
		usuarioAdministracionHandler,
		integranteHandler,
		proyectoHandler,
		invitacionProyectoHandler,
		cancionHandler,
		comentarioHandler,
		stemHandler,
		reporteHandler,
		tableroHandler,
		permisoHandler,
		rolPermisoHandler,
		authMiddleware,
		permisoMiddleware,
		corsAllowedOrigins,
	)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Servidor iniciado en http://localhost:%s", port)

	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}

// leerMegabytes lee un tamaño en MB de una variable de entorno, con un valor
// por defecto si no está definida. Un valor inválido corta el arranque: es
// preferible a aceptar archivos con un límite distinto al configurado.
func leerMegabytes(variable string, porDefecto int64) int64 {

	valor := os.Getenv(variable)

	if valor == "" {
		return porDefecto
	}

	megabytes, err := strconv.ParseInt(valor, 10, 64)

	if err != nil || megabytes <= 0 {
		log.Fatalf("%s debe ser un entero positivo (MB), se recibió %q", variable, valor)
	}

	return megabytes
}

func leerSegundos(variable string, porDefecto int) time.Duration {

	valor := os.Getenv(variable)

	if valor == "" {
		return time.Duration(porDefecto) * time.Second
	}

	segundos, err := strconv.Atoi(valor)

	if err != nil || segundos <= 0 {
		log.Fatalf("%s debe ser un entero positivo (segundos), se recibió %q", variable, valor)
	}

	return time.Duration(segundos) * time.Second
}

// configurarZonaHoraria fija time.Local según TZ (por defecto Argentina), para
// que las fechas mostradas al usuario (p. ej. en los reportes PDF) salgan en
// hora local y no en la UTC del contenedor.
func configurarZonaHoraria() {

	zona, err := resolverZonaHoraria(os.Getenv("TZ"))

	if err != nil {
		log.Fatal(err)
	}

	time.Local = zona
}

func resolverZonaHoraria(nombre string) (*time.Location, error) {

	if nombre == "" {
		nombre = zonaHorariaPorDefecto
	}

	zona, err := time.LoadLocation(nombre)

	if err != nil {
		return nil, fmt.Errorf("TZ inválida %q: %w", nombre, err)
	}

	return zona, nil
}
