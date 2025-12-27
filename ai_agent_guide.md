# ramorie - AI Agent Kullanım Rehberi

Bu rehber, bir Yapay Zeka (AI) ajanının `ramorie` komut satırı aracını otonom bir şekilde kullanarak geliştirme görevlerini yönetmesi, bilgi depolaması ve ilerlemesini kaydetmesi için tasarlanmıştır.

## Temel Konseptler

- **Project (Proje):** Tüm çalışmaların ana kapsayıcısıdır. Her görev veya bilgi bir projeye aittir. Her zaman aktif bir proje bağlamında çalışılmalıdır.
- **Task (Görev):** Temel çalışma birimidir. Bir görevin başlığı, açıklaması, durumu (`TODO`, `IN_PROGRESS`, `COMPLETED`) ve önceliği vardır.
- **Annotation (Not):** Bir göreve eklenen ek bilgidir. Detaylı düşünceler, yapılan işlemlerin kayıtları (loglar), hata mesajları veya `elaborate` gibi komutlarla AI tarafından üretilen içerikler burada saklanır.
- **Memory (Hafıza):** Proje bağlamında bir bilgi tabanıdır. Tekrar kullanılabilecek komutlar, kod parçacıkları, konfigürasyon detayları gibi bilgileri depolamak için kullanılır.

## Anahtar Kural: ID Yönetimi

Neredeyse tüm komutlar bir **ID** ile çalışır. `create` veya `list` komutlarından dönen **kısa ID'leri (örn: `a6ba6295`)** bir sonraki komutta kullanmak için mutlaka yakala ve sakla.

**Önemli:** Bazı komutlar (`show`, `elaborate`) kısa ID ile çalışırken, bazıları (`complete` gibi) tam UUID gerektirebilir. Eğer "Invalid UUID format" hatası alırsan, `ramorie task show <kısa-id>` komutunu kullanarak tam UUID'yi al ve komutu onunla yeniden dene.

## Temel AI Agent İş Akışları

### 1. Yeni Bir Geliştirme Görevine Başlama

Kullanıcı yeni bir hedef veya görev verdiğinde, izlenecek adımlar:

1.  **Aktif Projeyi Kontrol Et:** `ramorie project list` komutu ile aktif bir proje olup olmadığını kontrol et. Aktif proje yoksa, kullanıcıdan `ramorie project use <id>` ile bir proje seçmesini iste.
2.  **Görevi Oluştur (Taskify):** Kullanıcının isteğini hemen bir göreve dönüştür.
    ```bash
    ramorie task create "Kullanıcının verdiği görev başlığı" --description "Görevin detaylı açıklaması ve hedefleri"
    ```
3.  **ID'yi Yakala:** Komutun çıktısından dönen kısa ID'yi (örn: `b0c94701`) hemen sakla. Bu ID, bu görevle ilgili tüm gelecekteki operasyonlar için kullanılacaktır.

### 2. Görevi Detaylandırma ve Planlama (Elaborate)

Bir görev oluşturulduktan hemen sonra, görevin nasıl yapılacağına dair bir plan oluşturmak için `elaborate` komutu kullanılmalıdır. Bu, ajanın "düşünme" ve "planlama" adımıdır.

1.  **Komutu Çalıştır:**
    ```bash
    ramorie task elaborate <görev-id>
    ```
2.  **Planı İncele:** Komut başarılı olduktan sonra, AI tarafından üretilen planı ve adımları görmek için `ramorie task show <görev-id>` komutunu çalıştır. Bu, görevi tamamlamak için izlenecek yol haritanı oluşturur.

### 3. Görev Üzerinde Çalışma

Kodlama, komut çalıştırma gibi aktif geliştirme adımları sırasında:

1.  **Görevi Başlat:** Çalışmaya başlamadan hemen önce, görevin durumunu `IN_PROGRESS` olarak güncelle. Bu, görevin aktif olarak ele alındığını belirtir.
    ```bash
    ramorie task start <görev-id>
    ```
