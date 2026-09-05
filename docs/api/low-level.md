# Низкоуровневый API

Эти пакеты нужны для собственного транспорта, event loop и wire-протокола.
Обычному приложению достаточно `gomax.NewClient` и `gomax.NewWebClient`.
Каждая строка ниже показывает назначение и минимальный вызов.

Эта страница описывает расширение библиотеки, а не обычный сценарий бота. Для
прикладного кода сначала используйте `gomax.NewClient` или
`gomax.NewWebClient`.

## `pkg/api`

| Функция/метод | Назначение | Пример |
|---|---|---|
| `NewAuthService` | Создать auth-сервис с `api.Invoker`. | `auth.NewAuthService(invoker)` |
| `NewBotsService` | Создать bot-сервис. | `bots.NewBotsService(invoker)` |
| `NewChatService` | Создать сервис чатов. | `chats.NewChatService(invoker)` |
| `NewMessageService` | Создать сервис сообщений. | `messages.NewMessageService(invoker)` |
| `NewSelfService` | Создать сервис профиля. | `selfapi.NewSelfService(invoker)` |
| `NewUploadService` | Создать сервис загрузок. | `uploads.NewUploadService(invoker)` |
| `NewUserService` | Создать сервис пользователей. | `users.NewUserService(invoker)` |
| `Invoker.Invoke` | Выполнить RPC по opcode. | `result, err := invoker.Invoke(ctx, op, payload)` |

## `pkg/dispatch`

Типы обработчиков задают сигнатуры callback-функций: `MessageHandler`,
`MessageEditHandler`, `MessageDeleteHandler`, `MessageReadHandler`,
`UserUpdateHandler`, `ReactionHandler`, `ChatUpdateHandler`,
`PresenceHandler`, `TypingHandler`, `DisconnectHandler`, `StartHandler`,
`EventHandler` и `MessagePredicate`.

### Регистрация обработчиков

| Метод | Назначение | Пример |
|---|---|---|
| `NewRouter` | Создать маршрутизатор. | `router := dispatch.NewRouter()` |
| `OnMessage` | Обработчик сообщений и фильтры. | `router.OnMessage(handler, predicate)` |
| `OnMessageEdit` | Обработчик изменения сообщения. | `router.OnMessageEdit(handler)` |
| `OnMessageDelete` | Обработчик удаления. | `router.OnMessageDelete(handler)` |
| `OnMessageRead` | Обработчик read-marker. | `router.OnMessageRead(handler)` |
| `OnUserUpdate` | Обработчик изменения пользователя. | `router.OnUserUpdate(handler)` |
| `OnReaction` | Обработчик реакции. | `router.OnReaction(handler)` |
| `OnChatUpdate` | Обработчик изменения чата. | `router.OnChatUpdate(handler)` |
| `OnPresence` | Обработчик presence. | `router.OnPresence(handler)` |
| `OnTyping` | Обработчик typing. | `router.OnTyping(handler)` |
| `OnDisconnect` | Обработчик disconnect. | `router.OnDisconnect(handler)` |
| `OnStart` | Обработчик старта. | `router.OnStart(handler)` |
| `OnEvent` | Обработчик raw event. | `router.OnEvent(handler)` |

### Ручная отправка событий

| Метод | Назначение | Пример |
|---|---|---|
| `DispatchMessage` | Передать сообщение обработчикам. | `router.DispatchMessage(ctx, msg)` |
| `DispatchMessageEdit` | Передать изменение сообщения. | `router.DispatchMessageEdit(ctx, msg)` |
| `DispatchMessageDelete` | Передать удаление. | `router.DispatchMessageDelete(ctx, chatID, messageID)` |
| `DispatchMessageRead` | Передать read-marker. | `router.DispatchMessageRead(ctx, event)` |
| `DispatchUserUpdate` | Передать изменение пользователя. | `router.DispatchUserUpdate(ctx, event)` |
| `DispatchReaction` | Передать реакцию. | `router.DispatchReaction(ctx, event)` |
| `DispatchChatUpdate` | Передать изменение чата. | `router.DispatchChatUpdate(ctx, chat)` |
| `DispatchPresence` | Передать presence. | `router.DispatchPresence(ctx, event)` |
| `DispatchTyping` | Передать typing. | `router.DispatchTyping(ctx, event)` |
| `DispatchDisconnect` | Передать ошибку соединения. | `router.DispatchDisconnect(ctx, err)` |
| `DispatchStart` | Передать событие готовности. | `router.DispatchStart(ctx)` |
| `DispatchEvent` | Передать raw event. | `router.DispatchEvent(ctx, event)` |

