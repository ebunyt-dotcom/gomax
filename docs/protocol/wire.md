# Протокол

Этот раздел нужен тем, кто отлаживает transport или добавляет новый RPC-метод. Для обычного приложения достаточно сервисов `Client`/`WebClient`.

## Транспорты

| Транспорт | Клиент | Назначение |
|---|---|---|
| TLS TCP | `NewClient` | Мобильный handshake и SMS |
| Binary WebSocket | `NewWebClient` | Web handshake и QR |

По умолчанию используются `api2.oneme.ru:443` и `wss://api.oneme.ru/websocket`.

## Бинарный frame

TCP и binary WebSocket используют 10-байтный заголовок Big-Endian:

| Байты | Поле | Размер |
|---|---|---:|
| `0` | `Version` | 1 |
| `1` | `Command` | 1 |
| `2..3` | `Sequence` | 2 |
| `4..5` | `Opcode` | 2 |
| `6` | `Flags` | 1 |
| `7..9` | `PayloadLen` | 3 |

Payload — MessagePack. Команды: `0 REQUEST`, `1 RESPONSE`, `2 EVENT`, `3 ERROR`. `Sequence` связывает ответ с запросом.

## Сжатие

- `0x00` — без сжатия;
- значения от `0x01` до `0x7f` — raw LZ4 block;
- `0xff` — Zstandard.

GoMax ограничивает размер frame и распакованного payload, чтобы повреждённые данные не приводили к неограниченному выделению памяти.

## Handshake и вход

Мобильный порядок:

1. `SESSION_INIT` (`6`) с `deviceId`, `mt_instanceid`, `userAgent`, `clientSessionId`;
2. `AUTH_REQUEST` (`17`) и `AUTH` (`18`) для SMS;
3. `LOGIN` (`19`) с token и sync-состоянием.

Web/QR порядок:

1. WebSocket `SESSION_INIT` (`6`);
2. `GET_QR` (`288`);
3. `GET_QR_STATUS` (`289`);
4. `LOGIN_BY_QR` (`291`);
5. `LOGIN` (`19`).

## Основные опкоды

| Код | Константа | Назначение |
|---:|---|---|
| 6 | `OpSessionInit` | Handshake |
| 16 | `OpProfile` | Профиль |
| 17/18/19 | `OpAuthRequest` / `OpAuth` / `OpLogin` | SMS и login |
| 48/49/53 | `OpChatInfo` / `OpChatHistory` / `OpChatsList` | Чаты и история |
| 50 | `OpChatMark` | Read markers |
| 55/57/58/59 | `OpChatUpdate` / `OpChatJoin` / `OpChatLeave` / `OpChatMembers` | Управление чатами |
| 64/66/67 | `OpMsgSend` / `OpMsgDelete` / `OpMsgEdit` | Сообщения |
| 77 | `OpChatMembersUpdate` | Участники и заявки |
| 80/82/87 | `OpPhotoUpload` / `OpVideoUpload` / `OpFileUpload` | Upload slots |
| 128..136 | `OpNotifMessage` .. `OpNotifAttach` | Push events |
| 178..181 | `OpMsgReaction` .. `OpMsgGetDetailedReactions` | Реакции |
| 272..277 | `OpFoldersGet` .. `OpNotifFolders` | Папки |
| 288/289/291 | `OpGetQr` / `OpGetQrStatus` / `OpLoginByQr` | QR-вход |

Полный каталог находится в [`pkg/protocol/opcode.go`](../../pkg/protocol/opcode.go).

## Расширение API

Для нового RPC используйте `Invoker`:

```go
type Invoker interface {
    Invoke(context.Context, protocol.Opcode, interface{}) (map[string]interface{}, error)
}
```

Добавьте opcode в `pkg/protocol/opcode.go`, payload в соответствующий сервис и обработку результата. Не обходите `ConnectionManager`: он отвечает за sequence, таймауты и корреляцию ответов.
