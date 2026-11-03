# Discord Bridge 4.7

Windows 10/11 x64 için tek EXE. GoodbyeDPI / WinDivert motoru programın içindedir.

## Kullanım

1. Eski Discord Bridge sürümünden tepsi menüsündeki durdurup çıkma seçeneğiyle çıkın.
2. `DiscordBridge_4_7_Stable.exe` dosyasını açın.
3. Otomatik veya Dengeli profilini seçip Bağlantıyı Başlat'a basın.
4. Discord'u Aç düğmesiyle Discord'u açın.

X düğmesi programı sistem tepsisine gizler. Çıkmak için tepsi menüsünü kullanın. Yalnızca son çalışan profil `%LOCALAPPDATA%\DiscordBridge\settings.json` içinde saklanır. İşlem sonuçları ekranda geçicidir; günlük dosyası yazılmaz.

## 4.7 arayüz ve gizlilik

- Sol menü gerçek sayfalar açar: DNS merkezi, geri yükleme, bağlantı kontrolü, işlem durumu ve motor kontrolü. Her sayfada işlemi başlatan ayrı düğme, görünür durum ve geçici sonuç alanı bulunur; Genel Bakış düğmesi ana ekrana döndürür.
- Koyu tema korunur; seçili menü vurgusu, daha dengeli boşluklar, hafif kart kenarları ve düzenli profil kartları eklenmiştir. Trafik değişimi araç sayfalarını yenilemez.
- `session.log` ve `goodbyedpi.log` yazılmaz. Eski sürümden kalan bu iki uygulama günlüğü açılışta temizlenir. Gizli metin alanında günlük biriktirilmez; sayfa kapanınca ayrıntı alanı temizlenir.
- Yalnızca son başarılı bağlantı profili kaydedilir. Sonraki açılışta otomatik mod bu profili önce dener; başarısız bir elle seçim kaydedilmiş çalışan profilin yerini almaz. Dil tercihi ve motor dosya yolu saklanmaz. Motorun işlem kimliği dosyaya yazılmaz; yalnızca uygulamanın kendi motoru Windows işlem grubuyla yönetilir.
- Yinelenen iki yöntem birleştirildi. Otomatik yanında yedi ayrı yöntem bulunur: Dengeli, Uyumlu, Esnek, Birleşik, Web desteği, Klasik ve Yedek. Eski `standard5` tercihi aynı yöntemi kullanan Dengeli'ye taşınır.
- Web desteği, UDP/443 (QUIC) trafiğini kısıtlayan bir yöntemdir; ilk otomatik taramada denenmez. Elle seçilebilir; daha önce çalışıp kaydedildiyse sonraki otomatik seçimde tercih edilebilir. Diğer profillerin ve yeni eklenen yöntemlerin her ağda çalışacağı garanti edilmez.

## 4.7: DNS ayarını değiştirmeden bağlantı

- Başlat düğmesi Windows/modem DNS adreslerini değiştirmez; eski otomatik Cloudflare uygulaması kaldırıldı.
- Sadece `discord.com`, `discord.gg`, `discordapp.com`, `discordapp.net`, `discord.media` ve bunların alt alan adlarına yapılan standart UDP DNS sorguları program içinde HTTPS üzerinden çözülür. Diğer alan adlarının sorguları mevcut DNS sunucusuna iletilir.
- Önce Cloudflare, erişilemezse Google Public DNS denenir. Sağlayıcıya bağlantı, ISP DNS'inden bağımsız sabit başlangıç adresleriyle kurulur; HTTPS sertifikası normal şekilde doğrulanır.
- X ve “Arka planda çalıştır” bağlantıyı tepside açık tutar. “DPI'ı Durdur ve Çık” DNS filtresini de kapatır. Başlangıçta eski engelleme yanıtlarını silmek için DNS önbelleği temizlenir.
- “DNS ayarları” düğmesi eski elle değiştirme seçeneğidir ve yalnızca kendi onay penceresi kabul edilirse ağ kartını değiştirir. Yeni bağlantı yöntemi bu düğmeye basmayı gerektirmez. Eski sürümde DNS değiştirdiyseniz “DNS geri al” yedeği korur.

Kapsam: bu katman varsayılan Windows DNS istemcisinin UDP/53 sorgularını işler. Yerel DNS proxy'leri (loopback), uygulamanın kendi DoH/DoT çözümlemesi, DNS over TCP ve IPv6 uzantı başlıklı DNS paketleri bu katmanda değiştirilmez. Aşırı büyük UDP yanıtlarında standart DNS kesilme işareti kullanılır; TCP'ye geçiş yapan sorgular mevcut DNS yolundan gider. Tüm ISS'lerde bağlantı garantisi verilmez. Gerçek Superonline/ExitLag bağlantısı kullanıcı ağı üzerinde ayrıca denenmelidir.

## Önceki 4.6 arayüz düzeltmeleri

