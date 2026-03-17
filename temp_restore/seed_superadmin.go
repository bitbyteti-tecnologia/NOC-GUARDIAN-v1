// seed_superadmin.go
// - Garante um usuário SUPERADMIN global no startup.
// - Lê SUPERADMIN_EMAIL e SUPERADMIN_PASSWORD do .env.
// - Se o email não existir, cria com a senha fornecida.
// - Se variáveis não definidas, não cria (evita senhas fracas por engano).

package main

import (
    "context"
    "log"
    "os"
)

func SeedSuperAdmin() error {
    email := os.Getenv("SUPERADMIN_EMAIL")
    pass := os.Getenv("SUPERADMIN_PASSWORD")
    if email == "" || pass == "" {
        log.Println("SeedSuperAdmin: SUPERADMIN_EMAIL/SUPERADMIN_PASSWORD não definidos; pulando seed.")
        return nil
    }
    var exists bool
    err := MasterConn.QueryRow(context.Background(),
        "SELECT EXISTS(SELECT 1 FROM users WHERE email=$1)", email).Scan(&exists)
    if err != nil {
        return err
    }
    if exists {
        log.Printf("SeedSuperAdmin: usuário %s já existe (ok).", email)
        return nil
    }
    hash, err := HashPassword(pass)
    if err != nil { return err }
    _, err = MasterConn.Exec(context.Background(),
        `INSERT INTO users (email, password_hash, role, tenant_id) VALUES ($1,$2,'superadmin',NULL)`,
        email, hash)
    if err == nil {
        log.Printf("SeedSuperAdmin: criado superadmin %s", email)
    }
    return err
}
