# BUG-02: `diffIndexes` ve `diffConstraints` — Sadece isimle eşleştiriyor

## Kritiklik: KRİTİK

## Dosya
`core/diff/engine.go` satır 130-182

## Mevcut Kod

```go
func (e *Engine) diffIndexes(tableName string, oldIndexes, newIndexes []domain.Index, diff *domain.Diff) {
    oldMap := make(map[string]domain.Index)
    for _, idx := range oldIndexes {
        oldMap[idx.Name] = idx
    }

    newMap := make(map[string]domain.Index)
    for _, idx := range newIndexes {
        newMap[idx.Name] = idx
        if _, exists := oldMap[idx.Name]; !exists {
            diff.IndexesAdded = append(diff.IndexesAdded, domain.IndexDiff{
                TableName: tableName,
                Index:     idx,
            })
        }
    }

    for _, idx := range oldIndexes {
        if _, exists := newMap[idx.Name]; !exists {
            diff.IndexesDropped = append(diff.IndexesDropped, domain.IndexDiff{
                TableName: tableName,
                Index:     idx,
            })
        }
    }
}
```

Aynı sorun `diffConstraints` için de geçerli (satır 157-182).

## Bug Açıklaması

`diffIndexes` ve `diffConstraints` fonksiyonları varlıkları **sadece `Name` alanıyla** eşleştiriyor. Bu şu anlama gelir:

- Aynı isme sahip ama **farklı `fields`** içeren bir index değişse → yakalanmaz.
- Aynı isme sahip ama `unique` değeri değişse → yakalanmaz.
- Aynı isme sahip ama `method` değişse (BTREE → GIN) → yakalanmaz.
- Constraint'lerde `onDelete` / `onUpdate` değişse → yakalanmaz.

## Etki

- Index özellikleri değişirseniz migration üretilmez.
- Constraint davranışı (CASCADE vs RESTRICT) değişirseniz migration üretilmez.
- Bu, **güvenlik ve veri bütünlüğü** açısından ciddi risk oluşturur.

## Beklenen Davranış

Aynı isimle bulunan index/constraint'lerin **içerikleri de karşılaştırılmalı**. Fark varsa:
- Eski kayıt `IndexesDropped`/`ConstraintsDropped` listesine
- Yeni kayıt `IndexesAdded`/`ConstraintsAdded` listesine
eklenmeli.

## Önerilen Düzeltme

```go
func (e *Engine) diffIndexes(tableName string, oldIndexes, newIndexes []domain.Index, diff *domain.Diff) {
    oldMap := make(map[string]domain.Index)
    for _, idx := range oldIndexes {
        oldMap[idx.Name] = idx
    }

    newMap := make(map[string]domain.Index)
    for _, idx := range newIndexes {
        newMap[idx.Name] = idx
        oldIdx, exists := oldMap[idx.Name]
        if !exists {
            // Yeni index eklendi
            diff.IndexesAdded = append(diff.IndexesAdded, domain.IndexDiff{
                TableName: tableName,
                Index:     idx,
            })
        } else if !reflect.DeepEqual(oldIdx, idx) {
            // Aynı isim ama içerik değişmiş: drop + add
            diff.IndexesDropped = append(diff.IndexesDropped, domain.IndexDiff{
                TableName: tableName,
                Index:     oldIdx,
            })
            diff.IndexesAdded = append(diff.IndexesAdded, domain.IndexDiff{
                TableName: tableName,
                Index:     idx,
            })
        }
    }

    for _, idx := range oldIndexes {
        if _, exists := newMap[idx.Name]; !exists {
            diff.IndexesDropped = append(diff.IndexesDropped, domain.IndexDiff{
                TableName: tableName,
                Index:     idx,
            })
        }
    }
}
```

Aynı mantık `diffConstraints` için de uygulanmalı.

## Test Senaryosu

1. Mevcut şema: `users` tablosunda `idx_name` → fields: `["name"]`, unique: `true`
2. Modeli güncelle: `idx_name` → fields: `["name", "email"]`, unique: `false`
3. `seq-migrate make` çalıştır
4. **Beklenen:** Index drop + add migration üretilmeli
5. **Mevcut davranış:** Değişiklik yakalanmaz
