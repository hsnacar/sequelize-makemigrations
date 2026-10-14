# BUG-08: `verifier.go` — Tüm migration dosyalarını sandbox'ta çalıştırıyor

## Kritiklik: YÜKSEK

## Dosya
`adapters/verifier/verifier.go` satır 73-83

## Mevcut Kod

```go
// 1. Replay all existing migrations in order
for (const file of existingFiles) {
    try {
        const mig = require(file);
        if (typeof mig.up === 'function') {
            await mig.up(queryInterface, Sequelize);
        }
    } catch (err) {
        console.error("FAIL_PREVIOUS_MIGRATION (" + path.basename(file) + "): " + err.message);
        process.exit(9);
    }
}
```

## Bug Açıklaması

`--verify` çalıştığında, projedeki **tüm migration dosyaları** sandbox'ta yeniden çalıştırılır. Bu şu sorunlara yol açar:

1. **Performans** — 50 migration varsa, hepsi SQLite'ta yeniden çalıştırılır.
2. **Güvenlik** — Kullanıcı migration'ları `require` edilir; zararlı kod çalıştırılabilir.
3. **Hata bağımlılığı** — Başka bir migration hata verirse, yeni migration'ı etkiler.
4. **Yanlış pozitif/negatif** — Mevcut migration'ların hataları yeni migration'ı başarısız yapar.

## Etki

- `--verify` çok yavaş çalışır (özellikle büyük projelerde).
- Mevcut migration'ların hataları yeni migration'ı etkiler.
- Sandbox güvenliği zayıf.

## Beklenen Davranış

Sadece **yeni migration** test edilmeli. Mevcut migration'ların şeması snapshot'tan重建 (reconstruct) edilmeli:

```go
// Sadece yeni migration'ı test et
async function testMigration() {
    const sequelize = new Sequelize({
        dialect: 'sqlite',
        storage: ':memory:',
        logging: false
    });

    const queryInterface = sequelize.getQueryInterface();
    const newMigration = require(newMigrationPath);

    // 1. Yeni migration'ı UP çalıştır
    try {
        await newMigration.up(queryInterface, Sequelize);
    } catch (err) {
        console.error("FAIL_UP: " + err.message);
        process.exit(10);
    }

    // 2. Yeni migration'ı DOWN çalıştır (reversibility check)
    try {
        await newMigration.down(queryInterface, Sequelize);
    } catch (err) {
        console.error("FAIL_DOWN: " + err.message);
        process.exit(11);
    }

    await sequelize.close();
    process.exit(0);
}
```

## Test Senaryosu

1. Projede 20 migration dosyası var
2. `seq-migrate make --verify` çalıştır
3. **Beklenen:** Sadece yeni migration test edilmeli (hızlı)
4. **Mevcut davranış:** 20 migration yeniden çalıştırılır (yavaş)
