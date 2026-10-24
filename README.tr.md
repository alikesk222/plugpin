# plugpin

plugpin, yapay zeka kod ajanlarının eklentilerini, hook'larını ve MCP
yapılandırmasını denetleyen salt okunur bir araçtır. Tek bir statik Go
ikilisidir. Varsayılan tarama çevrimdışıdır. `git` çalıştırmaz, ajanları
çalıştırmaz ve okuduğu kodu çalıştırmaz.

v0.1.0 kaynak koddaki sürüm dizesidir. Yayın etiketi henüz yoktur.

Yalnızca size ait olan veya incelemeye açıkça yetkili olduğunuz makinelerde
kullanın. Aracı güvenlik açıklarını istismar etmek veya yapılandırmayı
değiştirmek için kullanmayın. Ayrıntı:
[RESPONSIBLE_USE.md](RESPONSIBLE_USE.md).

English: [README.md](README.md).

## Ne denetler

| Kural | Önem | Anlamı |
| --- | --- | --- |
| PP001 | critical | Çözülen HEAD, sabitlenmiş SHA değildir |
| PP002 | critical | HEAD, adı 40 onaltılık karakter olan bir dala bağlıdır |
| PP003 | critical | Adı `FETCH_HEAD` olan bir dal vardır |
| PP004 | high | Ajan sürümü düzeltme tabanının altındadır |
| PP005 | info | Kurulum kaydındaki SHA, HEAD ile uyuşmaz |
| PP010 / PP011 | medium | Hook temel kayda göre yenidir veya komut özeti değişmiştir |
| PP020 / PP021 | medium | MCP sunucusu yenidir veya komut, argüman, taşıma ya da URL değişmiştir |
| PP040 | high | İstemci düzeltmesi olmayan bir ajanda otomatik güncelleme açıktır |

Bu sürümdaki tabanlar: Claude Code `2.1.179` (GHSA-pq7j-f95f-qfcv), Codex
`0.146.0`. Copilot CLI için istemci düzeltmesi yoktur. Gemini CLI için
düzeltme yoktur; v0.1 kurulumu tespit eder ve bir checkout varsa git
denetimlerini yine çalıştırır. Ayrıntı [docs/rules.md](docs/rules.md) ve
[docs/agents.md](docs/agents.md) içindedir.

## Kurulum

Etiketli yayın olmadığı için kaynaktan kurun. Go 1.27.3 veya daha yeni:

```sh
git clone https://github.com/alikesk222/plugpin
cd plugpin
go build -trimpath -ldflags '-s -w -X github.com/alikesk222/plugpin/internal/cli.Version=0.1.0' -o plugpin ./cmd/plugpin
```

`CGO_ENABLED=0` statik ikili üretir. Bir etiket yayınlandıktan sonra
`go install github.com/alikesk222/plugpin/cmd/plugpin@latest` o yayını izler.
Bugün izlemez.

## Kullanım

```sh
plugpin scan
plugpin scan --format json -o report.json
plugpin scan --format sarif -o plugpin.sarif
plugpin scan --redact-paths --fail-on high
plugpin baseline --approve -o baseline.json
plugpin diff --baseline baseline.json
plugpin rules
plugpin version
```

| Çıkış | Anlamı |
| --- | --- |
| 0 | `--fail-on` eşiğinde veya üstünde bulgu yok (varsayılan `medium`) |
| 1 | Eşikte veya üstünde bir bulgu var |
| 2 | Kullanım veya çalışma hatası |

`--exec-version` varsayılan olarak kapalıdır. Süreç başlatan tek bayraktır ve
yalnızca kurulu bir ajan ikilisine `--version` geçirir. `--ignore` virgülle
ayrılmış kural kimlikleri alır (`PP004,PP005`).

## Örnek

HEAD'i, sabitlenmiş SHA ile adlandırılmış bir dala bağlı olan bir fikstür.
`--redact-paths` açıktır:

