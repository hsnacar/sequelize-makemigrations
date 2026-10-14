# BUG-01: `hasColumnChanged` — Comment ve Precision/Scale karşılaştırması eksik

## Kritiklik: KRİTİK

## Dosya
`core/diff/engine.go` satır 102-128

## Mevcut Kod

```go
func hasColumnChanged(oldCol, newCol domain.Column) bool {
    if oldCol.Type.Name != newCol.Type.Name {
        return true
    }
    if oldCol.Type.Length != newCol.Type.Length {
        return true
    }
    if oldCol.AllowNull != newCol.AllowNull {
        return true
    }
    if oldCol.PrimaryKey != newCol.PrimaryKey {
        return true
    }
    if oldCol.AutoIncrement != newCol.AutoIncrement {
        return true
    }
    if !reflect.DeepEqual(oldCol.DefaultValue, newCol.DefaultValue) {
        return true
    }
    if oldCol.Unique != newCol.Unique {
        return true
    }
    if !reflect.DeepEqual(oldCol.References, newCol.References) {
        return true
    }
    return false
}
```

## Bug Açıklaması

`hasColumnChanged` fonksiyonu şu alanları hiç kontrol etmiyor:

1. **`Column.Comment`** — Sequelize modelinde bir sütunun `comment` değeri değiştirilirse, diff engine bunu fark etmez ve migration üretilmez.
2. **`DataType.Precision`** — `DECIMAL` gibi tiplerin precision değeri değişse bile yakalanmaz.
3. **`DataType.Scale`** — `DECIMAL` gibi tiplerin scale değeri değişse bile yakalanmaz.

## Etki

- Kullanıcı Sequelize modelinde sadece comment günceller → `seq-migrate make` çalıştırır → "No changes detected" mesajı alır → **Migration üretilmez**.
- DECIMAL sütununun precision/scale değişikliği yakalanmaz → **Veritabanı şeması ile migration senkronizasyonu bozulur**.

## Beklenen Davranış

`hasColumnChanged` fonksiyonu tüm anlamlı alanları karşılaştırmalı:
- `oldCol.Comment != newCol.Comment`
- `oldCol.Type.Precision != newCol.Type.Precision`
- `oldCol.Type.Scale != newCol.Type.Scale`

## Önerilen Düzeltme

```go
func hasColumnChanged(oldCol, newCol domain.Column) bool {
    if oldCol.Type.Name != newCol.Type.Name {
        return true
    }
    if oldCol.Type.Length != newCol.Type.Length {
        return true
    }
    if oldCol.Type.Precision != newCol.Type.Precision {
        return true
    }
    if oldCol.Type.Scale != newCol.Type.Scale {
        return true
    }
    if oldCol.AllowNull != newCol.AllowNull {
        return true
    }
    if oldCol.PrimaryKey != newCol.PrimaryKey {
        return true
    }
    if oldCol.AutoIncrement != newCol.AutoIncrement {
        return true
    }
    if !reflect.DeepEqual(oldCol.DefaultValue, newCol.DefaultValue) {
        return true
    }
    if oldCol.Unique != newCol.Unique {
        return true
    }
    if oldCol.Comment != newCol.Comment {
        return true
    }
    if !reflect.DeepEqual(oldCol.References, newCol.References) {
        return true
    }
    return false
}
```

## Test Senaryosu

1. Mevcut `.seq-schema.json` içinde `price` sütunu: `DECIMAL(10, 2), comment: "fiyat"`
2. Modeli güncelle: `DECIMAL(12, 4), comment: "güncel fiyat"`
3. `seq-migrate make` çalıştır
4. **Beklenen:** Column modified migration üretilmeli
5. **Mevcut davranış:** "No changes detected" — migration üretilmez
