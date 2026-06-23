# Go Torrent Client

[English](#english) | [Русский](#russian)

<a name="english"></a>
## English Documentation

### Overview
This is a command-line torrent client built in Go using the anacrolix/torrent library. It enables downloading files from either magnet links or torrent files with various configuration options including download/upload rate limiting, seeding options, and proxy support.

### Features
- Download torrents via magnet links or .torrent files
- Configure maximum number of peer connections
- Set download and upload speed limits
- Configure seeding options (ratio, enable/disable)
- Support for HTTP, HTTPS, and SOCKS5 proxies
- Real-time download progress with speed display (CLI)
- Graceful shutdown with CTRL+C
- Web UI daemon for managing multiple torrents simultaneously
- Pause, Resume, and Verify per torrent in the web UI
- Persistent state — torrents are restored after restart
- REST JSON API for all torrent operations

### Installation

#### Prerequisites
- Go 1.23 or later

#### Building from Source
```bash
# Clone the repository
git clone https://github.com/foxzi/gotorrentclient.git
cd gotorrentclient

# Build the binary
go build -o gotorrentclient main.go
```

### Usage

```bash
./gotorrentclient [options] <magnet link or torrent file path>
```

### Command-line Options

| Option | Default | Description |
|--------|---------|-------------|
| `--max-peers` | 50 | Maximum number of peers to connect to per torrent |
| `--download-dir` | "./downloads" | Directory where downloaded files will be saved |
| `--download-rate` | 0 (unlimited) | Maximum download rate in Mbps |
| `--upload-rate` | 0 (unlimited) | Maximum upload rate in Mbps |
| `--enable-seeding` | false | Enable seeding after download completes |
| `--seed-ratio` | 0 (unlimited) | Seed ratio (e.g., 1.0 means seed until you've uploaded as much as you've downloaded) |
| `--proxy` | "" | Proxy URL (supports HTTP, HTTPS, SOCKS5) |

### Examples

#### Download from a .torrent file
```bash
./gotorrentclient --download-dir="/home/user/downloads" my-torrent-file.torrent
```

#### Download from a magnet link
```bash
./gotorrentclient magnet:?xt=urn:btih:HASH&dn=NAME&tr=TRACKER
```

#### Limit download speed
```bash
./gotorrentclient --download-rate=5.5 magnet:?xt=urn:btih:HASH&dn=NAME&tr=TRACKER
```

#### Download through a proxy
```bash
# Using SOCKS5 proxy
./gotorrentclient --proxy="socks5://127.0.0.1:9050" magnet:?xt=urn:btih:HASH

# Using HTTP proxy
./gotorrentclient --proxy="http://proxy.example.com:8080" magnet:?xt=urn:btih:HASH
```

#### Combine multiple options
```bash
./gotorrentclient --max-peers=100 --download-rate=2.5 --download-dir="/media/downloads" --proxy="socks5://127.0.0.1:9050" magnet:?xt=urn:btih:HASH
```

#### Enable seeding with ratio
```bash
./gotorrentclient --enable-seeding --seed-ratio=1.5 magnet:?xt=urn:btih:HASH
# Will seed until upload/download ratio reaches 1.5
```

#### Limit upload speed while seeding
```bash
./gotorrentclient --enable-seeding --upload-rate=1.0 magnet:?xt=urn:btih:HASH
# Will seed with upload speed limited to 1 Mbps
```

#### Full example with seeding options
```bash
./gotorrentclient --download-rate=10.0 --upload-rate=2.0 --enable-seeding --seed-ratio=2.0 --max-peers=100 magnet:?xt=urn:btih:HASH
# Downloads at max 10 Mbps, seeds at max 2 Mbps until ratio reaches 2.0
```

### Web Interface

A built-in web daemon manages multiple torrents at once from a browser.

```bash
./gotorrentclient --web --username admin --password secret --download-dir ./downloads
# Open http://localhost:8080
```

Omitting credentials starts the server without authentication (anyone on the network can access it).

Credentials can also be set via `GTC_USERNAME` / `GTC_PASSWORD`, and the
listen address via `GTC_LISTEN`. See [docs/web-en.md](docs/web-en.md) for full details,
including the REST API reference.

<a name="russian"></a>
## Русская Документация

### Обзор
Это консольный торрент-клиент, написанный на языке Go с использованием библиотеки anacrolix/torrent. Он позволяет скачивать файлы как по magnet-ссылкам, так и через .torrent файлы с различными опциями настройки, включая ограничение скорости загрузки/раздачи, настройку сидирования и поддержку прокси.

### Функции
- Загрузка торрентов через magnet-ссылки или .torrent файлы
- Настройка максимального количества пиров
- Установка ограничения скорости загрузки и раздачи
- Настройка параметров сидирования (коэффициент раздачи, включение/выключение)
- Поддержка HTTP, HTTPS и SOCKS5 прокси
- Отображение прогресса загрузки со скоростью в реальном времени (CLI)
- Корректное завершение работы по CTRL+C
- Веб-демон для одновременного управления несколькими торрентами
- Пауза, возобновление и верификация торрентов в веб-интерфейсе
- Сохранение состояния — торренты восстанавливаются после перезапуска
- REST JSON API для всех операций с торрентами

### Установка

#### Требования
- Go 1.23 или новее

#### Сборка из исходного кода
```bash
# Клонирование репозитория
git clone https://github.com/foxzi/gotorrentclient.git
cd gotorrentclient

# Сборка исполняемого файла
go build -o gotorrentclient main.go
```

### Использование

```bash
./gotorrentclient [опции] <magnet-ссылка или путь к torrent-файлу>
```

### Параметры командной строки

| Опция | Значение по умолчанию | Описание |
|-------|---------|-------------|
| `--max-peers` | 50 | Максимальное количество пиров для подключения к каждому торренту |
| `--download-dir` | "./downloads" | Директория, в которую будут сохранены загруженные файлы |
| `--download-rate` | 0 (без ограничения) | Максимальная скорость загрузки в Мбит/с |
| `--upload-rate` | 0 (без ограничения) | Максимальная скорость раздачи в Мбит/с |
| `--enable-seeding` | false | Включить раздачу (сидирование) после завершения загрузки |
| `--seed-ratio` | 0 (без ограничения) | Коэффициент раздачи (например, 1.0 означает раздавать, пока не отдадите столько же, сколько скачали) |
| `--proxy` | "" | URL прокси-сервера (поддерживаются HTTP, HTTPS, SOCKS5) |

### Примеры

#### Загрузка из .torrent файла
```bash
./gotorrentclient --download-dir="/home/user/downloads" my-torrent-file.torrent
```

#### Загрузка по magnet-ссылке
```bash
./gotorrentclient magnet:?xt=urn:btih:ХЕШ&dn=ИМЯ&tr=ТРЕКЕР
```

#### Ограничение скорости загрузки
```bash
./gotorrentclient --download-rate=5.5 magnet:?xt=urn:btih:ХЕШ&dn=ИМЯ&tr=ТРЕКЕР
```

#### Загрузка через прокси
```bash
# Использование SOCKS5 прокси
./gotorrentclient --proxy="socks5://127.0.0.1:9050" magnet:?xt=urn:btih:ХЕШ

# Использование HTTP прокси
./gotorrentclient --proxy="http://proxy.example.com:8080" magnet:?xt=urn:btih:ХЕШ
```

#### Комбинирование нескольких опций
```bash
./gotorrentclient --max-peers=100 --download-rate=2.5 --download-dir="/media/downloads" --proxy="socks5://127.0.0.1:9050" magnet:?xt=urn:btih:ХЕШ
```

#### Включение раздачи с указанием коэффициента
```bash
./gotorrentclient --enable-seeding --seed-ratio=1.5 magnet:?xt=urn:btih:ХЕШ
# Будет раздавать, пока соотношение отданного к скачанному не достигнет 1.5
```

#### Ограничение скорости раздачи
```bash
./gotorrentclient --enable-seeding --upload-rate=1.0 magnet:?xt=urn:btih:ХЕШ
# Будет раздавать с ограничением скорости в 1 Мбит/с
```

#### Полный пример с опциями раздачи
```bash
./gotorrentclient --download-rate=10.0 --upload-rate=2.0 --enable-seeding --seed-ratio=2.0 --max-peers=100 magnet:?xt=urn:btih:ХЕШ
# Загрузка со скоростью до 10 Мбит/с, раздача со скоростью до 2 Мбит/с до достижения коэффициента 2.0
```
### Веб-интерфейс

Встроенный веб-демон управляет несколькими торрентами одновременно из браузера.

```bash
./gotorrentclient --web --username admin --password secret --download-dir ./downloads
# Откройте http://localhost:8080
```

Если не указать учётные данные, сервер запустится без аутентификации (интерфейс доступен всем в сети).

Учётные данные также можно задать через `GTC_USERNAME` / `GTC_PASSWORD`,
а адрес прослушивания — через `GTC_LISTEN`. Подробнее, включая справочник REST API: [docs/web-ru.md](docs/web-ru.md).
