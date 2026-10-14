# BUG-09: `bridge.js` — `references.model` string olmayan değerler için hata

## Kritiklik: YÜKSEK

## Dosya
`adapters/extractor/bridge.js` satır 85-88

## Mevcut Kod

```javascript
references: attr.references ? {
    model: typeof attr.references.model === 'string' ? attr.references.model 
        : (attr.references.model && (attr.references.model.tableName || attr.references.model.name)),
    key: attr.references.key || 'id'
} : null
```

## Bug Açıklaması

`attr.references.model` bir **Sequelize Model class**'ı olduğunda (örn. `references: { model: User }`), `tableName` veya `name` özelliği olmayabilir:

- Sequelize v6'da `User.tableName` vardır ama `User.name` sınıf adıdır (model adı değil).
- Sequelize v7'de `User.tableName` değişmiş olabilir.
- `User.name` → Sınıf adı ("User") döner, ama tablo adı "users" olabilir.

Bu durumda `model: undefined` döner → Go tarafında boş string → **FK migration'ları yanlış üretilir**.

## Etki

- `User` modelinin tablosu `users` iken, `model: "User"` yazılır → FK yanlış tabloya işaret eder.
- Migration çalıştırılırken "table User does not exist" hatası alınır.
- Sequelize v7'de `tableName` API'si değişmiş olabilir.

## Beklenen Davranış

Daha sağlam bir çözüm:

```javascript
references: attr.references ? {
    model: typeof attr.references.model === 'string' 
        ? attr.references.model 
        : (attr.references.model && (
            attr.references.model.tableName || 
            attr.references.model.name || 
            (attr.references.model.getTableName && attr.references.model.getTableName())
          )),
    key: attr.references.key || 'id'
} : null
```

Alternatif olarak, `sequelize.models` içinde model adına göre arama yapılabilir.

## Test Senaryosu

1. Model tanımı: `User.hasMany(Post, { foreignKey: 'userId' })`
2. `Post` modelinde: `userId: { references: { model: User, key: 'id' } }`
3. `seq-migrate make` çalıştır
4. **Beklenen:** `references: { model: 'users', key: 'id' }`
5. **Mevcut davranış:** `references: { model: undefined, key: 'id' }` → Boş string