## `pkg/protocol`

Основные интерфейсы:

```go
type Protocol interface {
    Version() uint8
    Encode(*OutboundFrame) ([]byte, error)
    Decode([]byte) (*InboundFrame, error)
}

type Decompressor interface {
    Decompress(src []byte, maxOutput int) ([]byte, error)
}
```

### Кодирование и кадры

| Функция/метод | Назначение | Пример |
|---|---|---|
| `NewMsgpackCodec` | Создать MessagePack codec. | `codec := protocol.NewMsgpackCodec()` |
| `MsgpackCodec.Encode` | Кодировать payload. | `data, err := codec.Encode(payload)` |
| `MsgpackCodec.Decode` | Декодировать payload. | `value, err := codec.Decode(data)` |
| `MsgpackCodec.Normalize` | Нормализовать decoded value. | `value = codec.Normalize(value)` |
| `MsgpackCodec.NormalizeKey` | Привести ключ map к строке. | `key := codec.NormalizeKey(rawKey)` |
| `Ext.EncodeMsgpack` | Кодировать custom extension. | `err := ext.EncodeMsgpack(enc)` |
| `NewRequest` | Создать request frame. | `frame := protocol.NewRequest(op, seq, payload)` |
| `NewEvent` | Создать event frame. | `frame := protocol.NewEvent(op, seq, payload)` |
| `InboundFrame.IsResponse` | Проверить response frame. | `if frame.IsResponse() {}` |
| `InboundFrame.IsEvent` | Проверить event frame. | `if frame.IsEvent() {}` |
| `InboundFrame.IsError` | Проверить error frame. | `if frame.IsError() {}` |
| `InboundFrame.ErrorString` | Получить текст ошибки frame. | `text := frame.ErrorString()` |
| `NewApiError` | Создать ошибку API из frame. | `err := protocol.NewApiError(frame)` |
| `ApiError.Error` | Получить текст API-ошибки. | `log.Println(err.Error())` |

### Header и протокол

| Функция/метод | Назначение | Пример |
|---|---|---|
| `DecodeHeader` | Разобрать 10-байтный header. | `header, err := protocol.DecodeHeader(data)` |
| `ExtractPayloadLen` | Прочитать длину payload. | `n, err := protocol.ExtractPayloadLen(data)` |
| `Header.Encode` | Закодировать header. | `data := header.Encode()` |
| `Header.EncodeTo` | Закодировать header в готовый buffer. | `err := header.EncodeTo(dst)` |
| `Command.String` | Получить имя команды. | `name := command.String()` |
| `Command.IsValid` | Проверить команду. | `ok := command.IsValid()` |
| `Opcode.String` | Получить имя opcode. | `name := opcode.String()` |
| `Opcode.IsNotification` | Проверить notification opcode. | `ok := opcode.IsNotification()` |
| `CountOpcodes` | Получить количество opcode. | `count := protocol.CountOpcodes()` |
| `NewTcpProtocol` | Создать TCP protocol. | `p, err := protocol.NewTcpProtocol()` |
| `TcpProtocol.Version` | Получить версию TCP protocol. | `version := p.Version()` |
| `TcpProtocol.Encode` | Закодировать TCP frame. | `raw, err := p.Encode(frame)` |
| `TcpProtocol.Decode` | Разобрать TCP frame. | `frame, err := p.Decode(raw)` |
| `NewWsProtocol` | Создать WebSocket protocol. | `p, err := protocol.NewWsProtocol(true)` |
| `WsProtocol.Version` | Получить версию WS protocol. | `version := p.Version()` |
| `WsProtocol.Encode` | Закодировать WS frame. | `raw, err := p.Encode(frame)` |
| `WsProtocol.Decode` | Разобрать WS frame. | `frame, err := p.Decode(raw)` |

