# Discord Bridge 4.6

Windows 10/11 x64 için tek EXE. GoodbyeDPI / WinDivert motoru programın içindedir.

## Kullanım

1. Eski Discord Bridge sürümünden tepsi menüsündeki “Programdan Çık” ile çıkın.
2. `DiscordBridge_4_6_Stable.exe` dosyasını açın.
3. Otomatik seçim veya Superonline + ExitLag profilini seçip Bağlantıyı Başlat'a basın.
4. Discord'u Aç düğmesiyle Discord'u açın.

X düğmesi programı sistem tepsisine gizler. Çıkmak için tepsi menüsünü kullanın. Ayarlar ve işlem günlükleri `%LOCALAPPDATA%\DiscordBridge` içinde tutulur.

## 4.6 değişiklikleri

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

ExitLag ayarları değiştirilmez. Discord kendiliğinden açılmaz. DNS değişikliğinden önce mevcut ayarlar yedeklenir. Superonline + ExitLag profili `-5 --native-frag --blacklist discord-domains.txt` kullanır. Grafik Discord'a özel değildir; bütün fiziksel ağ trafiğini gösterir.

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
