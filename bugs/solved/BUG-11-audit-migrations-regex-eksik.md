# BUG-11: `AuditMigrations` — Regex `down` handler pattern'lerini kaçırıyor

## Kritiklik: DÜŞÜK

## Dosya
`core/repair/repair.go` satır 19-22

## Mevcut Kod

```go
var (
    hasUpRegex   = regexp.MustCompile(`(async\s+up|up\s*:\s*function|up\s*:\s*async)`)
    hasDownRegex = regexp.MustCompile(`(async\s+down|down\s*:\s*function|down\s*:\s*async)`)
)
```

## Bug Açıklaması

Regex sadece şu pattern'leri yakalıyor:
- `async down(queryInterface, Sequelize)`
- `down: function(...)`
- `down: async function(...)`

Şu pattern'leri **kaçırıyor**:
- `down(queryInterface, Sequelize) { }` (async yok, arrow function)
- `down: async (queryInterface, Sequelize) => { }` (arrow function)
- `down: (queryInterface, Sequelize) => { }` (arrow function)

Bu false positive'a yol açır — down handler var olmasına rağmen "missing down" raporlanır.

## Etki

- `seq-migrate repair --fix-down` gereksiz dosya değişikliği yapar.
- `seq-migrate check` yanlış WARNING gösterir.
- Kullanıcı kafası karışır.

## Beklenen Davranış

Daha kapsamlı regex:

```go
var (
    hasUpRegex   = regexp.MustCompile(`(async\s+up\s*\(|up\s*:\s*(async\s*)?function|up\s*:\s*\()`)
    hasDownRegex = regexp.MustCompile(`(async\s+down\s*\(|down\s*:\s*(async\s*)?function|down\s*:\s*\()`)
)
```

## Test Senaryosu

1. Migration dosyası: `down: async (queryInterface) => { }`
2. `seq-migrate check` çalıştır
3. **Beklenen:** "Existing migration files are structurally valid"
4. **Mevcut davranış:** "Missing 'down' rollback function" — False positive
