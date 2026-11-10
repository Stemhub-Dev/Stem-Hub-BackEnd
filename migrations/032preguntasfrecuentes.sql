BEGIN;

-- =========================================================
-- PREGUNTAS FRECUENTES INICIALES (HU-AYS-B1)
-- =========================================================
-- El orden de carga es el orden en que se muestran las funcionalidades y sus
-- preguntas. Idempotente: no repite una pregunta que ya exista.

INSERT INTO preguntafrecuente (
    funcionalidadprincipal,
    preguntafrecuente,
    respuestafrecuente
)
SELECT nueva.funcionalidad, nueva.pregunta, nueva.respuesta
FROM (
    VALUES
    -- REGISTRO Y ACCESO
    (1, 'Registro y acceso', '¿Cómo creo una cuenta?',
     'En la pantalla de acceso elegí «Crear cuenta» y completá nombre, email y contraseña, que debe tener al menos 6 caracteres. También podés registrarte con tu cuenta de Google. Si usás email, vas a tener que verificarlo antes de poder ingresar.'),
    (2, 'Registro y acceso', 'No me llegó el correo de verificación, ¿qué hago?',
     'Revisá la carpeta de correo no deseado y confirmá que escribiste bien tu email. Si no aparece, usá la opción para reenviar el correo de verificación.'),
    (3, 'Registro y acceso', 'Olvidé mi contraseña, ¿cómo la recupero?',
     'En la pantalla de inicio de sesión elegí «Recuperar contraseña» e ingresá el email de tu cuenta. Te vamos a enviar un enlace para definir una contraseña nueva.'),
    (4, 'Registro y acceso', '¿Puedo cambiar el email de mi cuenta?',
     'No, el email de la cuenta no se puede modificar. Desde tu perfil sí podés cambiar tu nombre, tu descripción y tu foto.'),
    (5, 'Registro y acceso', '¿Cómo elimino mi cuenta?',
     'Escribinos a stemhubsoporte@gmail.com. Cuando nos pidas eliminarla, borramos tus datos personales y tu contenido dentro de los 30 días.'),

    -- GESTIÓN DE ARCHIVOS
    (6, 'Gestión de archivos', '¿Qué formatos de audio puedo subir?',
     'Podés subir canciones, versiones y stems en formato MP3, WAV o FLAC.'),
    (7, 'Gestión de archivos', '¿Cuál es el tamaño máximo de un archivo?',
     'Cada archivo de audio puede pesar hasta 100 MB. La portada de un proyecto y la foto de perfil pueden pesar hasta 5 MB.'),
    (8, 'Gestión de archivos', 'Si subo una versión nueva, ¿se pierde la anterior?',
     'No. Cada versión queda guardada en el historial de la canción, así que podés volver a escucharla cuando quieras y comparar dos versiones entre sí.'),
    (9, 'Gestión de archivos', '¿Cómo separo una canción en stems?',
     'En el detalle de la canción, elegí la versión y presioná «Separar Pistas». Podés pedir 2, 4 o 5 stems. El proceso puede tardar varios minutos y sigue en curso aunque salgas de la pantalla. Si la versión ya tiene stems generados con IA, no se puede volver a separar.'),
    (10, 'Gestión de archivos', '¿Puedo descargar los archivos de un proyecto?',
     'Sí. Si tenés permiso para verlos, podés descargar el audio de cada versión y cada stem desde el detalle de la canción.'),

    -- PERMISOS DE ACCESO
    (11, 'Permisos de acceso', '¿Qué diferencia hay entre Productor y Músico?',
     'Los dos pueden ver el proyecto, escuchar versiones y stems, separar pistas con IA, comentar y generar reportes. Además, el Productor puede crear y administrar canciones, versiones y stems, y eliminar comentarios de otros integrantes.'),
    (12, 'Permisos de acceso', '¿Quién es el propietario de un proyecto?',
     'Es la persona que lo creó, sin importar si participa como Productor o como Músico. El propietario es quien edita o elimina el proyecto, invita integrantes, cambia sus roles y los quita.'),
    (13, 'Permisos de acceso', '¿Tengo el mismo rol en todos mis proyectos?',
     'No, el rol es por proyecto. Podés ser Productor en uno y Músico en otro, por eso las opciones disponibles cambian según el proyecto en el que estés.'),
    (14, 'Permisos de acceso', '¿Cómo invito a alguien a mi proyecto?',
     'Si sos el propietario, abrí los integrantes del proyecto e invitá a la persona con su email y el rol que va a tener. Va a recibir un correo con la invitación, que vence a los 7 días.'),
    (15, 'Permisos de acceso', '¿Por qué no veo una opción que otro integrante sí ve?',
     'Las acciones disponibles dependen de tu rol en ese proyecto y de si sos su propietario. Si necesitás hacer algo que no te aparece, pedile al propietario que revise tu rol.'),
    (16, 'Permisos de acceso', '¿Qué pasa si me quitan de un proyecto?',
     'Dejás de tener acceso a su contenido. Tu cuenta de StemHub y tus otros proyectos no se modifican.'),

    -- ASPECTOS LEGALES DEL CONTENIDO
    (17, 'Aspectos legales del contenido', '¿Quién puede ver el contenido que subo?',
     'Cada proyecto solo es visible para sus integrantes. Los archivos de audio se entregan con enlaces temporales que vencen a los pocos minutos.'),
    (18, 'Aspectos legales del contenido', '¿Qué contenido puedo subir a StemHub?',
     'Subí únicamente material propio o para el que tengas autorización de sus titulares. Si creés que algún contenido afecta tus derechos, escribinos a stemhubsoporte@gmail.com.'),
    (19, 'Aspectos legales del contenido', '¿StemHub vende o cede mis datos o mi contenido?',
     'No. Usamos tus datos únicamente para brindarte el servicio, no los vendemos ni los cedemos, y no los usamos para publicidad.'),
    (20, 'Aspectos legales del contenido', '¿Mi música se envía a servicios de inteligencia artificial externos?',
     'No. La separación de pistas se hace en nuestros propios servidores y tus archivos de audio no se envían a terceros. Cuando alguien pide resumir los comentarios de una versión, enviamos a Gemini, de Google, el texto de esos comentarios, el nombre de sus autores y el nombre de la canción y la versión.'),
    (21, 'Aspectos legales del contenido', '¿Cómo ejerzo mis derechos sobre mis datos personales?',
     'Podés pedir acceso, rectificación, actualización o eliminación de tus datos escribiendo a stemhubsoporte@gmail.com. Encontrás el detalle en la Política de privacidad.')
) AS nueva(orden, funcionalidad, pregunta, respuesta)
WHERE NOT EXISTS (
    SELECT 1
    FROM preguntafrecuente pf
    WHERE LOWER(pf.preguntafrecuente) = LOWER(nueva.pregunta)
)
ORDER BY nueva.orden;

COMMIT;