### Сжатие

| Функция/метод | Назначение | Пример |
|---|---|---|
| `NewLZ4BlockDecompressor` | Создать LZ4 decompressor. | `d := protocol.NewLZ4BlockDecompressor()` |
| `LZ4BlockDecompressor.Decompress` | Распаковать LZ4 block. | `data, err := d.Decompress(src, limit)` |
| `NewZstdDecompressor` | Создать Zstandard decompressor. | `d, err := protocol.NewZstdDecompressor()` |
| `ZstdDecompressor.Decompress` | Распаковать Zstandard payload. | `data, err := d.Decompress(src, limit)` |
| `NewPayloadDecoder` | Создать decoder payload. | `d, err := protocol.NewPayloadDecoder(codec)` |
| `PayloadDecoder.Decode` | Распаковать и декодировать payload. | `payload, err := d.Decode(data, flags)` |

## `pkg/connection`

Основные интерфейсы и настройки:

```go
type Reader interface {
    ReadFrame() ([]byte, error)
}

type Connection interface {
    Start(context.Context) error
    Close() error
    WaitClosed() error
    SendRequest(context.Context, protocol.Opcode, any) (*protocol.InboundFrame, error)
    SendEvent(context.Context, protocol.Opcode, any) error
    Events() <-chan *protocol.InboundFrame
    Handshake(context.Context, any) (*protocol.InboundFrame, error)
    IsOpen() bool
}
```

`connection.Config` содержит `Interactive`, `PingInterval`, `PingTimeout`,
`RequestTimeout`, `EventsChanSize` и `ProtocolVersion`. Пакет также экспортирует
ошибки `ErrConnectionNotOpen`, `ErrConnectionAlreadyOpen`, `ErrPingTimeout`,
`ErrRequestCancelled`, `ErrConnectionClosed`, `ErrFrameTooLarge` и
`ErrIncompleteFrame`.

| Функция/метод | Назначение | Пример |
|---|---|---|
| `DefaultConfig` | Получить параметры connection manager. | `cfg := connection.DefaultConfig()` |
| `NewConnectionManager` | Создать manager RPC и keepalive. | `m := connection.NewConnectionManager(reader, transport, protocol, &cfg, onClose, onEvent)` |
| `ConnectionManager.Start` | Открыть циклы чтения и keepalive. | `err := m.Start(ctx)` |
| `ConnectionManager.Close` | Закрыть manager. | `err := m.Close()` |
| `ConnectionManager.Fail` | Завершить manager с ошибкой. | `m.Fail(err)` |
| `ConnectionManager.WaitClosed` | Дождаться закрытия. | `err := m.WaitClosed()` |
| `ConnectionManager.IsOpen` | Проверить состояние. | `if m.IsOpen() {}` |
| `ConnectionManager.Events` | Получить канал событий. | `for event := range m.Events() {}` |
| `ConnectionManager.Send` | Отправить frame. | `err := m.Send(ctx, frame)` |
| `ConnectionManager.SendEvent` | Отправить event opcode. | `err := m.SendEvent(ctx, op, payload)` |
| `ConnectionManager.SendRequest` | Отправить request и дождаться ответа. | `reply, err := m.SendRequest(ctx, op, payload)` |
| `ConnectionManager.Handshake` | Выполнить handshake. | `reply, err := m.Handshake(ctx, payload)` |
| `NewSeqGenerator` | Создать генератор sequence. | `seq := connection.NewSeqGenerator()` |
| `NewSeqGeneratorWithStart` | Создать генератор с начальным числом. | `seq := connection.NewSeqGeneratorWithStart(100)` |
| `SeqGenerator.Next` | Получить следующий sequence. | `n := seq.Next()` |
| `SeqGenerator.Current` | Прочитать текущий sequence. | `n := seq.Current()` |
| `SeqGenerator.Reset` | Установить sequence. | `seq.Reset(0)` |
| `NewPendingTracker` | Создать tracker ожиданий. | `pending := connection.NewPendingTracker()` |
| `PendingTracker.Create` | Создать waiter для sequence. | `ch := pending.Create(seq)` |
| `PendingTracker.Resolve` | Завершить waiter frame-ом. | `ok := pending.Resolve(seq, frame)` |
| `PendingTracker.Discard` | Отменить waiter. | `pending.Discard(seq)` |
| `PendingTracker.CancelAll` | Отменить все waiter-ы. | `pending.CancelAll()` |
| `PendingTracker.Count` | Получить число waiter-ов. | `n := pending.Count()` |
| `PendingTracker.Has` | Проверить sequence. | `ok := pending.Has(seq)` |
| `NewTCPReader` | Создать reader для TCP. | `reader := connection.NewTCPReader(t)` |
| `TCPReader.ReadFrame` | Прочитать TCP frame. | `data, err := reader.ReadFrame()` |
| `NewWSReader` | Создать reader для WebSocket. | `reader := connection.NewWSReader(t)` |
| `WSReader.ReadFrame` | Прочитать WS frame. | `data, err := reader.ReadFrame()` |

