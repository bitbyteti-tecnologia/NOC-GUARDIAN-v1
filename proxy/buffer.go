// buffer.go (SQLite)
// - Armazena payloads JSON em fila ("backfill") quando não há internet.
// - FlushBuffer lê em ordem e envia; se sucesso, apaga do buffer.

package main

import (
    "database/sql"
    "encoding/base64"
    "errors"
    "log"

    _ "modernc.org/sqlite"
    "github.com/valyala/fasthttp"
)

var db *sql.DB

func InitBuffer(path string) error {
    var err error
    db, err = sql.Open("sqlite", path)
    if err != nil { return err }
    _, err = db.Exec(`CREATE TABLE IF NOT EXISTS queue (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      payload BLOB NOT NULL
    );`)
    return err
}

func BufferAppend(payload []byte) error {
    _, err := db.Exec(`INSERT INTO queue (payload) VALUES (?)`, payload)
    return err
}

func FlushBuffer(client *fasthttp.Client, url, token string) error {
    rows, err := db.Query(`SELECT id, payload FROM queue ORDER BY id ASC`)
    if err != nil { return err }
    defer rows.Close()

    type item struct{ id int; payload []byte }
    var batch []item
    for rows.Next() {
        var it item
        if err := rows.Scan(&it.id, &it.payload); err != nil { return err }
        batch = append(batch, it)
    }
    for _, it := range batch {
        if err := sendOnce(client, url, token, it.payload); err != nil {
            return err // para na primeira falha
        }
        _, _ = db.Exec(`DELETE FROM queue WHERE id=?`, it.id)
    }
    return nil
}

func sendOnce(client *fasthttp.Client, url, token string, payload []byte) error {
    req := fasthttp.AcquireRequest()
    resp := fasthttp.AcquireResponse()
    defer fasthttp.ReleaseRequest(req)
    defer fasthttp.ReleaseResponse(resp)

    req.Header.SetMethod("POST")
    req.SetRequestURI(url)
    req.Header.Set("Authorization", "Bearer "+token)
    req.Header.Set("Content-Type", "application/json")
    req.SetBody(payload)

    if err := client.Do(req, resp); err != nil {
        return err
    }
    if resp.StatusCode() >= 300 {
        return errors.New("central returned status " + base64.StdEncoding.EncodeToString([]byte(resp.Body())))
    }
    return nil
}
