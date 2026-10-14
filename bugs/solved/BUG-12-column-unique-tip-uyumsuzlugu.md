# BUG-12: `Column.Unique` tipi `bool` ama dokümantasyonda `bool|string`

## Kritiklik: BİLİNMEYEN

## Dosya
`core/domain/schema.go` satır 31

## Mevcut Kod

```go
type Column struct {
    // ...
    Unique        bool        `json:"unique,omitempty"`
    // ...
}
```

## Dokümantasyon (AGENTS.md)

```
- **`Column`**:
  - `Unique` (bool or string)
```

## Bug Açıklaması

Sequelize'de `unique` alanı şu değerleri alabilir:

- `true` — basit unique constraint
- `false` — unique değil
- `"unique_constraint_name"` — isimli unique constraint (string)

Go struct'ında `bool` tipi, string değerleri temsil edemez. Bu durumda:

1. JSON serialize edilirken string `unique` değeri kaybolur.
2. Modelde `unique: "my_custom_idx"` yazılmışsa, `bool` alanı `false` görünür.
3. Diff engine bu constraint'i karşılaştıramaz.

## Etki

- İsimli unique constraint'ler kaybolur.
- Sequelize migration'larında isimli constraint desteği çalışmaz.
- Kullanıcı `unique: "idx_name"` yazarsa, migration'da `unique: true` olur → Constraint adı kaybolur.

## Beklenen Davranış

`Unique` alanı `interface{}` veya özel bir tip olmalı:

```go
type Column struct {
    // ...
    Unique        interface{} `json:"unique,omitempty"` // bool veya string
    // ...
}
```

Veya iki ayrı alan:

```go
type Column struct {
    // ...
    Unique        bool        `json:"unique,omitempty"`
    UniqueName    string      `json:"uniqueName,omitempty"`
    // ...
}
```

## Test Senaryosu

1. Model: `email: { type: STRING, unique: 'users_email_unique' }`
2. `seq-migrate init` çalıştır
3. `.seq-schema.json` içinde `unique` alanını kontrol et
4. **Beklenen:** `"unique": "users_email_unique"`
5. **Mevcut davranış:** `"unique": false` veya alan yok
