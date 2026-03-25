package queue

import (
    "context"
    "encoding/json"
    "errors"
    "log/slog"
    "time"

    amqp "github.com/rabbitmq/amqp091-go"

    "github.com/bitbyteti/noc-guardian/async/internal/config"
    "github.com/bitbyteti/noc-guardian/async/internal/models"
)

type RabbitMQ struct {
    cfg  config.Config
    conn *amqp.Connection
    ch   *amqp.Channel
}

func NewRabbitMQ(cfg config.Config) (*RabbitMQ, error) {
    conn, err := amqp.Dial(cfg.RabbitURL)
    if err != nil {
        return nil, err
    }
    ch, err := conn.Channel()
    if err != nil {
        _ = conn.Close()
        return nil, err
    }

    if err := ch.Confirm(false); err != nil {
        _ = ch.Close()
        _ = conn.Close()
        return nil, err
    }

    if err := declareTopology(ch, cfg); err != nil {
        _ = ch.Close()
        _ = conn.Close()
        return nil, err
    }

    return &RabbitMQ{cfg: cfg, conn: conn, ch: ch}, nil
}

func (r *RabbitMQ) Close() {
    if r.ch != nil {
        _ = r.ch.Close()
    }
    if r.conn != nil {
        _ = r.conn.Close()
    }
}

func declareTopology(ch *amqp.Channel, cfg config.Config) error {
    if err := ch.ExchangeDeclare(cfg.RabbitExchange, "direct", true, false, false, false, nil); err != nil {
        return err
    }
    if err := ch.ExchangeDeclare(cfg.RabbitRetryExchange, "direct", true, false, false, false, nil); err != nil {
        return err
    }
    if err := ch.ExchangeDeclare(cfg.RabbitDeadExchange, "direct", true, false, false, false, nil); err != nil {
        return err
    }

    queueArgs := amqp.Table{
        "x-dead-letter-exchange": cfg.RabbitDeadExchange,
        "x-dead-letter-routing-key": cfg.RabbitDeadRoutingKey,
    }
    if _, err := ch.QueueDeclare(cfg.RabbitQueue, true, false, false, false, queueArgs); err != nil {
        return err
    }
    if err := ch.QueueBind(cfg.RabbitQueue, cfg.RabbitRoutingKey, cfg.RabbitExchange, false, nil); err != nil {
        return err
    }

    retryArgs := amqp.Table{
        "x-message-ttl":             int32(cfg.RabbitRetryTTLMS),
        "x-dead-letter-exchange":    cfg.RabbitExchange,
        "x-dead-letter-routing-key": cfg.RabbitRoutingKey,
    }
    if _, err := ch.QueueDeclare(cfg.RabbitRetryQueue, true, false, false, false, retryArgs); err != nil {
        return err
    }
    if err := ch.QueueBind(cfg.RabbitRetryQueue, cfg.RabbitRetryRoutingKey, cfg.RabbitRetryExchange, false, nil); err != nil {
        return err
    }

    if _, err := ch.QueueDeclare(cfg.RabbitDeadQueue, true, false, false, false, nil); err != nil {
        return err
    }
    if err := ch.QueueBind(cfg.RabbitDeadQueue, cfg.RabbitDeadRoutingKey, cfg.RabbitDeadExchange, false, nil); err != nil {
        return err
    }

    return nil
}

func (r *RabbitMQ) PublishMetric(ctx context.Context, m models.Metric, headers amqp.Table) error {
    body, err := json.Marshal(m)
    if err != nil {
        return err
    }

    confirms := r.ch.NotifyPublish(make(chan amqp.Confirmation, 1))

    err = r.ch.PublishWithContext(
        ctx,
        r.cfg.RabbitExchange,
        r.cfg.RabbitRoutingKey,
        false,
        false,
        amqp.Publishing{
            ContentType:  "application/json",
            Body:         body,
            DeliveryMode: amqp.Persistent,
            Timestamp:    time.Now().UTC(),
            Headers:      headers,
        },
    )
    if err != nil {
        return err
    }

    select {
    case confirm := <-confirms:
        if !confirm.Ack {
            return errors.New("publish not confirmed")
        }
    case <-ctx.Done():
        return ctx.Err()
    }

    return nil
}

func (r *RabbitMQ) PublishRetry(ctx context.Context, body []byte, headers amqp.Table) error {
    confirms := r.ch.NotifyPublish(make(chan amqp.Confirmation, 1))

    err := r.ch.PublishWithContext(
        ctx,
        r.cfg.RabbitRetryExchange,
        r.cfg.RabbitRetryRoutingKey,
        false,
        false,
        amqp.Publishing{
            ContentType:  "application/json",
            Body:         body,
            DeliveryMode: amqp.Persistent,
            Timestamp:    time.Now().UTC(),
            Headers:      headers,
        },
    )
    if err != nil {
        return err
    }

    select {
    case confirm := <-confirms:
        if !confirm.Ack {
            return errors.New("retry publish not confirmed")
        }
    case <-ctx.Done():
        return ctx.Err()
    }

    return nil
}

func (r *RabbitMQ) PublishDead(ctx context.Context, body []byte, headers amqp.Table) error {
    confirms := r.ch.NotifyPublish(make(chan amqp.Confirmation, 1))

    err := r.ch.PublishWithContext(
        ctx,
        r.cfg.RabbitDeadExchange,
        r.cfg.RabbitDeadRoutingKey,
        false,
        false,
        amqp.Publishing{
            ContentType:  "application/json",
            Body:         body,
            DeliveryMode: amqp.Persistent,
            Timestamp:    time.Now().UTC(),
            Headers:      headers,
        },
    )
    if err != nil {
        return err
    }

    select {
    case confirm := <-confirms:
        if !confirm.Ack {
            return errors.New("dead-letter publish not confirmed")
        }
    case <-ctx.Done():
        return ctx.Err()
    }

    return nil
}

func (r *RabbitMQ) Consume(prefetch int) (<-chan amqp.Delivery, error) {
    if err := r.ch.Qos(prefetch, 0, false); err != nil {
        return nil, err
    }
    return r.ch.Consume(
        r.cfg.RabbitQueue,
        "",
        false,
        false,
        false,
        false,
        nil,
    )
}

func RetryCount(headers amqp.Table) int {
    if headers == nil {
        return 0
    }
    raw, ok := headers["x-retry-count"]
    if !ok {
        return 0
    }
    switch v := raw.(type) {
    case int32:
        return int(v)
    case int64:
        return int(v)
    case int:
        return v
    case float32:
        return int(v)
    case float64:
        return int(v)
    default:
        slog.Warn("unexpected retry header type", "type", raw)
        return 0
    }
}

func WithRetryCount(headers amqp.Table, count int) amqp.Table {
    if headers == nil {
        headers = amqp.Table{}
    }
    headers["x-retry-count"] = count
    return headers
}
