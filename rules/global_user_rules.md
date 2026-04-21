# Ramorie MCP - Global User Rules

> Bu içeriği IDE'nizin global user rules/memory bölümüne ekleyin.

---

## 🤖 Ramorie MCP Kullanım Kılavuzu

AI agentlar için JosephsBrain/Ramorie MCP aracının kullanım rehberi.

---

## 🎯 Ne İşe Yarar?

**Ramorie MCP**, görev ve bilgi yönetimi için MCP server:

- Görevleri takip et (task management)
- Bilgi parçalarını sakla (memory bank)
- Kararları kaydet (ADR - Architectural Decision Records)
- Projelerle organize et
- Aktif bağlam yönetimi (context packs)

---

## 📋 Temel Kullanım Kuralları

### Oturum Başlangıcı
```
1. setup_agent                → Oturumu başlat ve bağlamı al
2. task(action="list")        → Bekleyen görevleri gör
```

### Yeni İş Başlatma
```
1. create_task                → Görevi oluştur
   - description: Net açıklama
   - priority: H/M/L
2. start_task                 → Görevi başlat (aktif yap)
3. add_task_note              → İlk planı kaydet
```

### Çalışma Sırasında
```
- add_task_note               → Her ilerlemeyi kaydet
- update_progress             → Yüzdeyi güncelle (0-100)
- add_memory                  → Öğrenilenleri sakla
- create_decision             → Önemli kararları belgele
```

### Görev Tamamlama
```
1. add_task_note              → Son durumu kaydet
2. complete_task              → Görevi tamamla
```

---

## 🧠 Memory Bank Kullanımı

### Ne Zaman Kaydet?
- Çalışan kod pattern'leri
- Konfigürasyon snippet'ları
- API endpoint kullanımları
- Hata çözümleri
- Performans optimizasyonları

### Nasıl Ara?
```
recall "anahtar kelime"       → Mevcut bilgiyi ara
```

**Kural**: Kullanıcıya sormadan önce `recall` ile mevcut bilgiyi kontrol et!

---

## 📝 Karar Kayıtları (ADR)

### Ne Zaman Kaydet?
- Mimari değişiklikler
- Teknoloji seçimleri
- API tasarım kararları
- Güvenlik politikaları

### Format
```
create_decision:
  title: "Açıklayıcı başlık"
  area: "Backend/Frontend/Architecture/DevOps"
  context: "Neden bu karar alındı"
  consequences: "Sonuçları ve etkileri"
  status: "draft/proposed/approved"
```

---

## 🔄 Bağlam Yönetimi

### Active Context = Odak Noktası
- Her proje/özellik için ayrı context pack
- Konu değiştiğinde `activate_context_pack`
- Agent'ın nihai hedefini temsil eder

### Bağlam Değişikliği
```
1. stop_task                  → Mevcut görevi duraklat
2. activate_context_pack      → Yeni bağlamı aktifle
3. start_task                 → Yeni göreve başla
```

---

## ⚡ Hızlı Referans

| İşlem | Tool |
|-------|------|
| Görev oluştur | `create_task` |
| Görevi başlat | `start_task` |
| Not ekle | `add_task_note` |
| İlerleme güncelle | `update_progress` |
| Görevi tamamla | `complete_task` |
| Bilgi sakla | `add_memory` |
| Bilgi ara | `recall` |
| Karar kaydet | `create_decision` |
| Bağlam değiştir | `activate_context_pack` |

---

## 🚫 Yapılmaması Gerekenler

1. ❌ Görev oluşturmadan çalışmaya başlama
2. ❌ İlerlemeyi kaydetmeden uzun süre çalışma
3. ❌ Önemli kararları belgelemeden geçme
4. ❌ Bağlamı kontrol etmeden yeni işe başlama
5. ❌ `recall` kullanmadan kullanıcıya soru sorma

---

## 📊 İlerleme Takibi

| Yüzde | Aşama |
|-------|-------|
| 0% | Başlamadı |
| 25% | Planlama/Araştırma |
| 50% | İmplementasyon başladı |
| 75% | Test aşaması |
| 100% | Tamamlandı |

---

## 🔧 MCP Kurulumu

### Homebrew
```bash
brew install terzigolu/tap/ramorie
ramorie auth login
```

### MCP Config
```json
{
  "mcpServers": {
    "ramorie": {
      "command": "ramorie",
      "args": ["mcp", "serve"]
    }
  }
}
```

---

*Ramorie MCP v1.7.0 - 57 tool desteklenmektedir*
