# Video ASCII

Conversor de imágenes y videos a arte ASCII en Go.

## Características (Fase 1)

- Decodificación de imágenes en formatos PNG, JPEG y GIF.
- Redimensionamiento proporcional preservando la relación de aspecto de la fuente monoespaciada.
- Cálculo de luminancia según el estándar BT.709 y mapeo a rampa de caracteres ASCII.
- Interfaz de línea de comandos (CLI) con opciones de dimensiones, rampa de caracteres y exportación a archivo.
- Servidor HTTP con endpoint `POST /render/image` para integración con aplicaciones frontend.

## Requisitos

- Go 1.22 o superior

## Uso

### CLI

Ejecutar sobre una imagen:

```bash
cd back
go run ./cmd/asciirender ruta/a/la/imagen.png
```

Opciones disponibles:
- `-width <int>`: Ancho en caracteres (por defecto: 80).
- `-height <int>`: Alto en caracteres (0 para cálculo automático según aspecto).
- `-ramp <string>`: Cadena personalizada de caracteres (de claro a oscuro).
- `-output <string>`: Ruta del archivo donde guardar la salida ASCII.

### Servidor HTTP

Iniciar el servidor API:

```bash
cd back
go run ./cmd/asciirender serve -port 8080
```

Enviar una petición con una imagen:

```bash
curl -X POST -F "image=@ruta/a/la/imagen.png" "http://localhost:8080/render/image?width=80"
```

## Pruebas

Para ejecutar las pruebas unitarias:

```bash
cd back
go test ./...
```