- Ağ örnekleri bütün pencereyi ve düğmeleri yenilemez; yalnızca alttaki grafik alanını günceller.
- Değişmeyen durum ve günlük olayları ekranı yenilemez.
- Arka plan çizimi ve düğmeler tamponda çizilerek tek seferde ekrana aktarılır. Yuvarlatılmış düğmelerin köşeleri her çizimde doğru arka plan rengiyle temizlenir.
- Ana pencere, düğmelerin üstünü boyamaz. Ana düğmenin yazısı/rengi yalnızca gerçekten değiştiğinde güncellenir.
- Ağ ölçümü için düzenli PowerShell işlemi başlatılmaz. Windows IP Helper API'nin 64 bit fiziksel adaptör sayaçları kullanılır.
- Ağ geçmişi 60 örnekle sınırlıdır; adaptör sıfırlanması veya değişmesi yanlış trafik sıçraması üretmez.
- Küçük çözünürlüklü ekranlarda pencere ve kontroller ekran alanına sığacak şekilde birlikte ölçeklenir.
- Daha kompakt Discord tarzı düzen; yuvarlak seçimler, yumuşak butonlar ve küçük alt ağ grafiği.
- İşlem sırasında profil/dil değişimi kapatılır. Aynı profilin tekrar seçilmesi gereksiz kayıt ve çizim başlatmaz.
- Arka plan PowerShell işlemlerine 30 saniye sınırı eklendi. Pencere kapanırken grafik tamponu ve GDI kaynakları bırakılır.

ExitLag ayarları değiştirilmez. Discord kendiliğinden açılmaz. Varsayılan bağlantı akışı DNS ayarını değiştirmez. İsteğe bağlı eski elle DNS işlemi, değişiklikten önce mevcut ayarları yedekler. Dengeli profili önceki Superonline + ExitLag yöntemini (`-5 --native-frag`) korur. Grafik Discord'a özel değildir; bütün fiziksel ağ trafiğini gösterir.

## Doğrulama

Windows x64 hedefi için derleme tamamlandı. Altı otomatik test, race denetimi açıkken geçti: 10.000 trafik güncellemesinde yalnızca grafiğin yenilenmesi, birleşik olayların durum bilgisini koruması, değişmeyen olayların çizim istememesi, ağ kesintisi/geri gelmesi, adaptör değişimi/sayaç sıfırlanması ve Windows sayaç yapısının x64 yerleşimi. Gömülü motorun SHA-256 bütünlüğü ve üç PE bileşeni kontrol edildi.

GitHub Actions, Windows üzerinde gerçek pencere çizimi, grafik güncellemelerinin yalnızca ağ panelini etkilemesi, GDI kaynak kullanımı, profil/dil kontrolleri ve uygun bildirim alanı mevcutsa tepsiye gizleme/geri açılma kontrollerini çalıştırır. Sonuçlar ve Windows ekran görüntüsü ilgili Actions çalışmasında bulunur. Bu kontroller Superonline/ExitLag üzerinden gerçek bağlantı testinin yerini tutmaz. İlk herkese açık sürüm bu nedenle ön sürüm olarak yayımlanır.

## Kaynaktan derleme

Go 1.23+ ve Python 3 kurulu bir geliştirme bilgisayarında:

```
python build.py
```

Hazır EXE'yi kullanmak için Go/Python kurulumu gerekmez. Kaynak paketindeki `engine.zip`, önceki 4.5 sürümünün SHA-256 doğrulamalı gömülü motorunu aynen içerir. Paketleyici ek indirme yapmaz.

Durum modeli testleri Linux'ta veya Windows'ta `go test -buildvcs=false -race .` ile çalıştırılabilir; race desteği için uygun C derleyicisi gerekir. `go test -buildvcs=false .` standart testleri çalıştırır.

## Lisanslar

Bu uygulama Discord ile bağlantılı resmî bir ürün değildir. GoodbyeDPI Apache-2.0, WinDivert LGPL-3 lisanslıdır. Lisans metinleri bu pakette ve gömülü arşivde yer alır. EXE Authenticode ile imzalanmamıştır.

GoodbyeDPI: https://github.com/ValdikSS/GoodbyeDPI
WinDivert: https://github.com/basil00/WinDivert
Windows ağ sayaçları: https://learn.microsoft.com/en-us/windows/win32/api/netioapi/ns-netioapi-mib_if_row2

GitHub sürüm derlemesi Go 1.27.3 kullanır. Kaynak paketinde üçüncü taraf motor bileşenlerinin ilgili kaynak arşivleri ve lisansları da yer alır.

## Code signing policy

Signing preparation is documented in [Code signing policy](docs/CODE_SIGNING_POLICY.md) and [Privacy policy](docs/PRIVACY.md). The SignPath Foundation application was declined due to insufficient public visibility. No provider approval or signing certificate is claimed. Existing release files remain unsigned.
