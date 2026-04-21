# 🤖 Ramorie MCP Agent Guide

> Bu rehber, AI agentların Ramorie MCP (Model Context Protocol) server'ını kullanarak görev yönetimi, bilgi depolama ve karar kaydetme işlemlerini nasıl yapacağını açıklar.

## 📋 İçindekiler

1. [MCP Nedir ve Neden Kullanmalı?](#mcp-nedir)
2. [Kurulum](#kurulum)
3. [Temel Konseptler](#temel-konseptler)
4. [Agent Timeline (Aktivite Takibi)](#agent-timeline)
5. [MCP Tool Kategorileri](#mcp-tool-kategorileri)
6. [Agent İş Akışları](#agent-iş-akışları)
7. [Windsurf/Cursor Rules Entegrasyonu](#rules-entegrasyonu)
8. [Best Practices](#best-practices)
9. [Tam Tool Referansı](#tool-referansı)

---

## 🎯 MCP Nedir ve Neden Kullanmalı? {#mcp-nedir}

**Model Context Protocol (MCP)**, AI agentların harici araçlarla standart bir şekilde iletişim kurmasını sağlayan bir protokoldür.

### Neden Ramorie MCP?

| Özellik | Açıklama |
|---------|----------|
| **Persistent Memory** | Oturumlar arası bilgi saklama |
| **Task Tracking** | Görev durumu ve ilerleme takibi |
| **Decision Records** | Alınan kararların ADR formatında kaydı |
| **Context Management** | Aktif çalışma bağlamı yönetimi |
| **Multi-Agent Support** | Birden fazla agent aynı veriyi paylaşabilir |

### MCP vs Direct API

```
❌ Direct API: Her seferinde auth, endpoint, payload yönetimi
✅ MCP: Standart tool çağrısı, otomatik auth, tip güvenliği
```

---

## 🔧 Kurulum {#kurulum}

### Windsurf İçin

`~/.windsurf/settings.json` veya proje `.windsurf/mcp.json`:

```json
{
  "mcpServers": {
    "ramorie": {
      "command": "ramorie",
      "args": ["mcp", "serve"],
      "env": {}
    }
  }
}
```

### Cursor İçin

`.cursor/mcp.json`:

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

### Claude Desktop İçin

`~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "ramorie": {
      "command": "/usr/local/bin/ramorie",
      "args": ["mcp", "serve"]
    }
  }
}
```

### Codex CLI İçin

Codex `~/.codex/config.toml` dosyasını kullanır. Hazır snippet üretmek için:

```bash
ramorie mcp config --client codex
```

Çıktıyı config dosyanıza ekleyin:

```toml
[mcp_servers.ramorie]
command = "ramorie"
args = ["mcp", "serve"]
enabled = true
```

> Düzenleme sonrası Codex CLI'yi yeniden başlatın.

### API Key Ayarı

```bash
# İlk kurulum
ramorie auth login

# Veya manuel
echo '{"api_key": "your-api-key"}' > ~/.ramorie/config.json
```

---

## 📚 Temel Konseptler {#temel-konseptler}

### 1. Project (Proje)
Tüm çalışmaların kapsayıcısı. Her task, memory ve decision bir projeye aittir.

### 2. Task (Görev)
- **Status**: `TODO` → `IN_PROGRESS` → `COMPLETED`
- **Priority**: `H` (High), `M` (Medium), `L` (Low)
- **Progress**: 0-100 arası ilerleme yüzdesi
- **Annotations**: Göreve eklenen notlar

### 3. Memory (Hafıza)
Tekrar kullanılabilir bilgi parçaları. Kod snippetları, konfigürasyonlar, öğrenilen dersler.

### 4. Decision (Karar - ADR)
Architectural Decision Records. Önemli teknik kararların kaydı.

### 5. Context Pack (Aktif Bağlam)
Şu an üzerinde çalışılan konu/hedef. Agent'ın odak noktası.

### 6. Agent Timeline (Aktivite Akışı)
AI agentların tüm aktivitelerinin gerçek zamanlı izlendiği sistem. Her MCP operasyonu otomatik olarak kaydedilir.

---

## 📊 Agent Timeline (Aktivite Takibi) {#agent-timeline}

Agent Timeline, tüm AI agent aktivitelerini gerçek zamanlı olarak izleyen ve kaydeden sistemdir. Dashboard üzerinden (`/agent-timeline`) veya `get_agent_activity` tool'u ile erişilebilir.

### Neden Agent Timeline?

| Özellik | Açıklama |
|---------|----------|
| **Gerçek Zamanlı İzleme** | SSE ile anlık event bildirimleri |
| **Agent Tanımlama** | Hangi agent'ın hangi işlemi yaptığı görülebilir |
| **Oturum Takibi** | Session ID ile ilişkili aktiviteler gruplandırılır |
| **Filtreleme** | Event tipi, agent adı, entity tipi ile filtreleme |
| **Şifreleme** | Hassas veriler şifreli saklanır |

### İzlenen Event Tipleri

#### Task Events
| Event Type | Açıklama | Tetikleyen Tool |
|------------|----------|-----------------|
| `task_created` | Yeni görev oluşturuldu | `create_task` |
| `task_started` | Görev başlatıldı | `start_task` |
| `task_stopped` | Görev duraklatıldı | `stop_task` |
| `task_completed` | Görev tamamlandı | `complete_task` |
| `task_deleted` | Görev silindi | `delete_task` |
| `task_updated` | Görev güncellendi | `update_task` |
| `task_note_added` | Göreve not eklendi | `add_task_note` |
| `task_progress_updated` | İlerleme güncellendi | `update_progress` |

#### Subtask Events
| Event Type | Açıklama | Tetikleyen Tool |
|------------|----------|-----------------|
| `subtask_created` | Alt görev oluşturuldu | `create_subtask` |
| `subtask_updated` | Alt görev güncellendi | `update_subtask` |
| `subtask_completed` | Alt görev tamamlandı | `complete_subtask` |
| `subtask_deleted` | Alt görev silindi | `delete_subtask` |

#### Memory Events
| Event Type | Açıklama | Tetikleyen Tool |
|------------|----------|-----------------|
| `memory_created` | Hafıza oluşturuldu | `add_memory` |
| `memory_updated` | Hafıza güncellendi | `update_memory` |
| `memory_deleted` | Hafıza silindi | `delete_memory` |

#### Decision Events (ADR)
| Event Type | Açıklama | Tetikleyen Tool |
|------------|----------|-----------------|
| `decision_created` | Karar kaydedildi | `create_decision` |
| `decision_updated` | Karar güncellendi | `update_decision` |
| `decision_deleted` | Karar silindi | `delete_decision` |

#### Dependency Events
| Event Type | Açıklama | Tetikleyen Tool |
|------------|----------|-----------------|
| `dependency_added` | Bağımlılık eklendi | `add_task_dependency` |
| `dependency_removed` | Bağımlılık kaldırıldı | `remove_task_dependency` |

#### Context & Focus Events
| Event Type | Açıklama | Tetikleyen Tool |
|------------|----------|-----------------|
| `pack_created` | Context pack oluşturuldu | `create_context_pack` |
| `pack_updated` | Context pack güncellendi | `update_context_pack` |
| `pack_deleted` | Context pack silindi | `delete_context_pack` |

#### AI Operation Events
| Event Type | Açıklama | Tetikleyen Tool |
|------------|----------|-----------------|
| `ai_time_estimate` | Zaman tahmini yapıldı | `ai_estimate_time` |
| `ai_risk_analysis` | Risk analizi yapıldı | `ai_analyze_risks` |
| `ai_next_step` | Sonraki adım önerisi | `ai_next_step` |
| `ai_dependencies` | Bağımlılık analizi | `ai_find_dependencies` |

### Agent Timeline Kullanımı

#### Tool ile Sorgulama

```
get_agent_activity {
  "project": "my-project",     // Opsiyonel: Proje filtresi
  "agent_name": "Claude",       // Opsiyonel: Agent filtresi
  "event_type": "task_created", // Opsiyonel: Event tipi filtresi
  "limit": 20                   // Opsiyonel: Sonuç limiti (varsayılan: 10)
}
```

#### Dashboard Üzerinden

1. `https://ramorie.app/agent-timeline` adresine gidin
2. Sol taraftaki filtrelerden:
   - **Event Types**: Görmek istediğiniz event tiplerini seçin
   - **Entity Types**: task, memory, decision vb. filtreleyin
   - **Agent Names**: Belirli agent'ları filtreleyin
   - **Created Via**: MCP, API veya CLI kaynaklarını filtreleyin
3. Sağ üstteki "Refresh" butonu veya SSE ile otomatik güncelleme

### Agent Bilgilerinin Takibi

Her MCP çağrısında aşağıdaki bilgiler otomatik olarak kaydedilir:

| Bilgi | Açıklama | Örnek |
|-------|----------|-------|
| `agent_name` | Agent'ın adı | `Claude Opus 4.5` |
| `agent_model` | Model bilgisi | `claude-opus-4-5-20251101` |
| `agent_session_id` | Oturum ID'si | `uuid-v4` |
| `created_via` | Kaynak | `mcp` / `api` / `cli` |

**Not:** Bu bilgiler `setup_agent` tool'u çağrıldıktan sonra tüm API isteklerine otomatik olarak eklenir.

### Entity Tipleri

| Entity Type | Açıklama |
|-------------|----------|
| `task` | Görevler |
| `subtask` | Alt görevler |
| `memory` | Hafızalar |
| `decision` | Mimari kararlar (ADR) |
| `context_pack` | Bağlam paketleri |
| `project` | Projeler |
| `dependency` | Görev bağımlılıkları |

---

## 🛠️ MCP Tool Kategorileri {#mcp-tool-kategorileri}

### Tasks (16 tool)

| Tool | Açıklama | Zorunlu Parametreler |
|------|----------|---------------------|
| `create_task` | Yeni görev oluştur | `description` |
| `list_tasks` | Görevleri listele | - |
| `get_task` | Görev detayı | `taskId` |
| `start_task` | Görevi başlat (IN_PROGRESS + aktif) | `taskId` |
| `stop_task` | Görevi duraklat | `taskId` |
| `complete_task` | Görevi tamamla | `taskId` |
| `delete_task` | Görevi sil | `taskId` |
| `update_task_status` | Durum güncelle | `taskId`, `status` |
| `update_progress` | İlerleme güncelle | `taskId`, `progress` |
| `add_task_note` | Not ekle | `taskId`, `note` |
| `create_subtask` | Alt görev ekle | `parentTaskId`, `description` |
| `search_tasks` | Görevlerde ara | `query` |
| `get_next_tasks` | Öncelikli görevler | - |
| `bulk_start_tasks` | Toplu başlat | `taskIds` |
| `bulk_complete_tasks` | Toplu tamamla | `taskIds` |

### Subtasks (5 tool)

| Tool | Açıklama | Zorunlu Parametreler |
|------|----------|---------------------|
| `create_subtask` | Alt görev oluştur | `task_id`, `description` |
| `get_subtasks` | Alt görevleri listele | `task_id` |
| `update_subtask` | Alt görev güncelle | `subtask_id` |
| `complete_subtask` | Alt görevi tamamla | `subtask_id` |
| `delete_subtask` | Alt görevi sil | `subtask_id` |

### Task Dependencies (4 tool)

| Tool | Açıklama | Zorunlu Parametreler |
|------|----------|---------------------|
| `add_task_dependency` | Bağımlılık ekle | `task_id`, `depends_on_id` |
| `remove_task_dependency` | Bağımlılık kaldır | `task_id`, `depends_on_id` |
| `get_task_dependencies` | Bağımlılıkları getir | `task_id` |
| `get_task_dependents` | Bağımlı görevleri getir | `task_id` |

### Memories (9 tool)

| Tool | Açıklama | Zorunlu Parametreler |
|------|----------|---------------------|
| `add_memory` | Bilgi sakla | `content` |
| `list_memories` | Hafızaları listele | - |
| `get_memory` | Hafıza detayı | `memoryId` |
| `update_memory` | Hafıza güncelle | `memoryId` |
| `delete_memory` | Hafıza sil | `memoryId` |
| `recall` | Hafızada ara | `term` |
| `get_task_memories` | Görevin hafızaları | `taskId` |
| `memory_tasks` | Hafızanın görevleri | `memoryId` |
| `create_memory_task_link` | Görev-hafıza bağla | `taskId`, `memoryId` |

### Decisions (5 tool)

| Tool | Açıklama | Zorunlu Parametreler |
|------|----------|---------------------|
| `list_decisions` | Kararları listele | - |
| `get_decision` | Karar detayı | `decisionId` |
| `create_decision` | Yeni karar oluştur | `title` |
| `update_decision` | Karar güncelle | `decisionId` |
| `delete_decision` | Karar sil | `decisionId` |

### Projects (6 tool)

| Tool | Açıklama | Zorunlu Parametreler |
|------|----------|---------------------|
| `list_projects` | Projeleri listele | - |
| `create_project` | Yeni proje | `name` |
| `get_project` | Proje detayı | `projectId` |
| `update_project` | Proje güncelle | `projectId` |
| `delete_project` | Proje sil | `projectId` |

### Context Packs (7 tool)

| Tool | Açıklama | Zorunlu Parametreler |
|------|----------|---------------------|
| `list_context_packs` | Context'leri listele | - |
| `get_context_pack` | Context detayı | `packId` |
| `create_context_pack` | Yeni context | `name`, `type` |
| `update_context_pack` | Context güncelle | `packId` |
| `delete_context_pack` | Context sil | `packId` |
| `activate_context_pack` | Context aktifle | `packId` |
| `get_active_context_pack` | Aktif context | - |

### Organizations (3 tool)

| Tool | Açıklama | Zorunlu Parametreler |
|------|----------|---------------------|
| `list_organizations` | Organizasyonları listele | - |
| `get_organization` | Organizasyon detayı | `orgId` |
| `create_organization` | Yeni organizasyon | `name` |

### Reports & Analysis (7 tool)

| Tool | Açıklama |
|------|----------|
| `get_stats` | İstatistikler |
| `get_history` | Aktivite geçmişi |
| `get_agent_activity` | **Agent Timeline** - Agent aktivitelerini sorgula |
| `timeline` | Zaman çizelgesi |
| `export_project` | Proje raporu |
| `analyze_task_risks` | Risk analizi |
| `analyze_task_dependencies` | Bağımlılık analizi |

### Utilities (5 tool)

| Tool | Açıklama |
|------|----------|
| `duplicate_task` | Görev kopyala |
| `move_tasks_to_project` | Görevleri taşı |
| `list_contexts` | Eski context listesi |
| `create_context` | Eski context oluştur |
| `set_active_context` | Eski context aktifle |

---

## 🔄 Agent İş Akışları {#agent-iş-akışları}

### Workflow 1: Yeni Görev Başlatma

```
1. get_active_context_pack    → Mevcut bağlamı kontrol et
2. create_task                → Görevi oluştur
3. start_task                 → Görevi başlat (aktif yap)
4. add_task_note              → İlk planı not olarak ekle
```

### Workflow 2: Çalışma Sırasında

```
1. task(action="list")        → Bekleyen görevleri kontrol et
2. [Çalışma yap]
3. task(action="note")        → İlerlemeyi kaydet
4. task(action="progress")    → Yüzdeyi güncelle
5. remember()                 → Öğrenilenleri kaydet
```

### Workflow 3: Karar Alma

```
1. Önemli bir teknik karar alındığında:
2. create_decision            → Kararı kaydet
   - title: "Karar başlığı"
   - description: "Kısa açıklama"
   - context: "Neden bu karar alındı"
   - consequences: "Sonuçları ve etkileri"
   - area: "Backend/Frontend/Architecture/DevOps"
   - status: "draft/proposed/approved"
```

### Workflow 4: Görev Tamamlama

```
1. add_task_note              → Son durumu kaydet
2. complete_task              → Görevi tamamla
3. add_memory                 → Öğrenilenleri sakla (opsiyonel)
```

### Workflow 5: Bağlam Değişikliği

```
1. stop_task                  → Mevcut görevi duraklat
2. activate_context_pack      → Yeni bağlamı aktifle
3. get_next_tasks             → Yeni bağlamdaki görevleri al
4. start_task                 → Yeni göreve başla
```

---

## 📜 Windsurf/Cursor Rules Entegrasyonu {#rules-entegrasyonu}

### Windsurf Rules (.windsurfrules)

```markdown
# Ramorie MCP Kullanım Kuralları

## Görev Yönetimi
- Her yeni iş için `create_task` kullan
- Çalışmaya başlarken `start_task` çağır
- İlerlemeyi `add_task_note` ile kaydet
- Tamamlandığında `complete_task` kullan

## Bilgi Yönetimi
- Öğrenilen her şeyi `add_memory` ile kaydet
- Mevcut bilgiyi `recall` ile ara
- Görev-bilgi ilişkisini `create_memory_task_link` ile kur

## Karar Kayıtları
- Önemli teknik kararları `create_decision` ile kaydet
- ADR formatını kullan (context, consequences)
- Kararları `list_decisions` ile referans al

## Bağlam Yönetimi
- `get_active_context_pack` ile mevcut odağı kontrol et
- Konu değiştiğinde `activate_context_pack` kullan
```

### Cursor Rules (.cursorrules)

```markdown
# Ramorie MCP Integration

When working on tasks:
1. List pending work with `task(action="list")`
2. Log progress with `task(action="note")`
3. Save learnings with `remember()`
4. Record decisions with `remember()` (auto-detects type=decision)

Memory Bank Usage:
- Use `recall` before asking user for information
- Save reusable code/configs with `add_memory`
- Link memories to tasks with `create_memory_task_link`

Decision Recording:
- Record architectural decisions with `create_decision`
- Include context (why), consequences (impact)
- Use areas: Frontend, Backend, Architecture, DevOps
```

### Global Rules (user_rules)

```markdown
# JosephsBrain/Ramorie MCP Kullanım Kılavuzu

## Temel Prensipler
1. Her oturumda `get_active_context_pack` ile bağlamı kontrol et
2. Yeni iş başlarken `create_task` + `start_task` kullan
3. İlerlemeyi `add_task_note` ile düzenli kaydet
4. Öğrenilenleri `add_memory` ile sakla
5. Önemli kararları `create_decision` ile kaydet

## Tool Kullanım Sırası
- Okuma: get_* → list_* → recall
- Yazma: create_* → update_* → complete_*
- Bağlam: get_active_* → activate_* → start_*
```

---

## ✅ Best Practices {#best-practices}

### 1. Her Zaman Bağlamı Kontrol Et

```
İlk adım: get_active_context_pack
Eğer yoksa: list_context_packs → activate_context_pack
```

### 2. Görev Notlarını Düzenli Tut

```
✅ İyi: "Login API entegrasyonu tamamlandı. JWT token refresh eklendi."
❌ Kötü: "bitti"
```

### 3. Memory'leri Etiketle

```
✅ İyi: add_memory ile tags: ["auth", "jwt", "security"]
❌ Kötü: Etiketsiz, aranması zor bilgi
```

### 4. Kararları Detaylı Kaydet

```
✅ İyi:
- title: "JWT yerine Session-based auth"
- context: "Mobile app desteği için stateless gerekli değil"
- consequences: "Server-side session yönetimi gerekecek"
- area: "Architecture"

❌ Kötü:
- title: "Auth değişikliği"
```

### 5. İlerlemeyi Yüzde Olarak Takip Et

```
0%   → Başlamadı
25%  → Planlama/araştırma
50%  → Implementasyon başladı
75%  → Test aşaması
100% → Tamamlandı
```

---

## 📖 Tam Tool Referansı {#tool-referansı}

### create_task
```json
{
  "description": "Görev açıklaması (zorunlu)",
  "priority": "H/M/L (varsayılan: M)",
  "project": "Proje adı veya ID (varsayılan: aktif proje)"
}
```

### add_memory
```json
{
  "content": "Saklanacak bilgi (zorunlu)",
  "project": "Proje adı veya ID (varsayılan: aktif proje)"
}
```

### create_decision
```json
{
  "title": "Karar başlığı (zorunlu)",
  "description": "Kısa açıklama",
  "status": "draft/proposed/approved/deprecated",
  "area": "Frontend/Backend/Architecture/DevOps",
  "context": "Kararın bağlamı ve nedeni",
  "consequences": "Kararın sonuçları ve etkileri"
}
```

### create_context_pack
```json
{
  "name": "Context adı (zorunlu)",
  "type": "project/integration/decision/custom (zorunlu)",
  "description": "Açıklama",
  "status": "draft/published",
  "tags": ["tag1", "tag2"]
}
```

---

## 🔗 Kaynaklar

- **CLI Kurulum**: `brew install terzigolu/tap/ramorie`
- **API Docs**: https://jbraincli-go-backend-production.up.railway.app/swagger/
- **GitHub**: https://github.com/terzigolu/ramorie

---

*Bu guide v3.14.0 için güncellenmiştir. Toplam 57+ MCP tool desteklenmektedir.*

---

## 📝 Changelog

### v3.14.0
- ✨ Agent Timeline bölümü eklendi
- ✨ Tüm MCP operasyonları artık timeline'da görünüyor
- ✨ Yeni event tipleri: subtask, decision, dependency, task_progress_updated
- 📖 Subtasks ve Task Dependencies tool referansları eklendi
- 📖 `get_agent_activity` tool dokümantasyonu eklendi
