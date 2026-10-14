# BUG-06: `bridge.js` — `extractSchema()` async fonksiyona await edilmemiş çağrı

## Kritiklik: YÜKSEK

## Dosya
`adapters/extractor/bridge.js` satır 133

## Mevcut Kod

```javascript
extractSchema();
```

## Bug Açıklaması

`extractSchema()` bir `async` fonksiyon. Ancak son satırda `await` olmadan çağrılıyor:

```javascript
async function extractSchema() {
    // ... async işlemler ...
    process.stdout.write(JSON.stringify(schema, null, 2));
}

extractSchema();  // ← await yok!
```

Bu şu sorunlara yol açar:

1. **Unhandled promise rejection** — Fonksiyon içinde bir hata olursa (örn. `process.exit(1)` ile çıkış yapılması gereken durumlar), bu yakalanamaz.
2. **Race condition** — Node.js prosesi `extractSchema()` tamamlanmadan exit olabilir → `process.stdout.write` eksik kalır.
3. **JSON parse hatası** — Go tarafında `json.Unmarshal` bozuk/eksik JSON ile karşılaşır.

## Etki

- Go extractor `cmd.Output()` çağrısı eksik stdout ile döner.
- `"failed to parse schema IR from bridge output"` hatası alınır.
- Özellikle `process.exit(1)` çağrıları (models bulunamadı vs.) beklenen şekilde çalışmayabilir.

## Beklenen Davranış

Async fonksiyon doğru şekilde await edilmeli ve hatalar yakalanmalı:

```javascript
extractSchema().catch(err => {
    console.error(err);
    process.exit(1);
});
```

Veya top-level await kullanılabilir (ESM modülse):

```javascript
await extractSchema();
```

## Test Senaryosu

1. `models/` klasörü olmayan bir projede `seq-migrate make` çalıştır
2. **Beklenen:** "Could not locate Sequelize models" hatası
3. **Mevcut davranış:** Bazen boş output → Go tarafında JSON parse hatası