```
plugpin 0.1.0
schema: plugpin.report/v1
scanned_at: 2026-10-24T16:00:00Z
host: linux/amd64

Agents
  claude-code    2.1.281          ok               floor 2.1.179
  codex          unknown          not-installed    floor 0.146.0
  copilot-cli    unknown          not-installed    floor none
  gemini-cli     unknown          not-installed    floor none

Plugins
  claude-code  review-kit@acme-market
    git_dir: ~/.claude/plugins/marketplaces/acme-market
    pinned: 1d0697f6f1ca622527e0cd464df0e418052a4b77  head: 90d2c48fb1bd829a184a8880af08f5ca57f63b20  attached:refs/heads/1d0697f6f1ca622527e0cd464df0e418052a4b77  auto_update: true

Hooks
  claude-code  user  SessionStart  ~/.claude/settings.json
    echo ok

MCP servers
  (none)

Findings
  [critical] PP001  claude-code  review-kit@acme-market
    Resolved HEAD does not match the pinned SHA
    remediation: Disable the plugin, turn off auto-update, confirm the agent is on a fixed build, and reinstall so HEAD is detached at the pinned commit.
  [critical] PP002  claude-code  review-kit@acme-market
    HEAD is attached to a branch whose name is a 40-hex SHA
    remediation: Disable the plugin and reinstall from a verified commit. Do not check out a branch whose name is a 40-hex SHA.
  [info] PP005  claude-code  review-kit@acme-market
    Install record SHA does not match the git state on disk
    remediation: Compare the marketplace pin and the on-disk HEAD. A stale record alone is not a critical finding.

Summary
  critical=2 high=0 medium=0 info=1
```

JSON ve SARIF alanları [docs/formats.md](docs/formats.md) içindedir.

## Kapsam dışı

plugpin eklenti kaynağında zararlı yazılım aramaz, ajanın önüne vekil koymaz,
yapılandırmayı onarmaz ve uzaktaki bir filoyu taramaz. Ajanı güncellemenin
yerini tutmaz. Temiz bir rapor yalnızca [docs/agents.md](docs/agents.md)
içinde sayılan dosyaları kapsar. Kamuya açık belgelerde doğrulanmayan yollar
doğrulanmamış kalır ve yoksa atlanır.

## Karşılaştırma

v0.1 kapsamı çıkarılırken, diskteki eklenti HEAD'inin sabitlenmiş SHA'ya eşit
olduğunu birden fazla ajanda paketleyen başka bir araç bulunmadı. Hook ve MCP
temel kayıt sapması yeni bir fikir değildir. Yakın projeler ve onların odağı:

| Proje | Odak |
| --- | --- |
| [snyk/agent-scan](https://github.com/snyk/agent-scan) | Ajan becerisi ve MCP risk taraması |
| [HTS-ASPM/skills-scanner](https://github.com/HTS-ASPM/skills-scanner) | Beceri paketi taraması |
| [smakarim/tripwire](https://github.com/smakarim/tripwire) | Hook ve MCP tuzakları |
| [trailofbits/mcp-context-protector](https://github.com/trailofbits/mcp-context-protector) | Çalışma zamanı MCP koruması |
| [NVIDIA/SkillSpector](https://github.com/NVIDIA/SkillSpector) | Beceri incelemesi |
| [cisco-ai-defense/skill-scanner](https://github.com/cisco-ai-defense/skill-scanner) | Beceri taraması |
| [Uky0Yang/repo-trust-scan](https://github.com/Uky0Yang/repo-trust-scan) | Depo güven denetimi |
| Automater Plugin4Shell tatbikatı | Paketlenmiş bir denetçi değil, bir bash tatbikatı |
| [xai-org/plugin-marketplace](https://github.com/xai-org/plugin-marketplace) `validate-catalog.py` | Yayımcı tarafında katalog doğrulaması |

## Belgeler

- [docs/architecture.md](docs/architecture.md)
- [docs/agents.md](docs/agents.md)
- [docs/rules.md](docs/rules.md)
- [docs/formats.md](docs/formats.md)
- [docs/threat-model.md](docs/threat-model.md)
- [CONTRIBUTING.md](CONTRIBUTING.md)
- [SECURITY.md](SECURITY.md)

## Yol haritası

v0.2 adayları: `--verify-remote` (isteğe bağlı kalır; varsayılan tarama
çevrimdışı kalır), yolu doğrulandıktan sonra Windsurf MCP, Antigravity CLI
araştırması, bir IOC dosyası, filo kipi ve Sigma çıktısı. İmzalı yayınlar ve
Homebrew tap'i, bakımcı isteyene kadar beklemektedir. `.goreleaser.yaml`
içinde tap bölümü yoktur.

## Lisans

[MIT](LICENSE). Telif hakkı 2026 Ali Kesik.