## `pkg/transport`

`transport.Transport` — общий интерфейс TCP и WebSocket транспорта:

```go
type Transport interface {
    io.Closer
    Connect(context.Context) error
    Send([]byte) error
    Recv(int) ([]byte, error)
    Connected() bool
}
```

`TCPOptions` настраивает `Host`, `Port`, `UseSSL`, `ProxyURL`, `TLSConfig`,
`ConnectTimeout` и `CloseTimeout`. `WSOptions` настраивает `URL`, `Origin`,
`ProxyURL`, `TLSConfig`, таймауты и `MaxMessageSize`. Экспортируемые ошибки:
`ErrNotConnected`, `ErrAlreadyConnected`, `ErrClosed`, `ErrInvalidProxy` и
`ErrMessageTooLarge`.

| Функция/метод | Назначение | Пример |
|---|---|---|
| `DefaultTCPOptions` | Получить TCP defaults. | `opts := transport.DefaultTCPOptions()` |
| `NewTCPTransport` | Создать TCP/TLS transport. | `t := transport.NewTCPTransport(opts)` |
| `TCPTransport.Connect` | Подключиться. | `err := t.Connect(ctx)` |
| `TCPTransport.Send` | Отправить bytes. | `err := t.Send(data)` |
| `TCPTransport.Recv` | Прочитать bytes. | `data, err := t.Recv(size)` |
| `TCPTransport.Connected` | Проверить подключение. | `ok := t.Connected()` |
| `TCPTransport.Close` | Закрыть transport. | `err := t.Close()` |
| `DefaultWSOptions` | Получить WS defaults. | `opts := transport.DefaultWSOptions(url)` |
| `NewWebSocketTransport` | Создать WebSocket transport. | `t := transport.NewWebSocketTransport(opts)` |
| `WebSocketTransport.Connect` | Подключиться. | `err := t.Connect(ctx)` |
| `WebSocketTransport.Send` | Отправить binary message. | `err := t.Send(data)` |
| `WebSocketTransport.Recv` | Прочитать message. | `data, err := t.Recv(size)` |
| `WebSocketTransport.Connected` | Проверить подключение. | `ok := t.Connected()` |
| `WebSocketTransport.Close` | Закрыть transport. | `err := t.Close()` |
| `GetEmbeddedRootCAPEM` | Получить встроенный CA PEM. | `pem := transport.GetEmbeddedRootCAPEM()` |
| `GetRootCACertPool` | Создать CA pool. | `pool, err := transport.GetRootCACertPool()` |
| `NewTLSConfig` | Создать TLS config. | `tlsCfg, err := transport.NewTLSConfig("api2.oneme.ru")` |

## `pkg/fingerprint`

| Функция/метод | Назначение | Пример |
|---|---|---|
| `DefaultFingerprint` | Получить мобильный fingerprint profile. | `profile := fingerprint.DefaultFingerprint()` |
| `NewFingerprintGenerator` | Создать генератор. | `g := fingerprint.NewFingerprintGenerator(profile)` |
| `GenerateFingerprint` | Создать fingerprint для устройства. | `data, err := g.GenerateFingerprint(deviceID, callsSeed, "arm64-v8a")` |
