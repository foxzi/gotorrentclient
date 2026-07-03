# Пакеты deb / rpm и systemd-сервис

gotorrentclient можно установить как системный сервис из пакетов `.deb`
(Debian, Ubuntu, Raspberry Pi OS) или `.rpm` (Fedora, RHEL, Rocky, openSUSE).
Пакет ставит бинарник, unit-файл systemd, конфиг в `/etc` и создаёт
отдельного системного пользователя.

## Что кладёт пакет

| Путь | Назначение |
|------|-----------|
| `/usr/bin/gotorrentclient` | исполняемый файл |
| `/lib/systemd/system/gotorrentclient.service` | unit systemd |
| `/etc/gotorrentclient/config.yaml` | конфиг (сохраняется при обновлении) |
| `/var/lib/gotorrentclient/downloads` | каталог загрузок и состояния |

Сервис работает под пользователем `gotorrentclient` (создаётся при установке).
Конфиг доступен на чтение только root и группе сервиса (права `0640`).

## Сборка пакетов

Для сборки нужны Go и Docker (nfpm запускается в контейнере).

```bash
./packaging/build-packages.sh
```

Скрипт собирает статические бинарники под `amd64` и `arm64`, затем через
[nfpm](https://nfpm.goreleaser.com/) формирует `.deb` и `.rpm`. Готовые пакеты
складываются в `release/`. Версия берётся из `git describe`.

## Установка

Debian / Ubuntu / Raspberry Pi OS:

```bash
sudo dpkg -i gotorrentclient_<версия>_arm64.deb
```

Fedora / RHEL / Rocky:

```bash
sudo rpm -i gotorrentclient-<версия>.aarch64.rpm
```

Выберите пакет под свою архитектуру: `amd64` / `x86_64` для обычных ПК,
`arm64` / `aarch64` для Raspberry Pi и других ARM-плат.

## Запуск сервиса

По умолчанию сервис не запускается автоматически, чтобы дать возможность
сначала настроить конфиг.

```bash
sudo nano /etc/gotorrentclient/config.yaml   # задайте username/password
sudo systemctl enable --now gotorrentclient  # включить и запустить
```

Полезные команды:

```bash
systemctl status gotorrentclient      # состояние
journalctl -u gotorrentclient -f      # логи в реальном времени
sudo systemctl restart gotorrentclient
```

После запуска веб-интерфейс доступен на порту из `listen` (по умолчанию `:8080`).

> Внимание: если `username` и `password` пусты, веб-интерфейс открыт для всех.
> Задайте учётные данные перед публикацией в сеть.

## Обновление

Установите новый пакет поверх старого — файл `/etc/gotorrentclient/config.yaml`
не будет перезаписан (он помечен как конфигурационный). После обновления
перезапустите сервис:

```bash
sudo systemctl restart gotorrentclient
```

## Удаление

```bash
sudo dpkg -r gotorrentclient      # deb
sudo rpm -e gotorrentclient       # rpm
```

Каталог `/var/lib/gotorrentclient` с загрузками и пользователь `gotorrentclient`
намеренно остаются, чтобы не потерять данные. Для полной очистки:

```bash
sudo userdel gotorrentclient
sudo rm -rf /var/lib/gotorrentclient
```