2.  **Görev Detaylarını Güncelle:** Çalışma sırasında görevin başlığını, açıklamasını, durumunu veya önceliğini değiştirmen gerekirse `update` komutunu kullan:
    ```bash
    # Başlığı güncelle
    ramorie task update <görev-id> --title "Yeni Başlık"

    # Durumu güncelle
    ramorie task update <görev-id> --status IN_PROGRESS

    # Önceliği güncelle
    ramorie task update <görev-id> --priority H

    # Birden fazla özelliği aynı anda güncelle
    ramorie task update <görev-id> --title "Güncellenmiş Babout:blank#blockedaşlık" --status IN_PROGRESS --priority H

    # Kısa flag isimleri de kullanılabilir
    ramorie task update <görev-id> -t "Yeni Başlık" -s COMPLETED -P M
    ```
3.  **İlerlemeyi Not Al (Annotate):** Yaptığın her anlamlı işlemi (bir komut çalıştırmak, bir dosyayı düzenlemek, bir hata almak vb.) göreve not olarak ekle. Bu, hem ilerlemenin bir kaydını tutar hem de kullanıcıya ne yapıldığını şeffaf bir şekilde gösterir.
    ```bash
    ramorie annotate <görev-id> "Makefile düzenlendi ve 'build' komutu eklendi."
    ramorie annotate <görev-id> "Derleme sırasında 'redeclared function' hatası alındı. Çözülüyor."
    ```

### 4. Görevi Tamamlama

Görevin tüm gereksinimleri karşılandığında ve iş bittiğinde:

1.  **Görevi Bitir:**
    ```bash
    ramorie task complete <görev-id>
    ```

### 5. Bilgi Yönetimi (Hafıza)

Çalışma sırasında öğrenilen veya yeniden kullanılabilecek bilgileri saklamak için:

1.  **Bilgiyi Hatırla (Remember):**
    ```bash
    ramorie memory remember "Uygulamayı kurmak için 'make dev-install' komutu kullanılır."
    ```
2.  **Bilgiyi Geri Çağır (Recall):** Benzer bir problemle karşılaştığında veya bir bilgiye ihtiyaç duyduğunda hafızayı sorgula.
    ```bash
    ramorie memory recall "uygulama kurulumu"
    ```

## Komut Referansı

### Task Yönetimi

| Komut | Açıklama | Örnek |
|-------|----------|-------|
| `task create` | Yeni görev oluştur | `ramorie task create "Bug fix" --description "Login hatası"` |
| `task list` | Görevleri listele | `ramorie task list --status TODO` |
| `task show` | Görev detayını göster | `ramorie task show a6ba6295` |
| `task update` | Görev özelliklerini güncelle | `ramorie task update <id> --title "Yeni başlık" -s COMPLETED` |
| `task start` | Görevi IN_PROGRESS yap | `ramorie task start a6ba6295` |
| `task complete` | Görevi COMPLETED yap | `ramorie task complete a6ba6295` |
| `task elaborate` | AI ile görev planı oluştur | `ramorie task elaborate a6ba6295` |
| `annotate` | Göreve not ekle (tam UUID gerekli) | `ramorie annotate <tam-uuid> "İlerleme notu"` |

### Task Update Flag'leri

| Flag | Kısa Hali | Açıklama | Değerler |
|------|-----------|----------|----------|
| `--title` | `-t` | Görev başlığını güncelle | Herhangi bir string |
| `--description` | `-d` | Görev açıklamasını güncelle | Herhangi bir string |
| `--status` | `-s` | Görev durumunu güncelle | `TODO`, `IN_PROGRESS`, `COMPLETED` |
| `--priority` | `-P` | Görev önceliğini güncelle | `H` (High), `M` (Medium), `L` (Low) |
| `--progress` | - | İlerleme yüzdesini güncelle | 0-100 arası sayı |

### Memory Yönetimi

| Komut | Açıklama | Örnek |
|-------|----------|-------|
| `memory remember` | Bilgi sakla | `ramorie memory remember "Deploy komutu: make deploy"` |
| `memory recall` | Bilgi ara | `ramorie memory recall "deploy"` |

### Project Yönetimi

| Komut | Açıklama | Örnek |
|-------|----------|-------|
| `project list` | Projeleri listele | `ramorie project list` |
| `project use` | Aktif projeyi değiştir | `ramorie project use <project-id>` |

Bu rehberi takip ederek, bir AI ajanı `ramorie`'ı verimli ve insan-gözetimli bir şekilde kullanabilir.