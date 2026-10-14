# BUG-10: `loadSchemaSnapshot` — Bozuk JSON sessizce yok sayılıyor

## Kritiklik: YÜKSEK

## Dosya
`cmd/seq-migrate/main.go` satır 400-410

## Mevcut Kod

```go
func loadSchemaSnapshot(path string) *domain.Schema {
    data, err := os.ReadFile(path)
    if err != nil {
        return domain.NewSchema()
    }
    var schema domain.Schema
    if err := json.Unmarshal(data, &schema); err != nil {
        return domain.NewSchema()
    }
    return &schema
}
```

## Bug Açıklaması

`.seq-schema.json` dosyası bozuksa (manuel edit hatası, merge conflict, yanlış format), CLI bunu fark etmez ve **boş schema** döndürür. Bu şu anlama gelir:

- Mevcut tüm tablolar "yeni" olarak görünür.
- `TablesCreated` listesine tüm tablolar eklenir.
- **DROP TABLE olmadan CREATE TABLE** migration üretilir → Veritabanında çakışma.

## Etki

- Bozuk snapshot → Yanlış migration → Veritabanı hatası.
- Kullanıcı nedenini anlamaz.
- Veri kaybı riski (eski tablolar "yeni" olarak görünür).

## Beklenen Davranış

Hata durumunda kullanıcıya uyarı verilmeli ve işlem durmalı:

```go
func loadSchemaSnapshot(path string) (*domain.Schema, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        if os.IsNotExist(err) {
            return domain.NewSchema(), nil // İkinci çalıştırma, normal
        }
        return nil, fmt.Errorf("failed to read schema snapshot: %w", err)
    }
    var schema domain.Schema
    if err := json.Unmarshal(data, &schema); err != nil {
        return nil, fmt.Errorf("schema snapshot is corrupted (%s): %w\nRun 'seq-migrate repair --snapshot' to rebuild", path, err)
    }
    return &schema, nil
}
```

Çağrı yerinde:
```go
oldSchema, err := loadSchemaSnapshot(schemaFile)
if err != nil {
    fmt.Fprintf(os.Stderr, "❌ %v\n", err)
    os.Exit(1)
}
```

## Test Senaryosu

1. `.seq-schema.json` dosyasını boz (örn. `{ "version": 1, "tables": {` yaz)
2. `seq-migrate make` çalıştır
3. **Beklenen:** Hata mesajı + "repair --snapshot" önerisi
4. **Mevcut davranış:** Boş schema → Tüm tablolar "yeni" → CREATE TABLE hatası
