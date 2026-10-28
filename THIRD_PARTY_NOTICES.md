# Third-party components and provenance

Discord Bridge runs the bundled GoodbyeDPI executable as a separate process. These components are unmodified copies from the official `goodbyedpi-0.2.3rc3-2.zip` x86_64 directory.

Upstream release: https://github.com/ValdikSS/GoodbyeDPI/releases/tag/0.2.3rc3

Upstream ZIP: https://github.com/ValdikSS/GoodbyeDPI/releases/download/0.2.3rc3/goodbyedpi-0.2.3rc3-2.zip

Upstream ZIP SHA-256: `37f96b32d050dadcc930a639eba68e1ccd57ed5c04a5f77dfca908f01905a4c5`

| Component | SHA-256 |
| --- | --- |
| goodbyedpi.exe | `8d412b094bb9c137ff25ba9a794d1122ecc84bb776debff6c249723a13cc31cd` |
| WinDivert.dll | `6110bfa44667405179c3e15e12af1b62037e447ed59b054b19042032995e6c7e` |
| WinDivert64.sys | `e69b5ba3f0cd6cfb2983e442636e7f0b342b61b15264b0328317d4559c82cf50` |

## GoodbyeDPI

Author: ValdikSS and contributors. License: Apache-2.0, reproduced in `GOODBYEDPI_APACHE_2_LICENSE.txt`.

Corresponding source is supplied in `third_party/GoodbyeDPI-0.2.3rc3-source.zip`, obtained from the upstream tag: https://github.com/ValdikSS/GoodbyeDPI/tree/0.2.3rc3

## WinDivert

Author: basil00 / ReQrypt and contributors. License: LGPL-3.0, reproduced in `WINDIVERT_LGPL_3_LICENSE.txt`; the corresponding source archive includes the upstream COPYING/COPYING.LESSER texts and build instructions.

Corresponding source is supplied in `third_party/WinDivert-2.2.0-source.zip`, obtained from: https://github.com/basil00/WinDivert/tree/v2.2.0

The WinDivert DLL matches the official v2.2.0 x64 DLL. The driver is the signed variant distributed with the official GoodbyeDPI release; its full hash is recorded above. Original third-party copyright, license and source materials remain included. The DLL and driver are extracted as separate files next to the GoodbyeDPI executable; the complete wrapper source and offline rebuild script are also supplied.

No component is represented as authored by Discord Bridge. No affiliation with Discord, GoodbyeDPI or WinDivert is implied.
